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
	"strings"

	"llvm/bindings/go/llvm"
)

// IndexOrName can be an index (int) or field name (string).
type IndexOrName interface{}

// ValueIndexOrName can be an index (Value or int) or field name (string).
type ValueIndexOrName interface{}

// Builder is provided to the callback of Build() for building a function's body.
type Builder struct {
	function *Function       // The function being built.
	params   []*Value        // Function parameter values.
	entry    llvm.BasicBlock // Function entry block.
	exit     llvm.BasicBlock // Function exit block.
	result   llvm.Value      // Function return value.
	llvm     llvm.Builder
	m        *Module
}

// buildFailure is a special type thrown as a panic by fail().
// They are caught in Build().
type buildFailure string

func (f buildFailure) String() string { return string(f) }

func fail(msg string, args ...interface{}) {
	str := fmt.Sprintf(msg, args...)
	panic(buildFailure(str))
}

// Call invokes the function f with the specified arguments
func (b *Builder) Call(f *Function, args ...*Value) *Value {
	return b.call(f.llvm, f.Type.Signature, f.Name, args)
}

// CallIndirect invokes the function pointer f with the specified arguments
func (b *Builder) CallIndirect(f *Value, args ...*Value) *Value {
	var fty *FunctionType
	if ptrTy, ok := Underlying(f.Type()).(Pointer); ok {
		fty, _ = Underlying(ptrTy.Element).(*FunctionType)
	}
	if fty == nil {
		fail("CallIndirect() can only be called with function pointers. Got %v", f.Type())
	}
	return b.call(f.llvm, fty.Signature, f.name, args)
}

func (b *Builder) call(fn llvm.Value, sig Signature, name string, args []*Value) *Value {
	l := b.callArgs(sig, name, args)
	if sig.Result == b.m.Types.Void {
		name = ""
	}
	return b.val(sig.Result, b.llvm.CreateCall(fn, l, name))
}

// Invoke invokes the function f with the specified arguments.
// If an exception is thrown while calling f, then cleanup will be called before
// rethrowing the exception.
func (b *Builder) Invoke(f *Function, cleanup func(), args ...*Value) *Value {
	fn, sig, name := f.llvm, f.Type.Signature, f.Name

	l := b.callArgs(sig, name, args)

	then := b.m.ctx.AddBasicBlock(b.function.llvm, fmt.Sprintf("%v_nothrow", name))
	throw := b.m.ctx.AddBasicBlock(b.function.llvm, fmt.Sprintf("%v_catch", name))

	if sig.Result == b.m.Types.Void {
		name = ""
	}

	res := b.val(sig.Result, b.llvm.CreateInvoke(fn, l, then, throw, name))

	b.function.llvm.SetPersonality(b.m.exceptions.personalityFn.llvm)

	b.block(throw, llvm.BasicBlock{}, func() {
		lp := b.llvm.CreateLandingPad(b.m.exceptions.exceptionTy.llvmTy(), 0, "cleanup")
		lp.SetCleanup(true)
		cleanup()
		b.llvm.CreateResume(lp)
	})

	b.setInsertPointAtEnd(then)

	return res
}

// Throw throws an exception with the given value.
func (b *Builder) Throw(v *Value) {
	b.m.exceptions.throw(b, v)
}

func (b *Builder) callArgs(sig Signature, name string, args []*Value) []llvm.Value {
	if sig.Variadic {
		if g, e := len(args), len(sig.Parameters); g < e {
			fail("Got %d arguments, but needed %d to call %v", g, e, sig.string(name))
		}
	} else if g, e := len(args), len(sig.Parameters); g != e {
		fail("Got %d arguments, but needed %d to call %v", g, e, sig.string(name))
	}
	l := make([]llvm.Value, len(args))
	for i, a := range args {
		if a == nil {
			fail("Argument %d is nil when attempting to call %v", i, sig.string(name))
		}
		l[i] = a.llvm
		if i < len(sig.Parameters) {
			if g, e := a.ty, sig.Parameters[i]; g != e {
				fail("Incorrect argument type for parameter %d when calling %v: Got %v, expected %v",
					i, sig.string(name), g.TypeName(), e.TypeName())
			}
		}
	}
	return l
}

// Parameter returns i'th function parameter
func (b *Builder) Parameter(i int) *Value {
	return b.params[i]
}

// Undef returns a new undefined value of the specified type.
func (b *Builder) Undef(ty Type) *Value {
	return b.val(ty, llvm.Undef(ty.llvmTy()))
}

// Local returns a pointer to a new local variable with the specified name and
// type.
func (b *Builder) Local(name string, ty Type) *Value {
	block := b.llvm.GetInsertBlock()
	b.llvm.SetInsertPoint(b.entry, b.entry.FirstInstruction())
	local := b.llvm.CreateAlloca(ty.llvmTy(), "")
	b.setInsertPointAtEnd(block)
	return b.val(b.m.Types.Pointer(ty), local).SetName(name)
}

