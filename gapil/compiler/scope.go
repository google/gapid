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

type scope struct {
	*codegen.Builder
	parent      *scope
	parameters  map[*semantic.Parameter]*codegen.Value
	locals      map[*semantic.Local]*codegen.Value
	ctx         *codegen.Value // ExecutionContext*
	location    *codegen.Value // u32*
	globals     *codegen.Value // globals*
	arena       *codegen.Value // arena*
	locationIdx int
	onExitLogic []func()
}

func (s *scope) enter(f func(s *scope)) {
	locals := make(map[*semantic.Local]*codegen.Value, len(s.locals))
	for l, v := range s.locals {
		locals[l] = v
	}

	child := &scope{
		Builder:    s.Builder,
		parent:     s,
		parameters: s.parameters,
		locals:     locals,
		ctx:        s.ctx,
		location:   s.location,
		globals:    s.globals,
		arena:      s.arena,
	}

	f(child)

	child.exit()
}

func (s *scope) onExit(f func()) {
	s.onExitLogic = append(s.onExitLogic, f)
}

func (s *scope) exit() {
	for _, f := range s.onExitLogic {
		f()
	}
}
