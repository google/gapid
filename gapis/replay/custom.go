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

package replay

import (
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// Custom must conform to the atom.Atom interface.
var _ = atom.Atom(Custom(nil))

// Custom is an atom issuing custom replay operations to the replay builder b upon Replay().
type Custom func(ctx log.Context, s *gfxapi.State, b *builder.Builder) error

func (c Custom) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	if b == nil {
		return nil
	}
	return c(ctx, s, b)
}

// atom.Atom compliance
func (Custom) API() gfxapi.API       { return nil }
func (Custom) AtomFlags() atom.Flags { return 0 }
func (Custom) Extras() *atom.Extras  { return nil }
func (Custom) Class() binary.Class   { return nil }
