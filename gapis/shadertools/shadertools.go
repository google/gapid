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

//#include "cc/libmanager.h"
//#include <stdlib.h>
//#include <third_party/SPIRV-Reflect/spirv_reflect.h>
import "C"

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text"
)

var mutex sync.Mutex

// Instruction represents a SPIR-V instruction.
type Instruction struct {
	ID     uint32   // Result identifer.
	Opcode uint32   // Opcode.
	Words  []uint32 // Operands.
	Name   string   // Optional symbol name.
}

// CodeWithDebugInfo is the result returned by ConvertGlsl.
type CodeWithDebugInfo struct {
	SourceCode        string        // Modified GLSL.
	DisassemblyString string        // Diassembly of modified GLSL.
	Info              []Instruction // A set of SPIR-V debug instructions.
}

// FormatDebugInfo returns the instructions as a string.
func FormatDebugInfo(insts []Instruction, linePrefix string) string {
	var buffer bytes.Buffer
	for _, inst := range insts {
		buffer.WriteString(linePrefix)
		if inst.ID != 0 {
			buffer.WriteString(fmt.Sprintf("%%%-5v = ", inst.ID))
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

// ShaderType is the enumerator of shader types.
type ShaderType int

const (
	TypeVertex         = ShaderType(C.VERTEX)
	TypeTessControl    = ShaderType(C.TESS_CONTROL)
	TypeTessEvaluation = ShaderType(C.TESS_EVALUATION)
	TypeGeometry       = ShaderType(C.GEOMETRY)
	TypeFragment       = ShaderType(C.FRAGMENT)
	TypeCompute        = ShaderType(C.COMPUTE)
)

// ClientType is the enumerator of client types.
type ClientType int

const (
	OpenGL   = ClientType(C.OPENGL)
	OpenGLES = ClientType(C.OPENGLES)
	Vulkan   = ClientType(C.VULKAN)
)

func (t ShaderType) String() string {
	switch t {
	case TypeVertex:
		return "Vertex"
	case TypeTessControl:
		return "TessControl"
	case TypeTessEvaluation:
		return "TessEvaluation"
	case TypeGeometry:
		return "Geometry"
	case TypeFragment:
		return "Fragment"
	case TypeCompute:
		return "Compute"
	default:
		return "Unknown"
	}
}

// ConvertOptions controls how ConvertGlsl converts its passed-in GLSL source code.
type ConvertOptions struct {
	// The type of shader.
	ShaderType ShaderType
	// The target GLSL version (default 330).
	TargetGLSLVersion int
	// Shader source preamble.
	Preamble string
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
	// If true, let some minor invalid statements compile.
	Relaxed bool
	// If true, optimizations that require high-end GL versions, or extensions
	// will be stripped. These optimizations should have no impact on the end
	// result of the shader, but may impact performance.
	// Example: Early Fragment Test.
	StripOptimizations bool
}

// ConvertGlsl modifies the given GLSL according to the options specified via
// o and returns the modification status and result. Possible modifications
// includes creating output variables for input variables, prefixing all
// non-builtin symbols with a given prefix, etc.
func ConvertGlsl(source string, o *ConvertOptions) (CodeWithDebugInfo, error) {
	toFree := []unsafe.Pointer{}
	defer func() {
		for _, ptr := range toFree {
			C.free(ptr)
		}
	}()

	mutex.Lock()
	defer mutex.Unlock()

	cstr := func(s string) *C.char {
		out := C.CString(s)
		toFree = append(toFree, unsafe.Pointer(out))
		return out
	}

	opts := C.struct_convert_options_t{
		shader_type:            C.shader_type(o.ShaderType),
		preamble:               cstr(o.Preamble),
		prefix_names:           C.bool(o.PrefixNames),
		names_prefix:           cstr(o.NamesPrefix),
		add_outputs_for_inputs: C.bool(o.AddOutputsForInputs),
		output_prefix:          cstr(o.OutputPrefix),
		make_debuggable:        C.bool(o.MakeDebuggable),
		check_after_changes:    C.bool(o.CheckAfterChanges),
		disassemble:            C.bool(o.Disassemble),
		relaxed:                C.bool(o.Relaxed),
		strip_optimizations:    C.bool(o.StripOptimizations),
		target_glsl_version:    C.int(o.TargetGLSLVersion),
	}
	result := C.convertGlsl(cstr(source), C.size_t(len(source)), &opts)
	defer C.deleteGlslCodeWithDebug(result)

	ret := CodeWithDebugInfo{
		SourceCode:        C.GoString(result.source_code),
		DisassemblyString: C.GoString(result.disassembly_string),
	}

	if result.info != nil {
		cInsts := (*[1 << 30]C.struct_instruction_t)(unsafe.Pointer(result.info.insts))
		for i := 0; i < int(result.info.insts_num); i++ {
			cInst := cInsts[i]
			inst := Instruction{
				ID:     uint32(cInst.id),
				Opcode: uint32(cInst.opcode),
				Words:  make([]uint32, 0, cInst.words_num),
				Name:   C.GoString(cInst.name),
			}
			cWords := (*[1 << 30]C.uint32_t)(unsafe.Pointer(cInst.words))
			for j := 0; j < int(cInst.words_num); j++ {
				inst.Words = append(inst.Words, uint32(cWords[j]))
			}
			ret.Info = append(ret.Info, inst)
		}
	}

	if !result.ok {
		msg := []string{
			fmt.Sprintf("Failed to convert %v shader.", o.ShaderType),
		}
		if m := C.GoString(result.message); len(m) > 0 {
			msg = append(msg, m)
		}
		msg = append(msg, "Translated source:", text.LineNumber(C.GoString(result.source_code)))
		msg = append(msg, "Original source:", text.LineNumber(source))
		return ret, fault.Const(strings.Join(msg, "\n"))
	}

	return ret, nil
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

// CompileOptions controls how CompileGlsl compile its passed-in GLSL source code.
type CompileOptions struct {
	// The type of shader.
	ShaderType ShaderType
	// Either OpenGL, OpenGLES or Vulkan
	ClientType ClientType
	// Shader source preamble.
	Preamble string
}

// CompileGlsl compiles GLSL source code to SPIR-V binary words.
func CompileGlsl(source string, o CompileOptions) ([]uint32, error) {
	toFree := []unsafe.Pointer{}
	defer func() {
		for _, ptr := range toFree {
			C.free(ptr)
		}
	}()
	mutex.Lock()
	defer mutex.Unlock()

	cstr := func(s string) *C.char {
		out := C.CString(s)
		toFree = append(toFree, unsafe.Pointer(out))
		return out
	}

	opts := C.struct_compile_options_t{
		shader_type: C.shader_type(o.ShaderType),
		client_type: C.client_type(o.ClientType),
		preamble:    cstr(o.Preamble),
	}

	result := C.compileGlsl(cstr(source), &opts)
	defer C.deleteCompileResult(result)

	count := uint64(result.binary.words_num)
	words := make([]uint32, count)
	if result.ok {
		// TODO: Remove the following hack and encoding the data without using unsafe.
		data := (*[1 << 30]uint32)(unsafe.Pointer(result.binary.words))[:count:count]
		copy(words, data)
		return words, nil
	}
	msg := []string{
		fmt.Sprintf("Failed to compile %v shader.", o.ShaderType),
	}
	if m := C.GoString(result.message); len(m) > 0 {
		msg = append(msg, m)
	}
	msg = append(msg, "Source:", text.LineNumber(source))
	if len(o.Preamble) > 0 {
		msg = append(msg, "Preamble:", text.LineNumber(o.Preamble))
	}
	return words, fault.Const(strings.Join(msg, "\n"))
}

type DescriptorSets map[uint32]DescriptorSet
type DescriptorSet []DescriptorBinding

type DescriptorBinding struct {
	Set             uint32
	Binding         uint32
	SpirvId         uint32
	DescriptorType  uint32
	DescriptorCount uint32
	ShaderStage     uint32
}

func descriptorBindingLess(a DescriptorBinding, b DescriptorBinding) bool {
	if a.Set != b.Set {
		return a.Set < b.Set
	}
	if a.Binding != b.Binding {
		return a.Binding < b.Binding
	}
	// SpirvId is a unique identifier so we don't need to keep comparing after this
	return a.SpirvId < b.SpirvId
}

// ParseDescriptorSets determines what descriptor sets are implied by the shader
func ParseDescriptorSets(shader []uint32, entryPoint string) (DescriptorSets, error) {
	spvReflectErr := func(res C.SpvReflectResult) error {
		if res == C.SPV_REFLECT_RESULT_SUCCESS {
			return nil
		}
		return fmt.Errorf("SPIRV-Reflect failed with error code %v\n", res)
	}
	module := C.SpvReflectShaderModule{}
	cEntryPoint := C.CString(entryPoint)
	defer C.free(unsafe.Pointer(cEntryPoint))

	shaderPtr := unsafe.Pointer(nil)
	if len(shader) > 0 {
		shaderPtr = unsafe.Pointer(&shader[0])
	}
	if err := spvReflectErr(C.spvReflectCreateShaderModule(
		C.size_t(len(shader)*4),
		shaderPtr,
		&module)); err != nil {
		return nil, err
	}
	defer C.spvReflectDestroyShaderModule(&module)

	entryPointStruct := C.spvReflectGetEntryPoint(
		&module,
		cEntryPoint)
	if entryPointStruct == nil {
		return nil, fmt.Errorf("Entry point %v not found")
	}

	setCount := C.uint32_t(0)
	if err := spvReflectErr(C.spvReflectEnumerateEntryPointDescriptorSets(
		&module,
		cEntryPoint,
		&setCount,
		nil)); err != nil {
		return nil, err
	}
	sets := make([]*C.SpvReflectDescriptorSet, setCount)
	setsPtr := unsafe.Pointer(nil)
	if setCount > 0 {
		setsPtr = unsafe.Pointer(&sets[0])
	}
	if err := spvReflectErr(C.spvReflectEnumerateEntryPointDescriptorSets(
		&module,
		cEntryPoint,
		&setCount,
		(**C.SpvReflectDescriptorSet)(setsPtr),
	)); err != nil {
		return nil, err
	}

	res := DescriptorSets{}
	for _, set := range sets {
		bindings := make(DescriptorSet, set.binding_count)
		for i := C.uint32_t(0); i < set.binding_count; i++ {
			bindingPtr := uintptr(unsafe.Pointer(set.bindings)) +
				uintptr(i)*unsafe.Sizeof(*set.bindings)
			binding := *(**C.SpvReflectDescriptorBinding)(unsafe.Pointer(bindingPtr))
			// If it's an array, need to get total descriptor count
			descriptorCount := C.uint32_t(1)
			for j := C.uint32_t(0); j < binding.array.dims_count; j++ {
				descriptorCount *= binding.array.dims[j]
			}
			bindings[i] = DescriptorBinding{
				Set:             uint32(binding.set),
				Binding:         uint32(binding.binding),
				SpirvId:         uint32(binding.spirv_id),
				DescriptorType:  uint32(binding.descriptor_type),
				DescriptorCount: uint32(descriptorCount),
				ShaderStage:     uint32(entryPointStruct.shader_stage),
			}
		}
		sort.Slice(bindings, func(i, j int) bool {
			return descriptorBindingLess(bindings[i], bindings[j])
		})
		res[uint32(set.set)] = bindings
	}

	return res, nil
}

// ParseDescriptorSets determines what descriptor sets are implied by the shader
func ParseAllDescriptorSets(shader []uint32) (map[string]DescriptorSets, error) {
	out := make(map[string]DescriptorSets)
	spvReflectErr := func(res C.SpvReflectResult) error {
		if res == C.SPV_REFLECT_RESULT_SUCCESS {
			return nil
		}
		return fmt.Errorf("SPIRV-Reflect failed with error code %v\n", res)
	}
	module := C.SpvReflectShaderModule{}

	shaderPtr := unsafe.Pointer(nil)
	if len(shader) > 0 {
		shaderPtr = unsafe.Pointer(&shader[0])
	}
	if err := spvReflectErr(C.spvReflectCreateShaderModule(
		C.size_t(len(shader)*4),
		shaderPtr,
		&module)); err != nil {
		return nil, err
	}
	defer C.spvReflectDestroyShaderModule(&module)

	nEntryPoints := module.entry_point_count

	for i := uint32(0); i < uint32(nEntryPoints); i++ {
		entryPointStruct := module.entry_points

		setCount := C.uint32_t(0)
		if err := spvReflectErr(C.spvReflectEnumerateEntryPointDescriptorSets(
			&module,
			entryPointStruct.name,
			&setCount,
			nil)); err != nil {
			return nil, err
		}
		sets := make([]*C.SpvReflectDescriptorSet, setCount)
		setsPtr := unsafe.Pointer(nil)
		if setCount > 0 {
			setsPtr = unsafe.Pointer(&sets[0])
		}
		if err := spvReflectErr(C.spvReflectEnumerateEntryPointDescriptorSets(
			&module,
			entryPointStruct.name,
			&setCount,
			(**C.SpvReflectDescriptorSet)(setsPtr),
		)); err != nil {
			return nil, err
		}

		res := DescriptorSets{}
		for _, set := range sets {
			bindings := make(DescriptorSet, set.binding_count)
			for i := C.uint32_t(0); i < set.binding_count; i++ {
				bindingPtr := uintptr(unsafe.Pointer(set.bindings)) +
					uintptr(i)*unsafe.Sizeof(*set.bindings)
				binding := *(**C.SpvReflectDescriptorBinding)(unsafe.Pointer(bindingPtr))
				// If it's an array, need to get total descriptor count
				descriptorCount := C.uint32_t(1)
				for j := C.uint32_t(0); j < binding.array.dims_count; j++ {
					descriptorCount *= binding.array.dims[j]
				}
				bindings[i] = DescriptorBinding{
					Set:             uint32(binding.set),
					Binding:         uint32(binding.binding),
					SpirvId:         uint32(binding.spirv_id),
					DescriptorType:  uint32(binding.descriptor_type),
					DescriptorCount: uint32(descriptorCount),
					ShaderStage:     uint32(entryPointStruct.shader_stage),
				}
			}
			sort.Slice(bindings, func(i, j int) bool {
				return descriptorBindingLess(bindings[i], bindings[j])
			})
			res[uint32(set.set)] = bindings
		}
		out[C.GoString(entryPointStruct.name)] = res
	}

	return out, nil
}
