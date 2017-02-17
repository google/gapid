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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
)

// Extra is the interface implemented by atom 'extras' - additional information
// that can be placed inside an atom instance.
type Extra interface {
	binary.Object
}

// ExtraCast is automatically called by the generated decoders.
func ExtraCast(obj binary.Object) Extra { return obj.(Extra) }

// Extras is a list of Extra objects.
type Extras []Extra

// Aborted is an extra used to mark atoms which did not finish execution.
// This can be expected (e.g. GL error), or unexpected (failed assertion).
type Aborted struct {
	binary.Generate

	IsAssert bool
	Reason   string
}

func (a *Aborted) Convert(ctx log.Context, out atom_pb.Handler) error {
	return out(ctx, &atom_pb.Aborted{
		IsAssert: a.IsAssert,
		Reason:   a.Reason,
	})
}

func AbortedFrom(from *atom_pb.Aborted) Aborted {
	return Aborted{
		IsAssert: from.IsAssert,
		Reason:   from.Reason,
	}
}

func (extras *Extras) All() Extras {
	if extras == nil {
		return nil
	} else {
		return *extras
	}
}

// Add appends one or more extras to the list of extras.
func (extras *Extras) Add(es ...Extra) {
	if extras != nil {
		*extras = append(*extras, es...)
	}
}

// Aborted returns a pointer to the Aborted structure in the extras, or nil if not found.
func (extras *Extras) Aborted() *Aborted {
	for _, e := range extras.All() {
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
func (e *Extras) Convert(ctx log.Context, out atom_pb.Handler) error {
	for _, o := range e.All() {
		c, ok := o.(Convertible)
		if !ok {
			return ErrNotConvertible
		}
		if err := c.Convert(ctx, out); err != nil {
			return err
		}
	}
	return nil
}
