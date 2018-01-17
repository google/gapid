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
	"unsafe"

	"llvm/bindings/go/llvm"
)

func init() {
	llvm.InitializeNativeTarget()
	llvm.InitializeNativeAsmPrinter()
}

// Executor executes module functions.
type Executor struct {
	llvm     llvm.ExecutionEngine
	funcPtrs map[string]unsafe.Pointer
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

	if optimize {
		pass := llvm.NewPassManager()
		defer pass.Dispose()

		pass.AddFunctionInliningPass()
		pass.AddConstantPropagationPass()
		pass.AddInstructionCombiningPass()
		pass.AddPromoteMemoryToRegisterPass()
		pass.AddGVNPass()
		pass.AddCFGSimplificationPass()
		pass.AddAggressiveDCEPass()
		pass.Run(m.llvm)
	}

	return &Executor{
		llvm:     engine,
		funcPtrs: map[string]unsafe.Pointer{},
	}, nil
}

func (e *Executor) FunctionAddress(f Function) unsafe.Pointer {
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
