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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
)

// Injector is an implementation of Transformer that can inject atoms into the
// atom stream.
type Injector struct {
	injections map[atom.ID][]api.Cmd
}

// Inject emits the atom a with identifier id after the command with identifier
// after.
func (t *Injector) Inject(after atom.ID, cmd api.Cmd) {
	if t.injections == nil {
		t.injections = make(map[atom.ID][]api.Cmd)
	}
	t.injections[after] = append(t.injections[after], cmd)
}

func (t *Injector) Transform(ctx context.Context, id atom.ID, cmd api.Cmd, out Writer) {
	out.MutateAndWrite(ctx, id, cmd)

	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			out.MutateAndWrite(ctx, atom.NoID, injection)
		}
		delete(t.injections, id)
	}
}

func (t *Injector) Flush(ctx context.Context, out Writer) {}
