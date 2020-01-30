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

package asm

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/replay/opcode"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

type testPtrResolver struct{}

func (testPtrResolver) ResolveTemporaryPointer(value.TemporaryPointer) value.VolatilePointer {
	return 0
}
func (testPtrResolver) ResolveObservedPointer(ptr value.ObservedPointer) (protocol.Type, uint64) {
	return protocol.Type_VolatilePointer, uint64(ptr)
}
func (testPtrResolver) ResolvePointerIndex(value.PointerIndex) (protocol.Type, uint64) {
	return protocol.Type_VolatilePointer, 0
}

func test(ctx context.Context, Instructions []Instruction, expected ...interface{}) {
	buf := &bytes.Buffer{}
	b := endian.Writer(buf, device.LittleEndian)
	for _, instruction := range Instructions {
		err := instruction.Encode(testPtrResolver{}, b)
		ctx := log.V{
			"instruction": instruction,
			"type":        fmt.Sprintf("%T", instruction),
		}.Bind(ctx)
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
	got, err := opcode.Disassemble(buf, device.LittleEndian)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "got").ThatSlice(got).Equals(expected)
}

func TestCall(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Call{false, 0, 0x1234},
			Call{true, 1, 0x5678},
			Call{false, 1, 0x1234},
			Call{true, 0, 0x5678},
		},
		opcode.Call{PushReturn: false, ApiIndex: 0, FunctionID: 0x1234},
		opcode.Call{PushReturn: true, ApiIndex: 1, FunctionID: 0x5678},
		opcode.Call{PushReturn: false, ApiIndex: 1, FunctionID: 0x1234},
		opcode.Call{PushReturn: true, ApiIndex: 0, FunctionID: 0x5678},
	)
}

func TestPush_UnsignedNoExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.U32(0xaaaaa)}, // Repeating pattern of 1010
			Push{value.U32(0x55555)}, // Repeating pattern of 0101
		},
		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0xaaaaa},
		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x55555},
	)
}

func TestPush_UnsignedOneExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.U32(0x100000)},   // One bit beyond what can fit in a PushI
			Push{value.U32(0x4000000)},  // One bit beyond what can fit in a Extend payload
			Push{value.U32(0xaaaaaaaa)}, // 1010101010...
			Push{value.U32(0x55555555)}, // 0101010101...
		},
		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0},
		opcode.Extend{Value: 0x100000},

		opcode.PushI{DataType: protocol.Type_Uint32, Value: 1},
		opcode.Extend{Value: 0},

		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x2a},
		opcode.Extend{Value: 0x2aaaaaa},

		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x15},
		opcode.Extend{Value: 0x1555555},
	)
}

func TestPush_SignedPositiveNoExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.S32(0x2aaaa)}, // 0010101010...
			Push{value.S32(0x55555)}, // 1010101010...
		},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0x2aaaa},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0x55555},
	)
}

func TestPush_SignedPositiveOneExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.S32(0x80000)},    // One bit beyond what can fit in a PushI
			Push{value.S32(0x4000000)},  // One bit beyond what can fit in a Extend payload
			Push{value.S32(0x2aaaaaaa)}, // 0010101010...
			Push{value.S32(0x55555555)}, // 0101010101...
		},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0},
		opcode.Extend{Value: 0x80000},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 1},
		opcode.Extend{Value: 0},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 0x0a},
		opcode.Extend{Value: 0x2aaaaaa},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 0x15},
		opcode.Extend{Value: 0x1555555},
	)
}

func TestPush_SignedNegativeNoExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.S32(-1)},
			Push{value.S32(-0x55556)}, // Repeating pattern of 1010
			Push{value.S32(-0x2aaab)}, // Repeating pattern of 0101
		},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffff},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xaaaaa},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xd5555},
	)
}

func TestPush_SignedNegativeOneExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.S32(-0x100001)},  // One bit beyond what can fit in a PushI
			Push{value.S32(-0x4000001)}, // One bit beyond what can fit in a Extend payload

			Push{value.S32(-0x2aaaaaab)}, // 110101010...
			Push{value.S32(-0x55555556)}, // 101010101...
		},
		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffff},
		opcode.Extend{Value: 0x03efffff},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xffffe},
		opcode.Extend{Value: 0x03ffffff},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xffff5},
		opcode.Extend{Value: 0x1555555},

		opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffea},
		opcode.Extend{Value: 0x2aaaaaa},
	)
}

func TestPush_FloatNoExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.F32(-2.0)},
			Push{value.F32(-1.0)},
			Push{value.F32(-0.5)},
			Push{value.F32(0)},
			Push{value.F32(0.5)},
			Push{value.F32(1.0)},
			Push{value.F32(2.0)},
		},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x180},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x17f},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x17e},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x000},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x07e},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x07f},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x080},
	)
}

