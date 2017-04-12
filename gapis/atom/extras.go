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

// Package atom provides the fundamental types used to describe a capture stream.
package atom

import (
	"context"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/atom/atom_pb"
)

// Extra is the interface implemented by atom 'extras' - additional information
// that can be placed inside an atom instance.
type Extra interface{}

// Extras is a list of Extra objects.
type Extras []Extra

// Aborted is an extra used to mark atoms which did not finish execution.
// This can be expected (e.g. GL error), or unexpected (failed assertion).
type Aborted struct {
	IsAssert bool
	Reason   string
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *Aborted) (*atom_pb.Aborted, error) {
			return &atom_pb.Aborted{IsAssert: a.IsAssert, Reason: a.Reason}, nil
		},
		func(ctx context.Context, a *atom_pb.Aborted) (*Aborted, error) {
			return &Aborted{IsAssert: a.IsAssert, Reason: a.Reason}, nil
		},
	)
}

func (e *Extras) All() Extras {
	if e == nil {
		return nil
	}
	return *e
}

// Add appends one or more extras to the list of extras.
func (e *Extras) Add(es ...Extra) {
	if e != nil {
		*e = append(*e, es...)
	}
}

// Aborted returns a pointer to the Aborted structure in the extras, or nil if not found.
func (e *Extras) Aborted() *Aborted {
	for _, e := range e.All() {
		if e, ok := e.(*Aborted); ok {
			return e
		}
	}
	return nil
}

// Observations returns a pointer to the Observations structure in the extras,
// or nil if there are no observations in the extras.
func (e *Extras) Observations() *Observations {
	for _, o := range e.All() {
		if o, ok := o.(*Observations); ok {
			return o
		}
	}
	return nil
}

// GetOrAppendObservations returns a pointer to the existing Observations
// structure in the extras, or appends and returns a pointer to a new
// observations structure if the extras does not already contain one.
func (e *Extras) GetOrAppendObservations() *Observations {
	if o := e.Observations(); o != nil {
		return o
	}
	o := &Observations{}
	e.Add(o)
	return o
}

// WithExtras adds the given extras to an atom and returns it.
func WithExtras(a Atom, extras ...Extra) Atom {
	a.Extras().Add(extras...)
	return a
}

// Convert calls the Convert method on all the extras in the list.
func (e *Extras) Convert(ctx context.Context, out atom_pb.Handler) error {
	for _, o := range e.All() {
		m, err := protoconv.ToProto(ctx, o)
		if err != nil {
			return err
		}
		if err := out(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
