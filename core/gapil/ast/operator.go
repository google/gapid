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

package ast

import "sort"

const (
	// Runes with special meaning to the parser.
	Quote    = '"'
	Backtick = '`'
)

const (
	// The set of operators understood by the parser.
	OpUnknown       = "?"
	OpBlockStart    = "{"
	OpBlockEnd      = "}"
	OpIndexStart    = "["
	OpIndexEnd      = "]"
	OpSlice         = ":"
	OpListStart     = "("
	OpListSeparator = ","
	OpListEnd       = ")"
	OpAssign        = "="
	OpAssignPlus    = "+="
	OpAssignMinus   = "-="
	OpDeclare       = ":="
	OpMember        = "."
	OpExtends       = ":"
	OpAnnotation    = "@"
	OpInitialise    = ":"
	OpPointer       = "*"
	OpEQ            = "=="
	OpGT            = ">"
	OpLT            = "<"
	OpGE            = ">="
	OpLE            = "<="
	OpNE            = "!="
	OpOr            = "||"
	OpAnd           = "&&"
	OpPlus          = "+"
	OpMinus         = "-"
	OpMultiply      = "*"
	OpDivide        = "/"
	OpBitwiseAnd    = "&"
	OpBitwiseOr     = "|"
	OpBitShiftRight = ">>"
	OpBitShiftLeft  = "<<"
	OpRange         = ".."
	OpNot           = "!"
	OpIn            = "in"
	OpGeneric       = "!"
)

var (
	Operators       = []string{}            // all valid operator strings, sorted in descending length order
	UnaryOperators  = map[string]struct{}{} // the map of valid unary operators
	BinaryOperators = map[string]struct{}{} // the map of valid boolean operators
)

// UnaryOp represents any unary operation applied to an expression.
type UnaryOp struct {
	Operator   string // the operator being applied
	Expression Node   // the expression the operator is being applied to
}

func (UnaryOp) isNode() {}

// BinaryOp represents any binary operation applied to two expressions.
type BinaryOp struct {
	LHS      Node   // the expression on the left of the operator
	Operator string // the operator being applied
	RHS      Node   // the expression on the right of the operator
}

func (BinaryOp) isNode() {}

func init() {
	for _, op := range []string{
		OpUnknown,
		OpBlockStart,
		OpBlockEnd,
		OpIndexStart,
		OpIndexEnd,
		OpSlice,
		OpListStart,
		OpListSeparator,
		OpListEnd,
		OpAssign,
		OpAssignPlus,
		OpAssignMinus,
		OpDeclare,
		OpMember,
		OpExtends,
		OpAnnotation,
		OpInitialise,
		OpPointer,
		OpGeneric,
	} {
		Operators = append(Operators, op)
	}
	for _, op := range []string{
		OpEQ,
		OpGT,
		OpLT,
		OpGE,
		OpLE,
		OpNE,
		OpOr,
		OpAnd,
		OpPlus,
		OpMinus,
		OpMultiply,
		OpDivide,
		OpBitwiseAnd,
		OpBitwiseOr,
		OpBitShiftLeft,
		OpBitShiftRight,
		OpRange,
		OpIn,
	} {
		Operators = append(Operators, op)
		BinaryOperators[op] = struct{}{}
	}
	for _, op := range []string{
		OpNot,
	} {
		Operators = append(Operators, op)
		UnaryOperators[op] = struct{}{}
	}
	sort.Sort(opsByLength(Operators))
}

type opsByLength []string

func (a opsByLength) Len() int           { return len(a) }
func (a opsByLength) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a opsByLength) Less(i, j int) bool { return len(a[i]) > len(a[j]) }
