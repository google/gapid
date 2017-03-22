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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
)

func TestInjector(t *testing.T) {
	ctx := log.Testing(t)
	inputs := test.List(
		&test.AtomA{ID: 10},
		&test.AtomA{ID: 30},
		&test.AtomA{ID: 50},
		&test.AtomA{ID: 90},
		&test.AtomA{ID: 00},
		&test.AtomA{ID: 60},
	)
	expected := test.List(
		&test.AtomA{ID: 10},
		&test.AtomA{ID: 30},
		&test.AtomA{ID: atom.NoID, Flags: 1},
		&test.AtomA{ID: 50},
		&test.AtomA{ID: 90},
		&test.AtomA{ID: atom.NoID, Flags: 2},
		&test.AtomA{ID: atom.NoID, Flags: 3},
		&test.AtomA{ID: 00},
		&test.AtomA{ID: 60},
		&test.AtomB{ID: atom.NoID},
	)

	transform := &Injector{}
	transform.Inject(30, &test.AtomA{ID: atom.NoID, Flags: 1})
	transform.Inject(90, &test.AtomA{ID: atom.NoID, Flags: 2})
	transform.Inject(90, &test.AtomA{ID: atom.NoID, Flags: 3})
	transform.Inject(60, &test.AtomB{ID: atom.NoID})

	transform.Inject(40, &test.AtomA{ID: 100, Flags: 5}) // Should not be injected

	CheckTransform(ctx, t, transform, inputs, expected)
}

// CheckTransform checks that transfomer emits the expected atoms given inputs.
func CheckTransform(ctx context.Context, t *testing.T, transformer Transformer, inputs, expected test.AtomAtomIDList) {
	mw := &test.MockAtomWriter{}
	for _, in := range inputs {
		transformer.Transform(ctx, in.Id, in.Atom, mw)
	}
	transformer.Flush(ctx, mw)

	assert.With(ctx).ThatSlice(mw.IdAtoms).DeepEquals(expected)
}
