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

package compare

import "fmt"

// Path represents a path from the root of an object hierarchy
type Path []Fragment

// Fragment is an entry in a Path
type Fragment struct {
	// Operation holds the operation occurring at this level in the path.
	Operation interface{}
	// Reference holds the entry in the reference hierarchy to apply the operation to
	Reference interface{}
	// Value holds the equivalent entry in the value hierarchy to apply the the operation to
	Value interface{}
}

// MemberOp is the fragment operation type for member comparisons.
type MemberOp string

// IndexOp is the fragment operation type for array / slice index comparisons.
type IndexOp int

// EntryOp is the fragment operation type for map entry comparisons.
type EntryOp struct {
	Key interface{} // The map key
}

// LengthOp is the fragment operation type for array / slice length comparisons.
type LengthOp string

// TypeOp is the fragment operation type for type comparisons.
type TypeOp string

// NilOp is the fragment operation type for nil-equality comparisons.
type NilOp string

// MissingOp is the fragment operation type for absent entries in arrays,
// slices and maps.
type MissingOp string

const (
	Length = LengthOp("·length")
	Type   = TypeOp("·type")
	Nil    = NilOp("nil")
	Key    = MissingOp("key")
)

func (m MemberOp) Format(f fmt.State, r rune) { fmt.Fprint(f, ".", string(m)) }
func (i IndexOp) Format(f fmt.State, r rune)  { fmt.Fprintf(f, "[%v]", int(i)) }
func (e EntryOp) Format(f fmt.State, r rune)  { fmt.Fprintf(f, "[%v]", e.Key) }

func (p Path) with(op, reference, value interface{}) Path {
	r := make(Path, len(p)+1)
	copy(r, p)
	r[len(p)] = Fragment{op, reference, value}
	return r
}

// Member returns a new Path with a member access fragment appended.
func (p Path) Member(name string, reference, value interface{}) Path {
	return p.with(MemberOp(name), reference, value)
}

// Length returns a new Path with a length query fragment appended.
func (p Path) Length(reference, value interface{}) Path { return p.with(Length, reference, value) }

// Type returns a new Path with a type query fragment appended.
func (p Path) Type(reference, value interface{}) Path { return p.with(Type, reference, value) }

// Nil returns a new Path with a nil query fragment appended.
func (p Path) Nil(reference, value interface{}) Path { return p.with(Nil, reference, value) }

// Missing returns a new Path with a missing value fragment appended.
func (p Path) Missing(reference, value interface{}) Path { return p.with(Key, reference, value) }

// Index returns a new Path with an array/slice index fragment appended.
func (p Path) Index(i int, reference, value interface{}) Path {
	return p.with(IndexOp(i), reference, value)
}

// Entry returns a new Path with a map entry fragment appended.
func (p Path) Entry(key, reference, value interface{}) Path {
	return p.with(EntryOp{key}, reference, value)
}

// Diff returns a new Path with a terminal diff fragment appended.
func (p Path) Diff(reference, value interface{}) Path { return p.with(nil, reference, value) }

func (p Path) Format(f fmt.State, r rune) {
	if len(p) == 0 {
		return
	}
	last := p[len(p)-1]
	remains := p[:len(p)-1]
	if last.Operation != nil {
		fmt.Fprint(f, last.Operation, " ")
	}
	fmt.Fprintf(f, "⟦%+v⟧ != ⟦%+v⟧", last.Reference, last.Value)
	if len(remains) > 0 {
		fmt.Fprint(f, " for v")
		for _, e := range remains {
			fmt.Fprint(f, e.Operation)
		}
	}
}
