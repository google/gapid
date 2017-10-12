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

// Package vulkan implementes the API interface for the Vulkan graphics library.

package vulkan

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
)

type CommandBufferCommand struct {
	function func(context.Context, api.Cmd, api.CmdID, *api.GlobalState, *builder.Builder)
}

type QueuedCommand struct {
	initialCall      api.Cmd
	submit           api.Cmd
	submissionIndex  []uint64
	actualSubmission bool
}

type CommandBufferCommands []CommandBufferCommand
