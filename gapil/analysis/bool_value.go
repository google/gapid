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

import "github.com/google/gapid/gapil/semantic"

// Value interface compliance check.
var _ = Value(&BoolValue{})

// BoolValue is an implementation of Value that represents all the possible
// values of a boolean type.
type BoolValue struct {
	Possibility
}

// Print returns a textual representation of the value.
func (v *BoolValue) Print(results *Results) string {
	return v.String()
}

func (v *BoolValue) String() string {
	switch v.Possibility {
	case True:
		return "True"
	case Maybe:
		return "Maybe"
	case False:
		return "False"
	case Impossible:
		return "Impossible"
	default:
		return "<unknown bool value>"
	}
}

// Type returns semantic.BoolType.
func (v *BoolValue) Type() semantic.Type {
	return semantic.BoolType
}

// Not returns the logical negation of v.
func (v *BoolValue) Not() *BoolValue {
	return &BoolValue{v.Possibility.Not()}
}

// And returns the logical-and of v and o.
func (v *BoolValue) And(o *BoolValue) *BoolValue {
	if v == o {
		return v
	}
	return &BoolValue{v.Possibility.And(o.Possibility)}
}

// Or returns the logical-or of v and o.
func (v *BoolValue) Or(o *BoolValue) *BoolValue {
	if v == o {
		return v
	}
	return &BoolValue{v.Possibility.Or(o.Possibility)}
}

// Equivalent returns true iff v and o are equivalent.
// See Value for the definition of equivalency.
func (v *BoolValue) Equivalent(o Value) bool { return v.Possibility == o.(*BoolValue).Possibility }

// Equals returns the possibility of v equaling o.
// o must be of type BoolValue.
func (v *BoolValue) Equals(o Value) Possibility {
	if v == o && v.Valid() {
		return True
	}
	return v.Possibility.Equals(o.(*BoolValue).Possibility)
}

// Valid returns true if there is any possibility of this value equaling
// any other.
func (v *BoolValue) Valid() bool {
	return v.Possibility != Impossible
}

// Union returns the union of possibile values for v and o.
// o must be of type BoolValue.
func (v *BoolValue) Union(o Value) Value {
	if v == o {
		return v
	}
	return &BoolValue{v.Possibility.Union(o.(*BoolValue).Possibility)}
}

// Intersect returns the intersection of possibile values for v and o.
// o must be of type BoolValue.
func (v *BoolValue) Intersect(o Value) Value {
	if v == o {
		return v
	}
	return &BoolValue{v.Possibility.Intersect(o.(*BoolValue).Possibility)}
}

// Difference returns the possibile for v that are not found in o.
// o must be of type BoolValue.
func (v *BoolValue) Difference(o Value) Value {
	return &BoolValue{v.Possibility.Difference(o.(*BoolValue).Possibility)}
}

// Clone returns a new instance of BoolValue initialized from v.
func (v *BoolValue) Clone() Value {
	return &BoolValue{v.Possibility}
}
