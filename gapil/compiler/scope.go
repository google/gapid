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
)

// S is a nestable compiler scope.
// A scope holds parameters and local variables.
type S struct {
	// The scope can emit any instructions into the current scope block.
	*codegen.Builder

	// Ctx is a pointer to the active context (context_t*).
	Ctx *codegen.Value

	// Arena is a pointer to the current memory arena (arena*).
	Arena *codegen.Value
}

// If overrides codegen.Builder.If to ensure all the scopes are popped after
// onTrue reaches its last instruction.
func (s *S) If(cond *codegen.Value, onTrue func(s *S)) {
	s.Builder.If(cond, func() { onTrue(s) })
}

// IfElse overrides codegen.Builder.IfElse to ensure all the scopes are
// popped after onTrue and onFalse reach their last instruction.
func (s *S) IfElse(cond *codegen.Value, onTrue, onFalse func(s *S)) {
	s.Builder.IfElse(cond,
		func() { onTrue(s) },
		func() { onFalse(s) },
	)
}

// ForN overrides codegen.Builder.ForN to ensure all the scopes are popped after
// cb reaches its last instruction.
func (s *S) ForN(n *codegen.Value, cb func(s *S, iterator *codegen.Value) (cont *codegen.Value)) {
	s.Builder.ForN(n, func(iterator *codegen.Value) *codegen.Value {
		return cb(s, iterator)
	})
}
