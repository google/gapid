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
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
	"github.com/google/gapid/gapis/api/transform"
)

func TestEarlyTerminator(t *testing.T) {
	ctx := log.Testing(t)

	cb := test.CommandBuilder{Arena: test.Cmds.Arena}
	newCmd := func(id api.CmdID) api.Cmd {
		return cb.CmdTypeMix(uint64(id), 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, test.Voidᵖ(0x12345678), 100)
	}

	inputs := transform.NewCmdAndIDList(
		newCmd(10),
		newCmd(30),
		newCmd(20),
		newCmd(50),
		newCmd(90),
		newCmd(70),
		newCmd(80),
		newCmd(00),
		newCmd(60),
		newCmd(40),
	)
	expected := transform.NewCmdAndIDList(
		newCmd(10),
		newCmd(30),
		newCmd(20),
		newCmd(50),
		newCmd(90),
		newCmd(70),
	)

	et := transform.NewEarlyTerminator(test.API{}.ID())
	et.Add(ctx, 20, []uint64{0})
	et.Add(ctx, 50, []uint64{})
	et.Add(ctx, 70, []uint64{1})

	CheckTransform(ctx, t, et, inputs, expected)
}
