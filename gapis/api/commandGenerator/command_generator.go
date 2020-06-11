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

// Package commandGenerator includes the interface and the implementation
// for generating commands to process.
package commandGenerator

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// CommandGenerator is the interface to get commands from an arbitrary source.
// Implementers of this interface should be able to produce commands
// when asked until they cannot produce any new command.
type CommandGenerator interface {
	// GetNextCommand should return a cmd produced by CommandGenerator
	GetNextCommand(ctx context.Context) api.Cmd

	// IsEndOfCommands should return true when there is no more command
	// to be produced.
	IsEndOfCommands() bool
}
