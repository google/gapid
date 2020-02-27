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
)

type earlyTerminator struct {
	lastID api.CmdID
	done   bool
	api    api.ID
}

// NewEarlyTerminator returns a Terminator that will consume all commands of the
// given API type that come after the last command passed to Add.
func NewEarlyTerminator(api api.ID) Terminator {
	return &earlyTerminator{api: api}
}

func (t *earlyTerminator) Add(ctx context.Context, id api.CmdID, idx api.SubCmdIdx) error {
	if id > t.lastID {
		t.lastID = id
	}
	return nil
}

func (t *earlyTerminator) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) error {
	if t.done && (cmd.API() == nil || cmd.API().ID() == t.api) {
		return nil
	}

	if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
		return err
	}
	// Keep a.API() == nil so that we can test without an API
	if t.lastID == id {
		t.done = true
	}
	return nil
}

func (t *earlyTerminator) Flush(ctx context.Context, out Writer) error { return nil }
func (t *earlyTerminator) PreLoop(ctx context.Context, output Writer)  {}
func (t *earlyTerminator) PostLoop(ctx context.Context, output Writer) {}
func (t *earlyTerminator) BuffersCommands() bool                       { return false }
