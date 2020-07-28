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
	"reflect"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// Value represents the data flow analysis representation of a variable or
// expression. Unlike the value of a variable or expression at runtime, the
// static analysis representation of a value usually represents a range of
// possible runtime values. As such many of the value implementations are backed
// by some sort of set.
//
// Values are immutable. All transformation methods on a value should return a
// new value.
//
// Values that have go-equality can be considered to have algerbraic equality,
// even if that Value represents a wide range of possible runtime values.
type Value interface {
	// Print returns a textual representation of the value.
	Print(results *Results) string

	// Type returns the semantic type of the value.
	Type() semantic.Type

	// Equivalent returns true iff this and v are equivalent.
	// Unlike Equals() which returns the possibility of two values being equal,
	// Equivalent() returns true iff the set of possible values are exactly
	// equal.
	//
	// Examples:
	//     [5]  and  [5]  are equivalent.
	//    [1-5] and [1-5] are equivalent.
	//    [1-3] and  [1]  are not equivalent.
	//     [1]  and  [2]  are not equivalent.
	Equivalent(v Value) bool

	// Equals returns the possibility of this value being equal to v.
	// If this and v have go-equality then Equals() will return True.
	//
	// Examples:
	//     [5]  and  [5]  returns True.
	//    [1-5] and [1-5] returns Maybe.
	//    [1-3] and [5-7] returns False.
	//     [1]  and  [2]  returns False.
	Equals(v Value) Possibility

	// Valid returns true if there is any possibility of this value equaling
	// any other.
	Valid() bool

	// Clone returns a copy of this value with a unique instance.
	// The cloned value will not have go-equality with this and so will not be
	// considered to have algerbraic equality.
	Clone() Value

	// Union (∪) returns the values that are found in this or v.
	//
	// Examples:
	//     [5]  ∪  [5]  ->      [5]
	//    [1-3] ∪ [2-5] ->     [1-5]
	//    [1-3] ∪ [4-6] ->  [1-3] [4-6]
	//     [1]  ∪  [3]  ->   [1]   [3]
	Union(v Value) Value

	// Intersect (∩) returns the values that are found in both this and v.
	//
	// Examples:
	//     [5]  ∩  [5]  ->   [5]
	//    [1-3] ∩ [2-5] ->  [2-3]
	//    [1-3] ∩ [4-6] ->   [-]
	//     [1]  ∩  [3]  ->   [-]
	Intersect(v Value) Value

	// Difference (\) returns the values that are found in this but not found in v.
	//
	// Examples:
	//     [5]  \  [5]  ->   [-]
	//    [1-3] \ [2-5] ->   [1]
	//    [1-3] \ [4-6] ->  [1-3]
	//     [1]  \  [3]  ->   [1]
	Difference(v Value) Value
}

// Relational is the interface implemented by values that can perform relational
// comparisons.
type Relational interface {
	// GreaterThan returns the possibility of this value being greater than v.
	//
	// Examples:
	//     [5]  >  [5]  ->  False
	//    [1-3] > [2-5] ->  Maybe
	//    [1-3] > [4-6] ->  False
	//    [4-6] > [1-3] ->  True
	GreaterThan(v Value) Possibility

	// GreaterEqual returns the possibility of this value being greater or equal
	// to v.
	//
	// Examples:
	//     [5]  >=  [5]  ->  True
	//    [1-3] >= [2-5] ->  Maybe
	//    [1-3] >= [4-6] ->  False
	//    [4-6] >= [1-3] ->  True
	GreaterEqual(v Value) Possibility

	// LessThan returns the possibility of this value being less than v.
	//
	// Examples:
	//     [5]  <  [5]  ->  False
	//    [1-3] < [2-5] ->  Maybe
	//    [1-3] < [4-6] ->  True
	//    [4-6] < [1-3] ->  False
	LessThan(v Value) Possibility

	// LessEqual returns the possibility of this value being less than or equal
	// to v.
	//
	// Examples:
	//     [5]  <=  [5]  ->  True
	//    [1-3] <= [2-5] ->  Maybe
	//    [1-3] <= [4-6] ->  True
	//    [4-6] <= [1-3] ->  False
	LessEqual(v Value) Possibility
}

