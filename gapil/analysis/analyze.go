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

const maxPasses = 3

// Analyze performs static analysis on the API semantic tree.
func Analyze(api *semantic.API, mappings *semantic.Mappings) *Results {
	// Start by building a root scope for analysis.
	s := &scope{
		shared: &shared{
			mappings: mappings,
			literals: map[semantic.Expression]Value{},
			unknowns: map[semantic.Type]Value{},
			defaults: map[semantic.Type]Value{},
			reached:  map[ast.Node]struct{}{},
		},
		locals:     map[*semantic.Local]Value{},
		parameters: map[*semantic.Parameter]Value{},
		globals:    map[*semantic.Global]Value{},
		instances:  map[*semantic.Create]Value{},
	}
	// Populate builtin globals.
	for _, g := range semantic.BuiltinGlobals {
		s.globals[g] = s.unknownOf(g.Type)
	}
	// Populate all the globals with their default values.
	for _, global := range api.Globals {
		var value Value
		if global.Default != nil {
			value, _ = s.valueOf(global.Default)
		} else {
			value = s.defaultOf(global.Type)
		}
		s.globals[global] = value
	}

	// Mark all the return values of externs to be unknown.
	for _, f := range api.Externs {
		if f.Return != nil {
			s.parameters[f.Return] = s.unknownOf(f.Return.Type)
		}
	}

	// Perform passes over the entire API.
	// Each pass will flow through each of the commands and will build a model
	// of all the possible global variable values.
	// We do multiple passes because a the results of a pass may unlock new
	// conditionals for the next pass.
	// TODO: Don't use a hard-coded number of passes.
	for i := 0; i < maxPasses; i++ {
		semantic.Visit(api, s.traverse)
	}

	// The passes above should have marked all blocks and statements that can
	// be reached. Do a final traversal though the API to find all unreachable
	// blocks and statements.
	unreachables := findUnreachables(api, mappings, s.shared.reached)

	return &Results{
		Unreachables: unreachables,
		Globals:      s.globals,
		Parameters:   s.parameters,
		Instances:    s.instances,
	}
}

// nilOrMaybeTrue returns true if n is nil or could be true.
// n must be of type *BoolValue.
func (s *scope) nilOrMaybeTrue(n semantic.Expression) bool {
	if n == nil {
		return true
	}
	v, _ := s.valueOf(n)
	return v.(*BoolValue).MaybeTrue()
}

func (s *scope) publicFunction(n *semantic.Function) {
	// In a new scope...
	ss, pop := s.push()
	defer pop()

	// All command parameters are initially open.
	params := make(map[*semantic.Parameter]Value, len(n.FullParameters))
	for _, p := range n.FullParameters {
		v := ss.unknownOf(p.ExpressionType())
		ss.parameters[p] = v
		params[p] = v
	}

	// Push the callstack for the traveral of the command's statements.
	ss.callstack.enter(ss.shared.mappings.AST.CST(n.AST), n, params)
	defer ss.callstack.exit()

	// Process each statement
	if n.Block != nil {
		ss.traverse(n.Block)
	}

	// Feed parameter values that didn't lead to an abort back to the root scope.
	for p, v := range ss.parameters {
		s.parameters[p] = UnionOf(v, s.parameters[p])
	}
}

