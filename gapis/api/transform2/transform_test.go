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

package transform2

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/commandGenerator"
	"github.com/google/gapid/gapis/api/test"
)

func TestSingleTransformTransformChain(t *testing.T) {
	ctx := log.Testing(t)

	inputCmds := []api.Cmd{createNewCmd(0, 1)}

	generator := commandGenerator.NewLinearCommandGenerator(nil, inputCmds)
	transforms := []Transform{&multiplier_transform{}}
	writer := newTestWriter()
	chain := CreateTransformChain(generator, transforms, writer)

	for !chain.IsEndOfCommands() {
		chain.GetNextTransformedCommands(ctx)
	}

	assert.For(ctx, "TestTransformChain").ThatSlice(writer.output).IsLength(2)
}

func TestMultipleTransformTransformChain(t *testing.T) {
	ctx := log.Testing(t)

	inputCmds := []api.Cmd{createNewCmd(0, 1)}

	generator := commandGenerator.NewLinearCommandGenerator(nil, inputCmds)
	transforms := []Transform{&multiplier_transform{}, &multiplier_transform{}, &multiplier_transform{}}
	writer := newTestWriter()
	chain := CreateTransformChain(generator, transforms, writer)

	for !chain.IsEndOfCommands() {
		chain.GetNextTransformedCommands(ctx)
	}

	assert.For(ctx, "TestTransformChain").ThatSlice(writer.output).IsLength(8)
}

func createNewCmd(id api.CmdID, tag uint64) api.Cmd {
	cb := test.CommandBuilder{Arena: test.Cmds.Arena}
	newCmd := func(id api.CmdID, tag uint64) api.Cmd {
		return cb.CmdTypeMix(uint64(id), 10, 20, 30, 40, 50, 60, tag, 80, 90, 100, true, test.Voidᵖ(0x12345678), 100)
	}

	return newCmd(id, tag)
}

type testWriter struct {
	output []api.Cmd
}

func newTestWriter() *testWriter {
	return &testWriter{
		output: make([]api.Cmd, 0),
	}
}

func (writer *testWriter) State() *api.GlobalState {
	return nil
}

func (writer *testWriter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	writer.output = append(writer.output, cmd)
	return nil
}

// multiplier_transform is a test transform that injects
// two cmds per cmd it receives.
type multiplier_transform struct{}

func (t *multiplier_transform) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (t *multiplier_transform) EndTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (t *multiplier_transform) TransformCommand(ctx context.Context, id api.CmdID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return multiply(inputCommands), nil
}

func (t *multiplier_transform) ClearTransformResources(ctx context.Context) {
	// Do nothing
}

func (t *multiplier_transform) RequiresAccurateState() bool {
	return false
}

func multiply(inputCommands []api.Cmd) []api.Cmd {
	outputCmds := make([]api.Cmd, 0)

	for _, cmd := range inputCommands {
		outputCmds = append(outputCmds, cmd, cmd)
	}

	return outputCmds
}
