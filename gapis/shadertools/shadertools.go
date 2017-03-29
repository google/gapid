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

//#cgo LDFLAGS: -lspv -lkhronos -lstdc++ -lpthread -lm
//#include "cc/libmanager.h"
//#include <stdlib.h>
//// Workaround for https://github.com/golang/go/issues/8756 (fixed in go 1.8):
//#cgo windows LDFLAGS: -Wl,--allow-multiple-definition
import "C"

import (
	"bytes"
	"fmt"
	"unsafe"
)

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

func FormatDebugInfo(insts []Instruction, linePrefix string) string {
	var buffer bytes.Buffer
	for _, inst := range insts {
		buffer.WriteString(linePrefix)
		if inst.Id != 0 {
			buffer.WriteString(fmt.Sprintf("%%%-5v = ", inst.Id))
		} else {
			buffer.WriteString(fmt.Sprintf("       = "))
		}
		buffer.WriteString("Op")
		buffer.WriteString(OpcodeToString(inst.Opcode))
		for _, word := range inst.Words {
			buffer.WriteString(fmt.Sprintf(" %v", word))
		}
		if inst.Name != "" {
			buffer.WriteString(fmt.Sprintf(" \"%v\"", inst.Name))
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
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

// ConvertGlsl modifies the given GLSL according to the options specified via
// option and returns the modification status and result. Possible
// modifications includes creating output variables for input variables,
// prefixing all non-builtin symbols with a given prefix, etc.
func ConvertGlsl(source string, option *Option) CodeWithDebugInfo {
	np := C.CString(option.NamesPrefix)
	op := C.CString(option.OutputPrefix)
	opts := C.struct_options_t{
		is_fragment_shader:     C.bool(option.IsFragmentShader),
		is_vertex_shader:       C.bool(option.IsVertexShader),
		prefix_names:           C.bool(option.PrefixNames),
		names_prefix:           np,
		add_outputs_for_inputs: C.bool(option.AddOutputsForInputs),
		output_prefix:          op,
		make_debuggable:        C.bool(option.MakeDebuggable),
		check_after_changes:    C.bool(option.CheckAfterChanges),
		disassemble:            C.bool(option.Disassemble),
	}
	csource := C.CString(source)
	result := C.convertGlsl(csource, C.size_t(len(source)), &opts)
	C.free(unsafe.Pointer(np))
	C.free(unsafe.Pointer(op))
	C.free(unsafe.Pointer(csource))

	ret := CodeWithDebugInfo{
		Ok:                bool(result.ok),
		Message:           C.GoString(result.message),
		SourceCode:        C.GoString(result.source_code),
		DisassemblyString: C.GoString(result.disassembly_string),
	}
	if result.info != nil {
		c_insts := (*[1 << 30]C.struct_instruction_t)(unsafe.Pointer(result.info.insts))
		for i := 0; i < int(result.info.insts_num); i++ {
			c_inst := c_insts[i]
			inst := Instruction{
				Id:     uint32(c_inst.id),
				Opcode: uint32(c_inst.opcode),
				Words:  make([]uint32, 0, c_inst.words_num),
				Name:   C.GoString(c_inst.name),
			}
			c_words := (*[1 << 30]C.uint32_t)(unsafe.Pointer(c_inst.words))
			for j := 0; j < int(c_inst.words_num); j++ {
				inst.Words = append(inst.Words, uint32(c_words[j]))
			}
			ret.Info = append(ret.Info, inst)
		}
	}
	C.deleteGlslCodeWithDebug(result)
	return ret
}

// DisassembleSpirvBinary disassembles the given SPIR-V binary words by calling
// SPIRV-Tools and returns the disassembly. Returns an empty string if
// diassembling fails.
func DisassembleSpirvBinary(words []uint32) string {
	source := ""
	if len(words) > 0 {
		spirv := C.getDisassembleText((*C.uint32_t)(&words[0]), C.size_t(len(words)))
		source = C.GoString(spirv)
		C.deleteDisassembleText(spirv)
	}

	return source
}

// AssembleSpirvText assembles the given SPIR-V text chars by calling
// SPIRV-Tools and returns the slice for the encoded binary. Returns nil
// if assembling fails.
func AssembleSpirvText(chars string) []uint32 {
	text := C.CString(chars)
	spirv := C.assembleToBinary(text)
	C.free(unsafe.Pointer(text))

	if spirv == nil {
		return nil
	}

	count := uint64(spirv.words_num)
	words := make([]uint32, count)
	// TODO: Remove the following hack and encoding the data without using unsafe.
	data := (*[1 << 30]uint32)(unsafe.Pointer(spirv.words))[:count:count]
	copy(words, data)
	C.deleteBinary(spirv)

	return words
}

// OpcodeToString converts opcode number to human readable string.
func OpcodeToString(opcode uint32) string {
	return C.GoString(C.opcodeToString(C.uint32_t(opcode)))
}
