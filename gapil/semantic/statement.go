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
import "github.com/google/gapid/core/data/slice"

// Statement is the interface implemented by all statement types.
type Statement interface {
	Node
	isStatement() // A placeholder function that's implemented by all semantic statements.
}

// Statements is a list of statements.
type Statements []Statement

func (Statements) isNode() {}

// Replace replaces count statements starting from first with s.
func (l *Statements) Replace(first, count int, s ...Statement) {
	slice.Replace(l, first, count, s)
}

// Remove removes all occurances of s from the list of statements.
func (l *Statements) Remove(s Statement) {
	slice.Remove(l, s)
}

// InsertBefore inserts the statement s before the i'th statement.
func (l *Statements) InsertBefore(s Statement, i int) {
	slice.InsertBefore(l, i, s)
}

// Append adds the statement s at the end of the list.
func (l *Statements) Append(s Statement) {
	slice.Append(l, s)
}

// Last returns the last statement in the list.
func (l Statements) Last() Statement {
	if c := len(l); c > 0 {
		return l[c-1]
	}
	return nil
}

// Block represents a collection of statements, used as the body of other
// nodes.
type Block struct {
	AST        *ast.Block // the underlying syntax node this was built from
	Statements Statements // the set of statements this block represents
}

func (*Block) isNode()      {}
func (*Block) isStatement() {}

// Branch represents the basic conditional execution statement.
// If Condition is true we use the True block, otherwise the False block.
type Branch struct {
	AST       *ast.Branch // the underlying syntax node this was built from
	Condition Expression  // the condition to select on
	True      *Block      // use if Condition is true
	False     *Block      // used if Condition is false
}

func (*Branch) isNode()      {}
func (*Branch) isStatement() {}

// Switch represents a resolved ast.Switch statement.
type Switch struct {
	AST     *ast.Switch // the underlying syntax node this was built from
	Value   Expression  // the value to match the cases against
	Cases   []*Case     // the set of case statements to choose from
	Default *Block      // the block to use if no condition matches
}

func (*Switch) isNode()      {}
func (*Switch) isStatement() {}

// Case represents a possible choice in a switch.
type Case struct {
	AST         *ast.Case    // the underlying syntax node this was built from
	Annotations              // the annotations applied to the case
	Conditions  []Expression // the set of expressions to match the switch value against
	Block       *Block       // the block to use if a condition matches
}

func (*Case) isNode()      {}
func (*Case) isStatement() {}

// Iteration is the basic looping construct.
// It will set Iterator to each value from Iterable in turn, and run Block for each one.
type Iteration struct {
	AST      *ast.Iteration // the underlying syntax node this was built from
	Iterator *Local         // the iteration control variable
	From     Expression     // the expression to iterate from
	To       Expression     // the expression to iterate to
	Block    *Block         // the block to run for each entry from Iterable
}

func (*Iteration) isNode()      {}
func (*Iteration) isStatement() {}

// MapIteration is a loop over a map's key-value pairs.
// It will set KeyIterator and ValueIterator to each pair from Map in turn,
// set IndexIterator to 0 and increment on each loop, and run Block for each.
type MapIteration struct {
	AST           *ast.MapIteration // the underlying syntax node this was built from
	IndexIterator *Local            // the iteration index control variable
	KeyIterator   *Local            // the iteration key control variable
	ValueIterator *Local            // the iteration value control variable
	Map           Expression        // the map to iterate over
	Block         *Block            // the block to run for each k-v mapping
}

func (*MapIteration) isNode()      {}
func (*MapIteration) isStatement() {}

// Assign is the only "mutating" construct.
// It assigns the value from the rhs into the slot described by the lhs, as defined
// by the operator.
type Assign struct {
	AST      *ast.Assign // the underlying syntax node this was built from
	LHS      Expression  // the expression that gives the location to store into
	Operator string      // the assignment operator being applied
	RHS      Expression  // the value to store
}

func (*Assign) isNode()      {}
func (*Assign) isStatement() {}

// ArrayAssign represents assigning to a static-array index expression.
type ArrayAssign struct {
	AST      *ast.Assign // the underlying syntax node this was built from
	To       *ArrayIndex // the array index to assign to
	Operator string      // the assignment operator being applied
	Value    Expression  // the value to set in the array
}

func (*ArrayAssign) isNode()      {}
func (*ArrayAssign) isStatement() {}

// MapAssign represents assigning to a map index expression.
type MapAssign struct {
	AST      *ast.Assign // the underlying syntax node this was built from
	To       *MapIndex   // the map index to assign to
	Operator string      // the assignment operator being applied
	Value    Expression  // the value to set in the map
}

func (*MapAssign) isNode()      {}
func (*MapAssign) isStatement() {}

// MapRemove represents removing an element from a map.
type MapRemove struct {
	AST  *ast.Delete // the underlying syntax node this was built from
	Type *Map        // the value type of the map
	Map  Expression  // the expression that returns the map holding the key
	Key  Expression  // the map key to remove
}

func (*MapRemove) isNode()      {}
func (*MapRemove) isStatement() {}

// MapClear represents clearing a map
type MapClear struct {
	AST  *ast.Clear // the underlying syntax node this was built from
	Type *Map       // the value type of the map
	Map  Expression // the expression that returns the map holding the key
}

func (*MapClear) isNode()      {}
func (*MapClear) isStatement() {}

// SliceAssign represents assigning to a slice index expression.
type SliceAssign struct {
	AST      *ast.Assign // the underlying syntax node this was built from
	To       *SliceIndex // the slice index to assign to
	Operator string      // the assignment operator being applied
	Value    Expression  // the value to set in the slice
}

func (*SliceAssign) isNode()      {}
func (*SliceAssign) isStatement() {}

// DeclareLocal represents a local variable declaration statement.
// Variables cannot be modified after declaration.
type DeclareLocal struct {
	AST   *ast.DeclareLocal // the underlying syntax node this was built from
	Local *Local            // the local variable that was declared by this statement
}

func (*DeclareLocal) isNode()      {}
func (*DeclareLocal) isStatement() {}

// Return represents return statement for a function.
type Return struct {
	AST      *ast.Return // the underlying syntax node this was built from
	Function *Function   // the function this statement returns from
	Value    Expression  // the value to be returned
}

func (*Return) isNode()      {}
func (*Return) isStatement() {}

// Abort represents the abort statement, used to immediately terminate execution
// of a command, usually because of an error.
type Abort struct {
	AST       *ast.Abort // the underlying syntax node this was built from
	Function  *Function  // the function this is aborting
	Statement Statement
}

func (*Abort) isNode()      {}
func (*Abort) isStatement() {}

// Fence is a marker to indicate the point between all statements to be
// executed before (pre-fence) the call to the API function and all statements
// to be executed after (post-fence) the call to the API function.
//
// The Statement member is the first statement that is classified as post-fence,
// but may be nil if the fence is being added at the end of a function that has
// no post operations.
//
// Note that some statements are classified as both pre-fence and post-fence,
// and require logic to be executed either side of the API function call.
type Fence struct {
	AST       *ast.Fence // the underlying syntax node this was built from
	Statement Statement
	Explicit  bool // If true, then the fence was explicitly declared in the API file.
}

func (*Fence) isNode()      {}
func (*Fence) isStatement() {}
