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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
)

// Injector is an implementation of Transformer that can inject atoms into the
// atom stream.
type Injector struct {
	injections map[atom.ID][]atom.Atom
}

// Inject emits the atom a with identifier id after the atom with identifier
// after.
func (t *Injector) Inject(after atom.ID, a atom.Atom) {
	if t.injections == nil {
		t.injections = make(map[atom.ID][]atom.Atom)
	}
	t.injections[after] = append(t.injections[after], a)
}

func (t *Injector) Transform(ctx log.Context, id atom.ID, a atom.Atom, out Writer) {
	out.MutateAndWrite(ctx, id, a)

	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			out.MutateAndWrite(ctx, atom.NoID, injection)
		}
		delete(t.injections, id)
	}
}

func (t *Injector) Flush(ctx log.Context, out Writer) {}
