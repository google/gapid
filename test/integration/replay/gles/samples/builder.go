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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles"
	"github.com/google/gapid/gapis/memory"
)

type builder struct {
	atom.List
	state  *gfxapi.State
	lastID uint
}

func newBuilder(ctx context.Context) *builder {
	return &builder{
		state: gfxapi.NewStateWithEmptyAllocator(device.Little32),
	}
}

func (b *builder) newID() uint {
	b.lastID++
	return b.lastID
}

func (b *builder) newShaderID() gles.ShaderId   { return gles.ShaderId(b.newID()) }
func (b *builder) newProgramID() gles.ProgramId { return gles.ProgramId(b.newID()) }

// p returns a unique pointer. Meant to be used to generate
// pointers representing driver-side data, so the allocation
// itself is not relevant.
func (b *builder) p() memory.Pointer {
	base, err := b.state.Allocator.Alloc(8, 8)
	if err != nil {
		panic(err)
	}
	return memory.BytePtr(base, memory.ApplicationPool)
}

func (b *builder) data(ctx context.Context, v ...interface{}) atom.AllocResult {
	return atom.Must(atom.AllocData(ctx, b.state, v...))
}

func (b *builder) newEglContext(width, height int, eglShareContext memory.Pointer, preserveBuffersOnSwap bool) (eglContext, eglSurface, eglDisplay memory.Pointer) {
	eglContext = b.p()
	eglSurface = b.p()
	eglDisplay = b.p()
	eglConfig := b.p()

	// TODO: We don't observe attribute lists properly. We should.
	b.Add(
		gles.NewEglGetDisplay(gles.EGLNativeDisplayType(0), eglDisplay),
		gles.NewEglInitialize(eglDisplay, memory.Nullptr, memory.Nullptr, gles.EGLBoolean(1)),
		gles.NewEglCreateContext(eglDisplay, eglConfig, eglShareContext, b.p(), eglContext),
	)
	b.makeCurrent(eglDisplay, eglSurface, eglContext, width, height, preserveBuffersOnSwap)
	return eglContext, eglSurface, eglDisplay
}

func (b *builder) makeCurrent(eglDisplay, eglSurface, eglContext memory.Pointer, width, height int, preserveBuffersOnSwap bool) {
	eglTrue := gles.EGLBoolean(1)
	b.Add(atom.WithExtras(
		gles.NewEglMakeCurrent(eglDisplay, eglSurface, eglSurface, eglContext, eglTrue),
		gles.NewStaticContextState(),
		gles.NewDynamicContextState(width, height, preserveBuffersOnSwap),
	))
}

func (b *builder) program(ctx context.Context,
	vertexShaderID, fragmentShaderID gles.ShaderId,
	programID gles.ProgramId,
	vertexShaderSource, fragmentShaderSource string) {

	b.Add(gles.BuildProgram(ctx, b.state,
		vertexShaderID, fragmentShaderID,
		programID,
		vertexShaderSource, fragmentShaderSource)...)
}
