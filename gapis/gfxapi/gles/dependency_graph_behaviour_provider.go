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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

type uniformKey struct {
	program  *Program
	location UniformLocation
	count    GLsizei
}

func (k uniformKey) Parent() dependencygraph.StateKey { return uniformGroupKey{k.program} }

type uniformGroupKey struct {
	program *Program
}

func (k uniformGroupKey) Parent() dependencygraph.StateKey { return nil }

type vertexAttribKey struct {
	vertexArray *VertexArray
	location    AttributeLocation
}

func (k vertexAttribKey) Parent() dependencygraph.StateKey {
	return vertexAttribGroupKey{k.vertexArray}
}

type vertexAttribGroupKey struct {
	vertexArray *VertexArray
}

func (k vertexAttribGroupKey) Parent() dependencygraph.StateKey { return nil }

type renderbufferDataKey struct {
	renderbuffer *Renderbuffer
}

func (k renderbufferDataKey) Parent() dependencygraph.StateKey { return nil }

type renderbufferSubDataKey struct {
	renderbuffer *Renderbuffer
	region       Rect
}

func (k renderbufferSubDataKey) Parent() dependencygraph.StateKey {
	return renderbufferDataKey{k.renderbuffer}
}

type textureDataKey struct {
	texture *Texture
	id      TextureId // For debugging, as 0 is not unique identifier.
}

func (k textureDataKey) Parent() dependencygraph.StateKey { return nil }

type textureSizeKey struct {
	texture *Texture
	id      TextureId // For debugging, as 0 is not unique identifier.
}

func (k textureSizeKey) Parent() dependencygraph.StateKey { return nil }

type eglImageDataKey struct {
	image *EGLImage
}

func (k eglImageDataKey) Parent() dependencygraph.StateKey { return nil }

type eglImageSizeKey struct {
	image *EGLImage
}

func (k eglImageSizeKey) Parent() dependencygraph.StateKey { return nil }

type GlesDependencyGraphBehaviourProvider struct {
}

func newGlesDependencyGraphBehaviourProvider() *GlesDependencyGraphBehaviourProvider {
	return &GlesDependencyGraphBehaviourProvider{}
}

// GetBehaviourForAtom returns state reads/writes that the given atom performs.
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
func (*GlesDependencyGraphBehaviourProvider) GetBehaviourForAtom(
	ctx context.Context, s *gfxapi.State, id atom.ID, g *dependencygraph.DependencyGraph, a atom.Atom) dependencygraph.AtomBehaviour {
	b := dependencygraph.AtomBehaviour{}
	c := GetContext(s, a.Thread())
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
				b.Write(g, renderbufferDataKey{color})
			}
			b.Write(g, renderbufferDataKey{depth})
			b.Write(g, renderbufferDataKey{stencil})
		} else if a.AtomFlags().IsDrawCall() {
			b.Read(g, uniformGroupKey{c.Bound.Program})
			b.Read(g, vertexAttribGroupKey{c.Bound.VertexArray})
			for _, stateKey := range getAllUsedTextureData(ctx, a, s, c) {
				b.Read(g, stateKey)
			}
			fb := c.Bound.DrawFramebuffer
			for _, att := range fb.ColorAttachments {
				b.Modify(g, getAttachmentData(g, c, att))
			}
			b.Modify(g, getAttachmentData(g, c, fb.DepthAttachment))
			b.Modify(g, getAttachmentData(g, c, fb.StencilAttachment))
			// TODO: Write transform feedback buffers.
		} else {
			switch a := a.(type) {
			case *GlClear:
				fb := c.Bound.DrawFramebuffer
				if (a.Mask & GLbitfield_GL_COLOR_BUFFER_BIT) != 0 {
					for _, att := range fb.ColorAttachments {
						b.Read(g, getAttachmentSize(g, c, att))
						b.Write(g, getAttachmentData(g, c, att))
					}
				}
				if (a.Mask & GLbitfield_GL_DEPTH_BUFFER_BIT) != 0 {
					b.Read(g, getAttachmentSize(g, c, fb.DepthAttachment))
					b.Write(g, getAttachmentData(g, c, fb.DepthAttachment))
				}
				if (a.Mask & GLbitfield_GL_STENCIL_BUFFER_BIT) != 0 {
					b.Read(g, getAttachmentSize(g, c, fb.StencilAttachment))
					b.Write(g, getAttachmentData(g, c, fb.StencilAttachment))
				}
			case *GlCopyImageSubData:
				// TODO: This assumes whole-image copy.  Handle sub-range copies.
				if a.SrcTarget == GLenum_GL_RENDERBUFFER {
					b.Read(g, renderbufferDataKey{c.Objects.Shared.Renderbuffers[RenderbufferId(a.SrcName)]})
				} else {
					data, size := c.Objects.Shared.Textures[TextureId(a.SrcName)].getTextureDataAndSize()
					b.Read(g, data)
					b.Read(g, size)
				}
				if a.DstTarget == GLenum_GL_RENDERBUFFER {
					b.Write(g,
						renderbufferDataKey{c.Objects.Shared.Renderbuffers[RenderbufferId(a.DstName)]})
				} else {
					data, size := c.Objects.Shared.Textures[TextureId(a.DstName)].getTextureDataAndSize()
					b.Write(g, data)
					b.Write(g, size)
				}
			case *GlFramebufferTexture2D:
				b.Read(g, textureSizeKey{c.Objects.Shared.Textures[a.Texture], a.Texture})
				b.KeepAlive = true // Changes untracked state
			case *GlCompressedTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlCompressedTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.Modify(g, texData)
			case *GlTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, a, s, c.Bound.TextureUnit, a.Target)
				b.Modify(g, texData)
			case *GlUniform1fv:
				b.Write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform2fv:
				b.Write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform3fv:
				b.Write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniform4fv:
				b.Write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlUniformMatrix4fv:
				b.Write(g, uniformKey{c.Bound.Program, a.Location, a.Count})
			case *GlVertexAttribPointer:
				b.Write(g, vertexAttribKey{c.Bound.VertexArray, a.Location})
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
		return dependencygraph.AtomBehaviour{Aborted: true}
	}
	return b
}

