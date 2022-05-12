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

// Program is the output of a compilation.
type Program struct {
	// Codegen is the codegen module.
	Codegen *codegen.Module

	// Module is the global that holds the generated gapil_module structure.
	Module codegen.Global
}

// CommandInfo holds the generated execute function for a given command.
type CommandInfo struct {
	// The execute function for the given command.
	// The function has the signature: void (ctx*, Params*)
	Execute *codegen.Function

	// The Params structure that is passed to Execute.
	Parameters *codegen.Struct
}

// StructInfo holds the generated structure for a given structure type.
type StructInfo struct {
	Type *codegen.Struct
}

// MapInfo holds the generated map info for a given semantic map type.
type MapInfo struct {
	Type     *codegen.Struct // Maps are held as pointers to these structs
	Elements *codegen.Struct
	Key      codegen.Type
	Val      codegen.Type
	Element  codegen.Type
}

// Dump returns the full LLVM IR for the program.
func (p *Program) Dump() string {
	return p.Codegen.String()
}
