// Copyright (C) 2018 Google Inc.
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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
)

// copyDepthToColorGLES emmits commands that will copy the depth data of the
// current read framebuffer into an new framebuffer (which is left bound as the
// read framebuffer upon exit) as the color attachment in R32F format, allowing
// it to be read via glReadPixels.
func copyDepthToColorGLES(ctx context.Context, dID api.CmdID, thread uint64, s *api.GlobalState, out transform.Writer, t *tweaker, fbai fbai, width, height int32) {
	const (
		aScreenCoordsLocation AttributeLocation = 0
		aScreenCoords                           = "aScreenCoords"
		uTextureLocation      UniformLocation   = 0
		uTexture                                = "uTexture"

		vertexShaderSource string = `
					#version 300 es

					precision highp float;
					in vec2 aScreenCoords;
					out vec2 uv;

					void main() {
						uv = (aScreenCoords + 1.0) / 2.0;
						gl_Position = vec4(aScreenCoords.xy, 0., 1.);
					}`
		fragmentShaderSource string = `
					#version 300 es

					precision highp float;
					uniform sampler2D uTexture;
					in vec2 uv;
					out vec4 fragColor;

					void main() {
						fragColor = texture(uTexture, uv);
					}`
	)

	// 2D vertices positions for a full screen 2D triangle strip.
	positions := []float32{-1., -1., 1., -1., -1., 1., 1., 1.}
	ws, hs := GLsizei(width), GLsizei(height)
	wi, hi := GLint(width), GLint(height)

	cb := CommandBuilder{Thread: thread, Arena: s.Arena}

	// Create a framebuffer with a depth texture and blit.
	fb := t.glGenFramebuffer(ctx)
	tex := t.glGenTexture(ctx)
	t.glActiveTexture(ctx, GLenum_GL_TEXTURE0)
	t.glBindTexture_2D(ctx, tex)
	out.MutateAndWrite(ctx, dID, cb.GlTexParameteri(GLenum_GL_TEXTURE_2D, GLenum_GL_TEXTURE_MIN_FILTER, GLint(GLenum_GL_NEAREST)))
	out.MutateAndWrite(ctx, dID, cb.GlTexParameteri(GLenum_GL_TEXTURE_2D, GLenum_GL_TEXTURE_MAG_FILTER, GLint(GLenum_GL_NEAREST)))
	if fbai.internalFormat != GLenum_GL_NONE && fbai.internalFormat != fbai.sizedFormat {
		out.MutateAndWrite(ctx, dID, cb.GlTexImage2D(GLenum_GL_TEXTURE_2D, 0, GLint(fbai.internalFormat), ws, hs, 0, fbai.format, fbai.ty, memory.Nullptr))
	} else {
		out.MutateAndWrite(ctx, dID, cb.GlTexStorage2D(GLenum_GL_TEXTURE_2D, 1, fbai.sizedFormat, ws, hs))
	}
	t.glBindFramebuffer_Draw(ctx, fb)
	out.MutateAndWrite(ctx, dID, cb.GlFramebufferTexture2D(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_DEPTH_ATTACHMENT, GLenum_GL_TEXTURE_2D, tex, 0))
	out.MutateAndWrite(ctx, dID, cb.GlBlitFramebuffer(0, 0, wi, hi, 0, 0, wi, hi, GLbitfield_GL_DEPTH_BUFFER_BIT, GLenum_GL_NEAREST))

	// Detach the depth texture from the FB and attach a color renderbuffer.
	out.MutateAndWrite(ctx, dID, cb.GlFramebufferTexture2D(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_DEPTH_ATTACHMENT, GLenum_GL_TEXTURE_2D, 0, 0))
	rb := t.glGenRenderbuffer(ctx)
	t.glBindRenderbuffer(ctx, rb)
	out.MutateAndWrite(ctx, dID, cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, GLenum_GL_R32F, ws, hs))
	out.MutateAndWrite(ctx, dID, cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_COLOR_ATTACHMENT0, GLenum_GL_RENDERBUFFER, rb))

	// Temporarily change rasterizing/blending state and enable VAP 0.
	t.glDisable(ctx, GLenum_GL_BLEND)
	t.glDisable(ctx, GLenum_GL_CULL_FACE)
	t.glDisable(ctx, GLenum_GL_DEPTH_TEST)
	t.glDisable(ctx, GLenum_GL_SCISSOR_TEST)
	t.glDisable(ctx, GLenum_GL_STENCIL_TEST)
	t.makeVertexArray(ctx, aScreenCoordsLocation)
	t.glViewport(ctx, 0, 0, ws, hs)

	// Create a program, link, and use it.
	programID := t.makeProgram(ctx, vertexShaderSource, fragmentShaderSource)
	tmp0 := t.AllocData(ctx, aScreenCoords)
	out.MutateAndWrite(ctx, dID, cb.GlBindAttribLocation(programID, aScreenCoordsLocation, tmp0.Ptr()).
		AddRead(tmp0.Data()))
	tmp0.Free()

	attrib := MakeProgramResourceʳ(s.Arena)
	attrib.SetType(GLenum_GL_FLOAT_VEC2)
	attrib.SetName(aScreenCoords)
	attrib.SetArraySize(1)
	attrib.SetLocations(NewU32ːGLintᵐ(s.Arena).Add(0, 0))

	unif := MakeProgramResourceʳ(s.Arena)
	unif.SetType(GLenum_GL_SAMPLER_2D)
	unif.SetName(uTexture)
	unif.SetArraySize(1)
	unif.SetLocations(NewU32ːGLintᵐ(s.Arena).Add(0, 0))

	resources := MakeActiveProgramResourcesʳ(s.Arena)
	resources.SetProgramInputs(NewU32ːProgramResourceʳᵐ(s.Arena).Add(0, attrib))
	resources.SetDefaultUniformBlock(NewUniformIndexːProgramResourceʳᵐ(s.Arena).Add(UniformIndex(0), unif))

	extra := MakeLinkProgramExtra(s.Arena)
	extra.SetLinkStatus(GLboolean_GL_TRUE)
	extra.SetActiveResources(resources)
	out.MutateAndWrite(ctx, dID, api.WithExtras(cb.GlLinkProgram(programID), extra))
	tmp1 := t.AllocData(ctx, uTexture)
	out.MutateAndWrite(ctx, dID, cb.GlGetUniformLocation(programID, tmp1.Ptr(), uTextureLocation).
		AddRead(tmp1.Data()))
	tmp1.Free()
	t.glUseProgram(ctx, programID)

	// Create a buffer and fill it.
	bufferID := t.glGenBuffer(ctx)
	t.GlBindBuffer_ArrayBuffer(ctx, bufferID)

	tmp2 := t.AllocData(ctx, positions)
	out.MutateAndWrite(ctx, dID, cb.GlBufferData(GLenum_GL_ARRAY_BUFFER, GLsizeiptr(4*len(positions)), tmp1.Ptr(), GLenum_GL_STATIC_DRAW).
		AddRead(tmp2.Data()))
	tmp2.Free()

	// Render a textured quad.
	out.MutateAndWrite(ctx, dID, cb.GlVertexAttribPointer(aScreenCoordsLocation, 2, GLenum_GL_FLOAT, GLboolean(0), 0, memory.Nullptr))
	out.MutateAndWrite(ctx, dID, cb.GlDrawArrays(GLenum_GL_TRIANGLE_STRIP, 0, 4))

	// Leave the tweaker with the read fb binding containing the depth in the color attachemnt.
	t.glBindFramebuffer_Read(ctx, fb)
}
