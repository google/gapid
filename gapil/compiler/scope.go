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

package compiler

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/semantic"
)

// S is a nestable compiler scope.
// A scope holds parameters and local variables.
type S struct {
	// The scope can emit any instructions into the current scope block.
	*codegen.Builder

	// Ctx is a pointer to the active context (context_t*).
	Ctx *codegen.Value

	// Location is a pointer to the current source code location (uint32_t*).
	Location *codegen.Value

	// Globals is a pointer to the current API's global state variables.
	Globals *codegen.Value

	// Arena is a pointer to the current memory arena (arena*).
	Arena *codegen.Value

	parent      *S
	parameters  map[*semantic.Parameter]*codegen.Value
	locals      map[*semantic.Local]*codegen.Value
	locationIdx int
	onExitLogic []func()
}

func (s *S) enter(f func(*S)) {
	locals := make(map[*semantic.Local]*codegen.Value, len(s.locals))
	for l, v := range s.locals {
		locals[l] = v
	}

	child := &S{
		Builder:    s.Builder,
		parent:     s,
		parameters: s.parameters,
		locals:     locals,
		Ctx:        s.Ctx,
		Location:   s.Location,
		Globals:    s.Globals,
		Arena:      s.Arena,
	}

	f(child)

	child.exit()
}

// Return overrides codegen.Builder.Return to ensure all the scopes are
// popped before emitting the terminating instruction.
func (s *S) Return(val *codegen.Value) {
	for s := s; s != nil; s = s.parent {
		s.exit()
	}
	s.Builder.Return(val)
}

// If overrides codegen.Builder.If to ensure all the scopes are popped after
// onTrue reaches its last instruction.
func (s *S) If(cond *codegen.Value, onTrue func(s *S)) {
	s.Builder.If(cond, func() { s.enter(onTrue) })
}

// IfElse overrides codegen.Builder.IfElse to ensure all the scopes are
// popped after onTrue and onFalse reach their last instruction.
func (s *S) IfElse(cond *codegen.Value, onTrue, onFalse func(s *S)) {
	s.Builder.IfElse(cond,
		func() { s.enter(onTrue) },
		func() { s.enter(onFalse) },
	)
}

// ForN overrides codegen.Builder.ForN to ensure all the scopes are popped after
// cb reaches its last instruction.
func (s *S) ForN(n *codegen.Value, cb func(s *S, iterator *codegen.Value) (cont *codegen.Value)) {
	s.Builder.ForN(n, func(iterator *codegen.Value) *codegen.Value {
		var cont *codegen.Value
		s.enter(func(s *S) { cont = cb(s, iterator) })
		return cont
	})
}

// SwitchCase is a single condition and block used as a case statement in a
// switch.
type SwitchCase struct {
	Conditions func(*S) []*codegen.Value
	Block      func(*S)
}

// Switch overrides codegen.Builder.Switch to ensure all the scopes are
// popped after each condition and block reach their last instruction.
func (s *S) Switch(cases []SwitchCase, defaultCase func(s *S)) {
	cs := make([]codegen.SwitchCase, len(cases))
	for i, c := range cases {
		i, c := i, c
		cs[i] = codegen.SwitchCase{
			Conditions: func() []*codegen.Value {
				var out []*codegen.Value
				s.enter(func(s *S) { out = c.Conditions(s) })
				return out
			},
			Block: func() { s.enter(c.Block) },
		}
	}
	var dc func()
	if defaultCase != nil {
		dc = func() { s.enter(defaultCase) }
	}
	s.Builder.Switch(cs, dc)
}

func (s *S) onExit(f func()) {
	s.onExitLogic = append(s.onExitLogic, f)
}

func (s *S) exit() {
	for _, f := range s.onExitLogic {
		f()
	}
}
