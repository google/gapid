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

	"github.com/google/gapid/gapil/semantic"
)

// Value interface compliance checks.
var (
	_ = Value(&UntrackedValue{})
	_ = fieldHolder(&UntrackedValue{})
)

// UntrackedValue is an implementation of Value used for types that are not
// tracked by static analysis.
type UntrackedValue struct {
	// Ty is the type of the untracked value.
	Ty semantic.Type
}

// Print returns "<untracked>"
func (v *UntrackedValue) Print(*Results) string { return "<untracked>" }

// Type returns the semantic type of the untracked value.
func (v *UntrackedValue) Type() semantic.Type { return v.Ty }

// Equivalent returns true.
func (v *UntrackedValue) Equivalent(Value) bool { return true }

// Equals returns Maybe.
func (v *UntrackedValue) Equals(Value) Possibility { return Maybe }

// Valid returns true.
func (v *UntrackedValue) Valid() bool { return true }

// Union returns a new pointer to a UntrackedValue.
func (v *UntrackedValue) Union(o Value) Value { return v.Clone() }

// Intersect returns a new pointer to a UntrackedValue.
func (v *UntrackedValue) Intersect(o Value) Value { return v.Clone() }

// Difference returns a new pointer to a UntrackedValue.
func (v *UntrackedValue) Difference(o Value) Value { return v.Clone() }

// Clone returns a new pointer to a UntrackedValue.
func (v *UntrackedValue) Clone() Value { return &UntrackedValue{v.Ty} }

// field returns an unknown value of the field type.
func (v UntrackedValue) field(s *scope, name string) Value {
	switch ty := semantic.Underlying(v.Ty).(type) {
	case *semantic.Class:
		for _, f := range ty.Fields {
			if f.Name() == name {
				return s.unknownOf(f.Type)
			}
		}
	case *semantic.Reference:
		return UntrackedValue{ty.To}.field(s, name)
	}
	panic(fmt.Errorf("Type %v does not contain a field named \"%v\"", v.Ty.Name(), name))
}

// setField simply returns a new  a UntrackedValue.
func (v *UntrackedValue) setField(s *scope, name string, val Value) Value {
	return v.Clone()
}
