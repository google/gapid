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
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// equal returns the semantic expression: (lhs == rhs)
func equal(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpEQ, RHS: rhs}
}

// equal returns the semantic expression: (lhs != rhs)
func notequal(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpNE, RHS: rhs}
}

// gt returns the semantic expression: (lhs > rhs)
func gt(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpGT, RHS: rhs}
}

// ge returns the semantic expression: (lhs >= rhs)
func ge(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpGE, RHS: rhs}
}

// lt returns the semantic expression: (lhs < rhs)
func lt(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpLT, RHS: rhs}
}

// le returns the semantic expression: (lhs <= rhs)
func le(lhs, rhs semantic.Expression) semantic.Expression {
	return &semantic.BinaryOp{Type: semantic.BoolType, LHS: lhs, Operator: ast.OpLE, RHS: rhs}
}

// not returns the semantic expression: (!e)
func not(e semantic.Expression) semantic.Expression {
	return &semantic.UnaryOp{Type: semantic.BoolType, Expression: e, Operator: ast.OpNot}
}

// and returns the chained semantic expression: (((e[0] && e[1]) && e[2]) && ... e[len(e)-1])
// nil elements in the slice of e are ignored.
func and(e ...semantic.Expression) semantic.Expression {
	var out semantic.Expression
	for _, e := range e {
		switch {
		case e == nil:
			// skip
		case out == nil:
			out = e
		default:
			out = &semantic.BinaryOp{Type: semantic.BoolType, LHS: out, Operator: ast.OpAnd, RHS: e}
		}
	}
	return out
}

// or returns the chained semantic expression: (((e[0] || e[1]) || e[2]) || ... e[len(e)-1])
// nil elements in the slice of e are ignored.
func or(e ...semantic.Expression) semantic.Expression {
	var out semantic.Expression
	for _, e := range e {
		switch {
		case e == nil:
			// skip
		case out == nil:
			out = e
		default:
			out = &semantic.BinaryOp{Type: semantic.BoolType, LHS: out, Operator: ast.OpOr, RHS: e}
		}
	}
	return out
}
