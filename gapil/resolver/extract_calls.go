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

// extractCalls moves all call expressions to subroutines out to locals.
// This is done so that there is an oppotunity to test for abort() before
// executing the rest of the expression.
func extractCalls(rv *resolver, block *semantic.Block) {
	if block == nil {
		return
	}
	rv.with(semantic.VoidType, func() {
		for i := 0; i < len(block.Statements); i++ {
			var parent interface{}
			inSelect := false
			var traverse func(n semantic.Node)
			var traverseExpressions func(n []semantic.Expression)
			visit := func(n semantic.Node) semantic.Node {
				switch n := n.(type) {
				case *semantic.Call:
					traverseExpressions(n.Arguments)

					if !n.Target.Function.Subroutine {
						return n // Can't extract a call to a non-subroutine.
					}
					if parent == nil {
						return n // Can't extract a call any more than a call statement.
					}
					if _, ok := parent.(*semantic.DeclareLocal); ok {
						return n // No point extracting a call when it's already just an assignment.
					}
					if inSelect {
						rv.errorf(n, "Cannot call subroutines inside select expressions.")
						return n
					}
					decl := rv.declareTemporaryLocal(n)
					block.Statements.InsertBefore(decl, i)
					i++ // +1 for new injected statement
					return decl.Local
				case *semantic.Block:
					rv.with(semantic.VoidType, func() { extractCalls(rv, n) })
				case *semantic.Select:
					wasInSelect := false
					inSelect = true
					traverse(n)
					inSelect = wasInSelect
				case semantic.Type, *semantic.Callable, semantic.Invalid:
					// Don't traverse into these.
				default:
					traverse(n)
				}
				return n
			}
			with := func(p interface{}, f func()) {
				oldParent := parent
				parent = p
				f()
				parent = oldParent
			}
			traverse = func(n semantic.Node) {
				with(n, func() { semantic.Replace(n, visit) })
			}
			traverseExpressions = func(n []semantic.Expression) {
				with(n, func() {
					for i, a := range n {
						n[i] = visit(a).(semantic.Expression)
					}
				})
			}
			traverse(block.Statements[i])
		}
	})
}
