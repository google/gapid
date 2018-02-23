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

func (s *S) onExit(f func()) {
	s.onExitLogic = append(s.onExitLogic, f)
}

func (s *S) exit() {
	for _, f := range s.onExitLogic {
		f()
	}
}