func getAllUsedTextureData(ctx context.Context, a atom.Atom, s *gfxapi.State, c *Context) (stateKeys []dependencygraph.StateKey) {
	// Look for samplers used by the current program.
	if prog := c.Bound.Program; prog != nil {
		for _, activeUniform := range prog.ActiveUniforms {
			// Optimization - skip the two most common types which we know are not samplers.
			if activeUniform.Type != GLenum_GL_FLOAT_VEC4 && activeUniform.Type != GLenum_GL_FLOAT_MAT4 {
				target, _ := subGetTextureTargetFromSamplerType(ctx, a, nil, s, GetState(s), a.Thread(), nil, activeUniform.Type)
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
						if tu := c.Objects.TextureUnits[TextureUnitId(unit)]; tu != nil {
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

func getTextureDataAndSize(ctx context.Context, a atom.Atom, s *gfxapi.State, unit *TextureUnit, target GLenum) (dependencygraph.StateKey, dependencygraph.StateKey) {
	tex, err := subGetBoundTextureForUnit(ctx, a, nil, s, GetState(s), a.Thread(), nil, unit, target)
	if tex == nil || err != nil {
		log.E(ctx, "Can not find texture %v in unit %v", target, unit)
		return nil, nil
	}
	return tex.getTextureDataAndSize()
}

func (tex *Texture) getTextureDataAndSize() (dependencygraph.StateKey, dependencygraph.StateKey) {
	if tex.EGLImage != nil {
		return eglImageDataKey{tex.EGLImage}, eglImageSizeKey{tex.EGLImage}
	} else {
		return textureDataKey{tex, tex.ID}, textureSizeKey{tex, tex.ID}
	}
}

func getAttachmentData(g *dependencygraph.DependencyGraph, c *Context, att FramebufferAttachment) (key dependencygraph.StateKey) {
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
			if tex.EGLImage != nil {
				key = eglImageDataKey{tex.EGLImage}
			} else {
				key = textureDataKey{tex, tex.ID}
			}
		}
	}
	if key != nil {
		g.SetRoot(key)
	}
	return
}

func getAttachmentSize(g *dependencygraph.DependencyGraph, c *Context, att FramebufferAttachment) (key dependencygraph.StateKey) {
	if att.Type == GLenum_GL_TEXTURE {
		tex := att.Texture
		if tex != nil {
			if tex.EGLImage != nil {
				key = eglImageSizeKey{tex.EGLImage}
			} else {
				key = textureSizeKey{tex, tex.ID}
			}
		}
	}
	if key != nil {
		g.SetRoot(key)
	}
	return
}
