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

package semantic

import "github.com/google/gapid/gapil/ast"

// Length represents a length of object expression.
// Object must be of either pointer, slice, map or string type.
// The length expression is allowed to be of any numeric type
type Length struct {
	AST    *ast.Call  // the underlying syntax node this was built from
	Object Expression // the object go get the length of
	Type   Type       // the resolved type of the length operation
}

func (*Length) isNode()       {}
func (*Length) isExpression() {}

// ExpressionType implements Expression
func (l *Length) ExpressionType() Type { return l.Type }

// Assert represents a runtime assertion.
// Assertions are also used to infer required behavior from the expressions.
type Assert struct {
	AST       *ast.Call  // the underlying syntax node this was built from
	Condition Expression // the condition is being asserted must be true
	Message   string
}

func (*Assert) isNode()      {}
func (*Assert) isStatement() {}

// Cast represents a type reinterpret expression.
type Cast struct {
	AST    *ast.Call  // the underlying syntax node this was built from
	Object Expression // the expression to cast the result of
	Type   Type       // the type to cast to
}

func (*Cast) isNode()       {}
func (*Cast) isExpression() {}

// ExpressionType implements Expression
func (c *Cast) ExpressionType() Type { return c.Type }

// New represents a call to new.
type New struct {
	AST  *ast.Call // the underlying syntax node this was built from
	Type *Reference
}

func (*New) isNode()       {}
func (*New) isExpression() {}

// ExpressionType implements Expression
func (n *New) ExpressionType() Type { return n.Type }

// Create represents a call to new on a class type.
type Create struct {
	AST         *ast.Call // the underlying syntax node this was built from
	Type        *Reference
	Initializer *ClassInitializer
}

func (*Create) isNode()       {}
func (*Create) isExpression() {}

// ExpressionType implements Expression
func (n *Create) ExpressionType() Type { return n.Type }

// Make represents a call to make.
type Make struct {
	AST  *ast.Call // the underlying syntax node this was built from
	Type *Slice
	Size Expression
}

func (*Make) isNode()       {}
func (*Make) isExpression() {}

// ExpressionType implements Expression
func (m *Make) ExpressionType() Type { return m.Type }

// Clone represents a call to clone.
type Clone struct {
	AST   *ast.Call // the underlying syntax node this was built from
	Slice Expression
	Type  *Slice
}

func (*Clone) isNode()       {}
func (*Clone) isExpression() {}

// ExpressionType implements Expression
func (m *Clone) ExpressionType() Type { return m.Type }

// Read represents a call to read.
type Read struct {
	AST   *ast.Call // the underlying syntax node this was built from
	Slice Expression
}

func (*Read) isNode()      {}
func (*Read) isStatement() {}

// Write represents a call to write.
type Write struct {
	AST   *ast.Call // the underlying syntax node this was built from
	Slice Expression
}

func (*Write) isNode()      {}
func (*Write) isStatement() {}

// Copy represents a call to copy.
type Copy struct {
	AST *ast.Call // the underlying syntax node this was built from
	Src Expression
	Dst Expression
}

func (*Copy) isNode()      {}
func (*Copy) isStatement() {}

// Print represents a call to print.
type Print struct {
	AST       *ast.Call    // the underlying syntax node this was built from
	Arguments []Expression // The parameters to print
}

func (*Print) isNode()      {}
func (*Print) isStatement() {}
