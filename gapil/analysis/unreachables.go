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

// findUnreachables returns the list of blocks and statements that are found
// by traversing api which are not found in reached.
func findUnreachables(
	api *semantic.API,
	mappings *semantic.Mappings,
	reached map[ast.Node]struct{}) []Unreachable {

	// Output list of unreachable blocks and statements.
	unreachables := []Unreachable{}

	// Blocks and statements that have already been reported as unreachable.
	reported := map[ast.Node]struct{}{}

	// check looks to see if n has been marked as reached. If n has not been
	// reached then it is added to the list of unreachables and check returns
	// false.
	check := func(n semantic.Node) bool {
		for _, a := range mappings.SemanticToAST[n] {
			if _, ok := reached[a]; !ok {
				// Ensure this is reported only once.
				if _, ok := reported[a]; !ok {
					reported[a] = struct{}{}
					at := mappings.AST.CST(a)
					unreachables = append(unreachables, Unreachable{At: at, Node: n})
				}
				return false
			}
		}
		return true
	}

	// The recursive traversal function.
	var traverse func(n semantic.Node)
	traverse = func(n semantic.Node) {
		switch n := n.(type) {
		case *semantic.Function:
			if n.GetAnnotation("ignore_unreachables") == nil {
				semantic.Visit(n, traverse)
			}

		case *semantic.Block:
			if check(n) {
				for _, s := range n.Statements {
					if !check(s) {
						break
					}
					traverse(s)
				}
			}

		case semantic.Type, *semantic.Callable:
			// Not interested in these.

		default:
			semantic.Visit(n, traverse)
		}
	}

	// Peform the traversal and find all unreachables.
	traverse(api)

	return unreachables
}
