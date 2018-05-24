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
	program  Programʳ
	location UniformLocation
	count    GLsizei
}

func (k uniformKey) Parent() dependencygraph.StateKey { return uniformGroupKey{k.program} }

type uniformGroupKey struct {
	program Programʳ
}

func (k uniformGroupKey) Parent() dependencygraph.StateKey { return nil }

type vertexAttribKey struct {
	vertexArray VertexArrayʳ
	location    AttributeLocation
}

func (k vertexAttribKey) Parent() dependencygraph.StateKey {
	return vertexAttribGroupKey{k.vertexArray}
}

type vertexAttribGroupKey struct {
	vertexArray VertexArrayʳ
}

func (k vertexAttribGroupKey) Parent() dependencygraph.StateKey { return nil }

type renderbufferDataKey struct {
	renderbuffer Renderbufferʳ
}

func (k renderbufferDataKey) Parent() dependencygraph.StateKey { return nil }

type renderbufferSubDataKey struct {
	renderbuffer     Renderbufferʳ
	regionX, regionY GLint
	regionW, regionH GLsizei
}

func (k renderbufferSubDataKey) Parent() dependencygraph.StateKey {
	return renderbufferDataKey{k.renderbuffer}
}

type textureDataKey struct {
	texture Textureʳ
	id      TextureId // For debugging, as 0 is not unique identifier.
	level   GLint
	layer   GLint
}

func (k textureDataKey) Parent() dependencygraph.StateKey {
	return textureDataGroupKey{k.texture, k.id}
}

// represents data for all levels and layers in texture
type textureDataGroupKey struct {
	texture Textureʳ
	id      TextureId // For debugging, as 0 is not unique identifier.
}

func (k textureDataGroupKey) Parent() dependencygraph.StateKey { return nil }

type textureSizeKey struct {
	texture Textureʳ
	id      TextureId // For debugging, as 0 is not unique identifier.
	layer   GLint
	level   GLint
}

func (k textureSizeKey) Parent() dependencygraph.StateKey { return nil }

type eglImageDataKey struct {
	image EGLImageʳ
}

func (k eglImageDataKey) Parent() dependencygraph.StateKey { return nil }

type eglImageSizeKey struct {
	image EGLImageʳ
}

func (k eglImageSizeKey) Parent() dependencygraph.StateKey { return nil }

type GlesDependencyGraphBehaviourProvider struct{}

func newGlesDependencyGraphBehaviourProvider() *GlesDependencyGraphBehaviourProvider {
	return &GlesDependencyGraphBehaviourProvider{}
}

