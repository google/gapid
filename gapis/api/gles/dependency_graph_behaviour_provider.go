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
	"github.com/google/gapid/gapis/api"
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
	level   GLint
	layer   GLint
}

func (k textureDataKey) Parent() dependencygraph.StateKey { return nil }

type textureSizeKey struct {
	texture *Texture
	id      TextureId // For debugging, as 0 is not unique identifier.
	layer   GLint
	level   GLint
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

// GetBehaviourForAtom returns state reads/writes that the given command
// performs.
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
	ctx context.Context, s *api.GlobalState, id api.CmdID, cmd api.Cmd, g *dependencygraph.DependencyGraph) dependencygraph.AtomBehaviour {
	b := dependencygraph.AtomBehaviour{}
	c := GetContext(s, cmd.Thread())
	if c != nil && c.Info.Initialized {
		_, isEglSwapBuffers := cmd.(*EglSwapBuffers)
		// TODO: We should also be considering eglSwapBuffersWithDamageKHR here
		// too, but this is nearly exculsively used by the Android framework,
		// which also loves to do partial framebuffer updates. Unfortunately
		// we do not currently know whether the framebuffer is invalidated
		// between calls to eglSwapBuffersWithDamageKHR as the OS now uses
		// the EGL_EXT_buffer_age extension, which we do not track. For now,
		// assume that eglSwapBuffersWithDamageKHR calls are coming from the
		// framework, and that the framebuffer is reused between calls.
		// BUG: https://github.com/google/gapid/issues/846.
		if isEglSwapBuffers {
			// Get default renderbuffers
			fb := c.Objects.Framebuffers.Get(0)
			color := fb.ColorAttachments.Get(0).Renderbuffer
			depth := fb.DepthAttachment.Renderbuffer
			stencil := fb.StencilAttachment.Renderbuffer
			if !c.Info.PreserveBuffersOnSwap {
				b.Write(g, renderbufferDataKey{color})
			}
			b.Write(g, renderbufferDataKey{depth})
			b.Write(g, renderbufferDataKey{stencil})
		} else if cmd.CmdFlags(ctx, id, s).IsDrawCall() {
			b.Read(g, uniformGroupKey{c.Bound.Program})
			b.Read(g, vertexAttribGroupKey{c.Bound.VertexArray})
			for _, stateKey := range getAllUsedTextureData(ctx, cmd, id, s, c) {
				b.Read(g, stateKey)
			}
			c.Bound.DrawFramebuffer.ForEachAttachment(func(name GLenum, att FramebufferAttachment) {
				data, size := att.dataAndSize(g, c)
				b.Read(g, size)
				b.Modify(g, data)
			})
			// TODO: Write transform feedback buffers.
		} else if cmd.CmdFlags(ctx, id, s).IsClear() {
			switch cmd := cmd.(type) {
			case *GlClearBufferfi:
				clearBuffer(g, &b, cmd.Buffer, cmd.Drawbuffer, c)
			case *GlClearBufferfv:
				clearBuffer(g, &b, cmd.Buffer, cmd.Drawbuffer, c)
			case *GlClearBufferiv:
				clearBuffer(g, &b, cmd.Buffer, cmd.Drawbuffer, c)
			case *GlClearBufferuiv:
				clearBuffer(g, &b, cmd.Buffer, cmd.Drawbuffer, c)
			case *GlClear:
				if (cmd.Mask & GLbitfield_GL_COLOR_BUFFER_BIT) != 0 {
					for i, _ := range c.Bound.DrawFramebuffer.ColorAttachments.Range() {
						clearBuffer(g, &b, GLenum_GL_COLOR, i, c)
					}
				}
				if (cmd.Mask & GLbitfield_GL_DEPTH_BUFFER_BIT) != 0 {
					clearBuffer(g, &b, GLenum_GL_DEPTH, 0, c)
				}
				if (cmd.Mask & GLbitfield_GL_STENCIL_BUFFER_BIT) != 0 {
					clearBuffer(g, &b, GLenum_GL_STENCIL, 0, c)
				}
			default:
				log.E(ctx, "Unknown clear command: %v", cmd)
			}
		} else {
			switch cmd := cmd.(type) {
			case *GlCopyImageSubData:
				// TODO: This assumes whole-image copy.  Handle sub-range copies.
				if cmd.SrcTarget == GLenum_GL_RENDERBUFFER {
					b.Read(g, renderbufferDataKey{c.Objects.Renderbuffers.Get(RenderbufferId(cmd.SrcName))})
				} else {
					data, size := c.Objects.Textures.Get(TextureId(cmd.SrcName)).dataAndSize(cmd.SrcLevel, 0)
					b.Read(g, data)
					b.Read(g, size)
				}
				if cmd.DstTarget == GLenum_GL_RENDERBUFFER {
					b.Write(g,
						renderbufferDataKey{c.Objects.Renderbuffers.Get(RenderbufferId(cmd.DstName))})
				} else {
					data, size := c.Objects.Textures.Get(TextureId(cmd.DstName)).dataAndSize(cmd.DstLevel, 0)
					b.Write(g, data)
					b.Write(g, size)
				}
			case *GlFramebufferTexture2D:
				b.Read(g, textureSizeKey{c.Objects.Textures.Get(cmd.Texture), cmd.Texture, cmd.Level, 0})
				b.KeepAlive = true // Changes untracked state
			case *GlCompressedTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, cmd, id, s, c.Bound.TextureUnit, cmd.Target, cmd.Level)
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlCompressedTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, cmd, id, s, c.Bound.TextureUnit, cmd.Target, cmd.Level)
				b.Modify(g, texData)
			case *GlTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, cmd, id, s, c.Bound.TextureUnit, cmd.Target, cmd.Level)
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, cmd, id, s, c.Bound.TextureUnit, cmd.Target, cmd.Level)
				b.Modify(g, texData)
			case *GlGenerateMipmap:
				tex, err := subGetBoundTextureOrErrorInvalidEnum(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, cmd.Target)
				if err != nil {
					log.E(ctx, "Can not find bound texture %v", cmd.Target)
				}
				for levelIndex, level := range tex.Levels.Range() {
					for layerIndex := range level.Layers.Range() {
						data, size := tex.dataAndSize(levelIndex, layerIndex)
						if levelIndex == 0 {
							b.Read(g, data)
							b.Read(g, size)
						} else {
							b.Read(g, size)
							b.Write(g, data)
						}
					}
				}
			case *GlUniform1fv:
				b.Write(g, uniformKey{c.Bound.Program, cmd.Location, cmd.Count})
			case *GlUniform2fv:
				b.Write(g, uniformKey{c.Bound.Program, cmd.Location, cmd.Count})
			case *GlUniform3fv:
				b.Write(g, uniformKey{c.Bound.Program, cmd.Location, cmd.Count})
			case *GlUniform4fv:
				b.Write(g, uniformKey{c.Bound.Program, cmd.Location, cmd.Count})
			case *GlUniformMatrix4fv:
				b.Write(g, uniformKey{c.Bound.Program, cmd.Location, cmd.Count})
			case *GlVertexAttribPointer:
				b.Write(g, vertexAttribKey{c.Bound.VertexArray, cmd.Location})
			default:
				// Force all unhandled atoms to be kept alive.
				b.KeepAlive = true
			}
		}
	} else /* c == nil */ {
		b.KeepAlive = true
	}
	if err := cmd.Mutate(ctx, id, s, nil /* builder */); err != nil {
		log.W(ctx, "Command %v %v: %v", id, cmd, err)
		return dependencygraph.AtomBehaviour{Aborted: true}
	}
	return b
}

