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

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapil/semantic"
)

// Value interface compliance checks.
var (
	_ = Value(&UintValue{})
	_ = SetRelational(&UintValue{})
)

// newUintRange returns a new UintValue that has the possible values
// [minValue - maxValue).
func newUintRange(ty *semantic.Builtin, minValue, maxValue uint64) *UintValue {
	return &UintValue{
		Ty: ty,
		Ranges: interval.U64SpanList{
			interval.U64Span{
				Start: minValue,
				End:   maxValue + 1,
			},
		},
	}
}

// newUintValue returns a new UintValue that has the single possible value [v]
func newUintValue(ty *semantic.Builtin, v uint64) *UintValue {
	return newUintRange(ty, v, v)
}

// newUintValue returns a new UintValue that has the single possible value [v]
// as transformed into the unsigned range.
func newInt8Value(v int8) *UintValue {
	return newUintValue(semantic.Int8Type, uint64(v)+0x80)
}

// newUintValue returns a new UintValue that has the single possible value [v]
// as transformed into the unsigned range.
func newInt16Value(v int16) *UintValue {
	return newUintValue(semantic.Int16Type, uint64(v)+0x8000)
}

// newUintValue returns a new UintValue that has the single possible value [v]
// as transformed into the unsigned range.
func newInt32Value(v int32) *UintValue {
	return newUintValue(semantic.Int32Type, uint64(v)+0x80000000)
}

// newUintValue returns a new UintValue that has the single possible value [v]
// as transformed into the unsigned range.
func newInt64Value(v int64) *UintValue {
	return newUintValue(semantic.Int64Type, uint64(v)+0x8000000000000000)
}

// UintValue is an implementation of Value that represents all the possible
// values of an unsigned integer type.
// Although UintValue naturally represents unsigned integers, it can also be
// used to represent signed values by shifting the signed value into the
// unsigned range.
type UintValue struct {
	Ty     *semantic.Builtin
	Ranges interval.U64SpanList
}

// Print returns a textual representation of the value.
func (v *UintValue) Print(results *Results) string {
	return v.String()
}

func uintBias(ty semantic.Type) func(v uint64) interface{} {
	switch semantic.Underlying(ty) {
	case semantic.Int8Type:
		return func(v uint64) interface{} { return int64(v - 0x80) }
	case semantic.Int16Type:
		return func(v uint64) interface{} { return int64(v - 0x8000) }
	case semantic.Int32Type:
		return func(v uint64) interface{} { return int64(v - 0x80000000) }
	case semantic.Int64Type:
		return func(v uint64) interface{} { return int64(v - 0x8000000000000000) }
	default:
		return func(v uint64) interface{} { return v }
	}
}

func (v *UintValue) String() string {
	bias := uintBias(v.Ty)
	parts := make([]string, len(v.Ranges))
	for i, r := range v.Ranges {
		if r.End > r.Start+1 {
			parts[i] = fmt.Sprintf("[%#x-%#x]", bias(r.Start), bias(r.End-1))
		} else {
			parts[i] = fmt.Sprintf("[%#x]", bias(r.Start))
		}
	}
	return strings.Join(parts, " ")
}

// Type returns the semantic type of the integer value represented by v.
func (v *UintValue) Type() semantic.Type {
	return v.Ty
}

// GreaterThan returns the possibility of v being greater than o.
// o must be of type *UintValue.
func (v *UintValue) GreaterThan(o Value) Possibility {
	a, b := v.span(), o.(*UintValue).span()
	switch {
	case a.Start > b.End-1:
		return True
	case a.End-1 <= b.Start:
		return False
	default:
		return Maybe
	}
}

// GreaterEqual returns the possibility of v being greater or equal to o.
// o must be of type *UintValue.
func (v *UintValue) GreaterEqual(o Value) Possibility {
	a, b := v.span(), o.(*UintValue).span()
	switch {
	case a.Start >= b.End-1:
		return True
	case a.End-1 < b.Start:
		return False
	default:
		return Maybe
	}
}

// LessThan returns the possibility of v being less than o.
// o must be of type *UintValue.
func (v *UintValue) LessThan(o Value) Possibility {
	a, b := v.span(), o.(*UintValue).span()
	switch {
	case a.End-1 < b.Start:
		return True
	case a.Start >= b.End-1:
		return False
	default:
		return Maybe
	}
}

// LessEqual returns the possibility of v being less than or equal to o.
// o must be of type *UintValue.
func (v *UintValue) LessEqual(o Value) Possibility {
	a, b := v.span(), o.(*UintValue).span()
	switch {
	case a.End-1 <= b.Start:
		return True
	case a.Start > b.End-1:
		return False
	default:
		return Maybe
	}
}

// SetGreaterThan returns a new value that represents the range of possible
// values in v that are greater than the lowest in o.
// o must be of type *UintValue.
func (v *UintValue) SetGreaterThan(o Value) Value {
	b := o.(*UintValue).span()
	out := v.Clone().(*UintValue)
	interval.Remove(&out.Ranges, interval.U64Span{Start: 0, End: b.Start + 1})
	return out
}

