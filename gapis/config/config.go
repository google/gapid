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
	DebugDeadCodeElimination   = false
	LogExtrasInTransforms      = false // Logs all atoms' extras together with transforms
	LogMemoryInExtras          = false // Logs all atoms' read/write memory observation together with extras
	LogTransformsToFile        = false
	SeparateMutateStates       = false
)
