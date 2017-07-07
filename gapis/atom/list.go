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

package atom

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// List is a list of atoms.
type List struct {
	Atoms []api.Cmd
}

func NewList(atoms ...api.Cmd) *List {
	return &List{Atoms: atoms}
}

// WriteTo writes all atoms in the list to w, terminating with a single EOS
// atom.
func (l *List) WriteTo(ctx context.Context, w Writer) {
	for i, a := range l.Atoms {
		w.Write(ctx, api.CmdID(i), a)
	}
}

// Clone makes and returns a shallow copy of the atom list.
func (l *List) Clone() *List {
	c := &List{Atoms: make([]api.Cmd, len(l.Atoms))}
	copy(c.Atoms, l.Atoms)
	return c
}

// Add appends a to the end of the atom list, returning the id of the last added
// atom.
func (l *List) Add(c ...api.Cmd) api.CmdID {
	l.Atoms = append(l.Atoms, c...)
	return api.CmdID(len(l.Atoms) - 1)
}

// AddAt adds c to the list before the atom at id.
func (l *List) AddAt(c api.Cmd, id api.CmdID) {
	l.Atoms = append(l.Atoms, nil)
	copy(l.Atoms[id+1:], l.Atoms[id:])
	l.Atoms[id] = c
}
