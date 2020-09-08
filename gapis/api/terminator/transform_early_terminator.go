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

package terminator

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

var _ Terminator = &earlyTerminator{}

type earlyTerminator struct {
	lastID     api.CmdID
	terminated bool
}

// NewEarlyTerminator returns a Terminator that will consume all commands
// after the last command passed to Add.
func NewEarlyTerminator() Terminator {
	return &earlyTerminator{
		lastID:     0,
		terminated: false,
	}
}

func (earlyTerminatorTransform *earlyTerminator) Add(ctx context.Context, id api.CmdID, idx api.SubCmdIdx) error {
	if id > earlyTerminatorTransform.lastID {
		earlyTerminatorTransform.lastID = id
	}

	return nil
}

func (earlyTerminatorTransform *earlyTerminator) RequiresAccurateState() bool {
	return false
}

func (earlyTerminatorTransform *earlyTerminator) RequiresInnerStateMutation() bool {
	return false
}

func (earlyTerminatorTransform *earlyTerminator) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (earlyTerminatorTransform *earlyTerminator) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	return nil
}

func (earlyTerminatorTransform *earlyTerminator) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (earlyTerminatorTransform *earlyTerminator) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if earlyTerminatorTransform.terminated {
		return nil, nil
	}

	if id.GetID() == earlyTerminatorTransform.lastID {
		earlyTerminatorTransform.terminated = true
	}

	return inputCommands, nil
}

func (earlyTerminatorTransform *earlyTerminator) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}