// GetBehaviourForCommand returns state reads/writes that the given command
// performs.
//
// Writes: Write dependencies keep commands alive. Each command must correctly
// report all its writes or it must set the keep-alive flag. If a write is
// missing then the liveness analysis will remove the command since it seems
// unneeded.
// It is fine to omit related/overloaded commands that write to the same state
// as long as they are marked as keep-alive.
//
// Reads: For each state write, all commands that could possibly read it must be
// implemented. This makes it more difficult to do only partial implementations.
// It is fine to overestimate reads, or to read parent state (i.e. superset).
//
func (*GlesDependencyGraphBehaviourProvider) GetBehaviourForCommand(
	ctx context.Context, s *api.GlobalState, id api.CmdID, cmd api.Cmd, g *dependencygraph.DependencyGraph) dependencygraph.CmdBehaviour {
	b := dependencygraph.CmdBehaviour{}
	c := GetContext(s, cmd.Thread())
	if err := cmd.Mutate(ctx, id, s, nil /* builder */); err != nil {
		log.W(ctx, "Command %v %v: %v", id, cmd, err)
		return dependencygraph.CmdBehaviour{Aborted: true}
	}
	if !c.IsNil() && c.Other().Initialized() {
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
			fb := c.Objects().Framebuffers().Get(0)
			color := fb.ColorAttachments().Get(0).Renderbuffer()
			depth := fb.DepthAttachment().Renderbuffer()
			stencil := fb.StencilAttachment().Renderbuffer()
			if !c.Other().PreserveBuffersOnSwap() {
				b.Write(g, renderbufferDataKey{color})
			}
			b.Write(g, renderbufferDataKey{depth})
			b.Write(g, renderbufferDataKey{stencil})
		} else if cmd.CmdFlags(ctx, id, s).IsDrawCall() {
			b.Read(g, uniformGroupKey{c.Bound().Program()})
			b.Read(g, vertexAttribGroupKey{c.Bound().VertexArray()})
			for _, stateKey := range getAllUsedTextureData(ctx, cmd, id, s, c) {
				b.Read(g, stateKey)
			}
			c.Bound().DrawFramebuffer().ForEachAttachment(func(name GLenum, att FramebufferAttachment) {
				data, size := att.dataAndSize(g, c)
				b.Read(g, size)
				b.Modify(g, data)
			})
			// TODO: Write transform feedback buffers.
		} else if cmd.CmdFlags(ctx, id, s).IsClear() {
			switch cmd := cmd.(type) {
			case *GlClearBufferfi:
				clearBuffer(g, &b, cmd.Buffer(), cmd.Drawbuffer(), c)
			case *GlClearBufferfv:
				clearBuffer(g, &b, cmd.Buffer(), cmd.Drawbuffer(), c)
			case *GlClearBufferiv:
				clearBuffer(g, &b, cmd.Buffer(), cmd.Drawbuffer(), c)
			case *GlClearBufferuiv:
				clearBuffer(g, &b, cmd.Buffer(), cmd.Drawbuffer(), c)
			case *GlClear:
				if (cmd.Mask() & GLbitfield_GL_COLOR_BUFFER_BIT) != 0 {
					for i := range c.Bound().DrawFramebuffer().ColorAttachments().All() {
						clearBuffer(g, &b, GLenum_GL_COLOR, i, c)
					}
				}
				if (cmd.Mask() & GLbitfield_GL_DEPTH_BUFFER_BIT) != 0 {
					clearBuffer(g, &b, GLenum_GL_DEPTH, 0, c)
				}
				if (cmd.Mask() & GLbitfield_GL_STENCIL_BUFFER_BIT) != 0 {
					clearBuffer(g, &b, GLenum_GL_STENCIL, 0, c)
				}
			default:
				log.E(ctx, "Unknown clear command: %v", cmd)
			}
		} else {
			switch cmd := cmd.(type) {
			case *GlCopyImageSubData:
				// TODO: This assumes whole-image copy.  Handle sub-range copies.
				// TODO: This does not handle multiple layers well.
				if cmd.SrcTarget() == GLenum_GL_RENDERBUFFER {
					b.Read(g, renderbufferDataKey{c.Objects().Renderbuffers().Get(RenderbufferId(cmd.SrcName()))})
				} else {
					data, size := c.Objects().Textures().Get(TextureId(cmd.SrcName())).dataAndSize(cmd.SrcLevel(), 0)
					b.Read(g, data)
					b.Read(g, size)
				}
				if cmd.DstTarget() == GLenum_GL_RENDERBUFFER {
					b.Write(g,
						renderbufferDataKey{c.Objects().Renderbuffers().Get(RenderbufferId(cmd.DstName()))})
				} else {
					data, size := c.Objects().Textures().Get(TextureId(cmd.DstName())).dataAndSize(cmd.DstLevel(), 0)
					b.Write(g, data)
					b.Write(g, size)
				}
			case *GlFramebufferTexture2D:
				b.Read(g, textureSizeKey{c.Objects().Textures().Get(cmd.Texture()), cmd.Texture(), cmd.Level(), 0})
				b.KeepAlive = true // Changes untracked state
			case *GlCompressedTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, cmd, id, s, c.Bound().TextureUnit(), cmd.Target(), cmd.Level())
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlCompressedTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, cmd, id, s, c.Bound().TextureUnit(), cmd.Target(), cmd.Level())
				b.Modify(g, texData)
			case *GlTexImage2D:
				texData, texSize := getTextureDataAndSize(ctx, cmd, id, s, c.Bound().TextureUnit(), cmd.Target(), cmd.Level())
				b.Modify(g, texData)
				b.Write(g, texSize)
			case *GlTexSubImage2D:
				texData, _ := getTextureDataAndSize(ctx, cmd, id, s, c.Bound().TextureUnit(), cmd.Target(), cmd.Level())
				b.Modify(g, texData)
			case *GlGenerateMipmap:
				tex, err := subGetBoundTextureOrErrorInvalidEnum(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, cmd.Target())
				if err != nil {
					log.E(ctx, "Can not find bound texture %v", cmd.Target())
				}
				if baseLevel, ok := tex.Levels().Lookup(0); ok {
					for layerIndex := range baseLevel.Layers().All() {
						data, size := tex.dataAndSize(0, layerIndex)
						b.Read(g, data)
						b.Read(g, size)
						// Overestimate the number of levels to 31
						for levelIndex := GLint(1); levelIndex < 32; levelIndex++ {
							data, size := tex.dataAndSize(levelIndex, layerIndex)
							b.Write(g, size)
							b.Write(g, data)
						}
					}
				}
			case *GlUniform1fv:
				b.Write(g, uniformKey{c.Bound().Program(), cmd.Location(), cmd.Count()})
			case *GlUniform2fv:
				b.Write(g, uniformKey{c.Bound().Program(), cmd.Location(), cmd.Count()})
			case *GlUniform3fv:
				b.Write(g, uniformKey{c.Bound().Program(), cmd.Location(), cmd.Count()})
			case *GlUniform4fv:
				b.Write(g, uniformKey{c.Bound().Program(), cmd.Location(), cmd.Count()})
			case *GlUniformMatrix4fv:
				b.Write(g, uniformKey{c.Bound().Program(), cmd.Location(), cmd.Count()})
			case *GlVertexAttribPointer:
				b.Write(g, vertexAttribKey{c.Bound().VertexArray(), cmd.Location()})
			default:
				// Force all unhandled commands to be kept alive.
				b.KeepAlive = true
			}
		}
	} else /* c == nil */ {
		b.KeepAlive = true
	}
	return b
}

