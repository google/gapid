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

package analysis

import (
	"fmt"
	"strings"

	"github.com/google/gapid/gapil/semantic"
)

// Value interface compliance checks.
var (
	_ = Value(&ReferenceValue{})
	_ = fieldHolder(&ReferenceValue{})
)

// ReferenceValue is an implementation of Value that represents all the possible
// references of a reference value type.
// As references are an indirection, this value only holds the set of references
// not the set of referenced values. The actual referenced values are held by
// the scope.
// ReferenceValues uses *semantic.Create as the reference handles.
type ReferenceValue struct {
	// The reference type.
	Ty semantic.Type
	// Unknown is the unknown value of the referenced type.
	Unknown Value
	// Assignments is the set of all assignments made to this value.
	Assignments map[*semantic.Create]struct{}
}

func (v *ReferenceValue) String() string {
	if len(v.Assignments) == 0 {
		return "<nil>"
	}
	pointers := make([]string, 0, len(v.Assignments))
	for a := range v.Assignments {
		pointers = append(pointers, fmt.Sprintf("%p", a))
	}
	return "ref!" + strings.Join(pointers, ", ")
}

// Print returns a textual representation of the value.
func (v *ReferenceValue) Print(results *Results) string {
	if len(v.Assignments) == 0 {
		return "<nil>"
	}
	values := make([]Value, 0, len(v.Assignments))
	for a := range v.Assignments {
		values = append(values, results.Instances[a])
	}
	return "ref!" + UnionOf(values...).Print(results)
}

// Type returns the semantic reference type of the value.
func (v *ReferenceValue) Type() semantic.Type { return v.Ty }

// Equivalent returns true iff v and o are equivalent.
// Unlike Equals() which returns the possibility of two values being equal,
// Equivalent() returns true iff the set of possible field values are exactly
// equal.
// o must be of type *ReferenceValue.
func (v *ReferenceValue) Equivalent(o Value) bool {
	a, b := v, o.(*ReferenceValue)
	if len(a.Assignments) != len(b.Assignments) {
		return false
	}
	for c := range a.Assignments {
		if _, ok := b.Assignments[c]; !ok {
			return false
		}
	}
	for c := range b.Assignments {
		if _, ok := a.Assignments[c]; !ok {
			return false
		}
	}
	return true
}

// Equals returns the possibility of the reference value v being equal to o.
// o must be of type *ReferenceValue.
func (v *ReferenceValue) Equals(o Value) Possibility {
	if v == o && v.Valid() {
		return True
	}
	a, b := v, o.(*ReferenceValue)
	if len(a.Assignments) == len(b.Assignments) {
		switch len(a.Assignments) {
		case 0:
			return True
		case 1:
			for s := range a.Assignments {
				if _, ok := b.Assignments[s]; ok {
					return True
				}
			}
		}
	}
	return Maybe // There's always null == null
}

// Valid returns true.
func (v *ReferenceValue) Valid() bool {
	return true
}

// Union (∪) returns the reference value with all the assignments of v and o.
// o must be of type *ReferenceValue.
func (v *ReferenceValue) Union(o Value) Value {
	a, b := v, o.(*ReferenceValue)
	out := a.Clone().(*ReferenceValue)
	for s := range b.Assignments {
		out.Assignments[s] = struct{}{}
	}
	return out
}

// Intersect (∩) returns the reference value the assignments that are found in
// both v and o.
// o must be of type *ReferenceValue.
func (v *ReferenceValue) Intersect(o Value) Value {
	a, b := v, o.(*ReferenceValue)
	out := a.Clone().(*ReferenceValue)
	for s := range a.Assignments {
		if _, ok := b.Assignments[s]; !ok {
			delete(out.Assignments, s)
		}
	}
	return out
}

// Difference (\) returns the reference value with assignments that are found in
// v but not found in o.
// o must be of type *ReferenceValue.
func (v *ReferenceValue) Difference(o Value) Value {
	a, b := v, o.(*ReferenceValue)
	out := &ReferenceValue{Ty: v.Ty, Unknown: v.Unknown, Assignments: map[*semantic.Create]struct{}{}}
	for s := range a.Assignments {
		if _, ok := b.Assignments[s]; !ok {
			out.Assignments[s] = struct{}{}
		}
	}
	return out
}

// Clone returns a copy of v with a unique pointer.
func (v *ReferenceValue) Clone() Value {
	out := &ReferenceValue{Ty: v.Ty, Unknown: v.Unknown, Assignments: map[*semantic.Create]struct{}{}}
	for s := range v.Assignments {
		out.Assignments[s] = struct{}{}
	}
	return out
}

// field returns the union of all the fields with the specfied name across all
// assignments to v.
func (v *ReferenceValue) field(s *scope, name string) Value {
	candidates := make([]Value, 0, len(v.Assignments))
	for a := range v.Assignments {
		// Check the assignment has an instance. No instance can happen when the
		// assignment took place in a sibling block which has not been merged
		// into the common scope yet.
		// In this particular case, the instance is inaccessible to this block,
		// so skipping it is the correct thing to do.
		if fh, ok := s.getInstance(a).(fieldHolder); ok {
			field := fh.field(s, name)
			candidates = append(candidates, field)
		}
	}
	if len(candidates) == 0 {
		return v.Unknown.(fieldHolder).field(s, name)
	}
	return UnionOf(candidates...)
}

// setField calls setField across all assignments to v and updating the value
// in the scope's instances map.
func (v *ReferenceValue) setField(s *scope, name string, val Value) Value {
	for a := range v.Assignments {
		// Check the assignment has an instance. No instance can happen when the
		// assignment took place in a sibling block which has not been merged
		// into the common scope yet.
		// In this particular case, the instance is inaccessible to this block,
		// so skipping it is the correct thing to do.
		if fh, ok := s.getInstance(a).(fieldHolder); ok {
			s.instances[a] = fh.setField(s, name, val)
		}
	}
	return v
}
