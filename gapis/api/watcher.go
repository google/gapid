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
	"math/bits"

	"github.com/google/gapid/gapis/memory"
)

// StateWatcher provides callbacks to track state effects
type StateWatcher interface {
	// OnBeginCmd is called at the beginning of each API call
	OnBeginCmd(ctx context.Context, cmdID CmdID, cmd Cmd)

	// OnEndCmd is called at the end of each API call
	OnEndCmd(ctx context.Context, cmdID CmdID, cmd Cmd)

	// OnBeginSubCmd is called at the beginning of each subcommand execution
	OnBeginSubCmd(ctx context.Context, subCmdIdx SubCmdIdx, recordIdx RecordIdx)

	// OnEndSubCmd is called at the end of each subcommand execution
	OnEndSubCmd(ctx context.Context)

	// OnGet is called when a fragment of state (field, map key, array index) is read
	OnReadFrag(ctx context.Context, owner RefObject, f Fragment, v RefObject, track bool)

	// OnSet is called when a fragment of state (field, map key, array index) is written
	OnWriteFrag(ctx context.Context, owner RefObject, f Fragment, old RefObject, new RefObject, tracke bool)

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

	// DropForwardDependency is called to abandon a previously opened
	// forward dependency, without actually adding the forward dependency.
	// See `OpenForwardDependency` for an explanation of forward dependencies.
	DropForwardDependency(ctx context.Context, dependencyID interface{})

	// OnRecordSubCmd is called when a subcommand is recorded.
	OnRecordSubCmd(ctx context.Context, recordIdx RecordIdx)
}

type RecordIdx []uint64

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

func (f FieldFragment) DenseIndex() int { return 1 + f.Field.FieldIndex() }

func (FieldFragment) fragment() {}

// ArrayIndexFragment is a Fragment identifying an array index.
// This corresponds to syntax such as `myArray[3]`.
type ArrayIndexFragment struct {
	ArrayIndex int
}

func (f ArrayIndexFragment) DenseIndex() int            { return 1 + f.ArrayIndex }
func (f ArrayIndexFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[%d]", f.ArrayIndex) }

func (ArrayIndexFragment) fragment() {}

// MapIndexFragment is a Fragment identifying a map index.
// This corresponds to syntax such as `myMap["foo"]`
type MapIndexFragment struct {
	MapIndex interface{}
}

func (f MapIndexFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[%v]", f.MapIndex) }

func (MapIndexFragment) fragment() {}

// CompleteFragment is a Fragment identifying the entire object (all fields),
// map (all key/value pairs) or array (all values).
type CompleteFragment struct{}

func (CompleteFragment) DenseIndex() int            { return 0 }
func (CompleteFragment) Format(s fmt.State, r rune) { fmt.Fprintf(s, "[*]") }

func (CompleteFragment) fragment() {}

// Field identifies a field in an API object
type Field interface {
	FieldName() string
	FieldIndex() int
	ClassName() string
}

type FragmentMap interface {
	Get(Fragment) (interface{}, bool)
	Set(Fragment, interface{})
	Delete(Fragment)
	Clear()
	ForeachFrag(func(Fragment, interface{}) error) error
	EmptyClone() FragmentMap
}

type DenseFragment interface {
	DenseIndex() int
}

type denseFragmentMapEntry struct {
	frag  Fragment
	value interface{}
}

type DenseFragmentMap struct {
	Values []denseFragmentMapEntry
}

func NewDenseFragmentMap(cap int) *DenseFragmentMap {
	return &DenseFragmentMap{
		Values: make([]denseFragmentMapEntry, cap),
	}
}

func (m DenseFragmentMap) Get(f Fragment) (interface{}, bool) {
	if d, ok := f.(DenseFragment); ok {
		i := d.DenseIndex()
		if i < len(m.Values) {
			a := m.Values[i]
			if a.frag != nil {
				if a.frag != f {
					panic("Collision in DenseFragmentmap")
				}
				return a.value, true
			}
			return nil, false
		}
		return nil, false
	} else {
		panic("DenseFragmentMap used with non-dense fragment")
	}
}

func (m *DenseFragmentMap) Set(f Fragment, v interface{}) {
	if d, ok := f.(DenseFragment); ok {
		i := d.DenseIndex()
		n := len(m.Values)
		if i >= n {
			n := 1 << uint(bits.Len(uint(i)))
			newVals := make([]denseFragmentMapEntry, n)
			copy(newVals, m.Values)
			m.Values = newVals
		}
		a := &m.Values[i]
		if a.frag != nil && a.frag != f {
			panic("Collision in DenseFragmentMap")
		}
		a.frag = f
		a.value = v
	} else {
		panic("DenseFragmentMap used with non-dense fragment")
	}
}

func (m DenseFragmentMap) Delete(f Fragment) {
	if d, ok := f.(DenseFragment); ok {
		i := d.DenseIndex()
		if i < len(m.Values) {
			m.Values[i] = denseFragmentMapEntry{}
		}
	} else {
		panic("DenseFragmentMap used with non-dense fragment")
	}
}

func (m DenseFragmentMap) ForeachFrag(f func(Fragment, interface{}) error) error {
	for _, v := range m.Values {
		if v.frag == nil {
			continue
		}
		if err := f(v.frag, v.value); err != nil {
			return err
		}
	}
	return nil
}

func (m DenseFragmentMap) Clear() {
	for _, e := range m.Values {
		e.frag = nil
	}
}

func (m DenseFragmentMap) EmptyClone() FragmentMap {
	return NewDenseFragmentMap(len(m.Values))
}

type SparseFragmentMap struct {
	Map map[Fragment]interface{}
}

func NewSparseFragmentMap() *SparseFragmentMap {
	return &SparseFragmentMap{
		Map: make(map[Fragment]interface{}),
	}
}

func (m SparseFragmentMap) Get(f Fragment) (interface{}, bool) {
	v, ok := m.Map[f]
	return v, ok
}

func (m *SparseFragmentMap) Set(f Fragment, v interface{}) {
	m.Map[f] = v
}

func (m SparseFragmentMap) Delete(f Fragment) {
	delete(m.Map, f)
}

func (m SparseFragmentMap) ForeachFrag(f func(Fragment, interface{}) error) error {
	for k, v := range m.Map {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (m SparseFragmentMap) Clear() {
	m.Map = make(map[Fragment]interface{})
}

func (m SparseFragmentMap) EmptyClone() FragmentMap {
	return NewSparseFragmentMap()
}

type RefObject interface {
	Reference
	NewFragmentMap() FragmentMap
}

func (NilReference) NewFragmentMap() FragmentMap { panic("NewFragmentMap called on NilReference") }