// traverse is the visitor function for all API functions, blocks and
// statements.
func (s *scope) traverse(n semantic.Node) {
	switch n := n.(type) {
	case *semantic.Function:
		if n.Subroutine || n.Extern {
			return // Only interested in commands.
		}

		s.publicFunction(n)

	case *semantic.Assert:
		// For anything below an assert to execute, the assertion must have been
		// true.
		s.considerTrue(n.Condition)

	case *semantic.DeclareLocal:
		v, _ := s.valueOf(n.Local.Value)
		s.locals[n.Local] = v

	case *semantic.Branch:
		cond, _ := s.valueOf(n.Condition)

		flows := []*scope{}
		defer func() { s.setUnion(flows...) }()

		if cond.(*BoolValue).MaybeTrue() {
			// Create a new scope for evaluating the true block.
			bs, pop := s.push()
			defer pop()
			// The if-condition must have been true to enter this block.
			bs.considerTrue(n.Condition)
			// Evaluate the true block.
			bs.traverse(n.True)
			if bs.abort == nil {
				flows = append(flows, bs)
			} else {
				// If the true block resulted in an abort then only the false
				// block allows execution to continue below the if-statement.
				s.considerFalse(n.Condition)
			}
		}

		if cond.(*BoolValue).MaybeFalse() {
			if n.False != nil {
				// Create a new scope for evaluating the false block.
				bs, pop := s.push()
				defer pop()
				// The if-condition must have been false to enter this block.
				bs.considerFalse(n.Condition)
				// Evaluate the false block.
				bs.traverse(n.False)
				if bs.abort == nil {
					flows = append(flows, bs)
				} else {
					// If the false block resulted in an abort then only the true
					// block allows execution to continue below the if-statement.
					s.considerTrue(n.Condition)
				}
			} else {
				// Not always true, and no false block means we might
				// be able to flow over the if-statement.
				flows = append(flows, s)
			}
		}

	case *semantic.Assign:
		rhs, _ := s.valueOf(n.RHS)
		if _, ok := n.LHS.(*semantic.Ignore); ok {
			return // '_ = RHS'
		}
		_, set := s.valueOf(n.LHS)
		if n.Operator == ast.OpAssign {
			set(rhs)
		} else {
			// TODO: Composite assignments
			set(s.unknownOf(n.LHS.ExpressionType()))
		}

	case *semantic.MapAssign:
		m, set := s.valueOf(n.To.Map)
		k, _ := s.valueOf(n.To.Index)
		v, _ := s.valueOf(n.Value)
		if m, ok := m.(*MapValue); ok {
			set(m.Put(k, v))
		}

	case *semantic.MapRemove:
		// TODO: Put an entry that says the map did not contain the key.

	case *semantic.MapClear:
		m, set := s.valueOf(n.Map)
		if m, ok := m.(*MapValue); ok {
			set(m.Clear())
		}

	case *semantic.Abort:
		s.abort = n

	case *semantic.Block:
		// Mark the block as reached, update the callstack location.
		s.setCurrentNode(n)
		for _, n := range n.Statements {
			if s.abort != nil {
				break
			}
			// Mark the statement as reached, update the callstack location.
			s.setCurrentNode(n)
			// Process the statement.
			s.traverse(n)
		}

	case *semantic.Iteration:
		// Create a new scope for evaluating the iteration block.
		s, pop := s.push()
		defer pop()
		s.locals[n.Iterator] = s.unknownOf(n.Iterator.Type) // TODO: smarter range on iterator.
		s.traverse(n.Block)

	case *semantic.MapIteration:
		// Create a new scope for evaluating the iteration block.
		s, pop := s.push()
		defer pop()
		s.locals[n.IndexIterator] = s.unknownOf(n.IndexIterator.Type) // TODO: smarter range on iterator.
		s.locals[n.KeyIterator] = s.unknownOf(n.KeyIterator.Type)     // TODO: smarter range on iterator.
		s.locals[n.ValueIterator] = s.unknownOf(n.ValueIterator.Type) // TODO: smarter range on iterator.
		s.traverse(n.Block)

	case *semantic.Switch:
		// Create a scope for evaluating the switch cases
		ss, pop := s.push()
		defer pop()

		flows := []*scope{}
		defer func() { s.setUnion(flows...) }()

		for _, kase := range n.Cases {
			for _, cond := range kase.Conditions {
				// For each case and condition...

				// Create an semantic expression that is true if the condition
				// passes.
				isTrue := equal(cond, n.Value)
				v, _ := s.valueOf(isTrue)
				possibility := v.(*BoolValue).Possibility
				if !possibility.MaybeTrue() {
					continue // Unreachable
				}
				// Create a new scope to evaluate this case condition.
				cs, pop := ss.push()
				// The case condition must have been true to enter this block.
				cs.considerTrue(isTrue)
				// Process the block's statements.
				cs.traverse(kase.Block)
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
					return // No later conditions can be reached.
				}
			}
		}
		if n.Default != nil {
			// Default block can be reached.
			// Create a new scope to evaluate the default block.
			cs, pop := ss.push()
			// Process the block's statements.
			cs.traverse(n.Default)
			pop()
			if cs.abort == nil {
				flows = append(flows, cs)
			}
		} else {
			// No default case? Then we're flowing over the switch.
			flows = append(flows, s)
		}

	case *semantic.Call:
		s.valueOf(n)

	case *semantic.Return:
		if n.Value != nil {
			// Store the return value in the scope.
			s.returnVal, _ = s.valueOf(n.Value)
		}

	case semantic.Type, *semantic.Callable:
		// Not interested in these.

	default:
		semantic.Visit(n, s.traverse)
	}
}

