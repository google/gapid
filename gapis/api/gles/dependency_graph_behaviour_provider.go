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

type shaderProgramKeyKind int

const (
	everything shaderProgramKeyKind = iota
	uniforms
	binding
	linkstate
	compilestate
	source
)

type programKey struct {
	program Programʳ
	kind    shaderProgramKeyKind
}

func (k programKey) Parent() dependencygraph.StateKey {
	switch k.kind {
	case uniforms, binding, linkstate:
		return programKey{k.program, everything}
	}
	return nil
}

type shaderKey struct {
	shader Shaderʳ
	kind   shaderProgramKeyKind
}

func (k shaderKey) Parent() dependencygraph.StateKey {
	switch k.kind {
	case compilestate, source:
		return shaderKey{k.shader, everything}
	}
	return nil
}

type uniformKey struct {
	program  Programʳ
	location UniformLocation
	count    GLsizei
}

func (k uniformKey) Parent() dependencygraph.StateKey { return programKey{k.program, uniforms} }

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
	if err := cmd.Mutate(ctx, id, s, nil /* builder */, nil /* watcher */); err != nil {
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
			switch {
			case !c.Bound().Program().IsNil():
				b.Read(g, programKey{c.Bound().Program(), everything})
			case !c.Bound().Pipeline().IsNil():
				p := c.Bound().Pipeline()
				b.Read(g, programKey{p.VertexShader(), everything})
				b.Read(g, programKey{p.TessControlShader(), everything})
				b.Read(g, programKey{p.TessEvaluationShader(), everything})
				b.Read(g, programKey{p.GeometryShader(), everything})
				b.Read(g, programKey{p.FragmentShader(), everything})
				b.Read(g, programKey{p.ComputeShader(), everything})
			}
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
				if cmd.SrcTarget() == GLenum_GL_RENDERBUFFER {
					b.Read(g, renderbufferDataKey{c.Objects().Renderbuffers().Get(RenderbufferId(cmd.SrcName()))})
				} else {
					for layer := GLsizei(0); layer < cmd.SrcDepth(); layer++ {
						data, size := c.Objects().Textures().Get(TextureId(cmd.SrcName())).dataAndSize(cmd.SrcLevel(), GLint(layer)+cmd.SrcZ())
						b.Read(g, data)
						b.Read(g, size)
					}
				}
				if cmd.DstTarget() == GLenum_GL_RENDERBUFFER {
					b.Write(g,
						renderbufferDataKey{c.Objects().Renderbuffers().Get(RenderbufferId(cmd.DstName()))})
				} else {
					for layer := GLsizei(0); layer < cmd.SrcDepth(); layer++ {
						data, size := c.Objects().Textures().Get(TextureId(cmd.DstName())).dataAndSize(cmd.DstLevel(), GLint(layer)+cmd.DstZ())
						b.Write(g, data)
						b.Write(g, size)
					}
				}
			case *GlFramebufferTexture2D:
				var layer GLint
				switch target := cmd.TextureTarget(); target {
				case GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_X,
					GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Y, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Y,
					GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_Z, GLenum_GL_TEXTURE_CUBE_MAP_NEGATIVE_Z:
					layer = GLint(target - GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X)
				}
				b.Read(g, textureSizeKey{c.Objects().Textures().Get(cmd.Texture()), cmd.Texture(), cmd.Level(), layer})
				b.KeepAlive = true // Changes untracked state
			case *GlFramebufferTexture:
				if t := c.Objects().Textures().Get(cmd.Texture()); !t.IsNil() {
					for layer := range t.Levels().Get(cmd.Level()).Layers().All() {
						b.Read(g, textureSizeKey{t, cmd.Texture(), cmd.Level(), layer})
					}
				}
				b.KeepAlive = true // Changes untracked state
			case *GlFramebufferTextureLayer:
				b.Read(g, textureSizeKey{c.Objects().Textures().Get(cmd.Texture()), cmd.Texture(), cmd.Level(), cmd.Layer()})
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
				tex, err := subGetBoundTextureOrErrorInvalidEnum(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, nil, cmd.Target())
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
			case *GlShaderSource:
				b.Write(g, shaderKey{c.Objects().Shaders().Get(cmd.Shader()), source})
			case *GlCompileShader:
				s := c.Objects().Shaders().Get(cmd.Shader())
				b.Read(g, shaderKey{s, source})
				b.Write(g, shaderKey{s, compilestate})
			case *GlLinkProgram:
				p := c.Objects().Programs().Get(cmd.Program())
				for _, s := range p.Shaders().All() {
					b.Read(g, shaderKey{s, compilestate})
				}
				b.Write(g, programKey{p, linkstate})
			case *GlGetUniformLocation:
				// Treat this as a modify so it is coupled with the glLinkProgram call.
				b.Modify(g, programKey{c.Objects().Programs().Get(cmd.Program()), linkstate})
			case *GlUseProgram:
				p := c.Objects().Programs().Get(cmd.Program())
				b.Read(g, programKey{p, linkstate})
				b.Write(g, programKey{p, binding})
			case *GlBindProgramPipeline:
				p := c.Objects().Pipelines().Get(cmd.Pipeline())
				b.Read(g, programKey{p.VertexShader(), linkstate})
				b.Read(g, programKey{p.TessControlShader(), linkstate})
				b.Read(g, programKey{p.TessEvaluationShader(), linkstate})
				b.Read(g, programKey{p.GeometryShader(), linkstate})
				b.Read(g, programKey{p.FragmentShader(), linkstate})
				b.Read(g, programKey{p.ComputeShader(), linkstate})
				b.KeepAlive = true // TODO: complete separable shader support
			case *GlUniform1f:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform2f:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform3f:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform4f:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform1i:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform2i:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform3i:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform4i:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform1ui:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform2ui:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform3ui:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform4ui:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), 1})
			case *GlUniform1fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform2fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform3fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform4fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform1iv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform2iv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform3iv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform4iv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform1uiv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform2uiv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform3uiv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniform4uiv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix2fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix3fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix4fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix2x3fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix3x2fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix2x4fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix4x2fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix3x4fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlUniformMatrix4x3fv:
				p := getBoundProgram(c)
				b.Read(g, programKey{p, binding})
				b.Write(g, uniformKey{p, cmd.Location(), cmd.Count()})
			case *GlVertexAttribPointer:
				b.Write(g, vertexAttribKey{c.Bound().VertexArray(), cmd.Location()})
			case *GlEGLImageTargetTexture2DOES:
				img := GetState(s).EGLImages().Get(EGLImageKHR(cmd.Image()))
				if !img.IsNil() && img.Target() == EGLenum_EGL_GL_TEXTURE_2D {
					if sc := GetState(s).EGLContexts().Get(img.Context()); !sc.IsNil() {
						data, size := sc.Objects().Textures().Get(TextureId(img.Buffer())).dataAndSize(0, 0)
						b.Read(g, data)
						b.Read(g, size)
					}
				}
				data, size := getTextureDataAndSize(ctx, cmd, id, s, c.Bound().TextureUnit(), cmd.Target(), 0)
				b.Write(g, data)
				b.Write(g, size)
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

