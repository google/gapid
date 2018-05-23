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

package transform

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/testcmd"
)

func TestInjector(t *testing.T) {
	ctx := log.Testing(t)
	inputs := testcmd.List(
		&testcmd.A{ID: 10},
		&testcmd.A{ID: 30},
		&testcmd.A{ID: 50},
		&testcmd.A{ID: 90},
		&testcmd.A{ID: 00},
		&testcmd.A{ID: 60},
	)
	expected := testcmd.List(
		&testcmd.A{ID: 10},
		&testcmd.A{ID: 30},
		&testcmd.A{ID: api.CmdNoID, Flags: 1},
		&testcmd.A{ID: 50},
		&testcmd.A{ID: 90},
		&testcmd.A{ID: api.CmdNoID, Flags: 2},
		&testcmd.A{ID: api.CmdNoID, Flags: 3},
		&testcmd.A{ID: 00},
		&testcmd.A{ID: 60},
		&testcmd.B{ID: api.CmdNoID},
	)

	transform := &Injector{}
	transform.Inject(30, &testcmd.A{ID: api.CmdNoID, Flags: 1})
	transform.Inject(90, &testcmd.A{ID: api.CmdNoID, Flags: 2})
	transform.Inject(90, &testcmd.A{ID: api.CmdNoID, Flags: 3})
	transform.Inject(60, &testcmd.B{ID: api.CmdNoID})

	transform.Inject(40, &testcmd.A{ID: 100, Flags: 5}) // Should not be injected

	CheckTransform(ctx, t, transform, inputs, expected)
}

// CheckTransform checks that transfomer emits the expected commands given
// inputs.
func CheckTransform(ctx context.Context, t *testing.T, transformer Transformer, inputs, expected testcmd.CmdAndIDList) {
	mw := &testcmd.Writer{}
	for _, in := range inputs {
		transformer.Transform(ctx, in.Id, in.Cmd, mw)
	}
	transformer.Flush(ctx, mw)

	assert.With(ctx).ThatSlice(mw.CmdsAndIDs).DeepEquals(expected)
}