// considerTrue restricts the scope's values so that n == true.
func (s *scope) considerTrue(n semantic.Expression) {
	switch n := n.(type) {
	case *semantic.Local:
		s.locals[n] = &BoolValue{True}

	case *semantic.Parameter:
		s.parameters[n] = &BoolValue{True}

	case *semantic.BinaryOp:
		switch n.Operator {
		case ast.OpAnd:
			// LHS and RHS must both be true.
			s.considerTrue(n.LHS)
			s.considerTrue(n.RHS)

		case ast.OpOr:
			// LHS or RHS can be true.
			// Evaluate each in separate scopes and then set this scope to the
			// union of both possibilities.
			lhs, _ := s.push()
			rhs, _ := s.push()
			lhs.considerTrue(n.LHS)
			rhs.considerTrue(n.RHS)
			s.setUnion(lhs, rhs)

		case ast.OpEQ:
			// LHS is equal to RHS.
			// Set both to be the intersection of LHS and RHS.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			intersection := lhs.Intersect(rhs)
			if setLHS != nil {
				setLHS(intersection)
			}
			if setRHS != nil {
				setRHS(intersection)
			}

		case ast.OpNE:
			// LHS is not equal to RHS.
			// Remove RHS values from the LHS and vice-versa.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			if setLHS != nil {
				setLHS(lhs.Difference(rhs))
			}
			if setRHS != nil {
				setRHS(rhs.Difference(lhs))
			}

		case ast.OpLT:
			// LHS is less than RHS.
			// Set LHS to be less than RHS, and RHS to be greater than LHS.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			if lhs, ok := lhs.(SetRelational); ok && setLHS != nil {
				setLHS(lhs.SetLessThan(rhs))
			}
			if rhs, ok := rhs.(SetRelational); ok && setRHS != nil {
				setRHS(rhs.SetGreaterThan(lhs))
			}

		case ast.OpLE:
			// LHS is less than or equal to RHS.
			// Set LHS to be less than or equal to RHS, and RHS to be greater
			// than or equal to RHS.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			if lhs, ok := lhs.(SetRelational); ok && setLHS != nil {
				setLHS(lhs.SetLessEqual(rhs))
			}
			if rhs, ok := rhs.(SetRelational); ok && setRHS != nil {
				setRHS(rhs.SetGreaterEqual(lhs))
			}

		case ast.OpGT:
			// LHS is greater than RHS.
			// Set LHS to be greater than RHS, and RHS to be less than LHS.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			if lhs, ok := lhs.(SetRelational); ok && setLHS != nil {
				setLHS(lhs.SetGreaterThan(rhs))
			}
			if rhs, ok := rhs.(SetRelational); ok && setRHS != nil {
				setRHS(rhs.SetLessThan(lhs))
			}

		case ast.OpGE:
			// LHS is greater than or equal to RHS.
			// Set LHS to be greater than or equal to RHS, and RHS to be less
			// than or equal to RHS.
			lhs, setLHS := s.valueOf(n.LHS)
			rhs, setRHS := s.valueOf(n.RHS)
			if lhs, ok := lhs.(SetRelational); ok && setLHS != nil {
				setLHS(lhs.SetGreaterEqual(rhs))
			}
			if rhs, ok := rhs.(SetRelational); ok && setRHS != nil {
				setRHS(rhs.SetLessEqual(lhs))
			}
		}

	case *semantic.BitTest:
		lhs, _ := s.valueOf(n.Bits)
		rhs, setRHS := s.valueOf(n.Bitfield)
		if setRHS != nil {
			setRHS(rhs.Union(lhs)) // TODO: Not really true, need more complex bit logic.
		}

	case *semantic.MapContains:
		// TODO: Put an entry that says the map contained the key.

	case *semantic.UnaryOp:
		switch n.Operator {
		case ast.OpNot:
			s.considerFalse(n.Expression)
		}
	}
}

// considerTrue restricts the scope's values so that n == false.
func (s *scope) considerFalse(n semantic.Expression) {
	switch n := n.(type) {
	case *semantic.Local:
		s.locals[n] = &BoolValue{False}

	case *semantic.Parameter:
		s.parameters[n] = &BoolValue{False}

	case *semantic.BinaryOp:
		switch n.Operator {
		case ast.OpAnd: // !(lhs && rhs) <=> !lhs || !rhs
			s.considerTrue(or(not(n.LHS), not(n.RHS)))

		case ast.OpOr: // !(lhs || rhs) <=> !lhs && !rhs
			s.considerFalse(n.LHS)
			s.considerFalse(n.RHS)

		case ast.OpEQ: // !(lhs == rhs) <=> lhs != rhs
			s.considerTrue(notequal(n.LHS, n.RHS))

		case ast.OpNE: // !(lhs != rhs) <=> lhs == rhs
			s.considerTrue(equal(n.LHS, n.RHS))

		case ast.OpLT: // !(lhs < rhs) <=> lhs >= rhs
			s.considerTrue(ge(n.LHS, n.RHS))

		case ast.OpLE: // !(lhs <= rhs) <=> lhs > rhs
			s.considerTrue(gt(n.LHS, n.RHS))

		case ast.OpGT: // !(lhs > rhs) <=> lhs <= rhs
			s.considerTrue(le(n.LHS, n.RHS))

		case ast.OpGE: // !(lhs >= rhs) <=> lhs < rhs
			s.considerTrue(lt(n.LHS, n.RHS))
		}

	case *semantic.BitTest:
		lhs, _ := s.valueOf(n.Bits)
		rhs, setRHS := s.valueOf(n.Bitfield)
		if setRHS != nil {
			setRHS(rhs.Union(lhs)) // TODO: Not really true, need more complex bit logic.
		}

	case *semantic.MapContains:
		// TODO: Put an entry that says the map did not contain the key.

	case *semantic.UnaryOp:
		switch n.Operator {
		case ast.OpNot:
			s.considerTrue(n.Expression)
		}
	}
}
