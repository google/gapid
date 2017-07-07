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
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
)

type MockAtomWriter struct {
	S       *api.State
	Atoms   []atom.Atom
	IdAtoms AtomAtomIDList
}

func (m *MockAtomWriter) State() *api.State {
	return m.S
}

func (m *MockAtomWriter) MutateAndWrite(ctx context.Context, id atom.ID, a atom.Atom) {
	if m.S != nil {
		a.Mutate(ctx, m.S, nil)
	}
	m.Atoms = append(m.Atoms, a)
	m.IdAtoms = append(m.IdAtoms, AtomAtomID{a, id})
}

type AtomAtomID struct {
	Atom atom.Atom
	Id   atom.ID
}

type AtomAtomIDList []AtomAtomID

// List takes a mix of Atoms and AtomAtomIDs and returns a AtomAtomIDList.
// Atoms are transformed into AtomAtomIDs by using the ID field as the atom
// id.
func List(atoms ...interface{}) AtomAtomIDList {
	l := AtomAtomIDList{}
	for _, a := range atoms {
		switch a := a.(type) {
		case *AtomA:
			l = append(l, AtomAtomID{a, a.ID})
		case *AtomB:
			l = append(l, AtomAtomID{a, a.ID})
		case AtomAtomID:
			l = append(l, a)
		default:
			panic("list only accepts types testAtom[AB] or AtomAtomID")
		}
	}
	return l
}

func (l *AtomAtomIDList) Write(ctx context.Context, id atom.ID, a atom.Atom) {
	*l = append(*l, AtomAtomID{a, id})
}

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}