// LocalInit returns a new local variable with the specified name and initial value.
func (b *Builder) LocalInit(name string, val *Value) *Value {
	local := b.Local(name, val.ty)
	local.Store(val)
	return local
}

// If builds an if statement.
func (b *Builder) If(cond *Value, onTrue func()) {
	b.IfElse(cond, onTrue, nil)
}

// IfElse builds an if-else statement.
func (b *Builder) IfElse(cond *Value, onTrue, onFalse func()) {
	trueBlock := b.m.ctx.AddBasicBlock(b.function.llvm, "if_true")
	var falseBlock llvm.BasicBlock
	if onFalse != nil {
		falseBlock = b.m.ctx.AddBasicBlock(b.function.llvm, "if_false")
	}
	endBlock := b.m.ctx.AddBasicBlock(b.function.llvm, "end_if")
	if onFalse == nil {
		falseBlock = endBlock
	}

	b.llvm.CreateCondBr(cond.llvm, trueBlock, falseBlock)

	b.block(trueBlock, endBlock, onTrue)

	if onFalse != nil {
		b.block(falseBlock, endBlock, onFalse)
	}

	b.setInsertPointAtEnd(endBlock)
}

// While builds a logic block with the following psuedocode:
//
// while test() {
//   loop()
// }
//
func (b *Builder) While(test func() *Value, loop func()) {
	testBlock := b.m.ctx.AddBasicBlock(b.function.llvm, "while_test")
	loopBlock := b.m.ctx.AddBasicBlock(b.function.llvm, "while_loop")
	exitBlock := b.m.ctx.AddBasicBlock(b.function.llvm, "while_exit")

	b.llvm.CreateBr(testBlock)

	b.block(testBlock, llvm.BasicBlock{}, func() {
		cond := test()
		if !b.IsBlockTerminated() {
			b.llvm.CreateCondBr(cond.llvm, loopBlock, exitBlock)
		}
	})

	b.block(loopBlock, testBlock, loop)

	b.setInsertPointAtEnd(exitBlock)
}

// ForN builds a logic block with the following psuedocode:
//
// for it := 0; it < n; it++ {
//   cont := cb()
//   if cont == false { break; }
// }
//
// If cb returns nil then the loop will never exit early.
func (b *Builder) ForN(n *Value, cb func(iterator *Value) (cont *Value)) {
	one := llvm.ConstInt(n.Type().llvmTy(), 1, false)
	zero := b.Zero(n.Type())
	iterator := b.LocalInit("loop_iterator", zero)

	test := b.m.ctx.AddBasicBlock(b.function.llvm, "for_n_test")
	loop := b.m.ctx.AddBasicBlock(b.function.llvm, "for_n_loop")
	exit := b.m.ctx.AddBasicBlock(b.function.llvm, "for_n_exit")

	b.llvm.CreateBr(test)

	b.block(test, llvm.BasicBlock{}, func() {
		done := b.llvm.CreateICmp(llvm.IntSLT, iterator.Load().llvm, n.llvm, "for_n_condition")
		b.llvm.CreateCondBr(done, loop, exit)
	})

	b.block(loop, llvm.BasicBlock{}, func() {
		it := iterator.Load()
		cont := cb(it)
		if b.IsBlockTerminated() {
			return
		}
		b.llvm.CreateStore(b.llvm.CreateAdd(it.llvm, one, "for_n_iterator_inc"), iterator.llvm)
		if cont == nil {
			b.llvm.CreateBr(test)
		} else {
			assertTypesEqual(cont.ty, b.m.Types.Bool)
			b.llvm.CreateCondBr(cont.llvm, test, exit)
		}
	})

	b.setInsertPointAtEnd(exit)
}

// SwitchCase is a single condition and block used as a case statement in a
// switch.
type SwitchCase struct {
	Conditions func() []*Value
	Block      func()
}