func clearBuffer(g *dependencygraph.DependencyGraph, b *dependencygraph.CmdBehaviour, buffer GLenum, index GLint, c Contextʳ) {
	var data, size dependencygraph.StateKey
	switch buffer {
	case GLenum_GL_COLOR:
		data, size = c.Bound().DrawFramebuffer().ColorAttachments().Get(index).dataAndSize(g, c)
	case GLenum_GL_DEPTH:
		data, size = c.Bound().DrawFramebuffer().DepthAttachment().dataAndSize(g, c)
	case GLenum_GL_STENCIL:
		data, size = c.Bound().DrawFramebuffer().StencilAttachment().dataAndSize(g, c)
	case GLenum_GL_DEPTH_STENCIL:
		data, size = c.Bound().DrawFramebuffer().DepthAttachment().dataAndSize(g, c)
		b.Read(g, size)
		b.Write(g, data)
		data, size = c.Bound().DrawFramebuffer().StencilAttachment().dataAndSize(g, c)
	}
	b.Read(g, size)
	b.Write(g, data)
}

func getAllUsedTextureData(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, c Contextʳ) (stateKeys []dependencygraph.StateKey) {
	// Look for samplers used by the current program.
	if c.Bound().Program().IsNil() {
		return
	}
	for _, uniform := range c.Bound().Program().UniformLocations().All() {
		if uniform.Type() == GLenum_GL_FLOAT_VEC4 || uniform.Type() == GLenum_GL_FLOAT_MAT4 {
			continue // Optimization - skip the two most common types which we know are not samplers.
		}
		target, _ := subGetTextureTargetFromSamplerType(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, uniform.Type())
		if target == GLenum_GL_NONE {
			continue // Not a sampler type
		}
		units := AsU32ˢ(s.Arena, uniform.Values(), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		for _, unit := range units {
			if tu := c.Objects().TextureUnits().Get(TextureUnitId(unit)); !tu.IsNil() {
				tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, tu, target)
				if !tex.IsNil() && err == nil {
					if !tex.EGLImage().IsNil() {
						stateKeys = append(stateKeys, eglImageDataKey{tex.EGLImage()})
					} else {
						stateKeys = append(stateKeys, textureDataGroupKey{tex, tex.ID()})
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
	unit TextureUnitʳ,
	target GLenum,
	level GLint) (dependencygraph.StateKey, dependencygraph.StateKey) {

	tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, unit, target)
	if tex.IsNil() || err != nil {
		log.E(ctx, "Can not find texture %v in unit %v", target, unit)
		return nil, nil
	}
	layer := cubemapFaceToLayer(target)
	return tex.dataAndSize(level, layer)
}

func (tex Textureʳ) dataAndSize(level, layer GLint) (dependencygraph.StateKey, dependencygraph.StateKey) {
	if !tex.EGLImage().IsNil() {
		return eglImageDataKey{tex.EGLImage()}, eglImageSizeKey{tex.EGLImage()}
	}
	return textureDataKey{tex, tex.ID(), level, layer}, textureSizeKey{tex, tex.ID(), layer, level}
}

func (att FramebufferAttachment) dataAndSize(g *dependencygraph.DependencyGraph, c Contextʳ) (dataKey dependencygraph.StateKey, sizeKey dependencygraph.StateKey) {
	if att.Type() == GLenum_GL_RENDERBUFFER {
		rb := att.Renderbuffer()
		if !rb.IsNil() && !rb.Image().IsNil() && rb.Image().SizedFormat() != GLenum_GL_NONE {
			scissor := c.Pixel().Scissor()
			box := scissor.Box()
			if scissor.Test() == GLboolean_GL_TRUE && !box.EqualTo(0, 0, rb.Image().Width(), rb.Image().Height()) {
				box := scissor.Box()
				x, y, w, h := box.X(), box.Y(), box.Width(), box.Height()
				dataKey, sizeKey = renderbufferSubDataKey{rb, x, y, w, h}, nil
			} else {
				dataKey, sizeKey = renderbufferDataKey{rb}, nil
			}
		}
	}
	if att.Type() == GLenum_GL_TEXTURE {
		if tex := att.Texture(); !tex.IsNil() {
			// TODO: We should handle scissor here as well.
			dataKey, sizeKey = tex.dataAndSize(att.TextureLevel(), att.TextureLayer())
		}
	}
	if dataKey != nil {
		g.SetRoot(dataKey)
	}
	return
}
