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

// Function represents function like objects in the semantic graph.
type Function struct {
	owned
	AST            *ast.Function // the underlying syntax node this was built from
	Annotations                  // the annotations applied to the function
	Named                        // the name of the function
	Docs           Documentation // the documentation for the function
	Return         *Parameter    // the return parameter
	This           *Parameter    // the this parameter, missing for non method functions
	FullParameters []*Parameter  // all the parameters, including This at the start if valid, and Return at the end if not void
	Block          *Block        // the body of the function, missing for externs
	Signature      *Signature    // the type signature of the function
	Extern         bool          // true if this was declared as an extern
	Subroutine     bool          // true if this was declared as a subroutine
	Recursive      bool          // true if this function is part of a recursive chain
	Order          LogicalOrder  // the logical order of the statements relative to the fence
}

func (*Function) isNode() {}

// CallParameters returns the full set of parameters with the return value
// filtered out.
func (f *Function) CallParameters() []*Parameter {
	if f.Return.Type == VoidType {
		return f.FullParameters
	}
	return f.FullParameters[0 : len(f.FullParameters)-1]
}

// Parameter represents a single parameter declaration for a function.
type Parameter struct {
	AST         *ast.Parameter // the underlying syntax node this was built from
	Annotations                // the annotations applied to the parameter
	Function    *Function      // the function this parameter belongs to
	Named                      // the name of the parameter
	Docs        Documentation  // the documentation for the parameter
	Type        Type           // the type of the parameter
}

func (*Parameter) isNode()       {}
func (*Parameter) isExpression() {}

// ExpressionType implements Expression for parameter lookup.
func (p *Parameter) ExpressionType() Type { return p.Type }

// IsThis returns true if this parameter is the This parameter of it's function.
func (p *Parameter) IsThis() bool { return p == p.Function.This }

// IsReturn returns true if this parameter is the Return parameter of it's function.
func (p *Parameter) IsReturn() bool { return p == p.Function.Return }

// Observed represents the final observed value of an output parameter.
// It is never produced directly from the ast, but is inserted when inferring
// the value of an unknown from observed outputs.
type Observed struct {
	Parameter *Parameter // the output parameter to infer from
}

func (*Observed) isNode()       {}
func (*Observed) isExpression() {}

// ExpressionType implements Expression for observed parameter lookup.
func (e *Observed) ExpressionType() Type { return e.Parameter.Type }

// Callable wraps a Function declaration into a function value expression,
// optionally binding to an object if its a method.
type Callable struct {
	Object   Expression // the object to use as the this parameter for a method
	Function *Function  // the function this expression represents
}

func (*Callable) isNode()       {}
func (*Callable) isExpression() {} // TODO: Do we really want this as an expression?

// ExpressionType implements Expression returning the function type signature.
func (c *Callable) ExpressionType() Type {
	if c.Function.Signature != nil {
		return c.Function.Signature
	}
	return VoidType
}

// Call represents a function call. It binds an Callable to the arguments it
// will be passed.
type Call struct {
	AST       *ast.Call    // the underlying syntax node this was built from
	Target    *Callable    // the function expression this invokes
	Arguments []Expression // the arguments to pass to the function
	Type      Type         // the return type of the call
}

func (*Call) isNode()       {}
func (*Call) isExpression() {}
func (*Call) isStatement()  {}

// ExpressionType implements Expression returning the underlying function return type.
func (c *Call) ExpressionType() Type { return c.Type }

// Signature represents a callable type signature
type Signature struct {
	owned
	noMembers
	Named            // the full type name
	Return    Type   // the return type of the callable
	Arguments []Type // the required callable arguments
}

func (*Signature) isNode() {}
func (*Signature) isType() {}
