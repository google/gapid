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
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

// scope contains the full context information for analysis of a semantic node.
type scope struct {
	parent     *scope
	shared     *shared
	callstack  Callstack
	locals     map[*semantic.Local]Value
	parameters map[*semantic.Parameter]Value
	globals    map[*semantic.Global]Value
	instances  map[*semantic.Create]Value
	abort      *semantic.Abort
	returnVal  Value
}

// shared is the common data shared between all scopes.
type shared struct {
	mappings *resolver.Mappings
	literals map[semantic.Expression]Value
	unknowns map[semantic.Type]Value
	defaults map[semantic.Type]Value
	reached  map[ast.Node]struct{}
}

// push returns a new child scope with a copy of the s's values.
// pop merges the child scope global and instance values back into s.
func (s *scope) push() (child *scope, pop func()) {
	c := scope{
		parent:     s,
		shared:     s.shared,
		callstack:  s.callstack,
		locals:     make(map[*semantic.Local]Value, len(s.locals)),
		parameters: make(map[*semantic.Parameter]Value, len(s.parameters)),
		globals:    make(map[*semantic.Global]Value, len(s.globals)),
		instances:  make(map[*semantic.Create]Value, len(s.instances)),
	}
	return &c, func() {
		// Merge global and instance values back together from child branch
		for g, v := range c.globals {
			s.globals[g] = unionOf(s.getGlobal(g), v)
		}
		for i, v := range c.instances {
			s.instances[i] = unionOf(s.getInstance(i), v)
		}
	}
}

func (s *scope) getLocal(n *semantic.Local) Value {
	if v, ok := s.locals[n]; ok || s.parent == nil {
		return v
	}
	return s.parent.getLocal(n)
}

func (s *scope) getParameter(n *semantic.Parameter) Value {
	if v, ok := s.parameters[n]; ok || s.parent == nil {
		return v
	}
	return s.parent.getParameter(n)
}

func (s *scope) getGlobal(n *semantic.Global) Value {
	if v, ok := s.globals[n]; ok || s.parent == nil {
		return v
	}
	return s.parent.getGlobal(n)
}

func (s *scope) getInstance(n *semantic.Create) Value {
	if v, ok := s.instances[n]; ok || s.parent == nil {
		return v
	}
	return s.parent.getInstance(n)
}

// setUnion sets all the values in s to be a union of those in a and b.
// setUnion is used to merge the results of two child scopes.
func (s *scope) setUnion(a, b *scope) {
	locals := map[*semantic.Local]struct{}{}
	for n := range a.locals {
		locals[n] = struct{}{}
	}
	for n := range b.locals {
		locals[n] = struct{}{}
	}
	for n := range locals {
		s.locals[n] = unionOf(a.getLocal(n), b.getLocal(n))
	}

	parameters := map[*semantic.Parameter]struct{}{}
	for n := range a.parameters {
		parameters[n] = struct{}{}
	}
	for n := range b.parameters {
		parameters[n] = struct{}{}
	}
	for n := range parameters {
		s.parameters[n] = unionOf(a.getParameter(n), b.getParameter(n))
	}

	globals := map[*semantic.Global]struct{}{}
	for n := range a.globals {
		globals[n] = struct{}{}
	}
	for n := range b.globals {
		globals[n] = struct{}{}
	}
	for n := range globals {
		s.globals[n] = unionOf(a.getGlobal(n), b.getGlobal(n))
	}

	instances := map[*semantic.Create]struct{}{}
	for n := range a.instances {
		instances[n] = struct{}{}
	}
	for n := range b.instances {
		instances[n] = struct{}{}
	}
	for n := range instances {
		s.instances[n] = unionOf(a.getInstance(n), b.getInstance(n))
	}
}

// setCurrentNode marks the node n as reached and changes the scope's callstack
// to point to n.
func (s *scope) setCurrentNode(n semantic.Node) {
	for _, n := range s.shared.mappings.SemanticToAST[n] {
		s.shared.reached[n] = struct{}{}
	}
	if pn := s.shared.mappings.ParseNode(n); pn != nil {
		s.callstack.set(pn)
	}
}
