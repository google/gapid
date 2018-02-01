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
	"fmt"

	"llvm/bindings/go/llvm"
)

// Function represents a callable function.
type Function struct {
	Name string
	Type FunctionType
	llvm llvm.Value
	m    *Module
}

func (f Function) String() string {
	return fmt.Sprintf("%v %v(%v)", f.Type.Signature.Result, f.Name, f.Type.Signature.Parameters)
}

// SetInline makes this function prefer inlining
func (f Function) MakeInline() {
	kind := llvm.AttributeKindID("alwaysinline")
	attr := f.m.ctx.CreateEnumAttribute(kind, 0)
	f.llvm.AddFunctionAttr(attr)
}

// Build calls cb with a Builder that can construct the function body.
func (f Function) Build(cb func(*Builder)) (err error) {
	lb := f.m.ctx.NewBuilder()
	defer lb.Dispose()

	entryBlock := f.m.ctx.AddBasicBlock(f.llvm, "entry")
	firstExitBlock := f.m.ctx.AddBasicBlock(f.llvm, "exit")
	b := &Builder{
		function: &f,
		params:   make([]*Value, len(f.Type.Signature.Parameters)),
		entry:    entryBlock,
		exit:     firstExitBlock, // Note: Builder.exit may be updated with chained blocks.
		llvm:     lb,
		m:        f.m,
	}

	lb.SetInsertPointAtEnd(b.entry)

	for i, p := range f.llvm.Params() {
		b.params[i] = b.val(f.Type.Signature.Parameters[i], p)
	}

	if ty := f.Type.Signature.Result; ty != f.m.Types.Void {
		b.result = lb.CreateAlloca(ty.llvmTy(), "result")
		lb.CreateStore(llvm.ConstNull(ty.llvmTy()), b.result)
	}

	defer func() {
		if r := recover(); r != nil {
			if failure, ok := r.(buildFailure); ok {
				err = fmt.Errorf("%v", string(failure))
				panic(err) // TEMP
			} else {
				panic(r) // re-throw
			}
		}
	}()

	cb(b)

	if !b.IsBlockTerminated() {
		lb.CreateBr(firstExitBlock)
	}

	lb.SetInsertPointAtEnd(b.exit)

	if b.result.IsNil() {
		lb.CreateRetVoid()
	} else {
		lb.CreateRet(lb.CreateLoad(b.result, ""))
	}

	return nil
}
