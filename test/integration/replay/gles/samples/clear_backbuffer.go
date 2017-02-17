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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi/gles"
)

// ClearBackbuffer returns the atom list needed to create a context then clear,
// sequentially the backbuffer to red, green, blue and black.
func ClearBackbuffer(ctx log.Context) (atoms *atom.List, red, green, blue, black atom.ID) {
	b := newBuilder(ctx)
	b.newEglContext(64, 64, false)
	red = b.Add(
		gles.NewGlClearColor(1.0, 0.0, 0.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	green = b.Add(
		gles.NewGlClearColor(0.0, 1.0, 0.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	blue = b.Add(
		gles.NewGlClearColor(0.0, 0.0, 1.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	black = b.Add(
		gles.NewGlClearColor(0.0, 0.0, 0.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	return &b.List, red, green, blue, black
}
