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

package gles

import (
	"context"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// directCall is an atom-wrapper that replaces calls to the the wrapped atom's
// Replay() method with Call(), preventing any state-mutation or memory
// observations to be performed. This is used by the tests that check for GL
// errors that would otherwise be caught by the state-mutator.
type directCall struct {
	binary.Generate
	atom caller // The wrapped atom.
}

type caller interface {
	atom.Atom
	Call(ctx context.Context, s *gfxapi.State, b *builder.Builder)
}

func callerCast(obj binary.Object) caller {
	return obj.(caller)
}

func (c directCall) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	if b != nil {
		c.atom.Call(ctx, s, b)
		return nil
	}
	return c.atom.Mutate(ctx, s, nil)
}

// atom.Atom compliance
func (c directCall) API() gfxapi.API       { return c.atom.API() }
func (c directCall) AtomFlags() atom.Flags { return c.atom.AtomFlags() }
func (c directCall) Extras() *atom.Extras  { return c.atom.Extras() }
