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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
)

// ClearBackbuffer returns the command list needed to sequentially clear the
// backbuffer to red, green, blue and black.
func (b *Builder) ClearBackbuffer(ctx context.Context) (red, green, blue, black api.CmdID) {
	red = b.Add(
		b.CB.GlClearColor(1.0, 0.0, 0.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	) + 1
	green = b.Add(
		b.CB.GlClearColor(0.0, 1.0, 0.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	) + 1
	blue = b.Add(
		b.CB.GlClearColor(0.0, 0.0, 1.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	) + 1
	black = b.Add(
		b.CB.GlClearColor(0.0, 0.0, 0.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	) + 1
	return red, green, blue, black
}
