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

// Package shadertools wraps around external C code for manipulating shaders.
package shadertools

// Instruction represents a SPIR-V instruction.
type Instruction struct {
	Id     uint32   // Result id.
	Opcode uint32   // Opcode.
	Words  []uint32 // Operands.
	Name   string   // Optional symbol name.
}

// CodeWithDebugInfo is the result returned by ConvertGlsl.
type CodeWithDebugInfo struct {
	Ok                bool          // Whether the call succeeds.
	Message           string        // Error message if failed.
	SourceCode        string        // Modified GLSL.
	DisassemblyString string        // Diassembly of modified GLSL.
	Info              []Instruction // A set of SPIR-V debug instructions.
}

// Options controls how ConvertGlsl converts its passed-in GLSL source code.
type Option struct {
	// Whether the passed-in shader is of the fragment stage.
	IsFragmentShader bool
	// Whether the passed-in shader is of the vertex stage.
	IsVertexShader bool
	// Whether to add prefix to all non-builtin symbols.
	PrefixNames bool
	// The name prefix to be added to all non-builtin symbols.
	NamesPrefix string /* optional */
	// Whether to create a corresponding output variable for each input variable.
	AddOutputsForInputs bool
	// The name prefix of added output variables.
	OutputPrefix string /* optional */
	// Whether to make the generated GLSL code debuggable.
	MakeDebuggable bool
	// Whether to check the generated GLSL code compiles again.
	CheckAfterChanges bool
	// Whether to disassemble the generated GLSL code.
	Disassemble bool
}
