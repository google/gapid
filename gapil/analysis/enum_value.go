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
	_ = Value(&EnumValue{})
	_ = SetRelational(&EnumValue{})
)

// EnumValue is an implementation of Value that represents all the possible
// values of an enumerator.
type EnumValue struct {
	Ty      *semantic.Enum
	Numbers *UintValue
	Labels  map[uint64]string
}

// Print returns a textual representation of the value.
func (v *EnumValue) Print(results *Results) string {
	return v.String()
}

func (v *EnumValue) String() string {
	bias := uintBias(v.Ty)
	parts := []string{}
	add := func(i uint64) {
		s, ok := v.Labels[i]
		if !ok {
			s = fmt.Sprintf("%#x", bias(i))
		}
		parts = append(parts, s)
	}
	for _, r := range v.Numbers.Ranges {
		if r.End-r.Start < 10 {
			for i := r.Start; i != r.End; i++ {
				add(i)
			}
		} else {
			add(r.Start)
			parts = append(parts, "...")
			add(r.End - 1)
		}
	}

	return fmt.Sprintf("[%v]", strings.Join(parts, ", "))
}

// Type returns the semantic type of the integer value represented by v.
func (v *EnumValue) Type() semantic.Type {
	return v.Ty
}

// GreaterThan returns the possibility of v being greater than o.
// o must be of type *EnumValue.
func (v *EnumValue) GreaterThan(o Value) Possibility {
	return v.Numbers.GreaterThan(o.(*EnumValue).Numbers)
}

// GreaterEqual returns the possibility of v being greater or equal to o.
// o must be of type *EnumValue.
func (v *EnumValue) GreaterEqual(o Value) Possibility {
	return v.Numbers.GreaterEqual(o.(*EnumValue).Numbers)
}

// LessThan returns the possibility of v being less than o.
// o must be of type *EnumValue.
func (v *EnumValue) LessThan(o Value) Possibility {
	return v.Numbers.LessThan(o.(*EnumValue).Numbers)
}

// LessEqual returns the possibility of v being less than or equal to o.
// o must be of type *EnumValue.
func (v *EnumValue) LessEqual(o Value) Possibility {
	return v.Numbers.LessEqual(o.(*EnumValue).Numbers)
}

// SetGreaterThan returns a new value that represents the range of possible
// values in v that are greater than the lowest in o.
// o must be of type *EnumValue.
func (v *EnumValue) SetGreaterThan(o Value) Value {
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: v.Numbers.SetGreaterThan(o.(*EnumValue).Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// SetGreaterEqual returns a new value that represents the range of possible
// values in v that are greater than or equal to the lowest in o.
// o must be of type *EnumValue.
func (v *EnumValue) SetGreaterEqual(o Value) Value {
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: v.Numbers.SetGreaterEqual(o.(*EnumValue).Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// SetLessThan returns a new value that represents the range of possible
// values in v that are less than to the highest in o.
// o must be of type *EnumValue.
func (v *EnumValue) SetLessThan(o Value) Value {
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: v.Numbers.SetLessThan(o.(*EnumValue).Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// SetLessEqual returns a new value that represents the range of possible
// values in v that are less than or equal to the highest in o.
// o must be of type *EnumValue.
func (v *EnumValue) SetLessEqual(o Value) Value {
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: a.Numbers.SetLessEqual(b.Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// Equivalent returns true iff v and o are equivalent.
// Unlike Equals() which returns the possibility of two values being equal,
// Equivalent() returns true iff the set of possible values are exactly
// equal.
// o must be of type *EnumValue.
func (v *EnumValue) Equivalent(o Value) bool {
	if v == o {
		return true
	}
	a, b := v, o.(*EnumValue)
	if !a.Numbers.Equivalent(b.Numbers) {
		return false
	}
	if len(a.Labels) != len(b.Labels) {
		return false
	}
	for i, v := range a.Labels {
		if b.Labels[i] != v {
			return false
		}
	}
	return true
}

// Equals returns the possibility of v being equal to o.
// o must be of type *EnumValue.
func (v *EnumValue) Equals(o Value) Possibility {
	if v == o && v.Valid() {
		return True
	}
	a, b := v, o.(*EnumValue)
	return a.Numbers.Equals(b.Numbers)
}

// Valid returns true if there is any possibility of this value equaling
// any other.
func (v *EnumValue) Valid() bool {
	return v.Numbers.Valid()
}

// Union (∪) returns the values that are found in v or o.
// o must be of type *EnumValue.
func (v *EnumValue) Union(o Value) Value {
	if v == o {
		return v
	}
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: a.Numbers.Union(b.Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// Intersect (∩) returns the values that are found in both v and o.
// o must be of type *EnumValue.
func (v *EnumValue) Intersect(o Value) Value {
	if v == o {
		return v
	}
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: a.Numbers.Intersect(b.Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// Difference (\) returns the values that are found in v but not found in o.
// o must be of type *EnumValue.
func (v *EnumValue) Difference(o Value) Value {
	a, b := v, o.(*EnumValue)
	return &EnumValue{
		Numbers: a.Numbers.Difference(b.Numbers).(*UintValue),
		Labels:  a.joinLabels(b),
	}
}

// Clone returns a copy of v with a unique pointer.
func (v *EnumValue) Clone() Value {
	out := &EnumValue{
		Ty:      v.Ty,
		Numbers: v.Numbers.Clone().(*UintValue),
		Labels:  make(map[uint64]string, len(v.Labels)),
	}
	for i, s := range v.Labels {
		out.Labels[i] = s
	}
	return out
}

func (v *EnumValue) joinLabels(o *EnumValue) map[uint64]string {
	out := make(map[uint64]string, len(v.Labels)+len(o.Labels))
	for i, s := range v.Labels {
		out[i] = s
	}
	for i, s := range o.Labels {
		out[i] = s
	}
	return out
}
