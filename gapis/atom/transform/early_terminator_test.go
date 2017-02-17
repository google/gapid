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
	"github.com/google/gapid/gapis/atom/test"
)

func TestEarlyTerminator(t *testing.T) {
	ctx := log.Testing(t)
	inputs := test.List(
		&test.AtomA{ID: 10},
		&test.AtomA{ID: 30},
		&test.AtomA{ID: 20},
		&test.AtomA{ID: 50},
		&test.AtomA{ID: 90},
		&test.AtomA{ID: 70},
		&test.AtomA{ID: 80},
		&test.AtomA{ID: 00},
		&test.AtomA{ID: 60},
		&test.AtomA{ID: 40},
	)
	expected := test.List(
		&test.AtomA{ID: 10},
		&test.AtomA{ID: 30},
		&test.AtomA{ID: 20},
		&test.AtomA{ID: 50},
		&test.AtomA{ID: 90},
		&test.AtomA{ID: 70},
	)

	transform := &EarlyTerminator{}
	transform.Add(20)
	transform.Add(50)
	transform.Add(70)

	CheckTransform(ctx, t, transform, inputs, expected)
}
