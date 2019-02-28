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

// +build !headless

package replay

import (
	"sync"
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/integration/gles/snippets"
)

func TestClear(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)

	red, green, blue, black := b.ClearBackbuffer(ctx)
	c := buildAndMaybeExportCapture(ctx, b, "clear")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		done := &sync.WaitGroup{}
		done.Add(4)
		go checkColorBuffer(ctx, c, d, 64, 64, 0, "solid-red", red, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0, "solid-green", green, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0, "solid-blue", blue, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0, "solid-black", black, done)
		done.Wait()
	})
}
