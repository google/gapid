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

package arena_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
)

func TestArenaStats(t *testing.T) {
	ctx := log.Testing(t)

	a := arena.New()
	defer a.Dispose()

	assert.For(ctx, "empty arena").That(a.Stats()).Equals(arena.Stats{})

	a.Allocate(10, 4)
	assert.For(ctx, "num alloc").ThatInteger(a.Stats().NumAllocations).Equals(1)
	assert.For(ctx, "bytes alloc").ThatInteger(a.Stats().NumBytesAllocated).IsAtLeast(10)
}
