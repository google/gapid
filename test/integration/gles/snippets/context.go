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

package snippets

import (
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/memory"
)

// CreateContext appends the commands to create a new EGL context with the
// given parameters.
func (b *Builder) CreateContext(width, height int, shared, preserveBuffersOnSwap bool) (eglContext, eglSurface, eglDisplay memory.Pointer) {
	eglContext, eglSurface, eglDisplay = b.p(), b.p(), b.p()
	eglConfig := b.p()

	// TODO: We don't observe attribute lists properly. We should.
	b.Cmds = append(b.Cmds,
		b.CB.EglGetDisplay(gles.EGLNativeDisplayType(0), eglDisplay),
		b.CB.EglInitialize(eglDisplay, memory.Nullptr, memory.Nullptr, gles.EGLBoolean(1)),
		b.CB.EglCreateContext(eglDisplay, eglConfig, memory.Nullptr, b.p(), eglContext),
	)

	// Switch to new context which shares resources with the first one
	if shared {
		sharedContext := eglContext
		eglContext = b.p()
		b.Cmds = append(b.Cmds,
			b.CB.EglCreateContext(eglDisplay, eglConfig, sharedContext, b.p(), eglContext),
		)
	}

	b.makeCurrent(eglDisplay, eglSurface, eglContext, width, height, preserveBuffersOnSwap)
	return eglContext, eglSurface, eglDisplay
}

// SwapBuffers appends a call to eglSwapBuffers.
func (b *Builder) SwapBuffers() api.CmdID {
	return b.Add(b.CB.EglSwapBuffers(b.eglDisplay, b.eglSurface, gles.EGLBoolean(1)))
}

// ResizeBackbuffer appends the commands to resize the backbuffer to the given
// dimensions.
func (b *Builder) ResizeBackbuffer(width, height int) api.CmdID {
	b.makeCurrent(b.eglDisplay, b.eglSurface, b.eglContext, width, height, true)
	return b.Last()
}

// ClearColor appends a the commands to clear the backbuffer with the given
// color.
func (b *Builder) ClearColor(red, green, blue, alpha gles.GLfloat) api.CmdID {
	b.Add(
		b.CB.GlClearColor(red, green, blue, alpha),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	return b.Last()
}

// ClearDepth appends a the commands to clear the backbuffer depth map.
func (b *Builder) ClearDepth() api.CmdID {
	return b.Add(b.CB.GlClear(gles.GLbitfield_GL_DEPTH_BUFFER_BIT))
}

func (b *Builder) makeCurrent(eglDisplay, eglSurface, eglContext memory.Pointer, width, height int, preserveBuffersOnSwap bool) {
	a := b.state.Arena
	eglTrue := gles.EGLBoolean(1)
	b.Cmds = append(b.Cmds, api.WithExtras(
		b.CB.EglMakeCurrent(eglDisplay, eglSurface, eglSurface, eglContext, eglTrue),
		gles.NewStaticContextStateForTest(a),
		gles.NewDynamicContextStateForTest(a, width, height, preserveBuffersOnSwap),
	))
	b.eglDisplay = eglDisplay
	b.eglSurface = eglSurface
	b.eglContext = eglContext
}