// Switch builds a switch statement.
func (b *Builder) Switch(cases []SwitchCase, defaultCase func()) {
	tests := make([]llvm.BasicBlock, len(cases))
	blocks := make([]llvm.BasicBlock, len(cases))
	for i := range cases {
		tests[i] = b.m.ctx.AddBasicBlock(b.function.llvm, fmt.Sprintf("switch_case_%d_test", i))
		blocks[i] = b.m.ctx.AddBasicBlock(b.function.llvm, fmt.Sprintf("switch_case_%d_block", i))
	}

	var defaultBlock llvm.BasicBlock
	if defaultCase != nil {
		defaultBlock = b.m.ctx.AddBasicBlock(b.function.llvm, "switch_case_default")
		tests = append(tests, defaultBlock)
	}

	exit := b.m.ctx.AddBasicBlock(b.function.llvm, "end_switch")

	b.llvm.CreateBr(tests[0])

	for i, c := range cases {
		i, c := i, c
		b.block(tests[i], llvm.BasicBlock{}, func() {
			conds := c.Conditions()
			match := conds[0]
			for _, c := range conds[1:] {
				match = b.Or(match, c)
			}
			next := exit
			if i+1 < len(tests) {
				next = tests[i+1]
			}
			b.llvm.CreateCondBr(match.llvm, blocks[i], next)
		})
		b.block(blocks[i], exit, c.Block)
	}

	if defaultCase != nil {
		b.block(defaultBlock, exit, defaultCase)
	}

	b.setInsertPointAtEnd(exit)
}

// Return returns execution of the function with the given value
func (b *Builder) Return(val *Value) {
	if val != nil {
		assertTypesEqual(val.Type(), b.function.Type.Signature.Result)
		b.llvm.CreateStore(val.llvm, b.result)
	} else if !b.result.IsNil() {
		b.llvm.CreateStore(llvm.ConstNull(b.function.Type.Signature.Result.llvmTy()), b.result)
	}
	b.llvm.CreateBr(b.exit)
}

// IsBlockTerminated returns true if the last instruction is a terminator
// (unconditional jump). It is illegal to write another instruction after a
// terminator.
func (b *Builder) IsBlockTerminated() bool {
	return !b.llvm.GetInsertBlock().LastInstruction().IsATerminatorInst().IsNil()
}

// FuncAddr returns the pointer to the given function.
func (b *Builder) FuncAddr(f *Function) *Value {
	return b.val(b.m.Types.Pointer(f.Type), f.llvm)
}

// PrintfSpecifier returns the string and values that can be used to print v
// with printf.
func (b *Builder) PrintfSpecifier(v *Value) (string, []*Value) {
	t := v.Type()
	switch t {
	case b.m.Types.Bool:
		v = b.Select(v, b.Scalar("true"), b.Scalar("false"))
		return "%s", []*Value{v}
	case b.m.Types.Float32:
		return "%f", []*Value{v}
	case b.m.Types.Float64:
		return "%d", []*Value{v}
	case b.m.Types.Int, b.m.Types.Int8, b.m.Types.Int16, b.m.Types.Int32, b.m.Types.Int64:
		return "%lld", []*Value{v.Cast(b.m.Types.Int64)}
	case b.m.Types.Uint, b.m.Types.Uint8, b.m.Types.Uint16, b.m.Types.Uint32, b.m.Types.Uint64:
		return "%llu", []*Value{v.Cast(b.m.Types.Int64)}
	case b.m.Types.Uintptr:
		return "%p", []*Value{v.Cast(b.m.Types.Int64)}
	case b.m.Types.Size:
		return "%z", []*Value{v.Cast(b.m.Types.Int64)}
	case b.m.Types.Float32:
		return "%f", []*Value{v}
	case b.m.Types.Float64:
		return "%d", []*Value{v}
	}
	switch t := t.(type) {
	case Pointer:
		return "%p", []*Value{v}
	case *Struct:
		vals := []*Value{}
		sb := strings.Builder{}
		sb.WriteString(t.TypeName())
		sb.WriteString("{ ")
		for i, f := range t.Fields() {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt, val := b.PrintfSpecifier(v.Extract(i))
			sb.WriteString(f.Name)
			sb.WriteString(": ")
			sb.WriteString(fmt)
			vals = append(vals, val...)
		}
		sb.WriteString(" }")
		return sb.String(), vals
	}
	return fmt.Sprintf("<%v>", v.Type()), nil
}

// StructOf builds a struct value that holds all the values in v.
func (b *Builder) StructOf(name string, v []*Value) *Value {
	fields := make([]Field, len(v))
	for i, v := range v {
		fields[i].Type = v.Type()
	}
	s := b.Undef(b.m.Types.Struct(name, fields...))
	for i, v := range v {
		s = s.Insert(i, v)
	}
	return s
}

// block calls f to appends instructions to the specified block.
// If next is not nil and the f returns without terminating the block, then a
// unconditional jump to next is added to the block.
func (b *Builder) block(block, next llvm.BasicBlock, f func()) {
	b.setInsertPointAtEnd(block)

	f()

	if !next.IsNil() && !b.IsBlockTerminated() {
		b.llvm.CreateBr(next)
	}
}

func (b *Builder) setInsertPointAtEnd(block llvm.BasicBlock) {
	b.llvm.SetInsertPointAtEnd(block)
	// LLVM will clear the debug location on a insert point change.
	// Restore it to what we previously had.
	b.dbgRestoreLocation()
}
