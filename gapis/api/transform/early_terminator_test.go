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
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/testcmd"
)

func TestEarlyTerminator(t *testing.T) {
	ctx := log.Testing(t)
	inputs := NewCmdAndIDList(
		&testcmd.A{ID: 10},
		&testcmd.A{ID: 30},
		&testcmd.A{ID: 20},
		&testcmd.A{ID: 50},
		&testcmd.A{ID: 90},
		&testcmd.A{ID: 70},
		&testcmd.A{ID: 80},
		&testcmd.A{ID: 00},
		&testcmd.A{ID: 60},
		&testcmd.A{ID: 40},
	)
	expected := NewCmdAndIDList(
		&testcmd.A{ID: 10},
		&testcmd.A{ID: 30},
		&testcmd.A{ID: 20},
		&testcmd.A{ID: 50},
		&testcmd.A{ID: 90},
		&testcmd.A{ID: 70},
	)

	transform := NewEarlyTerminator(api.ID{})
	transform.Add(ctx, 0, 20, []uint64{0})
	transform.Add(ctx, 0, 50, []uint64{})
	transform.Add(ctx, 0, 70, []uint64{1})

	CheckTransform(ctx, t, transform, inputs, expected)
}
