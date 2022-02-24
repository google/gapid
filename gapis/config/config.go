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

// Package config contains a list of build configuration flags.
package config

const (
	// DebugReplay activates various debug logs related to replay.
	DebugReplay = false

	// DebugReplayBuilder activates various debug logs and checks related to the
	// creation of a replay payload.
	DebugReplayBuilder = false

	// DisableDeadCodeElimination prevents the early computation of the
	// dependency graph.
	DisableDeadCodeElimination = false

	// DeadSubCmdElimination prevents the elimination of subcommands in dead
	// code elimination.
	DeadSubCmdElimination = false

	// DebugDeadCodeElimination activates various debug logs related to dead
	// code elimination.
	DebugDeadCodeElimination = false

	// DumpReplayProfile dumps the perfetto trace of a replay profile.
	DumpReplayProfile = false

	// AllInitialCommandsLive forces all initial commands to be considered as
	// live when computing dead code elimination.
	AllInitialCommandsLive = false

	// LogExtrasInTransforms logs all commands' extras together with transforms.
	LogExtrasInTransforms = false

	// LogMemoryInExtras logs all commands' read/write memory observation
	// together with extras.
	LogMemoryInExtras = false

	// LogMappingsToFile logs all mappings at the end of the replay from
	// original trace handles to replay client handles (if handles are reused in
	// the trace it will only print the last mapping).  Only works for Vulkan.
	LogMappingsToFile = false

	// LogTransformsToFile logs all commands seen by each transform into a
	// separate file for each transform.
	LogTransformsToFile = false

	// LogTransformsToCapture creates a gfxtrace that stores the commands
	// obtained after all transforms have been applied.
	LogTransformsToCapture = false

	// LogInitialCmdsToCapture creates a gfxtrace that stores initial commands.
	LogInitialCmdsToCapture = false
)
