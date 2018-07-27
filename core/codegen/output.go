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

package codegen

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/google/gapid/core/app/linker"
	"github.com/google/gapid/core/os/device"

	"llvm/bindings/go/llvm"
)

func init() {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()
}

// Executor executes module functions.
type Executor struct {
	llvm     llvm.ExecutionEngine
	funcPtrs map[string]unsafe.Pointer
}

// Object compiles the module down to an object file.
func (m *Module) Object(optimize bool) ([]byte, error) {
	t, err := llvm.GetTargetFromTriple(m.triple.String())
	if err != nil {
		return nil, fmt.Errorf("Couldn't get target for triple '%v': %v", m.triple, err)
	}
	cpu := ""
	features := ""
	opt := llvm.CodeGenLevelNone
	if optimize {
		opt = llvm.CodeGenLevelDefault
	}
	reloc := llvm.RelocPIC
	model := llvm.CodeModelDefault
	tm := t.CreateTargetMachine(m.triple.String(), cpu, features, opt, reloc, model)
	defer tm.Dispose()

	td := tm.CreateTargetData()
	defer td.Dispose()
	m.validateTargetData(td)

	buf, err := tm.EmitToMemoryBuffer(m.llvm, llvm.ObjectFile)
	if err != nil {
		return nil, err
	}
	defer buf.Dispose()
	return buf.Bytes(), nil
}

func (m *Module) validateTargetData(td llvm.TargetData) {
	abi := m.target
	errs := []string{}
	check := func(llvm, gapid interface{}, name string) {
		if reflect.DeepEqual(llvm, gapid) {
			return
		}
		errs = append(errs, fmt.Sprintf("%v target mismatch for %v: %v (llvm) != %v (gapid)", name, abi.Name, llvm, gapid))
	}
	checkTD := func(ty Type, dtl *device.DataTypeLayout) {
		check(td.TypeStoreSize(ty.llvmTy()), uint64(dtl.Size), ty.String()+"-size")
		check(td.ABITypeAlignment(ty.llvmTy()), int(dtl.Alignment), ty.String()+"-align")
	}

	layout := abi.MemoryLayout
	isLE := td.ByteOrder() == llvm.LittleEndian
	check(isLE, layout.Endian == device.LittleEndian, "is-little-endian")
	check(td.PointerSize(), int(layout.Pointer.Size), "pointer-size")

	checkTD(m.Types.Pointer(m.Types.Int), layout.Pointer)
	checkTD(m.Types.Int, layout.Integer)
	checkTD(m.Types.Size, layout.Size)
	checkTD(m.Types.Int64, layout.I64)
	checkTD(m.Types.Int32, layout.I32)
	checkTD(m.Types.Int16, layout.I16)
	checkTD(m.Types.Int8, layout.I8)
	checkTD(m.Types.Float32, layout.F32)
	checkTD(m.Types.Float64, layout.F64)

	if len(errs) > 0 {
		panic(fmt.Errorf("%v has ABI mismatches!\n%v", abi.Name, strings.Join(errs, "\n")))
	}
}

// Optimize optimizes the module.
func (m *Module) Optimize() {
	pass := llvm.NewPassManager()
	defer pass.Dispose()

	// pass.AddFunctionInliningPass() // Way too slow with this pass
	pass.AddInstructionCombiningPass()
	pass.AddReassociatePass()
	pass.AddConstantPropagationPass()
	pass.AddPromoteMemoryToRegisterPass()
	pass.AddGVNPass()
	pass.AddCFGSimplificationPass()
	pass.AddAggressiveDCEPass()
	pass.Run(m.llvm)
}

// Executor constructs an executor.
func (m *Module) Executor(optimize bool) (*Executor, error) {
	if dbg := m.llvmDbg; dbg != nil {
		dbg.Finalize() // TODO: Needed?
	}

	if err := m.Verify(); err != nil {
		return nil, err
	}

	opts := llvm.NewMCJITCompilerOptions()
	if optimize {
		opts.SetMCJITOptimizationLevel(2)
	} else {
		opts.SetMCJITOptimizationLevel(0)
	}

	engine, err := llvm.NewMCJITCompiler(m.llvm, opts)
	if err != nil {
		return nil, err
	}

	var unresolved []string
	for _, f := range m.funcs {
		if f.built || strings.HasPrefix(f.Name, "llvm.") {
			continue
		}
		if linker.ProcAddress(f.Name) == 0 {
			unresolved = append(unresolved, fmt.Sprint(f))
		}
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		msg := fmt.Sprintf("Unresolved external functions:\n%v", strings.Join(unresolved, "\n"))
		fail(msg)
	}

	return &Executor{
		llvm:     engine,
		funcPtrs: map[string]unsafe.Pointer{},
	}, nil
}

func (e *Executor) FunctionAddress(f *Function) unsafe.Pointer {
	ptr, ok := e.funcPtrs[f.Name]
	if !ok {
		ptr = e.llvm.PointerToGlobal(f.llvm)
		e.funcPtrs[f.Name] = ptr
	}
	return ptr
}

// SizeOf returns the offset in bytes between successive objects of the
// specified type, including alignment padding.
func (e *Executor) SizeOf(t Type) int {
	return int(e.llvm.TargetData().TypeAllocSize(t.llvmTy()))
}

// AlignOf returns the preferred stack/global alignment for the specified type.
func (e *Executor) AlignOf(t Type) int {
	// TODO: Preferred alignment vs ABI alignment. Which one?
	return e.llvm.TargetData().PrefTypeAlignment(t.llvmTy())
}

func (e *Executor) FieldOffsets(s *Struct) []int {
	td := e.llvm.TargetData()
	out := make([]int, len(s.Fields))
	for i := range s.Fields {
		out[i] = int(td.ElementOffset(s.llvm, i))
	}
	return out
}

func (e *Executor) StructLayout(s *Struct) string {
	w := bytes.Buffer{}
	w.WriteString(s.Name)
	w.WriteString("{\n")
	e.writeStructLayout(s, &w, 0, "")
	w.WriteString("}")
	return w.String()
}

func (e *Executor) writeStructLayout(s *Struct, w *bytes.Buffer, base int, prefix string) {
	for i, o := range e.FieldOffsets(s) {
		f := s.Fields[i]
		w.WriteString(fmt.Sprintf(" 0x%.4x: ", base+o))
		w.WriteString(prefix)
		w.WriteString(f.Name)
		w.WriteRune('\n')
		if s, ok := f.Type.(*Struct); ok {
			e.writeStructLayout(s, w, base+o, prefix+f.Name+".")
		}
	}
}
