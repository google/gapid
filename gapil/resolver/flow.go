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

import "github.com/google/gapid/gapil/semantic"

type visitedFuncs map[*semantic.Function]bool

// resolveFenceOrder analyses comamnds and subroutines for the placement of the
// fence.
func resolveFenceOrder(rv *resolver, f *semantic.Function, visitedFuncs visitedFuncs) semantic.LogicalOrder {
	if f.Order.Resolved() {
		return f.Order // Already resolved
	}
	if visitedFuncs[f] {
		rv.errorf(f, "Cyclic dependeny found")
		return 0
	}
	visitedFuncs[f] = true
	if f.Extern {
		f.Order = semantic.Resolved
		return f.Order
	}
	t := &fenceTracker{
		resolver:     rv,
		orders:       map[semantic.Node]semantic.LogicalOrder{},
		visitedFuncs: visitedFuncs,
	}

	f.Order = t.analyse(f.Block)
	if !t.explicit {
		// No explicit fence added, see if we need to add one.
		if f.Subroutine && f.Order.Pre() && f.Order.Post() {
			// A subroutine that contains a fence.
			t.insertFence(f.Block)
		} else if !f.Subroutine {
			// All commands must contain a fence.
			if !t.insertFence(f.Block) {
				// No pre and post statements found, we have to append a fence.
				f.Block.Statements = append(f.Block.Statements, &semantic.Fence{})
			}
			f.Order = semantic.Resolved | semantic.Pre | semantic.Post
		}
	}
	return f.Order
}

type fenceTracker struct {
	resolver     *resolver
	explicit     bool // explcitly declared fence found.
	orders       map[semantic.Node]semantic.LogicalOrder
	visitedFuncs visitedFuncs
}

// isInternal returns true if n is a slice or pointer that's guaranteed to only
// hold internal memory.
func (t *fenceTracker) isInternal(n semantic.Expression) bool {
	switch n := n.(type) {
	case *semantic.Cast:
		return t.isInternal(n.Object)
	case *semantic.SliceIndex:
		return t.isInternal(n.Slice)
	case *semantic.SliceRange:
		return t.isInternal(n.Slice)
	case *semantic.Select:
		for _, c := range n.Choices {
			if !t.isInternal(c.Expression) {
				return false
			}
		}
		if n.Default != nil {
			if !t.isInternal(n.Default) {
				return false
			}
		}
		return true
	case *semantic.Local:
		return t.isInternal(n.Value)
	case *semantic.Member:
		return n.Field.IsInternal()
	case *semantic.Global:
		return n.IsInternal()
	case *semantic.Parameter:
		return n.IsInternal()
	case *semantic.Clone, *semantic.Make:
		return true
	case *semantic.Call:
		return n.Target.Function.IsInternal()
	default:
		return false
	}
}

// analyse traverses the statements and expressions for semantic node n and its
// children, assessing and returning whether each node is pre or post w.r.t the
// fence. An entry is added to the fenceTracker.orders map for each visited
// node.
func (t *fenceTracker) analyse(n semantic.Node) semantic.LogicalOrder {
	o := semantic.Resolved
	switch n := n.(type) {
	case *semantic.Fence:
		if !n.Explicit {
			t.resolver.icef(n.AST, "unexpected fence found")
		}
		if t.explicit {
			t.resolver.errorf(n, "multiple explicit fences found")
		}
		t.explicit = true
		o = semantic.Pre | semantic.Post
	case *semantic.Read:
		o = semantic.Pre | t.analyse(n.Slice)
	case *semantic.Unknown:
		o = semantic.Post
	case *semantic.Write:
		o = semantic.Post | t.analyse(n.Slice)
	case *semantic.Assign:
		if t.isInternal(n.LHS) && !t.isInternal(n.RHS) {
			t.resolver.errorf(n, "Assigning from a non-internal to an internal")
		}
		t.analyse(n.LHS)
		t.analyse(n.RHS)
	case *semantic.Copy:
		if !t.isInternal(n.Src) {
			o |= semantic.Pre // Reading from a non-internal is pre-fence.
		}
		if !t.isInternal(n.Dst) {
			o |= semantic.Post // Writing to a non-internal is post-fence.
		}
	case *semantic.SliceIndex:
		if !t.isInternal(n.Slice) {
			o |= semantic.Pre // Reading an element from a non-internal is pre-fence.
		}
		o |= t.analyse(n.Index) | t.analyse(n.Slice)
	case *semantic.SliceAssign:
		if !t.isInternal(n.To) {
			o |= semantic.Post // Writing to an element of a non-internal is post-fence.
		}
		o |= t.analyse(n.Value)
	case *semantic.Return:
		if !n.Function.Subroutine {
			o = semantic.Post
		}
		if n.Value != nil {
			o |= t.analyse(n.Value)
		}
	case *semantic.Call:
		f := n.Target.Function
		o = resolveFenceOrder(t.resolver, f, t.visitedFuncs)
		if f.AST.Annotations.GetAnnotation("pre_fence") != nil {
			o |= semantic.Pre
		}
		if f.AST.Annotations.GetAnnotation("post_fence") != nil {
			o |= semantic.Post
		}
		semantic.Visit(n, func(c semantic.Node) { o |= t.analyse(c) })

		for i, a := range n.Arguments {
			if t.isInternal(f.FullParameters[i]) && !t.isInternal(a) {
				t.resolver.errorf(n, "Passing a non-internal argument to an internal parameter")
			}
		}
	case *semantic.Callable:
	case semantic.Type:
	// Don't traverse types
	default:
		semantic.Visit(n, func(c semantic.Node) { o |= t.analyse(c) })
	}
	t.orders[n] = o
	return o
}

func (t *fenceTracker) insertFence(n semantic.Node) (inserted bool) {
	switch n := n.(type) {
	case *semantic.Block:
		for i := 0; i < len(n.Statements); i++ {
			s := n.Statements[i]
			o := t.orders[s]
			if inserted {
				// fence already found, but continue looking over the statements
				// to ensure that no more pre-statements are found.
				if o.Pre() {
					t.resolver.errorf(s, "pre-statement after fence")
					return true
				}
				continue
			}

			if o.Post() {
				// first post statement or block containing post statement found.
				// insert a fence.
				switch {
				case !o.Pre(): // pre -> post transision between statements.
					n.Statements = append(n.Statements, nil)
					copy(n.Statements[i+1:], n.Statements[i:])
					n.Statements[i] = &semantic.Fence{}
					i++ // step past the newly inserted fence

				case t.insertFence(s): // inserted in sub-block

				default: // pre and post statement found.
					if call, ok := s.(*semantic.Call); ok {
						f := call.Target.Function
						if f.Subroutine && f.Order == o {
							break // Subroutine already contains the fence. All's good.
						}
					}
					if _, ok := s.(*semantic.Copy); ok {
						n.Statements[i] = &semantic.Fence{Statement: n.Statements[i]}
						break
					}
					t.resolver.errorf(s, "Only copy statements can be pre and post fence.")
				}
				inserted = true
			}
		}
		return inserted
	case *semantic.Iteration, *semantic.MapIteration, *semantic.Switch, *semantic.Branch:
		t.resolver.errorf(n, "fence not permitted in %T", n)
		return true
	default:
		return false
	}
}
