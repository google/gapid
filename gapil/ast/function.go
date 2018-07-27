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

// Function represents the declaration of a callable entity, any of "cmd", "sub"
// or "extern". Its structure is «return_type name(parameters) body»
// where parameters is a comma separated list and body is an optional block.
type Function struct {
	Annotations Annotations  // the annotations applied to this function
	Generic     *Generic     // the name of the function
	Parameters  []*Parameter // the parameters the function takes
	Block       *Block       // the body of the function if present
}

func (Function) isNode() {}

// Parameter represents a single parameter in the set of parameters for a Function.
// It has the structure «["in"|"out"|"inout"|"this"] type name»
type Parameter struct {
	Annotations Annotations // the annotations applied to this parameter
	This        bool        // true if the parameter is the this pointer of a method
	Type        Node        // the type of the parameter
	Name        *Identifier // the name the parameter as exposed to the body
}

func (Parameter) isNode() {}

// Call is an expression that invokes a function with a set of arguments.
// It has the structure «target(arguments)» where target must be a function and
// arguments is a comma separated list of expressions.
type Call struct {
	Target    Node   // the function to invoke
	Arguments []Node // the arguments to the function
}

func (Call) isNode() {}

// NamedArg represents a «name = value» expression as a function argument.
type NamedArg struct {
	Name  *Identifier // the name of the parameter this value is for
	Value Node        // the value to use for that parameter
}

func (NamedArg) isNode() {}
