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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

func TestSubAdd(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	env := executor.NewEnv(ctx, executor.Config{
		CaptureABI: device.AndroidARMv7a,
		Execute:    true,
	})
	defer env.Dispose()
	ctx = executor.PutEnv(ctx, env)
	cb := CommandBuilder{Thread: 0, Arena: env.State.Arena}
	api.MutateCmds(ctx, env.State, nil, nil, cb.CmdAdd(10, 20))
	got, err := GetState(env.State).Ints().Read(ctx, nil, env.State, nil)
	expected := []memory.Int{30}
	if assert.For(ctx, "err").ThatError(err).Succeeded() {
		assert.For(ctx, "got").ThatSlice(got).Equals(expected)
	}
}
