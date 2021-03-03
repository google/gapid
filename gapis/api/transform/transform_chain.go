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

package transform

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/commandGenerator"
	"github.com/google/gapid/gapis/config"
)

// TransformChain is responsible for running all the transforms on
// all the commands produced by command enerator
type TransformChain struct {
	transforms            []Transform
	out                   *commandCaptureWriter
	generator             commandGenerator.CommandGenerator
	hasBegun              bool
	hasEnded              bool
	currentCommandID      CommandID
	currentTransformIndex int
	mutator               StateMutator
}

// CreateTransformChain creates a transform chain that will run
// the required transforms on every command coming from commandGenerator
func CreateTransformChain(ctx context.Context, generator commandGenerator.CommandGenerator, transforms []Transform, out Writer) *TransformChain {
	chain := TransformChain{
		generator:        generator,
		transforms:       transforms,
		out:              createCommandCaptureWriter(out),
		hasBegun:         false,
		hasEnded:         false,
		currentCommandID: NewTransformCommandID(0),
		mutator:          nil,
	}

	chain.mutator = func(cmds []api.Cmd) error {
		return chain.stateMutator(ctx, cmds)
	}

	for _, t := range chain.transforms {
		if t.RequiresAccurateState() {
			// Melih TODO: Temporary check until we implement accurate state
			panic("Implement accurate state")
		}

		if t.RequiresInnerStateMutation() {
			t.SetInnerStateMutationFunction(chain.mutator)
		}
	}

	return &chain
}

func (chain *TransformChain) beginChain(ctx context.Context) error {

	chain.handleInitialState(chain.out.State()) // TODO: Why bother with the arg?
	var err error

	for _, transform := range chain.transforms {

		err = transform.BeginTransform(ctx, chain.out.State())

		if err != nil {
			log.W(ctx, "Begin Transform Error [%v] : %v", transform, err)
			return err
		}

		if transform.RequiresAccurateState() {
			// Melih TODO: Temporary check until we implement accurate state
			panic("Implement accurate state")
		}
	}

	for _, transform := range chain.transforms {
		transform.ClearTransformResources(ctx)
	}

	return nil
}

func (chain *TransformChain) endChain(ctx context.Context) error {

	chain.currentCommandID = NewEndCommandID()

	for i, transform := range chain.transforms {

		cmds, err := transform.EndTransform(ctx, chain.out.State())

		if err != nil {
			log.W(ctx, "End Transform Command Generation Error [%v] : %v", transform, err)
			return err
		}

		if cmds == nil {
			continue
		}

		if err = chain.transformCommands(ctx, chain.currentCommandID, cmds, i+1); err != nil {
			log.W(ctx, "End Transform Error [%v] : %v", transform, err)
			return err
		}
	}

	for _, transform := range chain.transforms {
		transform.ClearTransformResources(ctx)
	}

	return nil
}

func (chain *TransformChain) transformCommands(ctx context.Context, id CommandID, inputCmds []api.Cmd, beginTransformIndex int) error {

	cmds := inputCmds

	for i := beginTransformIndex; i < len(chain.transforms); i++ {

		chain.currentTransformIndex = i
		var err error
		cmds, err = chain.transforms[i].TransformCommand(ctx, id, cmds, chain.out.State())

		if err != nil {
			log.W(ctx, "Error on Transform on cmd [%v:%v] with transform [:%v:%v] : %v", id, cmds, i, chain.transforms[i], err)
			return err
		}

		if chain.transforms[i].RequiresAccurateState() {
			// Melih TODO: Temporary check until we implement accurate state
			panic("Implement accurate state")
		}

	}

	if err := mutateAndWrite(ctx, id, cmds, chain.out); err != nil {
		return err
	}

	if beginTransformIndex == 0 {
		for _, transform := range chain.transforms {
			transform.ClearTransformResources(ctx)
		}
	}

	return nil
}

func (chain *TransformChain) IsEndOfCommands() bool {
	return chain.hasEnded
}

