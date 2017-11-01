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

package samples

import (
	"context"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/memory"
)

// DrawTexturedSquare returns the command list needed to create a context then
// draw a textured square.
func DrawTexturedSquare(ctx context.Context, cb gles.CommandBuilder, sharedContext bool, ml *device.MemoryLayout) (cmds []api.Cmd, draw api.CmdID, swap api.CmdID) {
	squareVertices := []float32{
		-0.5, -0.5, 0.5,
		-0.5, +0.5, 0.5,
		+0.5, +0.5, 0.5,
		+0.5, -0.5, 0.5,
	}

	squareIndices := []uint16{
		0, 1, 2, 0, 2, 3,
	}

	textureVSSource := `
		precision mediump float;
		attribute vec3 position;
		varying vec2 texcoord;
		void main() {
			gl_Position = vec4(position, 1.0);
			texcoord = position.xy + vec2(0.5, 0.5);
		}`

	textureFSSource := `
		precision mediump float;
		uniform sampler2D tex;
		varying vec2 texcoord;
		void main() {
			gl_FragColor = texture2D(tex, texcoord);
		}`

	b := newBuilder(ctx, ml)
	vs, fs, prog, pos := b.newShaderID(), b.newShaderID(), b.newProgramID(), gles.AttributeLocation(0)
	eglContext, eglSurface, eglDisplay := b.newEglContext(128, 128, memory.Nullptr, false)
	texLoc := gles.UniformLocation(0)

	textureNames := []gles.TextureId{1}
	textureNamesPtr := b.data(ctx, textureNames)
	texData := make([]uint8, 3*64*64)
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			texData[y*64*3+x*3] = uint8(x * 4)
			texData[y*64*3+x*3+1] = uint8(y * 4)
			texData[y*64*3+x*3+2] = 255
		}
	}

	textureData := b.data(ctx, texData)
	squareIndicesPtr := b.data(ctx, squareIndices)
	squareVerticesPtr := b.data(ctx, squareVertices)

	// Build the program resource
	b.program(ctx, vs, fs, prog, textureVSSource, textureFSSource)
	b.cmds = append(b.cmds,
		api.WithExtras(
			cb.GlLinkProgram(prog),
			&gles.ProgramInfo{
				LinkStatus: gles.GLboolean_GL_TRUE,
				ActiveUniforms: gles.NewUniformIndexːActiveUniformᵐ().Add(0, gles.ActiveUniform{
					Type:      gles.GLenum_GL_SAMPLER_2D,
					Name:      "tex",
					ArraySize: 1,
					Location:  texLoc,
				}),
			}),
		cb.GlGetUniformLocation(prog, "tex", texLoc),
	)

	// Build the texture resource
	b.cmds = append(b.cmds,
		cb.GlGenTextures(1, textureNamesPtr.Ptr()).AddWrite(textureNamesPtr.Data()),
		cb.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		cb.GlTexImage2D(
			gles.GLenum_GL_TEXTURE_2D,
			0,
			gles.GLint(gles.GLenum_GL_RGB),
			64,
			64,
			0,
			gles.GLenum_GL_RGB,
			gles.GLenum_GL_UNSIGNED_BYTE,
			textureData.Ptr(),
		).AddRead(textureData.Data()),
	)

	// Switch to new context which shares resources with the first one
	if sharedContext {
		eglContext, eglSurface, eglDisplay = b.newEglContext(128, 128, eglContext, false)
	}

	// Render square using the build program and texture
	b.cmds = append(b.cmds,
		cb.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		cb.GlClearColor(0.0, 1.0, 0.0, 1.0),
		cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
		cb.GlUseProgram(prog),
		cb.GlActiveTexture(gles.GLenum_GL_TEXTURE0),
		cb.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		cb.GlUniform1i(texLoc, 0),
		cb.GlGetAttribLocation(prog, "position", gles.GLint(pos)),
		cb.GlEnableVertexAttribArray(pos),
		cb.GlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesPtr.Ptr()),
		cb.GlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesPtr.Ptr()).
			AddRead(squareIndicesPtr.Data()).
			AddRead(squareVerticesPtr.Data()),
	)
	draw = api.CmdID(len(b.cmds) - 1)
	b.cmds = append(b.cmds,
		cb.EglSwapBuffers(eglDisplay, eglSurface, gles.EGLBoolean(1)),
	)
	swap = api.CmdID(len(b.cmds) - 1)

	return b.cmds, draw, swap
}
