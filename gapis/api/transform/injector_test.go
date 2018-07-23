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

package transform_test

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
	"github.com/google/gapid/gapis/api/transform"
)

func TestInjector(t *testing.T) {
	ctx := log.Testing(t)

	cb := test.CommandBuilder{Arena: test.Cmds.Arena}
	newCmd := func(id api.CmdID, tag uint64) api.Cmd {
		return cb.CmdTypeMix(uint64(id), 10, 20, 30, 40, 50, 60, tag, 80, 90, 100, true, test.Voidᵖ(0x12345678), 100)
	}

	inputs := transform.NewCmdAndIDList(
		newCmd(10, 0),
		newCmd(30, 0),
		newCmd(50, 0),
		newCmd(90, 0),
		newCmd(00, 0),
		newCmd(60, 0),
	)
	expected := transform.NewCmdAndIDList(
		newCmd(10, 0),
		newCmd(30, 0),
		newCmd(api.CmdNoID, 1),
		newCmd(50, 0),
		newCmd(90, 0),
		newCmd(api.CmdNoID, 2),
		newCmd(api.CmdNoID, 3),
		newCmd(00, 0),
		newCmd(60, 0),
		newCmd(api.CmdNoID, 0),
	)

	transform := &transform.Injector{}
	transform.Inject(30, newCmd(api.CmdNoID, 1))
	transform.Inject(90, newCmd(api.CmdNoID, 2))
	transform.Inject(90, newCmd(api.CmdNoID, 3))
	transform.Inject(60, newCmd(api.CmdNoID, 0))

	transform.Inject(40, newCmd(100, 5)) // Should not be injected

	CheckTransform(ctx, t, transform, inputs, expected)
}

// CheckTransform checks that transfomer emits the expected commands given
// inputs.
func CheckTransform(ctx context.Context, t *testing.T, transformer transform.Transformer, inputs, expected transform.CmdAndIDList) {
	r := &transform.Recorder{}
	for _, in := range inputs {
		transformer.Transform(ctx, in.ID, in.Cmd, r)
	}
	transformer.Flush(ctx, r)

	assert.For(ctx, "CmdsAndIDs").ThatSlice(r.CmdsAndIDs).DeepEquals(expected)
}