func (chain *TransformChain) GetCurrentCommandID() CommandID {
	return chain.currentCommandID
}

func (chain *TransformChain) GetNumOfRemainingCommands() uint64 {
	return chain.generator.GetNumOfRemainingCommands()
}

func (chain *TransformChain) ProcessNextTransformedCommands(ctx context.Context) ([]api.Cmd, error) {

	if !chain.hasBegun {
		chain.hasBegun = true
		if err := chain.beginChain(ctx); err != nil {
			return nil, err
		}
	}

	if chain.generator.IsEndOfCommands() {
		if !chain.hasEnded {
			chain.hasEnded = true
			err := chain.endChain(ctx)
			if err != nil {
				return nil, err
			}
			return chain.out.GetAndResetCapturedCommands(ctx), err
		}

		return nil, nil
	}

	currentCommand := chain.generator.GetNextCommand(ctx)
	if !currentCommand.Terminated() {
		// Run only the commands that terminated during trace time.
		return nil, nil
	}
	if config.DebugReplay {
		log.I(ctx, "Transforming... (%v:%v)", chain.currentCommandID, currentCommand)
	}

	inputCmds := make([]api.Cmd, 0)
	inputCmds = append(inputCmds, currentCommand)
	err := chain.transformCommands(ctx, chain.currentCommandID, inputCmds, 0)
	if err != nil {
		log.E(ctx, "Replay error (%v:%v): %v", chain.currentCommandID, currentCommand, err)
	}

	chain.currentCommandID.Increment()
	return chain.out.GetAndResetCapturedCommands(ctx), err
}

func (chain *TransformChain) stateMutator(ctx context.Context, cmds []api.Cmd) error {
	// When we are mutating the state in the middle of the transform
	// we want to transform the rest of the transforms first
	beginTransformIndex := chain.currentTransformIndex + 1

	if err := chain.transformCommands(ctx, chain.currentCommandID, cmds, beginTransformIndex); err != nil {
		log.E(ctx, "state mutator error (%v:%v): %v", chain.currentCommandID, cmds, err)
		return err
	}

	// After we finish state mutation we have to restore the value of the currentTransformIndex
	// that has modified during the state mutation, so that we can continue the current transform
	// properly.
	chain.currentTransformIndex = beginTransformIndex - 1
	return nil
}

func (chain *TransformChain) handleInitialState(state *api.GlobalState) (*api.GlobalState, error) {
	for _, t := range chain.transforms {
		if t.RequiresAccurateState() {
			// Melih TODO: Temporary check until we implement accurate state
			panic("Implement accurate state")
		}
	}

	return state, nil
}

func (chain *TransformChain) State() *api.GlobalState {
	return chain.out.State()
}

func mutateAndWrite(ctx context.Context, id CommandID, cmds []api.Cmd, out Writer) error {
	cmdID := api.CmdID(0)
	if id.GetCommandType() == TransformCommand {
		cmdID = id.GetID()
	}

	for i, cmd := range cmds {
		if err := out.MutateAndWrite(ctx, cmdID, cmd); err != nil {
			log.W(ctx, "State mutation error in %v, [%v:%v]) : %v", id.GetCommandType(), cmdID, i, cmd)
			return err
		}
	}

	return nil
}

type commandCaptureWriter struct {
	out  Writer
	cmds []api.Cmd
}

func createCommandCaptureWriter(out Writer) *commandCaptureWriter {
	writer := commandCaptureWriter{
		out:  out,
		cmds: nil,
	}
	return &writer
}

func (w *commandCaptureWriter) State() *api.GlobalState {
	return w.out.State()
}

func (w *commandCaptureWriter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	w.cmds = append(w.cmds, cmd)
	return w.out.MutateAndWrite(ctx, id, cmd)
}

func (w *commandCaptureWriter) GetAndResetCapturedCommands(ctx context.Context) []api.Cmd {
	ret := w.cmds
	w.cmds = nil
	return ret
}
