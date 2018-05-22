// Copyright (C) 2018 Google Inc.
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
	"fmt"

	"github.com/google/gapid/gapis/memory"
)

// StateWatcher provides callbacks to track state effects
type StateWatcher interface {
	// OnBeginCmd is called at the beginning of each API call
	OnBeginCmd(ctx context.Context, cmdID CmdID, cmd Cmd)

	// OnEndCmd is called at the end of each API call
	OnEndCmd(ctx context.Context, cmdID CmdID, cmd Cmd)

	// OnGet is called when a fragment of state (field, map key, array index) is read
	OnGet(ctx context.Context, owner Reference, f Fragment, v Reference)

	// OnSet is called when a fragment of state (field, map key, array index) is written
	OnSet(ctx context.Context, owner Reference, f Fragment, old Reference, new Reference)

	// OnWriteSlice is called when writing to a slice
	OnWriteSlice(ctx context.Context, s memory.Slice)

	// OnReadSlice is called when reading from a slice
	OnReadSlice(ctx context.Context, s memory.Slice)

	// OnWriteObs is called when a memory write observations become visible
	OnWriteObs(ctx context.Context, obs []CmdObservation)

	// OnReadObs is called when a memory read observations become visible
	OnReadObs(ctx context.Context, obs []CmdObservation)

	// OpenForwardDependency is called to begin a forward dependency.
	// When `CloseForwardDependency` is called later with the same `dependencyID`,
	// a dependency is added from the current command node during the
	// `OpenForwardDependency` to the current command node during the
	// `CloseForwardDependency` call.
	// Each `OpenForwardDependency` call should have at most one matching
	// `CloseForwardDependency` call; additional `CloseForwardDependency`
	// calls with the same `dependencyID` will **not** result in additional
	// forward dependencies.
	OpenForwardDependency(ctx context.Context, dependencyID interface{})

	// CloseForwardDependency is called to end a forward dependency.
	// See `OpenForwardDependency` for an explanation of forward dependencies.
	CloseForwardDependency(ctx context.Context, dependencyID interface{})
}

// Fragment is an interface which marks types which identify pieces of API objects.
// All of the implementations appear below.
type Fragment interface {
	fragment()
}

// FieldFragment is a Fragment identifying a field member of an API object.
// This corresponds to API syntax such as `myObj.fieldName`.
type FieldFragment struct {
	Field
}

func (f FieldFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, ".%s", f.Field.FieldName()) }

func (FieldFragment) fragment() {}

// ArrayIndexFragment is a Fragment identifying an array index.
// This corresponds to syntax such as `myArray[3]`.
type ArrayIndexFragment struct {
	Index int
}

func (f ArrayIndexFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[%d]", f.Index) }

func (ArrayIndexFragment) fragment() {}

// MapIndexFragment is a Fragment identifying a map index.
// This corresponds to syntax such as `myMap["foo"]`
type MapIndexFragment struct {
	Index interface{}
}

func (f MapIndexFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[%v]", f.Index) }

func (MapIndexFragment) fragment() {}

// CompleteFragment is a Fragment identifying the entire object (all fields),
// map (all key/value pairs) or array (all values).
type CompleteFragment struct{}

func (f CompleteFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[*]") }

func (CompleteFragment) fragment() {}

// Field identifies a field in an API object
type Field interface {
	FieldName() string
	ClassName() string
}
