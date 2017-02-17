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

package test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
)

func TestSubAdd(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator()
	NewCmdAdd(10, 20).Mutate(ctx, s, nil)
	got := GetState(s).Ints.Read(ctx, nil, s, nil)
	expected := []int64{30}
	assert.With(ctx).ThatSlice(got).Equals(expected)
}