// SetGreaterEqual returns a new value that represents the range of possible
// values in v that are greater than or equal to the lowest in o.
// o must be of type *UintValue.
func (v *UintValue) SetGreaterEqual(o Value) Value {
	b := o.(*UintValue).span()
	out := v.Clone().(*UintValue)
	interval.Remove(&out.Ranges, interval.U64Span{Start: 0, End: b.Start})
	return out
}

// SetLessThan returns a new value that represents the range of possible
// values in v that are less than to the highest in o.
// o must be of type *UintValue.
func (v *UintValue) SetLessThan(o Value) Value {
	b := o.(*UintValue).span()
	out := v.Clone().(*UintValue)
	interval.Remove(&out.Ranges, interval.U64Span{Start: b.End - 1, End: ^uint64(0)})
	return out
}

// SetLessEqual returns a new value that represents the range of possible
// values in v that are less than or equal to the highest in o.
// o must be of type *UintValue.
func (v *UintValue) SetLessEqual(o Value) Value {
	b := o.(*UintValue).span()
	out := v.Clone().(*UintValue)
	interval.Remove(&out.Ranges, interval.U64Span{Start: b.End, End: ^uint64(0)})
	return out
}

// Equivalent returns true iff v and o are equivalent.
// Unlike Equals() which returns the possibility of two values being equal,
// Equivalent() returns true iff the set of possible values are exactly
// equal.
// o must be of type *UintValue.
func (v *UintValue) Equivalent(o Value) bool {
	if v == o {
		return true
	}
	a, b := v, o.(*UintValue)
	if len(a.Ranges) != len(b.Ranges) {
		return false
	}
	for i, r := range a.Ranges {
		if b.Ranges[i] != r {
			return false
		}
	}
	return true
}

// Equals returns the possibility of v being equal to o.
// o must be of type *UintValue.
func (v *UintValue) Equals(o Value) Possibility {
	if v == o && v.Valid() {
		return True
	}
	a, b := v, o.(*UintValue)
	if len(a.Ranges) == 1 && len(b.Ranges) == 1 && // Only 1 interval in LHS and RHS
		a.Ranges[0].Start == a.Ranges[0].End-1 && // Only 1 value in LHS
		b.Ranges[0].Start == b.Ranges[0].End-1 && // Only 1 value in RHS
		a.Ranges[0].Start == b.Ranges[0].Start { // Value the same in LHS and RHS
		return True
	}
	for _, span := range b.Ranges {
		if _, c := interval.Intersect(&a.Ranges, span); c > 0 {
			return Maybe
		}
	}
	return False
}

// Valid returns true if there is any possibility of this value equaling
// any other.
func (v *UintValue) Valid() bool {
	return len(v.Ranges) > 0
}

// Union (∪) returns the values that are found in v or o.
// o must be of type *UintValue.
func (v *UintValue) Union(o Value) Value {
	if v == o {
		return v
	}
	a, b := v, o.(*UintValue)
	out := a.Clone().(*UintValue)
	for _, span := range b.Ranges {
		interval.Merge(&out.Ranges, span, true)
	}
	return out
}

// Intersect (∩) returns the values that are found in both v and o.
// o must be of type *UintValue.
func (v *UintValue) Intersect(o Value) Value {
	if v == o {
		return v
	}
	a, b := v, o.(*UintValue)
	out := &UintValue{
		Ty:     v.Ty,
		Ranges: interval.U64SpanList{},
	}
	for _, spanB := range b.Ranges {
		f, c := interval.Intersect(&a.Ranges, spanB)
		for _, spanA := range a.Ranges[f : f+c] {
			span := interval.U64Span{
				Start: u64.Max(spanA.Start, spanB.Start),
				End:   u64.Min(spanA.End, spanB.End),
			}
			interval.Merge(&out.Ranges, span, true)
		}
	}
	return out
}

// Difference (\) returns the values that are found in v but not found in o.
// o must be of type *UintValue.
func (v *UintValue) Difference(o Value) Value {
	a, b := v, o.(*UintValue)
	if !v.maybeOverlaps(b) {
		return v
	}
	out := a.Clone().(*UintValue)
	for _, span := range b.Ranges {
		interval.Remove(&out.Ranges, span)
	}
	return out
}

// Clone returns a copy of v with a unique pointer.
func (v *UintValue) Clone() Value {
	out := &UintValue{
		Ty:     v.Ty,
		Ranges: make(interval.U64SpanList, len(v.Ranges)),
	}
	copy(out.Ranges, v.Ranges)
	return out
}

// span returns the range of the lowest and highest possible values.
func (v *UintValue) span() interval.U64Span {
	if len(v.Ranges) == 0 {
		return interval.U64Span{}
	}
	return interval.U64Span{
		Start: v.Ranges[0].Start,
		End:   v.Ranges[len(v.Ranges)-1].End,
	}
}

// maybeOverlaps returns true if v and o potentially share values.
func (v *UintValue) maybeOverlaps(o *UintValue) bool {
	a, b := v.span(), o.span()
	if a.End <= b.Start || b.End <= a.Start {
		return false
	}
	return true
}
