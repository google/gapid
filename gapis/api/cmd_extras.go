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

package api

import (
	"context"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/atom/atom_pb"
)

// CmdExtra is the interface implemented by command 'extras' - additional
// information that can be placed inside a command.
type CmdExtra interface{}

// CmdExtras is a list of CmdExtra objects.
type CmdExtras []CmdExtra

// Aborted is an CmdExtra used to mark atoms which did not finish execution.
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

func (e *CmdExtras) All() CmdExtras {
	if e == nil {
		return nil
	}
	return *e
}

// Add appends one or more CmdExtras to the list of CmdExtras.
func (e *CmdExtras) Add(es ...CmdExtra) {
	if e != nil {
		*e = append(*e, es...)
	}
}

// Aborted returns a pointer to the Aborted structure in the CmdExtras, or nil if not found.
func (e *CmdExtras) Aborted() *Aborted {
	for _, e := range e.All() {
		if e, ok := e.(*Aborted); ok {
			return e
		}
	}
	return nil
}

// Observations returns a pointer to the CmdObservations structure in the
// CmdExtras, or nil if there are no observations in the CmdExtras.
func (e *CmdExtras) Observations() *CmdObservations {
	for _, o := range e.All() {
		if o, ok := o.(*CmdObservations); ok {
			return o
		}
	}
	return nil
}

// GetOrAppendObservations returns a pointer to the existing Observations
// structure in the CmdExtras, or appends and returns a pointer to a new
// observations structure if the CmdExtras does not already contain one.
func (e *CmdExtras) GetOrAppendObservations() *CmdObservations {
	if o := e.Observations(); o != nil {
		return o
	}
	o := &CmdObservations{}
	e.Add(o)
	return o
}

// WithExtras adds the given extras to a command and returns it.
func WithExtras(a Cmd, extras ...CmdExtra) Cmd {
	a.Extras().Add(extras...)
	return a
}

// Convert calls the Convert method on all the extras in the list.
func (e *CmdExtras) Convert(ctx context.Context, out atom_pb.Handler) error {
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
