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

package compiler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/testutils"
	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

// Aliases
type cmd = testutils.Cmd

var (
	D = testutils.Encode
	P = testutils.Pad
	R = testutils.R
	W = testutils.W
)

func TestExecutor(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	// 32-bit address since we want this to be able to
	// represent addresses in the ARM abi
	ptrA := uint64(0x0000000004030000)

	c := &capture.GraphicsCapture{
		Observed: interval.U64RangeList{
			{First: ptrA, Count: 0x10000},
		},
	}
	u32Data := [1024]uint32{}
	for i := range u32Data {
		u32Data[i] = uint32(i)
	}
	res0x1234, _ := database.Store(ctx, D(uint16(0x1234)))
	res0x6789, _ := database.Store(ctx, D(uint16(0x6789)))
	resU32Data, _ := database.Store(ctx, D(u32Data))
	resHelloWorld, _ := database.Store(ctx, []byte("Hello World\x00"))

	resPointerTo500, _ := database.Store(ctx, D(
		uint32(uint32(ptrA)+500),
	))

	resPointee, _ := database.Store(ctx, D(
		uint64(0xdeadbeefdeadbeef),
		uint16(0xffee),
	))

	resArmPodStruct, _ := database.Store(ctx, D(
		uint32(0x00010203),
		uint32(0xdeadbeef),
		uint16(0x0a0b),
		// 6 bytes of padding
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		uint64(0xbadf00dbadf00d00),
		uint32(0x31323334),
	))

	resArmStructInStruct, _ := database.Store(ctx, D(
		uint32(0xaabbccdd),
		uint16(0xfefe),
		// 2 bytes padding
		[]byte{0x00, 0x00},
		uint64(0xdadadadadabcdabc),
		uint16(0xaabb),
		// 6 bytes padding
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		uint16(0x4253),
	))

	for _, test := range []test{
		////////////////////////////////////////////////////////
		// Types                                              //
		////////////////////////////////////////////////////////
		{
			name: "Types.Primitives",
			src: `
u64  a =  0x4000000000000004
u32  b =  0x30000003
u16  c =  0x2002
u8   d =  0x11
s64  e = -0x4000000000000004
s32  f = -0x30000003
s16  g = -0x2002
s8   h = -0x11
f64  i = 1
f32  j = 1
bool k = true
`,
			expected: expected{
				data: D(
					uint64(0x4000000000000004),
					uint32(0x30000003),
					uint16(0x2002),
					uint8(0x11), P(1),
					int64(-0x4000000000000004),
					int32(-0x30000003),
					int16(-0x2002),
					int8(-0x11), P(1),
					float64(1),
					float32(1),
					true, P(3),
				),
			},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.Bool",
			src:      `bool a = as!bool(as!any(true))`,
			expected: expected{data: D(true)},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.U8",
			src:      `u8 a = as!u8(as!any(as!u8(10)))`,
			expected: expected{data: D(uint8(10))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.S8",
			src:      `s8 a = as!s8(as!any(as!s8(-11)))`,
			expected: expected{data: D(int8(-11))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.U16",
			src:      `u16 a = as!u16(as!any(as!u16(1000)))`,
			expected: expected{data: D(uint16(1000))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.S16",
			src:      `s16 a = as!s16(as!any(as!s16(-1001)))`,
			expected: expected{data: D(int16(-1001))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.F32",
			src:      `f32 a = as!f32(as!any(as!f32(65536)))`,
			expected: expected{data: D(float32(65536))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.U32",
			src:      `u32 a = as!u32(as!any(as!u32(10001)))`,
			expected: expected{data: D(uint32(10001))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.S32",
			src:      `s32 a = as!s32(as!any(as!s32(-10002)))`,
			expected: expected{data: D(int32(-10002))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.F64",
			src:      `f64 a = as!f64(as!any(as!f64(65536)))`,
			expected: expected{data: D(float64(65536))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.U64",
			src:      `u64 a = as!u64(as!any(as!u64(100001)))`,
			expected: expected{data: D(uint64(100001))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Any.PackAndUnpack.S64",
			src:      `s64 a = as!s64(as!any(as!s64(-100002)))`,
			expected: expected{data: D(int64(-100002))},
		}, { ///////////////////////////////////////////////////
			name: "Types.Arrays",
			src: `
type u32[4] u32x4
u32x4 i = u32x4(1, 2, 3, 4)
`,
			expected: expected{
				data: D(uint32(1), uint32(2), uint32(3), uint32(4)),
			},
		}, { ///////////////////////////////////////////////////
			name: "Types.Bitfields",
			src: `
bitfield B {
	X = 1,
	Y = 2,
	Z = 4
}
B i = Y
`,
			expected: expected{data: D(uint32(2))},
		}, { ///////////////////////////////////////////////////
			name: "Types.Enums",
			src: `
enum E {
	X = 1,
	Y = 2,
	Z = 3
}
E i = Y
`,
			expected: expected{data: D(uint32(2))},
		}, { ///////////////////////////////////////////////////
			name:     "Types.String.Empty",
			src:      `string s`,
			expected: expected{},
		}, { ///////////////////////////////////////////////////
			name:     "Types.Map.Empty",
			src:      `map!(u32, u32) m`,
			expected: expected{numAllocs: 1}, // Maps are automatically allocated.
		},
		////////////////////////////////////////////////////////
		// Expressions                                        //
		////////////////////////////////////////////////////////
		{
			name: "Expressions.ArrayIndex",
			src: `
type u32[4] u32x4
u32 i = u32x4(1, 2, 3, 4)[3]
`,
			expected: expected{data: D(uint32(4))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.Add",
			src: `
u32 i = 8
f32 f = 8
cmd void Add() {
	i = i + 1
	f = f + 0.5
}`,
			cmds:     []cmd{{N: "Add"}},
			expected: expected{data: D(uint32(9), float32(8.5))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.Subtract",
			src: `
u32 i = 8
f32 f = 8
cmd void Subtract() {
	i = i - 1
	f = f - 0.5
}`,
			cmds:     []cmd{{N: "Subtract"}},
			expected: expected{data: D(uint32(7), float32(7.5))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.Multiply",
			src: `
u32 i = 8
f32 f = 8
cmd void Multiply() {
	i = i * 2
	f = f * 1.5
}`,
			cmds:     []cmd{{N: "Multiply"}},
			expected: expected{data: D(uint32(16), float32(12))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.Divide",
			src: `
u32 i = 8
f32 f = 8
cmd void Divide() {
	i = i / 2
	f = f / 0.5
}`,
			cmds:     []cmd{{N: "Divide"}},
			expected: expected{data: D(uint32(4), float32(16))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.Equal",
			src: `
bool[6] r
cmd void Equal(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0] = x == 1
	r[1] = x == 2
	r[2] = y == 3
	r[3] = y == 4
	r[4] = s == "Hello World"
	r[5] = s == "Goodbye World"
}`,
			cmds: []cmd{{
				N: "Equal",
				D: D(uint32(2), float32(3), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{false, true, true, false, true, false})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.NotEqual",
			src: `
bool[6] r
cmd void NotEqual(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0] = x != 1
	r[1] = x != 2
	r[2] = y != 3
	r[3] = y != 4
	r[4] = s != "Hello World"
	r[5] = s != "Goodbye World"
}`,
			cmds: []cmd{{
				N: "NotEqual",
				D: D(uint32(2), float32(3), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{true, false, false, true, false, true})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LessThan",
			src: `
bool[11] r
cmd void LessThan(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0]  = x < 1
	r[1]  = x < 2
	r[2]  = y < 3
	r[3]  = y < 4
	r[4]  = s < "Hello Ant"
	r[5]  = s < "Hello Antelope"
	r[6]  = s < "Hello"
	r[7]  = s < "Hello World"
	r[8]  = s < "Hello World!"
	r[9]  = s < "Hello Yak"
	r[10] = s < "Hello Zorilla"
}`,
			cmds: []cmd{{
				N: "LessThan",
				D: D(uint32(1), float32(3), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{
				false /* x < 1 */, true, /* x < 2 */
				false /* y < 3 */, true, /* y < 4 */
				false, // s < "Hello Ant"
				false, // s < "Hello Antelope"
				false, // s < "Hello"
				false, // s < "Hello World"
				true,  // s < "Hello World!"
				true,  // s < "Hello Yak"
				true,  // s < "Hello Zorilla"
			})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LessEqual",
			src: `
bool[11] r
cmd void LessEqual(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0]  = x <= 1
	r[1]  = x <= 2
	r[2]  = y <= 3
	r[3]  = y <= 4
	r[4]  = s <= "Hello Ant"
	r[5]  = s <= "Hello Antelope"
	r[6]  = s <= "Hello"
	r[7]  = s <= "Hello World"
	r[8]  = s <= "Hello World!"
	r[9]  = s <= "Hello Yak"
	r[10] = s <= "Hello Zorilla"
}`,
			cmds: []cmd{{
				N: "LessEqual",
				D: D(uint32(2), float32(4), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{
				false /* x <= 1 */, true, /* x <= 2 */
				false /* y <= 3 */, true, /* y <= 4 */
				false, // s <= "Hello Ant"
				false, // s <= "Hello Antelope"
				false, // s <= "Hello"
				true,  // s <= "Hello World"
				true,  // s <= "Hello World!"
				true,  // s <= "Hello Yak"
				true,  // s <= "Hello Zorilla"
			})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.GreaterThan",
			src: `
bool[11] r
cmd void GreaterThan(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0]  = x > 1
	r[1]  = x > 2
	r[2]  = y > 3
	r[3]  = y > 4
	r[4]  = s > "Hello Ant"
	r[5]  = s > "Hello Antelope"
	r[6]  = s > "Hello"
	r[7]  = s > "Hello World"
	r[8]  = s > "Hello World!"
	r[9]  = s > "Hello Yak"
	r[10] = s > "Hello Zorilla"
}`,
			cmds: []cmd{{
				N: "GreaterThan",
				D: D(uint32(2), float32(4), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{
				true /* x > 1 */, false, /* x > 2 */
				true /* y > 3 */, false, /* y > 4 */
				true,  // s > "Hello Ant"
				true,  // s > "Hello Antelope"
				true,  // s > "Hello"
				false, // s > "Hello World"
				false, // s > "Hello World!"
				false, // s > "Hello Yak"
				false, // s > "Hello Zorilla"
			})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.GreaterEqual",
			src: `
bool[11] r
cmd void GreaterEqual(u32 x, f32 y, char* z) {
	s := as!string(z)
	r[0]  = x >= 1
	r[1]  = x >= 2
	r[2]  = y >= 3
	r[3]  = y >= 4
	r[4]  = s >= "Hello Ant"
	r[5]  = s >= "Hello Antelope"
	r[6]  = s >= "Hello"
	r[7]  = s >= "Hello World"
	r[8]  = s >= "Hello World!"
	r[9]  = s >= "Hello Yak"
	r[10] = s >= "Hello Zorilla"
}`,
			cmds: []cmd{{
				N: "GreaterEqual",
				D: D(uint32(1), float32(3), ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D([]bool{
				true /* x >= 1 */, false, /* x >= 2 */
				true /* y >= 3 */, false, /* y >= 4 */
				true,  // s >= "Hello Ant"
				true,  // s >= "Hello Antelope"
				true,  // s >= "Hello"
				true,  // s >= "Hello World"
				false, // s >= "Hello World!"
				false, // s >= "Hello Yak"
				false, // s >= "Hello Zorilla"
			})},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.ShiftLeft",
			src: `
u32 i = 2
cmd void ShiftLeft() {	i = i << 2 }`,
			cmds:     []cmd{{N: "ShiftLeft"}},
			expected: expected{data: D(uint32(8))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.ShiftRight",
			src: `
u32 i = 8
cmd void ShiftRight() { i = i >> 2 }`,
			cmds:     []cmd{{N: "ShiftRight"}},
			expected: expected{data: D(uint32(2))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.BitwiseOr",
			src: `
u32 i = 8
cmd void BitwiseOr() { i = i | 2 }`,
			cmds:     []cmd{{N: "BitwiseOr"}},
			expected: expected{data: D(uint32(10))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.BitwiseAnd",
			src: `
u32 i = 7
cmd void BitwiseAnd() { i = i & 6 }`,
			cmds:     []cmd{{N: "BitwiseAnd"}},
			expected: expected{data: D(uint32(6))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LogicalOr",
			src: `
bool a  bool b  bool c  bool d
cmd void LogicalOr() {
	a = false || false
	b = true  || false
	c = false || true
	d = true  || true
}`,
			cmds:     []cmd{{N: "LogicalOr"}},
			expected: expected{data: D(false, true, true, true)},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LogicalAnd",
			src: `
bool a  bool b  bool c  bool d
cmd void LogicalAnd() {
	a = false && false
	b = true  && false
	c = false && true
	d = true  && true
}`,
			cmds:     []cmd{{N: "LogicalAnd"}},
			expected: expected{data: D(false, false, false, true)},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LogicalOr.ShortCircuit",
			src: `
bool a  bool b
sub bool True() { return true }
sub bool SideEffect() { b = true  return true }
cmd void LogicalOrShortCircuit() { a = True() || SideEffect() }`,
			cmds:     []cmd{{N: "LogicalOrShortCircuit"}},
			expected: expected{data: D(true, false)},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.LogicalAnd.ShortCircuit",
			src: `
bool a  bool b
sub bool False() { return false }
sub bool SideEffect() { b = true  return true }
cmd void LogicalAndShortCircuit() {	a = False() && SideEffect() }`,
			cmds:     []cmd{{N: "LogicalAndShortCircuit"}},
			expected: expected{data: D(false, false)},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.BinaryOp.StringConcat",
			src: `
u32 i
cmd void StringConcat() { i = len(" 3 " + "four") }`,
			cmds:     []cmd{{N: "StringConcat"}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.BinaryOp.NullStringConcat",
			src: `
u32 i
cmd void NullStringConcat() { i = len(as!string(null) + "four") }`,
			cmds:     []cmd{{N: "NullStringConcat"}},
			expected: expected{data: D(uint32(4))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.BitTest",
			src: `
bitfield B {
	X = 1
	Y = 2
	Z = 4
}
bool hasX
bool hasY
bool hasZ
cmd void BitTest() {
	b := X | Z
	hasX = X in b
	hasY = Y in b
	hasZ = Z in b
}`,
			cmds:     []cmd{{N: "BitTest"}},
			expected: expected{data: D(true, false, true)},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Call.Extern",
			src: `
u64  p
bool q
extern u64  extern_a(u64 a, f32 b, bool c)
extern bool extern_b(string s)
cmd void CallExterns(u64 a, f32 b, bool c) {
	p = extern_a(as!u64(10), 20.0, true)
	q = extern_b("meow")
}`,
			cmds: []cmd{{N: "CallExterns"}},
			expected: expected{
				data: D(30, true, P(7)),
				externCalls: []interface{}{
					externA{10, 20.0, true},
					externB{"meow"},
				},
			},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Call.Subroutine",
			src: `
u32 i = 2
sub u32 doAdd(u32 a, u32 b) { return a + b }
cmd void CallSubroutine(u32 a, u32 b) { i = doAdd(a, b) }`,
			cmds:     []cmd{{N: "CallSubroutine", D: D(uint32(3), uint32(4))}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Cast",
			src: `
u32 i
f32 f
u8* p
cmd void Cast(f32* ptr) {
	v := 1234.5678
	i = as!u32(v)
	f = as!f32(v)
	p = as!u8*(ptr)
}`,
			cmds: []cmd{{N: "Cast", D: D(uint64(0x12345678))}},
			expected: expected{data: D(
				uint32(1234),
				float32(1234.5678),
				uint64(0x12345678),
			)},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Cast.PointerToString",
			src: `
u32 i = 0
cmd void PointerToString(char* str) {
	i = len(as!string(str))
}`,
			cmds: []cmd{{
				N: "PointerToString",
				D: D(ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D(uint32(11))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.ClassInitializer.Class",
			src: `
class C {
	u32 a
	u32 b
	f32 c
}
C c
cmd void ClassInitializerClass() {
	c = C(a: 3, c: 5.0)
}
`,
			cmds:     []cmd{{N: "ClassInitializerClass"}},
			expected: expected{data: D(uint32(3), uint32(0), float32(5.0))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.ClassInitializer.Reference",
			src: `
class C {
	u32 a
	u32 b
	f32 c
}
C c
cmd void ClassInitializerReference() {
	r := new!C(a: 3, c: 5.0)
	c.a = r.a
	c.b = r.b
	c.c = r.c
}
`,
			cmds:     []cmd{{N: "ClassInitializerReference"}},
			expected: expected{data: D(uint32(3), uint32(0), float32(5.0))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Clone",
			src: `
u8 i
cmd void clone_slice(u8* ptr) {
	o := ptr[6:11] // 'W' 'o' 'r' 'l' 'd'
	c := clone(o)
	i = c[2] // 'r'
}
`,
			cmds: []cmd{{
				N: "clone_slice",
				D: D(ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{data: D(byte('r'))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Create",
			src: `
class C {
	u32 i
	f32 f
}
u32 i
f32 f
cmd void create_class() {
	r := new!C(2, 3)
	i = r.i
	f = r.f
}`,
			cmds:     []cmd{{N: "create_class"}},
			expected: expected{data: D(uint32(2), float32(3))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.EnumEntry",
			src: `
enum E {
	X = 1
	Y = 2
	Z = 4
}
E e
cmd void enum_entry() {
	e = X | Y | Z
}
`,
			cmds:     []cmd{{N: "enum_entry"}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Float32Value",
			src:      `f32 f = 3.141592653589793`,
			expected: expected{data: D(uint32(0x40490fdb))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Float64Value",
			src:      `f64 f = 3.141592653589793`,
			expected: expected{data: D(uint64(0x400921fb54442d18))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Global",
			src: `
u32 a = 10
u32 b
cmd void global() { b = a }
`,
			cmds:     []cmd{{N: "global"}},
			expected: expected{data: D(uint32(10), uint32(10))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Ignore",
			src:  `cmd void ignore() { _ = 20 }`,
			cmds: []cmd{{N: "ignore"}},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Int64Value",
			src:      `s64 i = -9223372036854775808`,
			expected: expected{data: D(int64(-9223372036854775808))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Int32Value",
			src:      `s32 i = -2147483648`,
			expected: expected{data: D(int32(-2147483648))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Int16Value",
			src:      `s16 i = -32768`,
			expected: expected{data: D(int16(-32768))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Int8Value",
			src:      `s8 i = -128`,
			expected: expected{data: D(int8(-128))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Length.Map",
			src: `
u32 i = 0
class C { map!(u32, u32) m }
cmd void map_length() {
	c := C()
	c.m[1] = 10
	c.m[2] = 20
	c.m[1] = 30
	i = len(c.m)
}`,
			cmds:     []cmd{{N: "map_length"}},
			expected: expected{data: D(uint32(2))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Length.String",
			src:      `u32 i = len("123456789")`,
			expected: expected{data: D(uint32(9))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Local",
			src: `
s32 i
cmd void local() {
	l := 123
	i = l
}`,
			cmds:     []cmd{{N: "local"}},
			expected: expected{data: D(int32(123))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Make",
			src: `
u32 v0
u32 v1
u32 v2
cmd void create_slice() {
	a := make!u32(3)
	a[1] = 10
	b := make!u32(3)
	b[2] = 20
	c := make!u32(3)
	c[0] = 30
	v0 = a[0] + b[0] + c[0]
	v1 = a[1] + b[1] + c[1]
	v2 = a[2] + b[2] + c[2]
}`,
			cmds:     []cmd{{N: "create_slice"}},
			expected: expected{data: D(uint32(30), uint32(10), uint32(20))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.MapContains",
			src: `
bool[3] v
class C { map!(u32, u32) m }
cmd void map_contains() {
	c := C()
	c.m[1] = 1
	c.m[3] = 3
	v[0] = 1 in c.m
	v[1] = 2 in c.m
	v[2] = 3 in c.m
}`,
			cmds:     []cmd{{N: "map_contains"}},
			expected: expected{data: D(true, false, true)},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.MapIndex.U32",
			src: `
u32[3] v
class C { map!(u32, u32) m }
cmd void map_index() {
	c := C()
	c.m[1] = 1
	c.m[3] = 3
	v[0] = c.m[1]
	v[1] = c.m[2]
	v[2] = c.m[3]
}`,
			cmds:     []cmd{{N: "map_index"}},
			expected: expected{data: D([]uint32{1, 0, 3})},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.MapIndex.Class",
			src: `
u32[3] v
class S { u32 i }
class C { map!(u32, S) m }
cmd void map_index() {
	c := C()
	c.m[1] = S(1)
	c.m[3] = S(3)
	v[0] = c.m[1].i
	v[1] = c.m[2].i
	v[2] = c.m[3].i
}`,
			cmds:     []cmd{{N: "map_index"}},
			expected: expected{data: D([]uint32{1, 0, 3})},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Member",
			src: `
class Z {
	u64 a
	u8  b
}
class Y {
	u32 a
	Z   b
	u16 c
}
class X {
	f32 a
	Y   b
	u16 c
}
u64 i
cmd void member() {
	x := X()
	x.a = 1
	x.b.a = 2
	x.b.b.a = 3
	x.b.b.b = 4
	x.b.c = 5
	x.c = 6
	i = x.b.b.a + as!u64(x.b.c)
}
`,
			cmds:     []cmd{{N: "member"}},
			expected: expected{data: D(uint64(8))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Null",
			src: `
class C { u8 i }
u8*    p
ref!C  r
u32    i
f32    f
cmd void null_vals(u8* ptr) {
	p = ptr
	r = new!C()
	i = 10
	f = 11

	p = null
	r = null
	i = null
	f = null
}`,
			cmds:     []cmd{{N: "null_vals", D: D(ptrA)}},
			expected: expected{data: D(uint64(0), uintptr(0), uint32(0), float32(0))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Observed/Unknown",
			src: `
u32 i
cmd u32 observed() {
	x := ?
	i = x
	return x
}`,
			cmds:     []cmd{{N: "observed", D: D(uint32(32))}},
			expected: expected{data: D(uint32(32))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Parameter",
			src: `
u32 i = 2
cmd void add(u32 a, u32 b) { i = a + b }`,
			cmds:     []cmd{{N: "add", D: D(uint32(3), uint32(4))}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Parameter",
			src: `
u32 i = 2
cmd void add(u32 a, u32 b) { i = a + b }`,
			cmds:     []cmd{{N: "add", D: D(uint32(3), uint32(4))}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.PointerSlice",
			src: `
u32 a
u32 b
u32 c
cmd void pointer_slice(u32* ptr) {
	a = ptr[0]
	b = ptr[200:700][200]
	c = ptr[400:402][2]
}`,
			cmds: []cmd{{
				N: "pointer_slice",
				D: D(ptrA),
				E: R(ptrA, uint64(len(u32Data))*4, resU32Data),
			}},
			expected: expected{data: D(uint32(0), uint32(400), uint32(402))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Select.NoDefault",
			src: `
u32 i
cmd void select_expr(u32 v) {
	i += switch(v) {
		case 3: 2
		case 2: 1
		case 1: 3
	}
}`,
			cmds:     []cmd{{N: "select_expr", D: D(uint32(2))}},
			expected: expected{data: D(uint32(1))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Select.WithDefault",
			src: `
u32 i
cmd void select_expr(u32 v) {
	i += switch(v) {
		case 3: 2
		case 2: 1
		case 1: 3
		default: 4
	}
}`,
			cmds: []cmd{
				{N: "select_expr", D: D(uint32(2))},
				{N: "select_expr", D: D(uint32(7))},
			},
			expected: expected{data: D(uint32(5))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.SliceIndex",
			src: `
u32 a
u32 b
u32 c
cmd void slice_index(u32* ptr) {
	slice := ptr[100:200]
	a = slice[5]
	b = slice[10]
	c = slice[99]
}`,
			cmds: []cmd{{
				N: "slice_index",
				D: D(ptrA),
				E: R(ptrA, uint64(len(u32Data))*4, resU32Data),
			}},
			expected: expected{data: D(uint32(105), uint32(110), uint32(199))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.SliceRange",
			src: `
u32 a
u32 b
u32 c
cmd void slice_range(u32* ptr) {
	sliceA := ptr[0:1024]
	sliceB := sliceA[100:900]
	sliceC := sliceB[50:53]
	a = sliceC[0]
	b = sliceC[1]
	c = sliceC[2]
}`,
			cmds: []cmd{{
				N: "slice_range",
				D: D(ptrA),
				E: R(ptrA, uint64(len(u32Data))*4, resU32Data),
			}},
			expected: expected{data: D(uint32(150), uint32(151), uint32(152))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.StringValue",
			src:      `u32 i = len("abc")`,
			expected: expected{data: D(uint32(3))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Thread.Command",
			src: `
u64 a
cmd void thread_cmd() { a = $Thread }`,
			cmds: []cmd{{
				N: "thread_cmd",
				T: 123,
			}},
			expected: expected{data: D(uint64(123))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Thread.Subroutine",
			src: `
u64 a
sub void S() { a = $Thread }
cmd void thread_sub() { S() }`,
			cmds: []cmd{{
				N: "thread_sub",
				T: 123,
			}},
			expected: expected{data: D(uint64(123))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Uint64Value",
			src:      `u64 i = 18446744073709551615`,
			expected: expected{data: D(uint64(18446744073709551615))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Uint32Value",
			src:      `u32 i = 4294967295`,
			expected: expected{data: D(uint32(4294967295))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Uint16Value",
			src:      `u16 i = 65535`,
			expected: expected{data: D(uint16(65535))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.Uint8Value",
			src:      `u8 i = 255`,
			expected: expected{data: D(uint8(255))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.UnaryOp",
			src: `
bool b
cmd void not(bool v) { b = !v }
`,
			cmds:     []cmd{{N: "not", D: D(false)}},
			expected: expected{data: D(true)},
		},
		////////////////////////////////////////////////////////
		// Statements                                         //
		////////////////////////////////////////////////////////
		{
			name: "Statements.Abort.InCmd",
			src: `
u32 i = 0
cmd void AbortInCmd() {
	i = 5
	abort
	i = 9
}`,
			cmds: []cmd{{N: "AbortInCmd"}},
			expected: expected{
				data: D(uint32(5)),
				err:  api.ErrCmdAborted{},
			},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Abort.InSub",
			src: `
u32 i = 0
sub void call_abort() { abort }
cmd void AbortInSub() {
	i = 5
	call_abort()
	i = 9
}`,
			cmds: []cmd{{N: "AbortInSub"}},
			expected: expected{
				data: D(uint32(5)),
				err:  api.ErrCmdAborted{},
			},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Abort.InCmd.Cleanup",
			src: `
cmd void AbortInCmdCleanup() {
	s := "this string must be released before throwing the exception"
	abort
}`,
			cmds:     []cmd{{N: "AbortInCmdCleanup"}},
			expected: expected{err: api.ErrCmdAborted{}},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Abort.InSub.Cleanup",
			src: `
sub void call_abort() { abort }
cmd void AbortInSubCleanup() {
	s := "this string must be released in the exception cleanup"
	call_abort()
}`,
			cmds:     []cmd{{N: "AbortInSubCleanup"}},
			expected: expected{err: api.ErrCmdAborted{}},
		}, { /////////////////////////////////////////////////////
			name: "Statements.ArrayAssign",
			src: `
u32[5] i
cmd void ArrayAssign() {
	i[2] = 4
	i[4] = 7
}`,
			cmds:     []cmd{{N: "ArrayAssign"}},
			expected: expected{data: D([]uint32{0, 0, 4, 0, 7})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Assign.Global",
			src: `
class S {
	f32 a
	f32 b
	f32 c
}
u32 a = 10
u32 b = 10
u32 c = 10
S s = S(10, 20, 30)
cmd void AssignGlobal() {
	a  = 5
	b += 5
	c -= 5
	s.a  = 123
	s.b += 10
	s.c -= 10
}`,
			cmds:     []cmd{{N: "AssignGlobal"}},
			expected: expected{data: D([]uint32{5, 15, 5}, []float32{123, 30, 20})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Branch",
			src: `
u32 a
u32 b
cmd void Branch(bool c) {
	if (c) {
		a = 1
	} else {
		a = 2
	}
	if (!c) {
		b = 3
	} else {
		b = 4
	}
}`,
			cmds:     []cmd{{N: "Branch", D: D(true)}},
			expected: expected{data: D([]uint32{1, 4})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Call.Sub",
			src: `
u32 i
sub T add!T(T a, T b) {
	return a + b
}
cmd void CallSub(u32 a, u32 b) {
	i = add!u32(a, b)
}`,
			cmds:     []cmd{{N: "CallSub", D: D([]uint32{7, 9})}},
			expected: expected{data: D(uint32(16))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Copy",
			src: `
u32 i
cmd void Copy() {
	a := make!u32(5)
	b := make!u32(5)
	a[1] = 1
	copy(b, a)
	a[1] = 2
	i = b[1]
}`,
			cmds:     []cmd{{N: "Copy"}},
			expected: expected{data: D(uint32(1))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.DeclareLocal",
			src: `
s32 i
cmd void DeclareLocal() {
	a := 1
	b := a
	c := b
	i = a
}`,
			cmds:     []cmd{{N: "DeclareLocal"}},
			expected: expected{data: D(int32(1))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Fence",
			src: `
u16 a
u16 b
cmd void Fence(u16* ptr) {
	a = ptr[0]
	fence
	b = ptr[0]
}`,
			cmds: []cmd{{
				N: "Fence",
				D: D(ptrA),
				E: R(ptrA, 2, res0x1234).W(ptrA, 2, res0x6789),
			}},
			expected: expected{data: D([]uint16{0x1234, 0x6789})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Iteration.Range",
			src: `
s32[5] v
cmd void IterationRange() {
	for i in (0 .. 5) {
		v[i] = i
	}
}`,
			cmds:     []cmd{{N: "IterationRange"}},
			expected: expected{data: D([]int32{0, 1, 2, 3, 4})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Iteration.Map",
			src: `
u32[5] k_
u32[5] v_
u32[5] i_
class C{ map!(u32, u32) m }
cmd void IterationMap() {
	m := C().m
	m[0] = 40
	m[1] = 70
	m[2] = 90
	for i, k, v in m {
		k_[k] = k
		v_[k] = v
		i_[i] = as!u32(i)
	}
}`,
			cmds:     []cmd{{N: "IterationMap"}},
			expected: expected{data: D([]uint32{0, 1, 2, 0, 0, 40, 70, 90, 0, 0, 0, 1, 2, 0, 0})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Iteration.MapAssign",
			src: `
u32[5] v
class C{ map!(u32, u32) m }
cmd void MapAssign() {
	m := C().m
	m[1] = 71
	m[3] = 23
	m[32] = 12
	v[0] = m[1]
	v[1] = m[84]
	v[2] = m[32]
	v[3] = m[42]
	v[4] = m[3]
}`,
			cmds:     []cmd{{N: "MapAssign"}},
			expected: expected{data: D([]uint32{71, 0, 12, 0, 23})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Iteration.MapRemove",
			src: `
u32[10] v
u32 count
class C{ map!(u32, u32) m }
cmd void MapRemove() {
	m := C().m
	m[3] = 42
	m[12] = 114
	m[32] = 52
	m[19] = 33
	m[40] = 43
	m[86] = 90
	m[11] = 12
	m[89] = 92
	m[23] = 13
	m[14] = 31
	delete(m, 12)
	v[0] = m[3]
	v[1] = m[4]
	v[2] = m[12]
	v[3] = m[21]
	v[4] = m[32]
	v[5] = m[86]
	v[6] = m[11]
	v[7] = m[89]
	v[8] = m[23]
	v[9] = m[14]
	count = len(m)
}`,
			cmds:     []cmd{{N: "MapRemove"}},
			expected: expected{data: D([]uint32{42, 0, 0, 0, 52, 90, 12, 92, 13, 31, 9})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Iteration.MapRehash",
			src: `
u32 count
u32[17] _v
class C{ map!(u32, u32) m }
cmd void MapRehash() {
	m := C().m
	m[0] = 0
	m[1] = 1
	m[2] = 2
	m[3] = 3
	m[4] = 4
	m[5] = 5
	m[6] = 6
	m[7] = 7
	m[8] = 8
	m[9] = 9
	// We should hash here, add a few more things
	m[10] = 10
	m[11] = 11
	m[12] = 12
	m[13] = 13
	m[14] = 14
	m[15] = 15
	// If we haven't rehashed, we will write over random memory here
	m[16] = 16
	count = len(m)
	for _, k, v in m {
		_v[k] = v
	}
}`,
			cmds:     []cmd{{N: "MapRehash"}},
			expected: expected{data: D([]uint32{17, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.MapIteration",
			src: `
class S { u32 i }
class M { map!(u32, S) m }
u32[3] r
cmd void MapIteration() {
	m := M().m
	m[1] = S(5)
	m[2] = S(7)
	m[3] = S(9)
	for i, k, v in m {
		r[0] = r[0] + as!u32(i)
		r[1] = r[1] + k
		r[2] = r[2] + v.i
	}
}`,
			cmds:     []cmd{{N: "MapIteration"}},
			expected: expected{data: D([]uint32{3, 6, 21})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Read",
			src: `
cmd void Read(u32* ptr) {
	read(ptr[0:5])
}`, // TODO: test read callbacks
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
			}},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Return",
			src: `
u32 i
sub u32 double(u32 v) {
	return v * 2
}
cmd void Return() {
	i = double(10)
}`,
			cmds:     []cmd{{N: "Return"}},
			expected: expected{data: D(int32(20))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.SliceAssign.AppPool.WritesEnabled",
			src: `
u16 i = 0
cmd void SliceAssignAppPoolWritesEnabled(u16* ptr) {
	ptr[0] = 10
	i = ptr[0]
}`,
			cmds: []cmd{{
				N: "SliceAssignAppPoolWritesEnabled",
				D: D(ptrA),
				E: R(ptrA, 4, res0x1234),
			}},
			expected: expected{data: D(uint16(10))},
			settings: compiler.Settings{WriteToApplicationPool: true},
		}, { /////////////////////////////////////////////////////
			name: "Statements.SliceAssign.AppPool.WritesDisabled",
			src: `
u16 i = 0
cmd void SliceAssignAppPoolWritesDisabled(u16* ptr) {
	ptr[0] = 10
	i = ptr[0]
}`,
			cmds: []cmd{{
				N: "SliceAssignAppPoolWritesDisabled",
				D: D(ptrA),
				E: R(ptrA, 2, res0x1234),
			}},
			expected: expected{data: D(uint16(0x1234))},
			settings: compiler.Settings{WriteToApplicationPool: false},
		}, { /////////////////////////////////////////////////////
			name: "Statements.SliceAssign.NewPool.WritesEnabled",
			src: `
u16 i = 0
cmd void SliceAssignNewPoolWritesEnabled() {
	s := make!u16(4)
	s[0] = 10
	i = s[0]
}`,
			cmds:     []cmd{{N: "SliceAssignNewPoolWritesEnabled"}},
			expected: expected{data: D(uint16(10))},
			settings: compiler.Settings{WriteToApplicationPool: true},
		}, { /////////////////////////////////////////////////////
			name: "Statements.SliceAssign.NewPool.WritesDisabled",
			src: `
u16 i = 0
cmd void SliceAssignNewPoolWritesDisabled() {
	s := make!u16(4)
	s[0] = 10
	i = s[0]
}`,
			cmds:     []cmd{{N: "SliceAssignNewPoolWritesDisabled"}},
			expected: expected{data: D(uint16(10))},
			settings: compiler.Settings{WriteToApplicationPool: false},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Switch.NoDefault.Match",
			src: `
u32 i
cmd void SwitchNoDefaultMatch(u32 v) {
	switch(v) {
	case 7: i = 70
	case 1:	i = 10
	case 3:	i = 30
	}
}`,
			cmds:     []cmd{{N: "SwitchNoDefaultMatch", D: D(uint32(1))}},
			expected: expected{data: D(int32(10))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Switch.NoDefault.NoMatch",
			src: `
u32 i
cmd void SwitchNoDefaultNoMatch(u32 v) {
	switch(v) {
	case 7: i = 70
	case 1:	i = 10
	case 3:	i = 30
	}
}`,
			cmds:     []cmd{{N: "SwitchNoDefaultNoMatch", D: D(uint32(6))}},
			expected: expected{data: D(int32(0))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Switch.WithDefault.Match",
			src: `
u32 i
cmd void SwitchWithDefaultMatch(u32 v) {
	switch(v) {
	case 7:  i = 70
	case 1:  i = 10
	case 3:  i = 30
	default: i = 50
	}
}`,
			cmds:     []cmd{{N: "SwitchWithDefaultMatch", D: D(uint32(1))}},
			expected: expected{data: D(int32(10))},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Switch.WithDefault.NoMatch",
			src: `
u32 i
cmd void SwitchWithDefaultNoMatch(u32 v) {
	switch(v) {
	case 7:  i = 70
	case 1:  i = 10
	case 3:  i = 30
	default: i = 50
	}
}`,
			cmds:     []cmd{{N: "SwitchWithDefaultNoMatch", D: D(uint32(8))}},
			expected: expected{data: D(int32(50))},
		},
		////////////////////////////////////////////////////////
		// Misc                                               //
		////////////////////////////////////////////////////////
		{
			name: "Misc.EmptyString",
			src: `
u32 i
class S { string str }
cmd void EmptyString() {
	s := S()
	i = len(s.str)
}`,
			cmds:     []cmd{{N: "EmptyString"}},
			expected: expected{data: D(uint32(0))},
		}, { /////////////////////////////////////////////////////
			name: "Misc.MapsOfStructsOfMaps",
			src: `
u32 i

class SA { u32 i }
type map!(u32, SA) MA
class SB { MA m }
type map!(u32, SB) MB
class SC { MB m }

cmd void MapsOfStructsOfMaps() {
	c := SC()

	tmp := c.m[10]
	tmp.m[20] = SA(42)
	c.m[10] = tmp

	i = c.m[10].m[20].i
}
`,
			cmds:     []cmd{{N: "MapsOfStructsOfMaps"}},
			expected: expected{data: D(uint32(42))},
		},
		////////////////////////////////////////////////////////
		// Reference Counting                                 //
		////////////////////////////////////////////////////////
		{
			name: "RefCount.String.InState",
			src: `
string s
cmd void StringInState(char* str) {
	s = as!string(str)
	s = "another string"
	s = "and another"
}`,
			cmds: []cmd{{
				N: "StringInState",
				D: D(ptrA),
				E: R(ptrA, 12, resHelloWorld),
			}},
			expected: expected{numAllocs: 1},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringConcat",
			src: `
string s
cmd void StringConcat() {
	s += "a string"
	s += " with some"
	s += " text appended"
}
`,
			cmds:     []cmd{{N: "StringConcat"}},
			expected: expected{numAllocs: 1},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringFromSubroutine",
			src: `
sub string ReturnAString() { return "A string" }
cmd void StringFromSubroutine() { x := ReturnAString() }
`,
			cmds: []cmd{{N: "StringFromSubroutine"}},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringInAny",
			src: `
any a
cmd void StringInAny() {
	a = "purr"
	a = "meow"
	a = "hiss"
}
`,
			cmds:     []cmd{{N: "StringInAny"}},
			expected: expected{numAllocs: 2},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringInClass",
			src: `
class C { string s }
C c
cmd void StringInClass() {
	c = C("purr")
	c = C("meow")
	c = C("hiss")
}
`,
			cmds:     []cmd{{N: "StringInClass"}},
			expected: expected{numAllocs: 1},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringInClass",
			src: `
class C { string s }
C c
cmd void Assign() { c = C("purr") }
cmd void Clear() { c = null }
`,
			cmds:     []cmd{{N: "Assign"}, {N: "Clear"}},
			expected: expected{numAllocs: 0},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringInRef",
			src: `
class C { string s }
ref!C c
cmd void StringInRef() { c = new!C("purr") }
`,
			cmds:     []cmd{{N: "StringInRef"}},
			expected: expected{numAllocs: 2},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringInRef",
			src: `
class C { string s }
ref!C c
cmd void Assign() { c = new!C("purr") }
cmd void Clear() { c = null }
`,
			cmds:     []cmd{{N: "Assign"}, {N: "Clear"}},
			expected: expected{numAllocs: 0},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringKeyInMap",
			src: `
map!(string, u32) m
cmd void Assign() {
	m["one"] = 1
	m["two"] = 2
	m["one"] = 1
	m["two"] = 2
}
cmd void Clear() {
	delete(m, "one")
	delete(m, "two")
}
`,
			cmds:     []cmd{{N: "Assign"}, {N: "Clear"}},
			expected: expected{numAllocs: 2}, // map + map's elements
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringValueInMap",
			src: `
map!(u32, string) m
cmd void Assign() {
	m[1] = "one"
	m[2] = "two"
	m[1] = "one"
	m[2] = "two"
}
cmd void Clear() {
	delete(m, 1)
	delete(m, 2)
}
`,
			cmds:     []cmd{{N: "Assign"}, {N: "Clear"}},
			expected: expected{numAllocs: 2}, // map + map's elements
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ClearMapOnFree",
			src: `
class C { map!(string, string) m }
cmd void ClearMapOnFree() {
	c := C()
	c.m["one"] = "I"
	c.m["two"] = "II"
}
`,
			cmds:     []cmd{{N: "ClearMapOnFree"}},
			expected: expected{numAllocs: 0},
		}, { /////////////////////////////////////////////////////
			// Stress-test reference count releasing on variables declared
			// within nested scopes. Incorrectly handling scopes may try to
			// release a LLVM variable that was declared in upstream block,
			// causing a "Instruction does not dominate all uses!" error.
			name: "RefCount.ReleaseInNestedScopes",
			src: `
sub string DeepNestedLogic(u32 i) {
	if i == 0 {
		s := "cat"
		switch(s) {
			case "purr": {
				x := switch(i) {
					case 4:  "fluffy"
					default: "hiss"
				}
				y := x + "kitty"
			}
			case "dog":
				abort
			default:
				_ := "nap"
		}
	} else {
		s := "meow"
	}
	return "the-end"
}
cmd void ReleaseInNestedScopes(u32 i) {
	x := DeepNestedLogic(i)
}`,
			cmds: []cmd{{
				N: "ReleaseInNestedScopes",
				D: D(uint32(4)),
			}},
		},
		////////////////////////////////////////////////////////
		// Memory Layout                                      //
		////////////////////////////////////////////////////////
		{
			name: "MemoryLayout.Struct",
			src: `
class C {
	u64 a
	u32 b
	u16 c
	u8  d
}
C s = C(a: 1, b: 2, c: 3, d: 4)
`,
			expected: expected{
				data: D(
					uint64(1),
					uint32(2),
					uint16(3),
					uint8(4), P(1),
				),
			},
		},
		////////////////////////////////////////////////////////
		// Capture Memory Layout                              //
		////////////////////////////////////////////////////////
		{
			name: "CaptureMemoryLayout.Struct",
			src: `
class PodStruct {
	u32 a
	void* b
	u16 c
	u64 d
	size  e
}
PodStruct s
cmd void Read(PodStruct* input) {
	s = input[0]
}
`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
				E: R(ptrA, 28, resArmPodStruct),
			}},
			expected: expected{
				data: D(
					uint32(0x00010203),
					P(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					P(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
				),
			},
			settings: compiler.Settings{
				CaptureABI: device.AndroidARMv7a,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.Slice.Struct",
			src: `
class PodStruct {
	u32 a
	void* b
	u16 c
	u64 d
	size  e
}
PodStruct s
PodStruct s2
cmd void Read(PodStruct* input) {
	myS := input[0:1]
	s = myS[0]
	s2 = myS[1]
}
`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
				E: R(ptrA, 28, resArmPodStruct).R(ptrA+32, 28, resArmPodStruct),
			}},
			expected: expected{
				data: D(
					uint32(0x00010203),
					P(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					P(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
					uint32(0x00010203),
					P(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					P(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
				),
			},
			settings: compiler.Settings{
				CaptureABI: device.AndroidARMv7a,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.StructWithStruct",
			src: `
class SizeStruct {
	u64 a
	u16 b
}
class StructInStruct {
	size a
	u16 b
	SizeStruct c
	u16 d
}
StructInStruct s
cmd void Read(StructInStruct* input) {
	s = input[0]
}`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
				E: R(ptrA, 26, resArmStructInStruct),
			}},
			expected: expected{
				data: D(
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					P(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					P(6),
					uint16(0x4253),
					P(6),
				),
			},
			settings: compiler.Settings{
				CaptureABI: device.AndroidARMv7a,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.Slice.StructWithStruct",
			src: `
class SizeStruct {
	u64 a
	u16 b
}
class StructInStruct {
	size a
	u16 b
	SizeStruct c
	u16 d
}
StructInStruct s
StructInStruct s2
cmd void Read(StructInStruct* input) {
	mys := input[0:1]
	s = mys[0]
	s2 = mys[1]
}`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
				E: R(ptrA, 26, resArmStructInStruct).R(ptrA+32, 26, resArmStructInStruct),
			}},
			expected: expected{
				data: D(
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					P(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					P(6),
					uint16(0x4253),
					P(6),
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					P(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					P(6),
					uint16(0x4253),
					P(6),
				),
			},
			settings: compiler.Settings{
				CaptureABI: device.AndroidARMv7a,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.Write.Struct",
			src: `
class PodStruct {
	u32 a
	void* b
	u16 c
	u64 d
	size e
	int[4] f
}

cmd void Write(PodStruct* input, void* ptr) {
	p := PodStruct(0x00010203, ptr, 0x0a0b, 0xbadf00dbadf00d00, as!size(0x31323334))
	p.f[0] = as!int(1)
	p.f[1] = as!int(2)
	p.f[2] = as!int(3)
	p.f[3] = as!int(4)
	input[0] = p
}
`,
			cmds: []cmd{{
				N: "Write",
				D: D(ptrA, uint64(0xdeadbeef)),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x00010203), uint32(0xdeadbeef), uint16(0x0a0b)),
					ptrA + 16: D(uint64(0xbadf00dbadf00d00), uint32(0x31323334)),
					ptrA + 28: D(uint32(1), uint32(2), uint32(3), uint32(4)),
				},
			},
			settings: compiler.Settings{
				CaptureABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.WriteSlice.Struct",
			src: `
class PodStruct {
	u32 a
	void* b
	u16 c
	u64 d
	size  e
}

cmd void Write(PodStruct* input, void* ptr) {
	i := input[0:1]
	i[0] = PodStruct(0x00010203, ptr, 0x0a0b, 0xbadf00dbadf00d00, as!size(0x31323334))
	i[1] = PodStruct(0x00010203, ptr, 0x0a0b, 0xbadf00dbadf00d00, as!size(0x31323334))
}
`,
			cmds: []cmd{{
				N: "Write",
				D: D(ptrA, uint64(0xdeadbeef)),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x00010203), uint32(0xdeadbeef), uint16(0x0a0b)),
					ptrA + 16: D(uint64(0xbadf00dbadf00d00), uint32(0x31323334)),
					ptrA + 32: D(uint32(0x00010203), uint32(0xdeadbeef), uint16(0x0a0b)),
					ptrA + 48: D(uint64(0xbadf00dbadf00d00), uint32(0x31323334)),
				},
			},
			settings: compiler.Settings{
				CaptureABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.Write.StructWithStruct",
			src: `
class SizeStruct {
	u64 a
	u16 b
}
class StructInStruct {
	size a
	u16 b
	SizeStruct c
	u16 d
}
cmd void Read(StructInStruct* input) {
	s := StructInStruct(as!size(0x3abbccdd), 0xfefe, SizeStruct(0xdadadadadabcdabc, 0xaabb), 0x4253)
	input[0] = s
}`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x3abbccdd), uint16(0xfefe)),
					ptrA + 8:  D(uint64(0xdadadadadabcdabc), uint16(0xaabb)),
					ptrA + 24: D(uint16(0x4253)),
				},
			},
			settings: compiler.Settings{
				CaptureABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.Write.Slice.StructWithStruct",
			src: `
class SizeStruct {
	u64 a
	u16 b
}
class StructInStruct {
	size a
	u16 b
	SizeStruct c
	u16 d
}
cmd void Read(StructInStruct* input) {
	i := input[0:1]
	i[0] = StructInStruct(as!size(0x3abbccdd), 0xfefe, SizeStruct(0xdadadadadabcdabc, 0xaabb), 0x4253)
	i[1] = StructInStruct(as!size(0x3abbccdd), 0xfefe, SizeStruct(0xdadadadadabcdabc, 0xaabb), 0x4253)
}`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x3abbccdd), uint16(0xfefe)),
					ptrA + 8:  D(uint64(0xdadadadadabcdabc), uint16(0xaabb)),
					ptrA + 24: D(uint16(0x4253)),
					ptrA + 32: D(uint32(0x3abbccdd), uint16(0xfefe)),
					ptrA + 40: D(uint64(0xdadadadadabcdabc), uint16(0xaabb)),
					ptrA + 56: D(uint16(0x4253)),
				},
			},
			settings: compiler.Settings{
				CaptureABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		}, { /////////////////////////////////////////////////////
			name: "CaptureMemoryLayout.StructWithPointer",
			src: `
class SizeStruct {
	u64 a
	u16 b
}
class StructInStruct {
	SizeStruct* s
}
SizeStruct s
cmd void Read(StructInStruct* input) {
	myS := input[0]
	s = myS.s[0]
}`,
			cmds: []cmd{{
				N: "Read",
				D: D(ptrA),
				E: R(ptrA, 4, resPointerTo500).
					R(ptrA+500, 10, resPointee),
			}},
			expected: expected{
				data: D(
					uint64(0xdeadbeefdeadbeef),
					uint16(0xffee),
					P(6),
				),
			},
			settings: compiler.Settings{
				CaptureABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.run(log.SubTest(ctx, t), c)
		})
	}
}

type externA struct {
	I uint64
	F float32
	B bool
}

type externB struct {
	S string
}

type buffers map[uint64][]byte

type expected struct {
	data        []byte
	err         error
	numAllocs   int
	buffers     buffers
	externCalls []interface{}
}

type test struct {
	name     string
	dump     bool // used for debugging
	src      string
	cmds     []cmd
	expected expected
	settings compiler.Settings
}

func (t test) run(ctx context.Context, c capture.Capture) (succeeded bool) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic in test '%v':\n%v", t.name, r))
		}
	}()

	fmt.Printf("--- %s ---\n", t.name)

	processor := gapil.NewProcessor()
	processor.Loader = gapil.NewDataLoader([]byte(t.src))
	a, errs := processor.Resolve("test.api")
	if !assert.For(ctx, "Resolve").ThatSlice(errs).Equals(parse.ErrorList{}) {
		return false
	}

	t.settings.EmitExec = true

	program, err := compiler.Compile([]*semantic.API{a}, processor.Mappings, t.settings)
	if !assert.For(ctx, "Compile").ThatError(err).Succeeded() {
		return false
	}

	exec := executor.New(program, false)
	env := exec.NewEnv(ctx, c.(*capture.GraphicsCapture))
	defer env.Dispose()

	if t.dump {
		fmt.Println(program.Dump())
	}

	defer func() {
		if !succeeded {
			fmt.Println(program.Dump())
		}
	}()

	externCalls := []interface{}{}

	for i, cmd := range t.cmds {
		fmt.Printf("    > %s\n", cmd.N)
		testutils.ExternA = func(env *executor.Env, i uint64, f float32, b bool) uint64 {
			externCalls = append(externCalls, externA{i, f, b})
			return i + uint64(f)
		}
		testutils.ExternB = func(env *executor.Env, s string) bool {
			externCalls = append(externCalls, externB{s})
			return s == "meow"
		}
		err = env.Execute(ctx, &cmd, api.CmdID(0x1000+i))
		if !assert.For(ctx, "Execute(%v, %v)", i, cmd.N).ThatError(err).DeepEquals(t.expected.err) {
			return false
		}
	}

	if t.expected.data != nil {
		if !assert.For(ctx, "Globals").ThatSlice(env.Globals()).Equals(t.expected.data) {
			return false
		}
	}

	if t.expected.externCalls != nil {
		if !assert.For(ctx, "ExternCalls").ThatSlice(externCalls).Equals(t.expected.externCalls) {
			return false
		}
	}

	for k, v := range t.expected.buffers {
		rng := memory.Range{k, uint64(len(v))}
		storedBytes := env.GetBytes(rng)

		if !assert.For(ctx, "Buffers").ThatSlice(storedBytes).Equals(v) {
			return false
		}
	}

	stats := env.Arena.Stats()
	numContextAllocs := 3                                          // gapil_create_context: context, next_pool_id, globals
	numMemBlocks := c.(*capture.GraphicsCapture).Observed.Length() // observation allocations
	numOtherAllocs := numMemBlocks + numContextAllocs
	if !assert.For(ctx, "Allocations").That(stats.NumAllocations - numOtherAllocs).Equals(t.expected.numAllocs) {
		log.I(ctx, "Allocations: %v\n", stats)
		return false
	}
	return true
}
