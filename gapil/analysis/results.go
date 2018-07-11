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
	"fmt"
	"strings"

	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/semantic"
)

// Results holds the results of the static analysis.
type Results struct {
	// Unreachables is the list of unreachable blocks and statements.
	Unreachables []Unreachable
	// Globals is the map of semantic globals to their possible values.
	Globals map[*semantic.Global]Value
	// Parameters is the map of semantic parameters to their possible values.
	Parameters map[*semantic.Parameter]Value
	// Instances is the map of semantic create statements to the possible values
	// for those instances.
	Instances map[*semantic.Create]Value
}

// Unreachable represents an unreachable block or statement.
type Unreachable struct {
	At   cst.Node
	Node semantic.Node
}

// CallstackEntry is a single entry in a callstack.
type CallstackEntry struct {
	Location   cst.Node
	Function   *semantic.Function
	Parameters map[*semantic.Parameter]Value
}

func (e CallstackEntry) String() string {
	at, fun := "<unknown>", ""
	if e.Location != nil {
		at = e.Location.Tok().At()
	}
	if e.Function != nil {
		callParams := e.Function.CallParameters()
		params := make([]string, len(callParams))
		for i, p := range callParams {
			val := e.Parameters[p]
			params[i] = fmt.Sprintf("%v: %v", p.Name(), val)
		}
		fun = fmt.Sprintf("%v(%v)", e.Function.Name(), strings.Join(params, ", "))
	}
	return fmt.Sprintf("%v%v", at, fun)
}

// Callstack holds the callstack to a point in the API file.
type Callstack []CallstackEntry

func (c Callstack) String() string {
	lines := make([]string, len(c))
	for i, e := range c {
		lines[len(c)-i-1] = e.String()
	}
	return strings.Join(lines, "\n")
}

// At returns the parse node to the deepest point in the callstack.
func (c Callstack) At() cst.Node {
	if len(c) > 0 {
		return c[len(c)-1].Location
	}
	return nil
}

// clone makes a new copy of the callstack.
func (c Callstack) clone() Callstack {
	out := make(Callstack, len(c))
	copy(out, c)
	return out
}

// set changes the deepest poin in the callstack to point to loc.
func (c Callstack) set(loc cst.Node) {
	if len(c) > 0 {
		c[len(c)-1].Location = loc
	}
}

// enter pushes a new entry to the callstack.
func (c *Callstack) enter(loc cst.Node, f *semantic.Function, params map[*semantic.Parameter]Value) {
	*c = append(*c, CallstackEntry{
		Location:   loc,
		Function:   f,
		Parameters: params,
	})
}

// exit pops the callstack one level.
func (c *Callstack) exit() {
	*c = (*c)[:len(*c)-1]
}
