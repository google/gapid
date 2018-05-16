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

package snippets

import (
	"context"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/memory"
)

// Builder is used to build command snippets.
type Builder struct {
	cb               gles.CommandBuilder
	Cmds             []api.Cmd
	state            *api.GlobalState
	lastID           uint
	programResources map[gles.ProgramId]gles.ActiveProgramResourcesʳ
	eglDisplay       memory.Pointer
	eglSurface       memory.Pointer
	eglContext       memory.Pointer
}

// NewBuilder returns a new builder.
func NewBuilder(ctx context.Context, cb gles.CommandBuilder, ml *device.MemoryLayout) *Builder {
	return &Builder{
		cb:               cb,
		state:            api.NewStateWithEmptyAllocator(ml),
		programResources: map[gles.ProgramId]gles.ActiveProgramResourcesʳ{},
	}
}

// Add appends cmds to the command list, returning the command identifier of the
// first added command.
func (b *Builder) Add(cmds ...api.Cmd) api.CmdID {
	start := api.CmdID(len(b.Cmds))
	b.Cmds = append(b.Cmds, cmds...)
	return start
}

// Last returns the command identifier of the last added command.
func (b *Builder) Last() api.CmdID {
	return api.CmdID(len(b.Cmds) - 1)
}

func (b *Builder) newID() uint {
	b.lastID++
	return b.lastID
}

func (b *Builder) newShaderID() gles.ShaderId   { return gles.ShaderId(b.newID()) }
func (b *Builder) newProgramID() gles.ProgramId { return gles.ProgramId(b.newID()) }

// p returns a unique pointer. Meant to be used to generate
// pointers representing driver-side data, so the allocation
// itself is not relevant.
func (b *Builder) p() memory.Pointer {
	base, err := b.state.Allocator.Alloc(8, 8)
	if err != nil {
		panic(err)
	}
	return memory.BytePtr(base)
}

// Data allocates memory for the given values.
func (b *Builder) Data(ctx context.Context, v ...interface{}) api.AllocResult {
	return b.state.AllocDataOrPanic(ctx, v...)
}