// SetRelational is the interface implemented by values that produce new
// constrained values.
type SetRelational interface {
	// SetGreaterThan returns a new value that represents the range of possible
	// values in this value that are greater than the lowest in v.
	//
	// Examples:
	//    [1-9] and  [5]  ->  [6-9]
	//    [1-9] and [2-5] ->  [3-9]
	//    [1-3] and [4-6] ->   [-]
	//    [4-6] and [1-3] ->  [4-6]
	SetGreaterThan(v Value) Value

	// SetGreaterEqual returns a new value that represents the range of possible
	// values in this value that are greater than or equal to the lowest in v.
	//
	// Examples:
	//    [1-9] and  [5]  ->  [5-9]
	//    [1-9] and [2-5] ->  [2-9]
	//    [1-3] and [4-6] ->   [-]
	//    [4-6] and [1-3] ->  [4-6]
	SetGreaterEqual(v Value) Value

	// SetLessThan returns a new value that represents the range of possible
	// values in this value that are less than to the highest in v.
	//
	// Examples:
	//    [1-9] and  [5]  ->  [1-4]
	//    [1-9] and [2-5] ->  [1-4]
	//    [1-3] and [4-6] ->  [1-3]
	//    [4-6] and [1-3] ->   [-]
	SetLessThan(v Value) Value

	// SetLessEqual returns a new value that represents the range of possible
	// values in this value that are less than or equal to the highest in v.
	//
	// Examples:
	//    [1-9] and  [5]  ->  [1-5]
	//    [1-9] and [2-5] ->  [1-5]
	//    [1-3] and [4-6] ->  [1-3]
	//    [4-6] and [1-3] ->   [-]
	SetLessEqual(v Value) Value
}

// UnionOf returns the union of all the values in the slice vals.
// nil values in the slice are ignored.
func UnionOf(vals ...Value) Value {
	var out Value
	for _, v := range vals {
		switch {
		case v == nil:
			// skip
		case out == nil:
			out = v
		default:
			out = out.Union(v)
		}
	}
	return out
}

// unknownOf returns the unbounded value for the given type.
func (s *scope) unknownOf(ty semantic.Type) Value {
	if ty == nil {
		panic("unknownOf passed nil type")
	}

	if v, ok := s.shared.unknowns[ty]; ok {
		return v.Clone() // Clone a pre-cached copy of this value type.
	}

	var out Value
	// setOut assigns v to out and stores v into the shared unknowns cache.
	// Call this as soon as the output value has been built, and before any
	// additional calls to unknownOf to prevent infinite recursion in this
	// function.
	setOut := func(v Value) {
		out = v
		s.shared.unknowns[ty] = v
	}

	switch ty := ty.(type) {
	case *semantic.Pseudonym:
		setOut(s.unknownOf(ty.To))

	case *semantic.Class:
		class := &ClassValue{Class: ty, Fields: map[string]Value{}}
		setOut(class)
		for _, f := range ty.Fields {
			class.Fields[f.Name()] = s.unknownOf(f.Type)
		}

	case *semantic.Map:
		setOut(&MapValue{
			Map:        ty,
			KeyToValue: map[Value]Value{},
			ValueToKey: map[Value]Value{},
		})

	case *semantic.Enum:
		setOut(&EnumValue{
			Ty:      ty,
			Numbers: s.unknownOf(semantic.Uint64Type).(*UintValue),
			Labels:  Labels{},
		})

	case *semantic.Reference:
		// Fake a semantic create node for this unknown reference.
		create := &semantic.Create{Type: ty}
		s.instances[create] = s.unknownOf(ty.To)
		ref := &ReferenceValue{
			Ty: ty.To,
			Assignments: map[*semantic.Create]struct{}{
				create: {},
			},
		}
		setOut(ref)
		ref.Unknown = s.unknownOf(ty.To)

	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			setOut(&BoolValue{Maybe})

		case semantic.Int8Type, semantic.Uint8Type, semantic.CharType:
			setOut(newUintRange(ty, 0, 0xff))

		case semantic.Int16Type, semantic.Uint16Type:
			setOut(newUintRange(ty, 0, 0xffff))

		case semantic.Int32Type, semantic.Uint32Type:
			setOut(newUintRange(ty, 0, 0xffffffff))

		case semantic.Int64Type, semantic.Uint64Type,
			semantic.IntType, semantic.UintType,
			semantic.SizeType:
			// TODO: Fix interval package to allow for entire range!
			setOut(newUintRange(ty, 0, 0xfffffffffffffffe))
		}
	}
	if out == nil {
		setOut(&UntrackedValue{ty})
	}
	return out.Clone()
}

