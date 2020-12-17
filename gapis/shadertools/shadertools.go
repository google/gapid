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
//#include "spirv_reflect.h"
import "C"

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text"
)

var mutex sync.Mutex

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

func ExtractDebugSource(shader []uint32) (string, error) {
	spvReflectErr := func(res C.SpvReflectResult) error {
		if res == C.SPV_REFLECT_RESULT_SUCCESS {
			return nil
		}
		return fmt.Errorf("SPIRV-Reflect failed with error code %v\n", res)
	}

	if len(shader) == 0 {
		return "", errors.New("Empty Shader")
	}

	module := C.SpvReflectShaderModule{}

	shaderPtr := unsafe.Pointer(&shader[0])

	err := spvReflectErr(C.spvReflectCreateShaderModule(C.size_t(len(shader)*4), shaderPtr, &module))
	if err != nil {
		return "", err
	}
	defer C.spvReflectDestroyShaderModule(&module)

	if module.source_source == nil {
		return "", errors.New("No Source File Detected")
	}

	return C.GoString(module.source_source), nil
}

// ParseAllDescriptorSets determines what descriptor sets are implied by the
// shader, for all entry points of the shader.
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
	entryPoints := module.entry_points

	for i := uint32(0); i < uint32(nEntryPoints); i++ {
		// Access module.entry_points[i]: use C pointer as array, do pointer
		// arithmetic as documented in https://golang.org/pkg/unsafe/#Pointer
		entryPointStruct := (*C.SpvReflectEntryPoint)(unsafe.Pointer(uintptr(unsafe.Pointer(entryPoints)) + uintptr(i)*unsafe.Sizeof(*(entryPoints))))

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
