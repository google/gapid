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

import "github.com/google/gapid/core/data/deep"

// CmdExtra is the interface implemented by command 'extras' - additional
// information that can be placed inside a command.
type CmdExtra interface{}

// CmdExtras is a list of CmdExtra objects.
type CmdExtras []CmdExtra

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

// Replace replaces the `old` CmdExtra with the `new` CmdExtra.
// If `old` is not found, this does nothing.
func (e *CmdExtras) Replace(old, new CmdExtra) {
	if e != nil {
		for i, x := range *e {
			if x == old {
				(*e)[i] = new
				return
			}
		}
	}
}

// MustClone clones all of es, adding them to e.
// if there was an error, a panic is raised
func (e *CmdExtras) MustClone(es ...CmdExtra) {
	if e != nil {
		for _, ex := range es {
			i, err := deep.Clone(ex)
			if err != nil {
				panic(err)
			}
			*e = append(*e, i)
		}
	}
}

// CloneObservations makes a shallow clone of the extras, except for the
// observations, which are cloned.
func (e *CmdExtras) CloneObservations() {
	if e != nil {
		r := CmdExtras{}
		for _, x := range *e {
			if o, ok := x.(*CmdObservations); ok {
				r = append(r, &CmdObservations{o.Reads, o.Writes})
			} else {
				r = append(r, x)
			}
		}
		*e = r
	}
}

// Aborted returns a pointer to the ErrCmdAborted structure in the CmdExtras, or
// nil if not found.
func (e *CmdExtras) Aborted() *ErrCmdAborted {
	for _, e := range e.All() {
		if e, ok := e.(*ErrCmdAborted); ok {
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
