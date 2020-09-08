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

// Package transform contains the elements to be able to transform
// commands which consist of interfaces for individual transform operations
// and a transform chain to run all of them.
package transform

import (
	"context"

	"github.com/google/gapid/gapis/api"
)

// StateMutator is a function that allows transformation to mutate state during transformation
// TODO: Refactor the transforms that use this to remove this behaviour
type StateMutator func(cmds []api.Cmd) error

// Transform is the interface that wraps the basic Transform functionality.
// Implementers of this interface, should take a list of commands and a state
// so they can do the necessary operations with them to achieve the results
// they aim to do and emit a list of commands to pass it to the next transform.
type Transform interface {
	// BeginTransform is called before transforming any command.
	BeginTransform(ctx context.Context, inputState *api.GlobalState) error

	// EndTransform is called after all commands are transformed.
	EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error)

	// TransformCommand takes a given command list and a state.
	// It outputs a new set of commands after running the transformation.
	// Transforms must not modify the input commands:
	// they can add or remove commands in the command list,
	// but they must not edit the internals of the commands that they receive as input.
	TransformCommand(ctx context.Context, id CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error)

	// ClearTransformResources releases the resources that have been allocated during transform
	// Resources are needed for the state mutation, therefore this should be called after mutation.
	ClearTransformResources(ctx context.Context)

	// RequiresAccurateState returns true if the transform needs to observe the accurate state.
	RequiresAccurateState() bool

	// RequiresInnerStateMutation returns true if the transform needs to mutate state during the transformation
	RequiresInnerStateMutation() bool

	// SetInnerStateMutationFunction sets a mutator function for making a branch in the transform
	// to transform and mutate pending commands to be able to continue transform.
	SetInnerStateMutationFunction(stateMutator StateMutator)
}

// Writer is the interface which consumes the output of transforms.
// MutateAndWrite function in this interface will be called either
// when any transform needs an accurate state or after the all transforms
// have been processed for a certain command.
// This interface can be used when it is necessary to collect the results
// of the transforms e.g. to build instructions for gapir.
// Implementers of this interface should provide a state function that
// returns the global state and a MutateAndWrite method to acknowledge the
// current command that has been processed.
type Writer interface {
	// State returns the state object associated with this writer.
	State() *api.GlobalState
	// MutateAndWrite mutates the state object associated with this writer,
	// and it passes the command to further consumers.
	MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error
}
