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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/memory"
)

// undefinedFramebuffer adds a transform that will render a pattern into the
// color buffer at the end of each frame.
func undefinedFramebuffer(ctx context.Context, device *device.Instance) transform.Transformer {
	seenSurfaces := make(map[EGLSurface]bool)
	return transform.Transform("DirtyFramebuffer", func(ctx context.Context, i atom.ID, a atom.Atom, out transform.Writer) {
		out.MutateAndWrite(ctx, i, a)
		s := out.State()
		c := GetContext(s, a.Thread())
		if c == nil || !c.Info.Initialized {
			return // We can't do anything without a context.
		}
		if eglMakeCurrent, ok := a.(*EglMakeCurrent); ok && !seenSurfaces[eglMakeCurrent.Draw] {
			// Render the undefined pattern for new contexts.
			drawUndefinedFramebuffer(ctx, i, a, device, s, c, out)
			seenSurfaces[eglMakeCurrent.Draw] = true
		}
		if a.AtomFlags().IsStartOfFrame() {
			if c != nil && !c.Info.PreserveBuffersOnSwap {
				drawUndefinedFramebuffer(ctx, i, a, device, s, c, out)
			}
		}
	})
}

func drawUndefinedFramebuffer(ctx context.Context, id atom.ID, a atom.Atom, device *device.Instance, s *api.State, c *Context, out transform.Writer) error {
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
	cb := CommandBuilder{Thread: a.Thread()}
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
