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

package api

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/gapis/replay/builder"
)

// ForeachCmd calls the callback cb for each command in cmds.
// If cb returns an error (excluding Break) then the iteration will stop
// and the error will be returned. If cb returns Break then the iteration
// will stop and nil will be returned.
// ForeachCmd creates a non-cancellable sub-context to reduce cancellation
// complexity in the callback function.
// If cb panics, the error will be annotated with the panicing command index and
// command.
// onlyTerminated: skip commands that were not terminated as capture time

func ForeachCmd(ctx context.Context, cmds []Cmd, onlyTerminated bool, cb func(context.Context, CmdID, Cmd) error) error {
	ctx = status.Start(ctx, "ForeachCmd<count: %v>", len(cmds))
	defer status.Finish(ctx)

	var idx CmdID
	var cmd Cmd
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic at command %v:%v:\n%v", idx, cmd, r))
		}
	}()

	subctx := keys.Clone(context.Background(), ctx)
	for i, c := range cmds {
		// Only update once every 1000 commands
		if (i % 1000) == 999 {
			status.UpdateProgress(ctx, uint64(i), uint64(len(cmds)))
		}
		idx, cmd = CmdID(i), c
		if onlyTerminated && !cmd.Terminated() {
			continue
		}
		if err := cb(subctx, idx, cmd); err != nil {
			if err != Break {
				return err
			}
			return nil
		}
		if err := task.StopReason(ctx); err != nil {
			return err
		}
	}

	return nil
}

// MutateCmds calls Mutate on each of cmds.
func MutateCmds(ctx context.Context, state *GlobalState, builder *builder.Builder,
	watcher StateWatcher, cmds ...Cmd) error {
	return ForeachCmd(ctx, cmds, true, func(ctx context.Context, id CmdID, cmd Cmd) error {
		if err := cmd.Mutate(ctx, id, state, builder, watcher); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
		return nil
	})
}

// Break can be returned from the callback passed to ForeachCmd to stop
// iteration of the loop.
const Break tyBreak = tyBreak(0)

type tyBreak int

func (tyBreak) Error() string { return "<break>" }
