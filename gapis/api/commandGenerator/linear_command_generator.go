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

package commandGenerator

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

type linearCommandGenerator struct {
	commands []api.Cmd
	index    int
}

// NewLinearCommandGenerator generates a command generator that
// takes a list of cmds and returns them in order when GetNextCommand
// is called.
func NewLinearCommandGenerator(initialCommands []api.Cmd, realCommands []api.Cmd) CommandGenerator {
	return &linearCommandGenerator{
		commands: append(initialCommands, realCommands...),
		index:    0,
	}
}

func (generator *linearCommandGenerator) GetNextCommand(ctx context.Context) api.Cmd {
	if generator.index >= len(generator.commands) {
		return nil
	}

	currentCommand := generator.commands[generator.index]
	generator.index++
	return currentCommand
}

func (generator *linearCommandGenerator) IsEndOfCommands() bool {
	return generator.index >= len(generator.commands)
}
