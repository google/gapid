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

package replay

import (
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/integration/gles/snippets"
)

func TestDrawTexturedSquare(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(128, 128, false, false)
	draw, _ := b.DrawTexturedSquare(ctx)
	c := buildAndMaybeExportCapture(ctx, b, "textured-square")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 128, 128, 0.01, "textured-square", draw, nil)
	})
}

func TestDrawTexturedSquareWithSharedContext(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(128, 128, true, false)
	draw, _ := b.DrawTexturedSquare(ctx)
	c := buildAndMaybeExportCapture(ctx, b, "textured-square-shared-context")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 128, 128, 0.01, "textured-square", draw, nil)
	})
}

func TestDrawTriangle(t *testing.T) {
	test(t, "draw-triangle", generateDrawTriangleCapture)
}
