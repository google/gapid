// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gles

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
)

var dependencyGraphBuildCounter = benchmark.GlobalCounters.Duration("dependencyGraph.build")

// DependencyGraph represents dependencies between atoms.
// For each atom, we want to know what other atoms it depends on.
// Traversing of this graph allows us to find the set of live atoms.
//
// We could just store list of dependencies per each atom,
// however this is inefficient since draw calls tend to depend
// on large number of other atoms (almost the whole GLES state).
// We solve this problem by inserting nodes for state into the
// graph - each atom reads from state nodes and writes to others.
// The trick is making the state hierarchical, so one atom can
// depend on large subset of the state with a single reference.
//
// The graph keeps alternating between atom and state nodes:
//
//      Atom1
//     /  |  \    (writes of Atom1)
//   s01 s10 s11
//     \  |   |   (reads of Atom2)
//     Atom2  |
//        |   |   (writes of Atom2)
//       s10  |
//         \ /    (reads of Atom3)
//        Atom3
//
type DependencyGraph struct {
	atoms      []atom.Atom           // Atom list which this graph was build for.
	behaviours []AtomBehaviour       // State reads/writes for each atom (graph edges).
	roots      map[StateAddress]bool // State to mark live at requested atoms.
	addressMap addressMapping        // Remap state keys to integers for performance.
}

type AtomBehaviour struct {
	Read      []StateAddress // State read by an atom.
	Modify    []StateAddress // State read and written by an atom.
	Write     []StateAddress // State written by an atom.
	KeepAlive bool           // Force the atom to be live.
	Aborted   bool           // Mutation of this command aborts.
}

type addressMapping struct {
	address map[stateKey]StateAddress
	key     map[StateAddress]stateKey
	parent  map[StateAddress]StateAddress
}

func (g *DependencyGraph) Print(ctx context.Context, b *AtomBehaviour) {
	for _, read := range b.Read {
		key := g.addressMap.key[read]
		log.I(ctx, " - read [%v]%T%+v", read, key, key)
	}
	for _, modify := range b.Modify {
		key := g.addressMap.key[modify]
		log.I(ctx, " - modify [%v]%T%+v", modify, key, key)
	}
	for _, write := range b.Write {
		key := g.addressMap.key[write]
		log.I(ctx, " - write [%v]%T%+v", write, key, key)
	}
	if b.Aborted {
		log.I(ctx, " - aborted")
	}
}

// State key uniquely represents part of the GL state.
// Think of it as memory range (which stores the state data).
type stateKey interface {
	// Parent returns enclosing state (and this state is strict subset of it).
	// This allows efficient implementation of operations which access a lot state.
	Parent() stateKey
}

type StateAddress uint32

const nullStateAddress = StateAddress(0)

func GetDependencyGraph(ctx context.Context) (*DependencyGraph, error) {
	r, err := database.Build(ctx, &DependencyGraphResolvable{Capture: capture.Get(ctx)})
	if err != nil {
		return nil, fmt.Errorf("Could not calculate dependency graph: %v", err)
	}
	return r.(*DependencyGraph), nil
}

func (r *DependencyGraphResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}

	g := &DependencyGraph{
		atoms:      c.Atoms,
		behaviours: make([]AtomBehaviour, len(c.Atoms)),
		roots:      map[StateAddress]bool{},
		addressMap: addressMapping{
			address: map[stateKey]StateAddress{nil: nullStateAddress},
			key:     map[StateAddress]stateKey{nullStateAddress: nil},
			parent:  map[StateAddress]StateAddress{nullStateAddress: nullStateAddress},
		},
	}

	s := c.NewState()
	t0 := dependencyGraphBuildCounter.Start()
	for i, a := range g.atoms {
		g.behaviours[i] = g.getBehaviour(ctx, s, atom.ID(i), a)
	}
	dependencyGraphBuildCounter.Stop(t0)
	return g, nil
}

func (m *addressMapping) addressOf(state stateKey) StateAddress {
	if a, ok := m.address[state]; ok {
		return a
	}
	address := StateAddress(len(m.address))
	m.address[state] = address
	m.key[address] = state
	m.parent[address] = m.addressOf(state.Parent())
	return address
}