func getBoundProgram(c Contextʳ) Programʳ {
	switch {
	case !c.Bound().Program().IsNil():
		return c.Bound().Program()
	case !c.Bound().Pipeline().IsNil():
		return c.Bound().Pipeline().ActiveProgram()
	default:
		return NilProgramʳ
	}
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

func (pipeline Pipelineʳ) Uniforms() []Uniform {
	var uniforms []Uniform
	if pipeline.IsNil() {
		return uniforms
	}
	// TODO: These might not all be necessary.
	uniforms = append(uniforms, pipeline.VertexShader().Uniforms()...)
	uniforms = append(uniforms, pipeline.TessControlShader().Uniforms()...)
	uniforms = append(uniforms, pipeline.TessEvaluationShader().Uniforms()...)
	uniforms = append(uniforms, pipeline.GeometryShader().Uniforms()...)
	uniforms = append(uniforms, pipeline.FragmentShader().Uniforms()...)
	uniforms = append(uniforms, pipeline.ComputeShader().Uniforms()...)
	return uniforms
}

func (program Programʳ) Uniforms() []Uniform {
	if program.IsNil() {
		return nil
	}
	uniforms := make([]Uniform, 0, len(program.UniformLocations().All()))
	for _, u := range program.UniformLocations().All() {
		uniforms = append(uniforms, u)
	}
	return uniforms
}

func getAllUsedTextureData(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, c Contextʳ) (stateKeys []dependencygraph.StateKey) {

	// Look for samplers used by the current program/pipeline.

	// Get the uniforms in use.
	var uniforms []Uniform

	if !c.Bound().Program().IsNil() {
		// The bound Program is used if present.
		uniforms = c.Bound().Program().Uniforms()
	} else if !c.Bound().Pipeline().IsNil() {
		// Otherwise, the bound Pipeline is used, if present.
		uniforms = c.Bound().Pipeline().Uniforms()
	} else {
		// Otherwise, there is no bound Program nor Pipeline; no texture reads.
		return
	}

	for _, uniform := range uniforms {
		if uniform.Type() == GLenum_GL_FLOAT_VEC4 || uniform.Type() == GLenum_GL_FLOAT_MAT4 {
			continue // Optimization - skip the two most common types which we know are not samplers.
		}
		target, _ := subGetTextureTargetFromSamplerType(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, nil, uniform.Type())
		if target == GLenum_GL_NONE {
			continue // Not a sampler type
		}
		units := AsU32ˢ(s.Arena, uniform.Values(), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
		for _, unit := range units {
			if tu := c.Objects().TextureUnits().Get(TextureUnitId(unit)); !tu.IsNil() {
				tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, nil, tu, target)
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

	tex, err := subGetBoundTextureForUnit(ctx, cmd, id, nil, s, GetState(s), cmd.Thread(), nil, nil, unit, target)
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
