// Copyright (C) 2020 Google Inc.
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

package vulkan

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
)

// InsertionCommand is a temporary command
// that is expected to be replaced by a down-stream transform.
type InsertionCommand struct {
	cmdBuffer             VkCommandBuffer
	pendingCommandBuffers []VkCommandBuffer
	idx                   api.SubCmdIdx
	callee                api.Cmd
}

// Interface check
var _ api.Cmd = &InsertionCommand{}

func (*InsertionCommand) Mutate(ctx context.Context, cmd api.CmdID, g *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	if b != nil {
		return fmt.Errorf("This command should have been replaced before it got to the builder")
	}
	return nil
}

func (s *InsertionCommand) Thread() uint64 {
	return s.callee.Thread()
}

func (s *InsertionCommand) SetThread(c uint64) {
	s.callee.SetThread(c)
}

// CmdName returns the name of the command.
func (s *InsertionCommand) CmdName() string {
	return "CommandBufferInsertion"
}

func (s *InsertionCommand) CmdParams() api.Properties {
	return api.Properties{}
}

func (s *InsertionCommand) CmdResult() *api.Property {
	return nil
}

func (s *InsertionCommand) CmdFlags() api.CmdFlags {
	return 0
}

func (s *InsertionCommand) Extras() *api.CmdExtras {
	return nil
}

func (s *InsertionCommand) Clone(a arena.Arena) api.Cmd {
	return &InsertionCommand{
		s.cmdBuffer,
		append([]VkCommandBuffer{}, s.pendingCommandBuffers...),
		s.idx,
		s.callee.Clone(a),
	}
}

func (s *InsertionCommand) Alive() bool {
	return true
}

func (s *InsertionCommand) Terminated() bool {
	return true
}

func (s *InsertionCommand) SetTerminated(bool) {
}

func (s *InsertionCommand) API() api.API {
	return s.callee.API()
}
