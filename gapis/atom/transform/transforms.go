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
	"fmt"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/config"
)

// Transforms is a list of Transformer objects.
type Transforms []Transformer

// Transform sequentially transforms the atoms by each of the transformers in
// the list, before writing the final output to the output atom Writer.
func (l Transforms) Transform(ctx context.Context, atoms atom.List, out Writer) {
	var currentIndex int
	var currentAtom atom.Atom
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic at atom %v:%v:\n%v", currentIndex, currentAtom, r))
		}
	}()

	chain := out
	for i := len(l) - 1; i >= 0; i-- {
		s := out.State()
		if config.SeparateMutateStates {
			s = api.NewStateWithAllocator(s.Allocator, s.MemoryLayout)
		}
		chain = TransformWriter{s, l[i], chain}
	}
	for i, a := range atoms.Atoms {
		currentIndex, currentAtom = i, a
		chain.MutateAndWrite(ctx, atom.ID(i), a)
	}
	for p, ok := chain.(TransformWriter); ok; p, ok = chain.(TransformWriter) {
		chain = p.O
		p.T.Flush(ctx, chain)
	}
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
func Transform(name string, f func(ctx context.Context, id atom.ID, atom atom.Atom, output Writer)) Transformer {
	return transform{name, f}
}

type transform struct {
	N string                                                               // Transform name. Used for debugging.
	F func(ctx context.Context, id atom.ID, atom atom.Atom, output Writer) // The transform function.
}

func (t transform) Transform(ctx context.Context, id atom.ID, atom atom.Atom, output Writer) {
	t.F(ctx, id, atom, output)
}

func (t transform) Flush(ctx context.Context, output Writer) {}

func (t transform) Name() string { return t.N }

// TransformWriter implements the Writer interface, transforming each atom that
// is written with T, before writing the result to O.
type TransformWriter struct {
	S *api.State
	T Transformer
	O Writer
}

func (p TransformWriter) State() *api.State {
	return p.S
}

func (p TransformWriter) MutateAndWrite(ctx context.Context, id atom.ID, a atom.Atom) {
	if config.SeparateMutateStates {
		a.Mutate(ctx, p.S, nil /* no builder, just mutate */)
	}
	p.T.Transform(ctx, id, a, p.O)
}