// defaultOf returns the default value for the given type.
func (s *scope) defaultOf(ty semantic.Type) (out Value) {
	if ty == nil {
		panic("valueOf passed nil type")
	}

	if v, ok := s.shared.defaults[ty]; ok {
		return v.Clone() // Clone a pre-cached copy of this value type.
	}
	defer func() { s.shared.defaults[ty] = out.Clone() }()

	switch ty := ty.(type) {
	case *semantic.Pseudonym:
		return s.defaultOf(ty.To)

	case *semantic.Class:
		out := &ClassValue{Class: ty, Fields: map[string]Value{}}
		for _, f := range ty.Fields {
			if f.Default != nil {
				v, _ := s.valueOf(f.Default)
				out.Fields[f.Name()] = v
			} else {
				out.Fields[f.Name()] = s.defaultOf(f.Type)
			}
		}
		return out

	case *semantic.Map:
		return &MapValue{
			Map:        ty,
			KeyToValue: map[Value]Value{},
			ValueToKey: map[Value]Value{},
		}

	case *semantic.Enum:
		return &EnumValue{
			Ty:      ty,
			Numbers: s.defaultOf(semantic.Uint64Type).(*UintValue),
			Labels:  Labels{},
		}

	case *semantic.Reference:
		return &ReferenceValue{
			Ty:          ty.To,
			Unknown:     s.unknownOf(ty.To),
			Assignments: map[*semantic.Create]struct{}{},
		}

	case *semantic.Builtin:
		switch ty {
		case semantic.BoolType:
			return &BoolValue{False}

		case semantic.Int8Type, semantic.CharType:
			return newInt8Value(0)

		case semantic.Int16Type:
			return newInt16Value(0)

		case semantic.Int32Type:
			return newInt32Value(0)

		case semantic.Int64Type, semantic.IntType:
			return newInt64Value(0)

		case semantic.Uint8Type,
			semantic.Uint16Type,
			semantic.Uint32Type,
			semantic.Uint64Type,
			semantic.UintType,
			semantic.SizeType:
			return newUintValue(ty, 0)
		}
	}
	return &UntrackedValue{ty}
}

type fieldHolder interface {
	field(scope *scope, name string) Value
	setField(scope *scope, name string, val Value) Value
}

