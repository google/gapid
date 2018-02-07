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

package codegen_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/codegen/call"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/host"
)

func TestCodegen(t *testing.T) {
	ctx := log.Testing(t)

	hostABI := host.Instance(ctx).Configuration.ABIs[0]

	m := codegen.NewModule("test", hostABI)
	f := m.Function(m.Types.Int, "add", m.Types.Int, m.Types.Int)
	err := f.Build(func(b *codegen.Builder) {
		x := b.Parameter(0)
		y := b.Parameter(1)
		b.Return(b.Add(x, y))
	})
	if !assert.For(ctx, "f.Build").ThatError(err).Succeeded() {
		return
	}
	e, err := m.Executor(false)
	if !assert.For(ctx, "m.Executor(false)").ThatError(err).Succeeded() {
		return
	}
	x := call.III(e.FunctionAddress(f), 2, 3)
	assert.For(ctx, "call.III('add', 2, 3)").ThatInteger(x).Equals(5)
}
