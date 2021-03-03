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

package controlFlowGenerator

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/gapis/api/transform"
)

type linearControlFlowGenerator struct {
	chain *transform.TransformChain
}

// NewLinearControlFlowGenerator generates a simple control flow
// that takes initial and real commands and transforms all of them
func NewLinearControlFlowGenerator(chain *transform.TransformChain) ControlFlowGenerator {
	return &linearControlFlowGenerator{
		chain: chain,
	}
}

func (cf *linearControlFlowGenerator) TransformAll(ctx context.Context) error {
	numberOfCmds := cf.chain.GetNumOfRemainingCommands()
	ctx = status.Start(ctx, "Running LinearControlFlow <count:%v>", numberOfCmds)
	defer status.Finish(ctx)

	var id transform.CommandID
	defer recoverFromPanic(id)

	subctx := keys.Clone(context.Background(), ctx)
	for !cf.chain.IsEndOfCommands() {
		id = cf.chain.GetCurrentCommandID()
		if id.GetCommandType() == transform.TransformCommand {
			cmdID := uint64(id.GetID())
			if cmdID%100 == 99 {
				status.UpdateProgress(ctx, cmdID, numberOfCmds)
			}
		}

		if _, err := cf.chain.ProcessNextTransformedCommands(subctx); err != nil {
			return err
		}

		if err := task.StopReason(ctx); err != nil {
			return err
		}
	}

	return nil
}

func recoverFromPanic(id transform.CommandID) {
	r := recover()
	if r == nil {
		return
	}

	switch id.GetCommandType() {
	case transform.TransformCommand:
		panic(fmt.Errorf("Panic at command %v\n%v", id.GetID(), r))
	case transform.EndCommand:
		panic(fmt.Errorf("Panic at end command\n%v", r))
	default:
		panic(fmt.Errorf("Panic at Unknown command type\n%v", r))
	}
}
