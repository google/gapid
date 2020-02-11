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

package transform

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/config"
)

// Transforms is a list of Transformer objects.
type Transforms []Transformer

// TransformAll sequentially transforms the commands by each of the transformers in
// the list, before writing the final output to the output command Writer.
func (l Transforms) TransformAll(ctx context.Context, cmds []api.Cmd, numberOfInitialCommands uint64, out Writer) error {
	chain := out
	for i := len(l) - 1; i >= 0; i-- {
		s := chain.State()
		if config.SeparateMutateStates || (i+1 < len(l) && l[i+1].BuffersCommands()) {
			newState := api.NewStateWithAllocator(s.Allocator, s.MemoryLayout)
			newState.Memory = s.Memory.Clone()
			for k, v := range s.APIs {
				clonedState := v.Clone(newState.Arena)
				clonedState.SetupInitialState(ctx)
				newState.APIs[k] = clonedState
			}
			s = newState
		}
		chain = TransformWriter{s, l[i], chain}
	}
	err := api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		captureCmdID := api.CmdNoID
		if uint64(id) > numberOfInitialCommands {
			captureCmdID = id - api.CmdID(numberOfInitialCommands)
		}

		return chain.MutateAndWrite(ctx, captureCmdID, cmd)
	})
	if err != nil {
		return err
	}
	for p, ok := chain.(TransformWriter); ok; p, ok = chain.(TransformWriter) {
		chain = p.O
		if err := p.T.Flush(ctx, chain); err != nil {
			return err
		}
	}
	return nil
}

// Add is a convenience function for appending the list of Transformers t to the
// end of the Transforms list, after filtering out nil Transformers.
func (l *Transforms) Add(t ...Transformer) {
	for _, tr := range t {
		if tr != nil {
			*l = append(*l, tr)
		}
	}
}

// Prepend adds the given transformer to the beginning of the transform chain.
func (l *Transforms) Prepend(t Transformer) {
	*l = append([]Transformer{t}, *l...)
}

// Transform is a helper for building simple Transformers that are implemented
// by function f. name is used to identify the transform when logging.
func Transform(name string, f func(ctx context.Context, id api.CmdID, cmd api.Cmd, output Writer) error) Transformer {
	return transform{name, f}
}

type transform struct {
	N string                                                  // Transform name. Used for debugging.
	F func(context.Context, api.CmdID, api.Cmd, Writer) error // The transform function.
}

func (t transform) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, output Writer) error {
	return t.F(ctx, id, cmd, output)
}

func (t transform) Flush(ctx context.Context, output Writer) error { return nil }

func (t transform) Name() string { return t.N }

func (t transform) PreLoop(ctx context.Context, output Writer)  {}
func (t transform) PostLoop(ctx context.Context, output Writer) {}
func (t transform) BuffersCommands() bool                       { return false }

// TransformWriter implements the Writer interface, transforming each command
// that is written with T, before writing the result to O.
type TransformWriter struct {
	S *api.GlobalState
	T Transformer
	O Writer
}

func (p TransformWriter) State() *api.GlobalState {
	return p.S
}

func (p TransformWriter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	if config.SeparateMutateStates || p.O.State() != p.S {
		if err := cmd.Mutate(ctx, id, p.S, nil, nil /* no builder, no watcher, just mutate */); err != nil {
			return err
		}
	}
	return p.T.Transform(ctx, id, cmd, p.O)
}

// NotifyPreLoop notifies next transformer in the chain about the beginning of the loop
func (p TransformWriter) NotifyPreLoop(ctx context.Context) {
	p.T.PreLoop(ctx, p.O)
}

// NotifyPostLoop notifies next transformer in the chain about the end of the loop
func (p TransformWriter) NotifyPostLoop(ctx context.Context) {
	p.T.PostLoop(ctx, p.O)
}
