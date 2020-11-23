// Copyright (C) 2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License")
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

package resolve

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func createSingleCommandTrace(ctx context.Context) *path.Capture {
	h := &capture.Header{ABI: device.WindowsX86_64}
	cb := test.CommandBuilder{}
	cmds := []api.Cmd{
		cb.CmdTypeMix(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, test.Voidᵖ(0x12345678), 2),
	}
	p, err := capture.NewGraphicsCapture(ctx, "test", h, nil, cmds)
	if err != nil {
		log.F(ctx, true, "Couldn't create capture: %v", err)
	}
	path, err := p.Path(ctx)
	if err != nil {
		log.F(ctx, true, "Couldn't get capture path: %v", err)
	}
	return path
}

func createMultipleCommandTrace(ctx context.Context) *path.Capture {
	h := &capture.Header{ABI: device.WindowsX86_64}
	cb := test.CommandBuilder{}
	cmds := []api.Cmd{
		cb.CmdTypeMix(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, test.Voidᵖ(0x12345678), 2),
		cb.CmdTypeMix(1, 15, 25, 35, 45, 55, 65, 75, 85, 95, 105, false, test.Voidᵖ(0x87654321), 3),
		cb.CmdTypeMix(2, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, true, test.Voidᵖ(0xdeadfeed), 3),
	}
	p, err := capture.NewGraphicsCapture(ctx, "test", h, nil, cmds)
	if err != nil {
		log.F(ctx, true, "Couldn't create capture: %v", err)
	}
	path, err := p.Path(ctx)
	if err != nil {
		log.F(ctx, true, "Couldn't get capture path: %v", err)
	}
	return path
}

func TestDeleteSingleCommandTrace(t *testing.T) {
	ctx := log.Testing(t)
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := createSingleCommandTrace(ctx)
	ctx = capture.Put(ctx, p)

	newTracePath, err := Delete(ctx, p.Command(0).Path(), nil)
	assert.For(ctx, "Delete").ThatError(err).DeepEquals(nil)

	newCapture := newTracePath.GetCapture()
	newBoxedCommands, err := Get(ctx, newCapture.Commands().Path(), nil)
	newCommands := newBoxedCommands.(*service.Commands).List

	assert.For(ctx, "Deleted Commands").That(len(newCommands)).DeepEquals(0)
}

func TestDeleteMultipleCommandFirstElement(t *testing.T) {
	ctx := log.Testing(t)
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := createMultipleCommandTrace(ctx)
	ctx = capture.Put(ctx, p)

	commandPathsBoxed, _ := Get(ctx, p.Commands().Path(), nil)
	commandPaths := commandPathsBoxed.(*service.Commands).List

	var commands []*api.Command

	for i := 0; i < len(commandPaths); i++ {
		command, _ := Get(ctx, commandPaths[i].Path(), nil)
		commands = append(commands, command.(*api.Command))
	}

	newTracePath, err := Delete(ctx, commandPaths[0].Path(), nil)
	assert.For(ctx, "Delete").ThatError(err).DeepEquals(nil)

	newCapture := newTracePath.GetCapture()
	newBoxedCommands, err := Get(ctx, newCapture.Commands().Path(), nil)
	newCommands := newBoxedCommands.(*service.Commands).List

	assert.For(ctx, "Deleted Commands").That(len(newCommands)).DeepEquals(len(commandPaths) - 1)

	for i, test := range newCommands {
		boxedCommand, err := Get(ctx, test.Path(), nil)
		command := boxedCommand.(*api.Command)
		assert.For(ctx, "Get(%v) value", test).That(command).DeepEquals(commands[i+1])
		assert.For(ctx, "Get(%v) error", test).That(err).DeepEquals(nil)
	}
}

func TestDeleteMultipleCommandLastElement(t *testing.T) {
	ctx := log.Testing(t)
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := createMultipleCommandTrace(ctx)
	ctx = capture.Put(ctx, p)

	commandPathsBoxed, _ := Get(ctx, p.Commands().Path(), nil)
	commandPaths := commandPathsBoxed.(*service.Commands).List

	var commands []*api.Command

	for i := 0; i < len(commandPaths); i++ {
		command, _ := Get(ctx, commandPaths[i].Path(), nil)
		commands = append(commands, command.(*api.Command))
	}

	newTracePath, err := Delete(ctx, commandPaths[len(commandPaths)-1].Path(), nil)
	assert.For(ctx, "Delete").ThatError(err).DeepEquals(nil)

	newCapture := newTracePath.GetCapture()
	newBoxedCommands, err := Get(ctx, newCapture.Commands().Path(), nil)
	newCommands := newBoxedCommands.(*service.Commands).List

	assert.For(ctx, "Deleted Commands").That(len(newCommands)).DeepEquals(len(commandPaths) - 1)

	for i, test := range newCommands {
		boxedCommand, err := Get(ctx, test.Path(), nil)
		command := boxedCommand.(*api.Command)
		assert.For(ctx, "Get(%v) value", test).That(command).DeepEquals(commands[i])
		assert.For(ctx, "Get(%v) error", test).That(err).DeepEquals(nil)
	}
}
