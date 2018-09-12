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

package resolver

import (
	"strconv"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

func inferNumber(rv *resolver, in *ast.Number, infer semantic.Type) semantic.Expression {
	var out semantic.Expression
	switch infer {
	case semantic.Int8Type:
		if v, err := strconv.ParseInt(in.Value, 0, 8); err == nil {
			out = semantic.Int8Value(v)
		}
	case semantic.Uint8Type:
		if v, err := strconv.ParseUint(in.Value, 0, 8); err == nil {
			out = semantic.Uint8Value(v)
		}
	case semantic.Int16Type:
		if v, err := strconv.ParseInt(in.Value, 0, 16); err == nil {
			out = semantic.Int16Value(v)
		}
	case semantic.Uint16Type:
		if v, err := strconv.ParseUint(in.Value, 0, 16); err == nil {
			out = semantic.Uint16Value(v)
		}
	case semantic.Int32Type:
		if v, err := strconv.ParseInt(in.Value, 0, 32); err == nil {
			out = semantic.Int32Value(v)
		}
	case semantic.Uint32Type:
		if v, err := strconv.ParseUint(in.Value, 0, 32); err == nil {
			out = semantic.Uint32Value(v)
		}
	case semantic.Int64Type:
		if v, err := strconv.ParseInt(in.Value, 0, 64); err == nil {
			out = semantic.Int64Value(v)
		}
	case semantic.Uint64Type:
		if v, err := strconv.ParseUint(in.Value, 0, 64); err == nil {
			out = semantic.Uint64Value(v)
		}
	case semantic.Float32Type:
		if v, err := strconv.ParseFloat(in.Value, 32); err == nil {
			out = semantic.Float32Value(v)
		}
	case semantic.Float64Type:
		if v, err := strconv.ParseFloat(in.Value, 64); err == nil {
			out = semantic.Float64Value(v)
		}
	default:
		return nil
	}
	rv.mappings.Add(in, out)
	return out
}

func inferUnknown(rv *resolver, lhs semantic.Node, rhs semantic.Node) {
	u := findUnknown(rv, rhs)
	if u != nil && u.Inferred == nil {
		// This might fail to infer the expression (i.e. keeps it at nil).
		// That is fine as long as some future expression still infers it.
		u.Inferred = lhsToObserved(rv, lhs)

		// As the unknown has only just been resolved, there may be locals with
		// 'any' types. Fix these up now.
		rv.scope.Symbols.Visit(func(s string, n semantic.Node) {
			if n, ok := n.(*semantic.Local); ok {
				if n.Value == u {
					n.Type = u.ExpressionType()
				}
			}
		})
	}
}

func findUnknown(rv *resolver, rhs semantic.Node) *semantic.Unknown {
	switch rhs := rhs.(type) {
	case *semantic.Unknown:
		return rhs
	case *semantic.Cast:
		return findUnknown(rv, rhs.Object)
	case *semantic.Local:
		return findUnknown(rv, rhs.Value)
	default:
		return nil
	}
}

// lhsToObserved takes an expression that is a valid lhs of an assignment, and
// creates a new expression that would read the observed output.
// This is used when attempting to infer the value an Unknown had from the
// observed outputs.
// Returns nil if the value can not be inferred from this expression.
func lhsToObserved(rv *resolver, lhs semantic.Node) semantic.Expression {
	switch lhs := lhs.(type) {
	case *semantic.SliceIndex:
		o := lhsToObserved(rv, lhs.Slice)
		if o == nil {
			return nil
		}
		return &semantic.SliceIndex{Slice: o, Index: lhs.Index, Type: lhs.Type}
	case *semantic.PointerRange:
		o := lhsToObserved(rv, lhs.Pointer)
		if o == nil {
			return nil
		}
		return &semantic.PointerRange{Pointer: o, Range: lhs.Range, Type: lhs.Type}
	case *semantic.MapIndex:
		o := lhsToObserved(rv, lhs.Map)
		if o == nil {
			return nil
		}
		return &semantic.MapIndex{Map: o, Index: lhs.Index, Type: lhs.Type}
	case *semantic.Cast:
		o := lhsToObserved(rv, lhs.Object)
		if o == nil {
			return nil
		}
		return &semantic.Cast{Object: lhs.Object, Type: lhs.Type}
	case *semantic.Parameter:
		if f := rv.scope.function; f != nil && f.Subroutine && lhs == f.Return {
			rv.errorf(lhs, "unknowns cannot be used to infer return values in subroutines")
		}
		return &semantic.Observed{Parameter: lhs}
	case *semantic.Local:
		return lhsToObserved(rv, lhs.Value)
	case *semantic.Member:
		obj := lhsToObserved(rv, lhs.Object)
		return &semantic.Member{Object: obj, Field: lhs.Field}
	default:
		return nil
	}
}
