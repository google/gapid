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

// Block represents a linear sequence of statements, most often the contents
// of a {} pair.
type Block struct {
	Statements []Node // The set of statements that make up the block
}

func (Block) isNode() {}

// Branch represents an «"if" condition { trueblock } "else" { falseblock }» structure.
type Branch struct {
	Condition Node   // the condition to use to select which block is active
	True      *Block // the block to use if condition is true
	False     *Block // the block to use if condition is false
}

func (Branch) isNode() {}

// Iteration represents a «"for" variable "in" iterable { block }» structure.
type Iteration struct {
	Variable *Identifier // the variable to use for the iteration value
	Iterable Node        // the expression that produces the iterable to loop over
	Block    *Block      // the block to run once per item in the iterable
}

func (Iteration) isNode() {}

// MapIteration represents a «"for" i "," k "," v "in" map { block }» structure.
type MapIteration struct {
	IndexVariable *Identifier // The index variable to use for iteration
	KeyVariable   *Identifier // the key variable to use for iteration
	ValueVariable *Identifier // the value variable to use for iteration
	Map           Node        // the expression that produces the map to loop over
	Block         *Block      // the block to run once per k-v mapping
}

func (MapIteration) isNode() {}

// Switch represents a «"switch" value { cases }» structure.
// The first matching case is selected.
// If a switch is used as an expression, the case blocks must all be a single
// expression.
type Switch struct {
	Value   Node     // the value to match against
	Cases   []*Case  // the set of cases to match the value with
	Default *Default // the block which is used if no case is matched
}

func (Switch) isNode() {}

// Case represents a «"case" conditions: block» structure within a switch statement.
// The conditions are a comma separated list of expressions the switch statement
// value will be compared against.
type Case struct {
	Annotations Annotations // the annotations applied to this case
	Conditions  []Node      // the set of conditions that would select this case
	Block       *Block      // the block to run if this case is selected
}

func (Case) isNode() {}

// Default represents a «"default": block» structure within a switch statement.
type Default struct {
	Block *Block // the block to run if the default is selected
}

func (Default) isNode() {}

// Group represents the «(expression)» construct, a single parenthesized expression.
type Group struct {
	Expression Node // the expression within the parentheses
}

func (Group) isNode() {}

// DeclareLocal represents a «name := value» statement that declares a new
// immutable local variable with the specified value and inferred type.
type DeclareLocal struct {
	Name *Identifier // the name to give the new local
	RHS  Node        // the value to store in that local
}

func (DeclareLocal) isNode() {}

// Assign represents a «location {,+,-}= value» statement that assigns a value to
// an existing mutable location.
type Assign struct {
	LHS      Node   // the location to store the value into
	Operator string // the assignment operator being applied
	RHS      Node   // the value to store
}

func (Assign) isNode() {}

// Return represents the «"return" value» construct, that assigns the value to
// the result slot of the function.
type Return struct {
	Value Node // the value to return
}

func (Return) isNode() {}

// Member represents an expression that access members of objects.
// Always of the form «object.name» where object is an expression.
type Member struct {
	Object Node        // the object to get a member of
	Name   *Identifier // the name of the member to get
}

func (Member) isNode() {}

// Index represents any expression of the form «object[index]»
// Used for arrays, maps and bitfields.
type Index struct {
	Object Node // the object to index
	Index  Node // the index to lookup
}

func (Index) isNode() {}
