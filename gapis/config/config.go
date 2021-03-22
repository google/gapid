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
	DebugReplay                = false
	DebugReplayBuilder         = false
	DisableDeadCodeElimination = false
	DeadSubCmdElimination      = false
	DebugDeadCodeElimination   = false
	DebugDependencyGraph       = false
	DumpReplayProfile          = false
	DumpValidationTrace        = true
	AllInitialCommandsLive     = false
	LogExtrasInTransforms      = false // Logs all commands' extras together with transforms
	LogMemoryInExtras          = false // Logs all commands' read/write memory observation together with extras
	// Logs all mappings at the end of the replay from original trace
	// handles to replay client handles (if handles are reused in the trace
	// it will only print the last mapping).  Only works for Vulkan.
	LogMappingsToFile        = false
	LogTransformsToFile      = false
	LogTransformsToCapture   = false
	LogInitialCmdsIssues     = false
	LogInitialCmdsToCapture  = false
	SeparateMutateStates     = false
	CheckRebuiltStateMatches = false
)