func TestPush_FloatOneExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.F32(-3)},
			Push{value.F32(-1.75)},
			Push{value.F32(1.75)},
			Push{value.F32(3)},
		},
		opcode.PushI{DataType: protocol.Type_Float, Value: 0x180},
		opcode.Extend{Value: 0x400000},

		opcode.PushI{DataType: protocol.Type_Float, Value: 0x17F},
		opcode.Extend{Value: 0x600000},

		opcode.PushI{DataType: protocol.Type_Float, Value: 0x07F},
		opcode.Extend{Value: 0x600000},

		opcode.PushI{DataType: protocol.Type_Float, Value: 0x080},
		opcode.Extend{Value: 0x400000},
	)
}

func TestPush_DoubleNoExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.F64(-2.0)},
			Push{value.F64(-1.0)},
			Push{value.F64(-0.5)},
			Push{value.F64(0)},
			Push{value.F64(0.5)},
			Push{value.F64(1.0)},
			Push{value.F64(2.0)},
		},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0xc00},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0xbff},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0xbfe},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0x000},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0x3fe},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0x3ff},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0x400},
	)
}

func TestPush_DoubleExpand(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Push{value.F64(-3)},
			Push{value.F64(-1.75)},
			Push{value.F64(1) / 3},
			Push{value.F64(1.75)},
			Push{value.F64(3)},
		},
		opcode.PushI{DataType: protocol.Type_Double, Value: 0xc00},
		opcode.Extend{Value: 0x2000000},
		opcode.Extend{Value: 0x0},

		opcode.PushI{DataType: protocol.Type_Double, Value: 0xbff},
		opcode.Extend{Value: 0x3000000},
		opcode.Extend{Value: 0x0},

		opcode.PushI{DataType: protocol.Type_Double, Value: 0x3fd},
		opcode.Extend{Value: 0x1555555},
		opcode.Extend{Value: 0x1555555},

		opcode.PushI{DataType: protocol.Type_Double, Value: 0x3ff},
		opcode.Extend{Value: 0x3000000},
		opcode.Extend{Value: 0x0},

		opcode.PushI{DataType: protocol.Type_Double, Value: 0x400},
		opcode.Extend{Value: 0x2000000},
		opcode.Extend{Value: 0x0},
	)
}
func TestPop(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Pop{10},
		},
		opcode.Pop{Count: 10},
	)
}

func TestCopy(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Copy{100},
		},
		opcode.Copy{Count: 100},
	)
}

func TestClone(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Clone{100},
		},
		opcode.Clone{Index: 100},
	)
}

func TestLoad(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Load{protocol.Type_Uint16, value.ConstantPointer(0x10)},
			Load{protocol.Type_Uint16, value.ConstantPointer(0x123456)},

			Load{protocol.Type_Uint16, value.VolatilePointer(0x10)},
			Load{protocol.Type_Uint16, value.VolatilePointer(0x123456)},
		},
		opcode.LoadC{DataType: protocol.Type_Uint16, Address: 0x10},

		opcode.PushI{DataType: protocol.Type_ConstantPointer, Value: 0},
		opcode.Extend{Value: 0x123456},
		opcode.Load{DataType: protocol.Type_Uint16},

		opcode.LoadV{DataType: protocol.Type_Uint16, Address: 0x10},

		opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
		opcode.Extend{Value: 0x123456},
		opcode.Load{DataType: protocol.Type_Uint16},
	)
}

func TestStore(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Store{value.VolatilePointer(0x10)},
			Store{value.VolatilePointer(0x4000000)},
		},
		opcode.StoreV{Address: 0x10},

		opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 1},
		opcode.Extend{Value: 0},
		opcode.Store{},
	)
}

func TestStrcpy(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Strcpy{0x3000},
		},
		opcode.Strcpy{MaxSize: 0x3000},
	)
}

func TestResource(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Resource{10, value.ObservedPointer(0x10)},
			Resource{20, value.ObservedPointer(0x4050607)},
		},
		opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
		opcode.Resource{ID: 10},

		opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 1},
		opcode.Extend{Value: 0x50607},
		opcode.Resource{ID: 20},
	)
}

func TestPost(t *testing.T) {
	ctx := log.Testing(t)
	test(ctx,
		[]Instruction{
			Post{value.AbsolutePointer(0x10), 0x50},
			Post{value.VolatilePointer(0x4050607), 0x8090a0b},
		},
		opcode.PushI{DataType: protocol.Type_AbsolutePointer, Value: 0x10},
		opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x50},
		opcode.Post{},

		opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 1},
		opcode.Extend{Value: 0x50607},
		opcode.PushI{DataType: protocol.Type_Uint32, Value: 2},
		opcode.Extend{Value: 0x90a0b},
		opcode.Post{},
	)
}
