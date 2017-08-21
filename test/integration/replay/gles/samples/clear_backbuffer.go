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

// ClearBackbuffer returns the command list needed to create a context then
// clear, sequentially the backbuffer to red, green, blue and black.
func ClearBackbuffer(ctx context.Context, cb gles.CommandBuilder, ml *device.MemoryLayout) (cmds []api.Cmd, red, green, blue, black api.CmdID) {
	b := newBuilder(ctx, ml)
	b.newEglContext(64, 64, memory.Nullptr, false)
	b.cmds = append(b.cmds,
		cb.GlClearColor(1.0, 0.0, 0.0, 1.0),
		cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	red = api.CmdID(len(b.cmds) - 1)
	b.cmds = append(b.cmds,
		cb.GlClearColor(0.0, 1.0, 0.0, 1.0),
		cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	green = api.CmdID(len(b.cmds) - 1)
	b.cmds = append(b.cmds,
		cb.GlClearColor(0.0, 0.0, 1.0, 1.0),
		cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	blue = api.CmdID(len(b.cmds) - 1)
	b.cmds = append(b.cmds,
		cb.GlClearColor(0.0, 0.0, 0.0, 1.0),
		cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	black = api.CmdID(len(b.cmds) - 1)
	return b.cmds, red, green, blue, black
}
