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
	"context"

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
)

// Custom must conform to the api.Cmd interface.
var _ = api.Cmd(Custom{})

// Custom is a command issuing custom replay operations to the replay builder b
// upon Replay().
type Custom struct {
	T uint64 // The thread ID
	F func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error
}

func (c Custom) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState,
	b *builder.Builder, w api.StateWatcher) error {
	if b == nil {
		return nil
	}
	return c.F(ctx, s, b)
}

// api.Cmd compliance
func (cmd Custom) Thread() uint64            { return cmd.T }
func (cmd Custom) SetThread(t uint64)        { cmd.T = t }
func (Custom) CmdName() string               { return "<Custom>" }
func (Custom) API() api.API                  { return nil }
func (Custom) CmdParams() api.Properties     { return nil }
func (Custom) CmdResult() *api.Property      { return nil }
func (Custom) CmdFlags() api.CmdFlags        { return 0 }
func (Custom) Extras() *api.CmdExtras        { return nil }
func (cmd Custom) Clone(arena.Arena) api.Cmd { return Custom{cmd.T, cmd.F} }
func (Custom) Alive() bool                   { return false }
func (Custom) Terminated() bool              { return false }
func (Custom) SetTerminated(bool)            {}
