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

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
)

// undefinedFramebuffer adds a transform that will render a pattern into the
// color buffer at the end of each frame.
func undefinedFramebuffer(ctx context.Context, device *device.Instance) transform.Transformer {
	seenSurfaces := make(map[EGLSurface]bool)
	return transform.Transform("DirtyFramebuffer", func(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
		out.MutateAndWrite(ctx, id, cmd)
		s := out.State()
		c := GetContext(s, cmd.Thread())
		if c == nil || !c.Info.Initialized {
			return // We can't do anything without a context.
		}
		if eglMakeCurrent, ok := cmd.(*EglMakeCurrent); ok && !seenSurfaces[eglMakeCurrent.Draw] {
			// Render the undefined pattern for new contexts.
			drawUndefinedFramebuffer(ctx, id, cmd, device, s, c, out)
			seenSurfaces[eglMakeCurrent.Draw] = true
		}
		if cmd.CmdFlags(ctx, id, s).IsStartOfFrame() {
			if _, ok := cmd.(*EglSwapBuffersWithDamageKHR); ok {
				// TODO: This is a hack. eglSwapBuffersWithDamageKHR is nearly
				// exculsively used by the Android framework, which also loves
				// to do partial framebuffer updates. Unfortunately we do not
				// currently know whether the framebuffer is invalidated between
				// calls to eglSwapBuffersWithDamageKHR as the OS now uses the
				// EGL_EXT_buffer_age extension, which we do not track. For now,
				// assume that eglSwapBuffersWithDamageKHR calls are coming from
				// the framework, and that the framebuffer is reused between
				// calls.
				// BUG: https://github.com/google/gapid/issues/846.
				return
			}
			if c != nil && !c.Info.PreserveBuffersOnSwap {
				drawUndefinedFramebuffer(ctx, id, cmd, device, s, c, out)
			}
		}
	})
}

func drawUndefinedFramebuffer(ctx context.Context, id api.CmdID, cmd api.Cmd, device *device.Instance, s *api.GlobalState, c *Context, out transform.Writer) error {
	const (
		aScreenCoordsLocation AttributeLocation = 0

		vertexShaderSource string = `
					precision highp float;
					attribute vec2 aScreenCoords;
					varying vec2 uv;

					void main() {
						uv = aScreenCoords;
						gl_Position = vec4(aScreenCoords.xy, 0., 1.);
					}`
		fragmentShaderSource string = `
					precision highp float;
					varying vec2 uv;

					float F(float a) { return smoothstep(0.0, 0.1, a) * smoothstep(0.4, 0.3, a); }

					void main() {
						vec2 v = uv * 5.0;
						gl_FragColor = vec4(0.8, 0.9, 0.6, 1.0) * F(fract(v.x + v.y));
					}`
	)

	// 2D vertices positions for a full screen 2D triangle strip.
	positions := []float32{-1., -1., 1., -1., -1., 1., 1., 1.}

	dID := id.Derived()
	cb := CommandBuilder{Thread: cmd.Thread()}
	t := newTweaker(out, id, cb)

	// Temporarily change rasterizing/blending state and enable VAP 0.
	t.glDisable(ctx, GLenum_GL_BLEND)
	t.glDisable(ctx, GLenum_GL_CULL_FACE)
	t.glDisable(ctx, GLenum_GL_DEPTH_TEST)
	t.glDisable(ctx, GLenum_GL_SCISSOR_TEST)
	t.glDisable(ctx, GLenum_GL_STENCIL_TEST)
	t.makeVertexArray(ctx, aScreenCoordsLocation)

	programID := t.makeProgram(ctx, vertexShaderSource, fragmentShaderSource)

	out.MutateAndWrite(ctx, dID, cb.GlBindAttribLocation(programID, aScreenCoordsLocation, "aScreenCoords"))
	out.MutateAndWrite(ctx, dID, cb.GlLinkProgram(programID))
	t.glUseProgram(ctx, programID)

	bufferID := t.glGenBuffer(ctx)
	t.GlBindBuffer_ArrayBuffer(ctx, bufferID)

	tmp := t.AllocData(ctx, positions)
	out.MutateAndWrite(ctx, dID, cb.GlBufferData(GLenum_GL_ARRAY_BUFFER, GLsizeiptr(4*len(positions)), tmp.Ptr(), GLenum_GL_STATIC_DRAW).
		AddRead(tmp.Data()))

	out.MutateAndWrite(ctx, dID, cb.GlVertexAttribPointer(aScreenCoordsLocation, 2, GLenum_GL_FLOAT, GLboolean(0), 0, memory.Nullptr))
	out.MutateAndWrite(ctx, dID, cb.GlDrawArrays(GLenum_GL_TRIANGLE_STRIP, 0, 4))

	t.revert(ctx)

	return nil
}