// valueOf evaluates the expression n and returns a value and a function that
// can change the value within the scope s.
func (s *scope) valueOf(n semantic.Expression) (out Value, setter func(Value)) {
	if n == nil {
		panic("valueOf passed nil expression")
	}

	if l, ok := s.shared.literals[n]; ok {
		return l, nil // use cached literal value
	}

	switch n := n.(type) {
	case *semantic.Local:
		return s.getLocal(n), func(v Value) { s.locals[n] = v }

	case *semantic.Parameter:
		return s.getParameter(n), func(v Value) { s.parameters[n] = v }

	case *semantic.Global:
		return s.getGlobal(n), func(v Value) { s.globals[n] = v }

	case *semantic.Unknown:
		return s.valueOf(n.Inferred)

	case *semantic.Observed:
		return s.valueOf(n.Parameter)

	case semantic.BoolValue:
		defer func() { s.shared.literals[n] = out }()
		v := &BoolValue{False}
		if n {
			v.Possibility = True
		}
		return v, nil

	case semantic.Null:
		return s.defaultOf(n.Type), nil

	case *semantic.EnumEntry:
		v, _ := s.valueOf(n.Value)
		i, ok := semantic.AsUint64(n.Value)
		if !ok {
			panic(fmt.Errorf("EnumEntry value was of type %v", n.Value))
		}
		return &EnumValue{
			Ty:      n.Owner().(*semantic.Enum),
			Numbers: v.(*UintValue),
			Labels:  Labels{i: n.Name()},
		}, nil

	case *semantic.Cast:
		v, _ := s.valueOf(n.Object)
		to := semantic.Underlying(n.Type)
		if from, ok := v.(*UintValue); ok && semantic.IsInteger(to) {
			return &UintValue{Ty: to.(*semantic.Builtin), Ranges: from.Ranges}, nil
		}

	case semantic.Int8Value, semantic.Int16Value, semantic.Int32Value, semantic.Int64Value:
		defer func() { s.shared.literals[n] = out }()
		i := reflect.ValueOf(n).Int()
		switch n.(type) {
		case semantic.Int8Value:
			return newInt8Value(int8(i)), nil

		case semantic.Int16Value:
			return newInt16Value(int16(i)), nil

		case semantic.Int32Value:
			return newInt32Value(int32(i)), nil

		case semantic.Int64Value:
			return newInt64Value(int64(i)), nil
		}

	case semantic.Uint8Value, semantic.Uint16Value, semantic.Uint32Value, semantic.Uint64Value:
		defer func() { s.shared.literals[n] = out }()
		ty := n.ExpressionType().(*semantic.Builtin)
		i := reflect.ValueOf(n).Uint()
		return newUintValue(ty, i), nil

	case *semantic.Member:
		obj, set := s.valueOf(n.Object)
		m, ok := obj.(fieldHolder)
		if !ok {
			panic(fmt.Errorf("Attempted to index field on type %T", obj))
		}
		name := n.Field.Name()
		v := m.field(s, name)
		if set != nil {
			return v, func(v Value) { set(m.setField(s, name, v)) }
		}
		return v, nil

	case *semantic.ClassInitializer:
		c := s.defaultOf(n.Class).(*ClassValue)
		for _, f := range n.Fields {
			v, _ := s.valueOf(f.Value)
			c.Fields[f.Field.Name()] = v
		}
		return c, nil

	case *semantic.Create:
		s.instances[n], _ = s.valueOf(n.Initializer)
		return &ReferenceValue{
			Ty:      n.Type.To,
			Unknown: s.unknownOf(n.Type.To),
			Assignments: map[*semantic.Create]struct{}{
				n: {},
			},
		}, nil

	case *semantic.MapIndex:
		m, set := s.valueOf(n.Map)
		k, _ := s.valueOf(n.Index)
		return m.(*MapValue).Get(s, k), func(v Value) {
			set(m.(*MapValue).Put(k, v))
		}

	case *semantic.MapContains:
		m, _ := s.valueOf(n.Map)
		k, _ := s.valueOf(n.Key)
		return &BoolValue{m.(*MapValue).ContainsKey(k)}, nil

	case *semantic.BinaryOp:
		lhs, _ := s.valueOf(n.LHS)
		rhs, _ := s.valueOf(n.RHS)
		set := func(v Value) {
			if v, ok := v.(*BoolValue); ok {
				switch v.Possibility {
				case True:
					s.considerTrue(n)

				case False:
					s.considerFalse(n)
				}
			}
		}

		switch n.Operator {
		case ast.OpEQ:
			return &BoolValue{lhs.Equals(rhs)}, set

		case ast.OpNE:
			return &BoolValue{lhs.Equals(rhs).Not()}, set

		case ast.OpLT:
			if r, ok := lhs.(Relational); ok {
				return &BoolValue{r.LessThan(rhs)}, set
			}

		case ast.OpGT:
			if r, ok := lhs.(Relational); ok {
				return &BoolValue{r.GreaterThan(rhs)}, set
			}

		case ast.OpLE:
			if r, ok := lhs.(Relational); ok {
				return &BoolValue{r.LessEqual(rhs)}, set
			}

		case ast.OpGE:
			if r, ok := lhs.(Relational); ok {
				return &BoolValue{r.GreaterEqual(rhs)}, set
			}

		case ast.OpAnd:
			if r, ok := lhs.(*BoolValue); ok {
				return r.And(rhs.(*BoolValue)), set
			}

		case ast.OpOr:
			if r, ok := lhs.(*BoolValue); ok {
				return r.Or(rhs.(*BoolValue)), set
			}

		case ast.OpBitwiseOr:
			if r, ok := lhs.(*EnumValue); ok {
				return r.Union(rhs.(*EnumValue)), set
			}

		case ast.OpBitwiseAnd:
			if r, ok := lhs.(*EnumValue); ok {
				return r.Intersect(rhs.(*EnumValue)), set
			}
		}

	case *semantic.UnaryOp:
		expr, _ := s.valueOf(n.Expression)
		switch n.Operator {
		case ast.OpNot:
			return expr.(*BoolValue).Not(), nil
		}

	case *semantic.Select:
		// out is the union of all the reachable choice expressions.
		var out Value

		// Create a scope for evaluating the switch cases
		ss, pop := s.push()
		defer pop()

		flows := []*scope{}
		defer func() { s.setUnion(flows...) }()

		for _, choice := range n.Choices {
			for _, cond := range choice.Conditions {
				// For each choice and condition...

				// Create an semantic expression that is true if the condition
				// passes.
				isTrue := equal(cond, n.Value)
				v, _ := s.valueOf(isTrue)
				possibility := v.(*BoolValue).Possibility
				if !possibility.MaybeTrue() {
					continue // Unreachable
				}
				// Create a new scope to evaluate this choice condition.
				cs, pop := ss.push()
				// The choice condition must have been true to enter this block.
				cs.considerTrue(isTrue)
				// Evaluate the choice's expression and merge into the result.
				val, _ := cs.valueOf(choice.Expression)
				out = UnionOf(out, val)
				pop()
				if cs.abort == nil {
					flows = append(flows, cs)
				} else {
					// case resulted in an abort.
					// Statements below the switch can consider this case false.
					s.considerFalse(isTrue)
				}
				// Later conditions can't equal this condition
				ss.considerFalse(isTrue)
				if possibility == True {
					return out, nil // No later conditions can be reached.
				}
			}
		}
		if n.Default != nil {
			// Default block can be reached.
			// Create a new scope to evaluate the default block.
			cs, pop := ss.push()
			// Evaluate the default's expression and merge into the result.
			val, _ := cs.valueOf(n.Default)
			out = UnionOf(out, val)
			pop()
			if cs.abort != nil {
				// default resulted in an abort.
				// Statements below the switch can consider one of the case
				// conditions to be true.
				if _, set := s.valueOf(n.Value); set != nil {
					conditions := []Value{}
					for _, choice := range n.Choices {
						for _, cond := range choice.Conditions {
							v, _ := s.valueOf(cond)
							conditions = append(conditions, v)
						}
					}
					if len(conditions) > 0 {
						set(UnionOf(conditions...))
					}
				}
			}
		}
		if out == nil {
			// No choice or default was reachable
			out = s.unknownOf(n.Type)
		}
		return out, nil

	case *semantic.Call:
		f := n.Target.Function
		if f.Subroutine {
			// Calling a subroutine.
			// Evaluate the subroutine inline (without pushing a child scope).
			params := make(map[*semantic.Parameter]Value, len(n.Arguments))
			for i, v := range n.Arguments {
				p := f.FullParameters[i]
				// Evaluate the argument.
				val, set := s.valueOf(v)
				s.parameters[p] = val
				params[p] = val
				if set != nil {
					// Once the function has returned, constrain the argument
					// expressions to those that didn't result in an abort.
					defer func() { set(s.parameters[p]) }()
				}
			}

			// If this is recursive, then return unknown.
			for _, b := range s.callstack {
				if b.Function == f {
					return s.unknownOf(n.ExpressionType()), nil
				}
			}

			s.callstack.enter(s.shared.mappings.AST.CST(f.AST), f, params)
			defer s.callstack.exit()

			s.traverse(f.Block)

			if ret := s.returnVal; ret != nil {
				s.returnVal = nil
				return ret, nil
			}
		}
		if f.Extern {
			// If we're passing a callable to an extern, it's likely to be used
			// as a callback. Process this callable function as a public
			// function.
			for _, v := range n.Arguments {
				if f, ok := v.(*semantic.Callable); ok {
					s.publicFunction(f.Function)
				}
			}
		}
	}

	// Was not handled above. Fallback to unknown value.
	return s.unknownOf(n.ExpressionType()), nil
}