func (b *AtomBehaviour) read(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Read = append(b.Read, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) modify(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Modify = append(b.Modify, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) write(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Write = append(b.Write, g.addressMap.addressOf(state))
	}
}

type uniformKey struct {
	program  *Program
	location UniformLocation
	count    GLsizei
}

func (k uniformKey) Parent() stateKey { return uniformGroupKey{k.program} }

type uniformGroupKey struct {
	program *Program
}

func (k uniformGroupKey) Parent() stateKey { return nil }

type vertexAttribKey struct {
	vertexArray *VertexArray
	location    AttributeLocation
}

func (k vertexAttribKey) Parent() stateKey { return vertexAttribGroupKey{k.vertexArray} }

type vertexAttribGroupKey struct {
	vertexArray *VertexArray
}

func (k vertexAttribGroupKey) Parent() stateKey { return nil }

type renderbufferDataKey struct {
	renderbuffer *Renderbuffer
}

func (k renderbufferDataKey) Parent() stateKey { return nil }

type renderbufferSubDataKey struct {
	renderbuffer *Renderbuffer
	region       Rect
}

func (k renderbufferSubDataKey) Parent() stateKey { return renderbufferDataKey{k.renderbuffer} }

type textureDataKey struct {
	texture *Texture
	id      TextureId // For debugging, as 0 is not unique identifier.
}

func (k textureDataKey) Parent() stateKey { return nil }

type textureSizeKey struct {
	texture *Texture
	id      TextureId // For debugging, as 0 is not unique identifier.
}

func (k textureSizeKey) Parent() stateKey { return nil }

type eglImageDataKey struct {
	address GLeglImageOES
}

func (k eglImageDataKey) Parent() stateKey { return nil }

type eglImageSizeKey struct {
	address GLeglImageOES
}

func (k eglImageSizeKey) Parent() stateKey { return nil }

// getBehaviour returns state reads/writes that the given atom performs.
//
// Writes: Write dependencies keep atoms alive. Each atom must correctly report
// all its writes or it must set the keep-alive flag. If a write is missing
// then the liveness analysis will remove the atom since it seems unneeded.
// It is fine to omit related/overloaded commands that write to the same state
// as long as they are marked as keep-alive.
//
// Reads: For each state write, all commands that could possibly read it must be
// implemented. This makes it more difficult to do only partial implementations.
// It is fine to overestimate reads, or to read parent state (i.e. superset).
//
func (g *DependencyGraph) getBehaviour(ctx context.Context, s *gfxapi.State, id atom.ID, a atom.Atom) AtomBehaviour {
	b := AtomBehaviour{}
	c := GetContext(s)
	if c != nil && c.Info.Initialized {
		_, isEglSwapBuffers := a.(*EglSwapBuffers)
		_, isEglSwapBuffersWithDamageKHR := a.(*EglSwapBuffersWithDamageKHR)
		if isEglSwapBuffers || isEglSwapBuffersWithDamageKHR {
			// Get default renderbuffers
			fb := c.Objects.Framebuffers[0]
			color := fb.ColorAttachments[0].Renderbuffer
			depth := fb.DepthAttachment.Renderbuffer
			stencil := fb.StencilAttachment.Renderbuffer
			if !c.Info.PreserveBuffersOnSwap {
				b.write(g, renderbufferDataKey{color})
			}
			b.write(g, renderbufferDataKey{depth})
			b.write(g, renderbufferDataKey{stencil})
		} else if a.AtomFlags().IsDrawCall() {
			b.read(g, uniformGroupKey{c.Bound.Program})
			b.read(g, vertexAttribGroupKey{c.Bound.VertexArray})
			for _, stateKey := range getAllUsedTextureData(ctx, a, s, c) {
				b.read(g, stateKey)
			}
			fb := c.Bound.DrawFramebuffer
			for _, att := range fb.ColorAttachments {
				b.modify(g, getAttachmentData(g, c, att))
			}
			b.modify(g, getAttachmentData(g, c, fb.DepthAttachment))
			b.modify(g, getAttachmentData(g, c, fb.StencilAttachment))
			// TODO: Write transform feedback buffers.
		} else {
			switch a := a.(type) {
			case *GlClear:
				fb := c.Bound.DrawFramebuffer
				if (a.Mask & GLbitfield_GL_COLOR_BUFFER_BIT) != 0 {
					for _, att := range fb.ColorAttachments {
						b.read(g, getAttachmentSize(g, c, att))
						b.write(g, getAttachmentData(g, c, att))
					}
				}
				if (a.Mask & GLbitfield_GL_DEPTH_BUFFER_BIT) != 0 {
					b.read(g, getAttachmentSize(g, c, fb.DepthAttachment))
					b.write(g, getAttachmentData(g, c, fb.DepthAttachment))
				}
				if (a.Mask & GLbitfield_GL_STENCIL_BUFFER_BIT) != 0 {
					b.read(g, getAttachmentSize(g, c, fb.StencilAttachment))
					b.write(g, getAttachmentData(g, c, fb.StencilAttachment))
				}
			case *GlBindFramebuffer:
				// It may act as "resolve" of EGLImage - i.e. save the content in one context.
				b.KeepAlive = true
			case *GlFramebufferTexture2D:
				b.read(g, textureSizeKey{c.Objects.Shared.Textures[a.Texture], a.Texture})
				b.KeepAlive = true // Changes untracked state
			case *GlBindTexture:
				// It may act as "load" of EGLImage - i.e. load the content in other context.
				b.KeepAlive = true
			case *GlCompressedTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.modify(g, texData)
				b.write(g, texSize)
			case *GlCompressedTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.modify(g, texData)
			case *GlTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.modify(g, texData)
				b.write(g, texSize)
			case *GlTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.modify(g, texData)
			case *GlUniform1fv:
				b.write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform2fv:
				b.write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform3fv:
				b.write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform4fv:
				b.write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniformMatrix4fv:
				b.write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlVertexAttribPointer:
				b.write(g, vertexAttribKey{c.Bound.VertexArray, a.Location})
			default:
				// Force all unhandled atoms to be kept alive.
				b.KeepAlive = true
			}
		}
	} else /* c == nil */ {
		b.KeepAlive = true
	}
	if err := a.Mutate(ctx, s, nil /* builder */); err != nil {
		log.W(ctx, "Atom %v %v: %v", id, a, err)
		return AtomBehaviour{Aborted: true}
	}
	return b
}

func getAllUsedTextureData(ctx context.Context, a atom.Atom, s *gfxapi.State, c *Context) (stateKeys []stateKey) {
	// Look for samplers used by the current program.
	if prog := c.Bound.Program; prog != nil {
		for _, activeUniform := range prog.ActiveUniforms {
			// Optimization - skip the two most common types which we know are not samplers.
			if activeUniform.Type != GLenum_GL_FLOAT_VEC4 && activeUniform.Type != GLenum_GL_FLOAT_MAT4 {
				target, _ := subGetTextureTargetFromSamplerType(ctx, a, nil, s, GetState(s), nil, activeUniform.Type)
				if target == GLenum_GL_NONE {
					continue // Not a sampler type
				}
				for i := 0; i < int(activeUniform.ArraySize); i++ {
					uniform := prog.Uniforms[activeUniform.Location+UniformLocation(i)]
					units := AsU32ˢ(uniform.Value, s.MemoryLayout).Read(ctx, a, s, nil)
					if len(units) == 0 {
						units = []uint32{0} // The uniform was not set, so use default value.
					}
					for _, unit := range units {
						if tu := c.TextureUnits[TextureUnitId(unit)]; tu != nil {
							texData, _ := getTextureDataAndSize(ctx, a, s, tu, target)
							stateKeys = append(stateKeys, texData)
						}
					}
				}
			}
		}
	}
	return
}

func getTextureDataAndSize(ctx context.Context, a atom.Atom, s *gfxapi.State, unit *TextureUnit, target GLenum) (stateKey, stateKey) {
	tex, err := subGetBoundTextureForUnit(ctx, a, nil, s, GetState(s), nil, unit, target)
	if tex == nil || err != nil {
		log.E(ctx, "Can not find texture %v in unit %v", target, unit)
		return nil, nil
	}
	if !tex.EGLImage.IsNullptr() {
		return eglImageDataKey{tex.EGLImage}, eglImageSizeKey{tex.EGLImage}
	} else {
		return textureDataKey{tex, tex.ID}, textureSizeKey{tex, tex.ID}
	}
}

func getAttachmentData(g *DependencyGraph, c *Context, att FramebufferAttachment) (key stateKey) {
	if att.Type == GLenum_GL_RENDERBUFFER {
		rb := att.Renderbuffer
		if rb != nil && rb.InternalFormat != GLenum_GL_NONE {
			scissor := c.Pixel.Scissor
			fullBox := Rect{Width: rb.Width, Height: rb.Height}
			if scissor.Test == GLboolean_GL_TRUE && scissor.Box != fullBox {
				key = renderbufferSubDataKey{rb, scissor.Box}
			} else {
				key = renderbufferDataKey{rb}
			}
		}
	}
	if att.Type == GLenum_GL_TEXTURE {
		tex := att.Texture
		if tex != nil {
			// TODO: We should handle scissor here as well.
			if !tex.EGLImage.IsNullptr() {
				key = eglImageDataKey{tex.EGLImage}
			} else {
				key = textureDataKey{tex, tex.ID}
			}
		}
	}
	if key != nil {
		g.roots[g.addressMap.addressOf(key)] = true
	}
	return
}

func getAttachmentSize(g *DependencyGraph, c *Context, att FramebufferAttachment) (key stateKey) {
	if att.Type == GLenum_GL_TEXTURE {
		tex := att.Texture
		if tex != nil {
			if !tex.EGLImage.IsNullptr() {
				key = eglImageSizeKey{tex.EGLImage}
			} else {
				key = textureSizeKey{tex, tex.ID}
			}
		}
	}
	if key != nil {
		g.roots[g.addressMap.addressOf(key)] = true
	}
	return
}