func clearBuffer(g *dependencygraph.DependencyGraph, b *dependencygraph.AtomBehaviour, buffer GLenum, index GLint, c *Context) {
	var data, size dependencygraph.StateKey
	switch buffer {
	case GLenum_GL_COLOR:
		data, size = c.Bound.DrawFramebuffer.ColorAttachments.Get(index).dataAndSize(g, c)
	case GLenum_GL_DEPTH:
		data, size = c.Bound.DrawFramebuffer.DepthAttachment.dataAndSize(g, c)
	case GLenum_GL_STENCIL:
		data, size = c.Bound.DrawFramebuffer.StencilAttachment.dataAndSize(g, c)
	case GLenum_GL_DEPTH_STENCIL:
		data, size = c.Bound.DrawFramebuffer.DepthAttachment.dataAndSize(g, c)
		b.Read(g, size)
		b.Write(g, data)
		data, size = c.Bound.DrawFramebuffer.StencilAttachment.dataAndSize(g, c)
	}
	b.Read(g, size)
	b.Write(g, data)
}

func getAllUsedTextureData(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, c *Context) (stateKeys []dependencygraph.StateKey) {
	// Look for samplers used by the current program.
	if prog := c.Bound.Program; prog != nil {
		for _, activeUniform := range prog.ActiveUniforms.Range() {
			// Optimization - skip the two most common types which we know are not samplers.
			if activeUniform.Type != GLenum_GL_FLOAT_VEC4 && activeUniform.Type != GLenum_GL_FLOAT_MAT4 {
				target, _ := subGetTextureTargetFromSamplerType(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, activeUniform.Type)
				if target == GLenum_GL_NONE {
					continue // Not a sampler type
				}
				for i := 0; i < int(activeUniform.ArraySize); i++ {
					uniform := prog.Uniforms.Get(activeUniform.Location + UniformLocation(i))
					units := AsU32ˢ(uniform.Value, s.MemoryLayout).MustRead(ctx, cmd, s, nil)
					if len(units) == 0 {
						units = []uint32{0} // The uniform was not set, so use default value.
					}
					for _, unit := range units {
						if tu := c.Objects.TextureUnits.Get(TextureUnitId(unit)); tu != nil {
							tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, tu, target)
							if tex != nil && err == nil {
								for lvl, level := range tex.Levels.Range() {
									for lyr := range level.Layers.Range() {
										texData, _ := tex.dataAndSize(lvl, lyr)
										stateKeys = append(stateKeys, texData)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

func getTextureDataAndSize(
	ctx context.Context,
	cmd api.Cmd,
	id api.CmdID,
	s *api.GlobalState,
	unit *TextureUnit,
	target GLenum,
	level GLint) (dependencygraph.StateKey, dependencygraph.StateKey) {

	tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, unit, target)
	if tex == nil || err != nil {
		log.E(ctx, "Can not find texture %v in unit %v", target, unit)
		return nil, nil
	}
	layer := cubemapFaceToLayer(target)
	return tex.dataAndSize(level, layer)
}

func (tex *Texture) dataAndSize(level, layer GLint) (dependencygraph.StateKey, dependencygraph.StateKey) {
	if tex.EGLImage != nil {
		return eglImageDataKey{tex.EGLImage}, eglImageSizeKey{tex.EGLImage}
	} else {
		return textureDataKey{tex, tex.ID, level, layer}, textureSizeKey{tex, tex.ID, layer, level}
	}
}

func (att FramebufferAttachment) dataAndSize(g *dependencygraph.DependencyGraph, c *Context) (dataKey dependencygraph.StateKey, sizeKey dependencygraph.StateKey) {
	if att.Type == GLenum_GL_RENDERBUFFER {
		rb := att.Renderbuffer
		if rb != nil && rb.InternalFormat != GLenum_GL_NONE {
			scissor := c.Pixel.Scissor
			fullBox := Rect{Width: rb.Width, Height: rb.Height}
			if scissor.Test == GLboolean_GL_TRUE && scissor.Box != fullBox {
				dataKey, sizeKey = renderbufferSubDataKey{rb, scissor.Box}, nil
			} else {
				dataKey, sizeKey = renderbufferDataKey{rb}, nil
			}
		}
	}
	if att.Type == GLenum_GL_TEXTURE {
		tex := att.Texture
		if tex != nil {
			// TODO: We should handle scissor here as well.
			dataKey, sizeKey = tex.dataAndSize(att.TextureLevel, att.TextureLayer)
		}
	}
	if dataKey != nil {
		g.SetRoot(dataKey)
	}
	return
}
