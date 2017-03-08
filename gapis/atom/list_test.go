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

package atom_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
)

var testList = atom.NewList(
	&test.AtomA{},
	&test.AtomB{Bool: true},
	&test.AtomC{String: "Pizza"},
)

type writeRecord struct {
	id   atom.ID
	atom atom.Atom
}
type writeRecordList []writeRecord

func (t *writeRecordList) Write(ctx context.Context, id atom.ID, atom atom.Atom) {
	*t = append(*t, writeRecord{id, atom})
}

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func TestAtomListWriteTo(t *testing.T) {
	ctx := log.Testing(t)
	expected := writeRecordList{
		writeRecord{0, &test.AtomA{}},
		writeRecord{1, &test.AtomB{Bool: true}},
		writeRecord{2, &test.AtomC{String: "Pizza"}},
	}
	got := writeRecordList{}
	testList.WriteTo(ctx, &got)

	matched := len(expected) == len(got)

	if matched {
		for i := range expected {
			e, g := expected[i], got[i]
			if e.id != g.id || !reflect.DeepEqual(g.atom, e.atom) {
				matched = false
				break
			}
		}
	}

	if !matched {
		c := max(len(expected), len(got))
		for i := 0; i < c; i++ {
			if i > len(got) {
				t.Errorf("(%d) Expected: %#v Got: <nothing>", i, expected[i])
				continue
			}
			e, g := expected[i], got[i]
			if e.id != g.id || !reflect.DeepEqual(g.atom, e.atom) {
				t.Errorf("(%d) Expected: %#v Got: %#v", i, e, g)
				continue
			}
			t.Logf("(%d) Matched: %#v", i, g)
		}
	}
}
