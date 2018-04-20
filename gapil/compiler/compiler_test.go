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
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/testexterns"
	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

func TestExecutor(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	// 32-bit address since we want this to be able to
	// represent addresses in the ARM abi
	ptrA := uint64(0x0000000004030000)

	c := &capture.Capture{
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
					uint8(0x11), pad(1),
					int64(-0x4000000000000004),
					int32(-0x30000003),
					int16(-0x2002),
					int8(-0x11), pad(1),
					float64(1),
					float32(1),
					true, pad(3),
				),
			},
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
			cmds:     []cmd{{name: "Add"}},
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
			cmds:     []cmd{{name: "Subtract"}},
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
			cmds:     []cmd{{name: "Multiply"}},
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
			cmds:     []cmd{{name: "Divide"}},
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
				name:   "Equal",
				data:   D(uint32(2), float32(3), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
				name:   "NotEqual",
				data:   D(uint32(2), float32(3), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
				name:   "LessThan",
				data:   D(uint32(1), float32(3), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
				name:   "LessEqual",
				data:   D(uint32(2), float32(4), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
				name:   "GreaterThan",
				data:   D(uint32(2), float32(4), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
				name:   "GreaterEqual",
				data:   D(uint32(1), float32(3), ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
			cmds:     []cmd{{name: "ShiftLeft"}},
			expected: expected{data: D(uint32(8))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.ShiftRight",
			src: `
u32 i = 8
cmd void ShiftRight() { i = i >> 2 }`,
			cmds:     []cmd{{name: "ShiftRight"}},
			expected: expected{data: D(uint32(2))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.BitwiseOr",
			src: `
u32 i = 8
cmd void BitwiseOr() { i = i | 2 }`,
			cmds:     []cmd{{name: "BitwiseOr"}},
			expected: expected{data: D(uint32(10))},
		}, { ///////////////////////////////////////////////////
			name: "Expressions.BinaryOp.BitwiseAnd",
			src: `
u32 i = 7
cmd void BitwiseAnd() { i = i & 6 }`,
			cmds:     []cmd{{name: "BitwiseAnd"}},
			expected: expected{data: D(uint32(6))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.BinaryOp.StringConcat",
			src: `
u32 i
cmd void StringConcat() { i = len(" 3 " + "four") }`,
			cmds:     []cmd{{name: "StringConcat"}},
			expected: expected{data: D(uint32(7))},
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
			cmds:     []cmd{{name: "BitTest"}},
			expected: expected{data: D(true, false, true)},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Call.Extern",
			src: `
u64  p
bool q
extern u64  test_extern_a(u64 a, f32 b, bool c)
extern bool test_extern_b(string s)
cmd void CallExterns(u64 a, f32 b, bool c) {
	p = test_extern_a(as!u64(10), 20.0, true)
	q = test_extern_b("meow")
}`,
			cmds: []cmd{{name: "CallExterns"}},
			expected: expected{
				data: D(30, true, pad(7)),
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
			cmds:     []cmd{{name: "CallSubroutine", data: D(uint32(3), uint32(4))}},
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
			cmds: []cmd{{name: "Cast", data: D(uint64(0x12345678))}},
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
				name:   "PointerToString",
				data:   D(ptrA),
				extras: read(ptrA, 12, resHelloWorld),
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
			cmds:     []cmd{{name: "ClassInitializerClass"}},
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
			cmds:     []cmd{{name: "ClassInitializerReference"}},
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
				name:   "clone_slice",
				data:   D(ptrA),
				extras: read(ptrA, 20, resHelloWorld),
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
			cmds:     []cmd{{name: "create_class"}},
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
			cmds:     []cmd{{name: "enum_entry"}},
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
			cmds:     []cmd{{name: "global"}},
			expected: expected{data: D(uint32(10), uint32(10))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Ignore",
			src:  `cmd void ignore() { _ = 20 }`,
			cmds: []cmd{{name: "ignore"}},
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
			cmds:     []cmd{{name: "map_length"}},
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
			cmds:     []cmd{{name: "local"}},
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
			cmds:     []cmd{{name: "create_slice"}},
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
			cmds:     []cmd{{name: "map_contains"}},
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
			cmds:     []cmd{{name: "map_index"}},
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
			cmds:     []cmd{{name: "map_index"}},
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
			cmds:     []cmd{{name: "member"}},
			expected: expected{data: D(uint64(8))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Null",
			src: `
class C { u8 i }
u8*    p
string s
ref!C  r
u32    i
f32    f
cmd void null_vals(u8* ptr) {
	p = ptr
	s = "meow"
	r = new!C()
	i = 10
	f = 11

	p = null
	s = null
	r = null
	i = null
	f = null
}`,
			cmds:     []cmd{{name: "null_vals", data: D(ptrA)}},
			expected: expected{data: D(uintptr(0), uintptr(0), uintptr(0), uint32(0), float32(0))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Observed/Unknown",
			src: `
u32 i
cmd u32 observed() {
	x := ?
	i = x
	return x
}`,
			cmds:     []cmd{{name: "observed", data: D(uint32(32))}},
			expected: expected{data: D(uint32(32))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Parameter",
			src: `
u32 i = 2
cmd void add(u32 a, u32 b) { i = a + b }`,
			cmds:     []cmd{{name: "add", data: D(uint32(3), uint32(4))}},
			expected: expected{data: D(uint32(7))},
		}, { /////////////////////////////////////////////////////
			name: "Expressions.Parameter",
			src: `
u32 i = 2
cmd void add(u32 a, u32 b) { i = a + b }`,
			cmds:     []cmd{{name: "add", data: D(uint32(3), uint32(4))}},
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
				name:   "pointer_slice",
				data:   D(ptrA),
				extras: read(ptrA, uint64(len(u32Data))*4, resU32Data),
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
			cmds:     []cmd{{name: "select_expr", data: D(uint32(2))}},
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
				{name: "select_expr", data: D(uint32(2))},
				{name: "select_expr", data: D(uint32(7))},
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
				name:   "slice_index",
				data:   D(ptrA),
				extras: read(ptrA, uint64(len(u32Data))*4, resU32Data),
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
				name:   "slice_range",
				data:   D(ptrA),
				extras: read(ptrA, uint64(len(u32Data))*4, resU32Data),
			}},
			expected: expected{data: D(uint32(150), uint32(151), uint32(152))},
		}, { /////////////////////////////////////////////////////
			name:     "Expressions.StringValue",
			src:      `u32 i = len("abc")`,
			expected: expected{data: D(uint32(3))},
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
			cmds:     []cmd{{name: "not", data: D(false)}},
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
			cmds: []cmd{{name: "AbortInCmd"}},
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
			cmds: []cmd{{name: "AbortInSub"}},
			expected: expected{
				data: D(uint32(5)),
				err:  api.ErrCmdAborted{},
			},
		}, { /////////////////////////////////////////////////////
			name: "Statements.ArrayAssign",
			src: `
u32[5] i
cmd void ArrayAssign() {
	i[2] = 4
	i[4] = 7
}`,
			cmds:     []cmd{{name: "ArrayAssign"}},
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
			cmds:     []cmd{{name: "AssignGlobal"}},
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
			cmds:     []cmd{{name: "Branch", data: D(true)}},
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
			cmds:     []cmd{{name: "CallSub", data: D([]uint32{7, 9})}},
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
			cmds:     []cmd{{name: "Copy"}},
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
			cmds:     []cmd{{name: "DeclareLocal"}},
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
				name:   "Fence",
				data:   D(ptrA),
				extras: read(ptrA, 2, res0x1234).write(ptrA, 2, res0x6789),
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
			cmds:     []cmd{{name: "IterationRange"}},
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
			cmds:     []cmd{{name: "IterationMap"}},
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
			cmds:     []cmd{{name: "MapAssign"}},
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
			cmds:     []cmd{{name: "MapRemove"}},
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
			cmds:     []cmd{{name: "MapRehash"}},
			expected: expected{data: D([]uint32{17, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})},
		}, { /////////////////////////////////////////////////////
			name: "Statements.Read",
			src: `
cmd void Read(u32* ptr) {
	read(ptr[0:5])
}`, // TODO: test read callbacks
			cmds: []cmd{{name: "Read"}},
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
			cmds:     []cmd{{name: "Return"}},
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
				name:   "SliceAssignAppPoolWritesEnabled",
				data:   D(ptrA),
				extras: read(ptrA, 4, res0x1234),
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
				name:   "SliceAssignAppPoolWritesDisabled",
				data:   D(ptrA),
				extras: read(ptrA, 4, res0x1234),
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
			cmds:     []cmd{{name: "SliceAssignNewPoolWritesEnabled"}},
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
			cmds:     []cmd{{name: "SliceAssignNewPoolWritesDisabled"}},
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
			cmds:     []cmd{{name: "SwitchNoDefaultMatch", data: D(uint32(1))}},
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
			cmds:     []cmd{{name: "SwitchNoDefaultNoMatch", data: D(uint32(6))}},
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
			cmds:     []cmd{{name: "SwitchWithDefaultMatch", data: D(uint32(1))}},
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
			cmds:     []cmd{{name: "SwitchWithDefaultNoMatch", data: D(uint32(8))}},
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
			cmds:     []cmd{{name: "EmptyString"}},
			expected: expected{data: D(uint32(0))},
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
				name:   "StringInState",
				data:   D(ptrA),
				extras: read(ptrA, 12, resHelloWorld),
			}},
			expected: expected{numAllocs: 1},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringFromSubroutine",
			src: `
sub string ReturnAString() { return "A string" }
cmd void StringFromSubroutine() { x := ReturnAString() }
`,
			cmds: []cmd{{name: "StringFromSubroutine"}},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringInClass",
			src: `
class C { string s }
C c
cmd void StringInClass() { c = C("purr") }
`,
			cmds:     []cmd{{name: "StringInClass"}},
			expected: expected{numAllocs: 1},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringInClass",
			src: `
class C { string s }
C c
cmd void Assign() { c = C("purr") }
cmd void Clear() { c = null }
`,
			cmds:     []cmd{{name: "Assign"}, {name: "Clear"}},
			expected: expected{numAllocs: 0},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.StringInRef",
			src: `
class C { string s }
ref!C c
cmd void StringInRef() { c = new!C("purr") }
`,
			cmds:     []cmd{{name: "StringInRef"}},
			expected: expected{numAllocs: 2},
		}, { /////////////////////////////////////////////////////
			name: "RefCount.ReleaseStringInRef",
			src: `
class C { string s }
ref!C c
cmd void Assign() { c = new!C("purr") }
cmd void Clear() { c = null }
`,
			cmds:     []cmd{{name: "Assign"}, {name: "Clear"}},
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
			cmds:     []cmd{{name: "Assign"}, {name: "Clear"}},
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
			cmds:     []cmd{{name: "Assign"}, {name: "Clear"}},
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
			cmds:     []cmd{{name: "ClearMapOnFree"}},
			expected: expected{numAllocs: 0},
		}, { /////////////////////////////////////////////////////
			// Stress-test reference count releasing on variables declared
			// within nested scopes. Incorrectly handling scopes may try to
			// release a LLVM variable that was declared in upstream block,
			// causing a "Instruction does not dominate all uses!" error.
			name: "RefCount.ReleaseInNestedScopes",
			src: `
sub string CrazyNestedLogic(u32 i) {
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
	x := CrazyNestedLogic(i)
}`,
			cmds: []cmd{{
				name: "ReleaseInNestedScopes",
				data: D(uint32(4)),
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
					uint8(4), pad(1),
				),
			},
		},
		////////////////////////////////////////////////////////
		// Storage Memory Layout                              //
		////////////////////////////////////////////////////////
		{
			name: "StorageMemoryLayout.Struct",
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
				name:   "Read",
				data:   D(ptrA),
				extras: read(ptrA, 28, resArmPodStruct),
			}},
			expected: expected{
				data: D(
					uint32(0x00010203),
					pad(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					pad(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
				),
			},
			settings: compiler.Settings{
				StorageABI: device.AndroidARMv7a,
			},
		},
		{
			name: "StorageMemoryLayout.Slice.Struct",
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
				name:   "Read",
				data:   D(ptrA),
				extras: read(ptrA, 28, resArmPodStruct).read(ptrA+32, 28, resArmPodStruct),
			}},
			expected: expected{
				data: D(
					uint32(0x00010203),
					pad(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					pad(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
					uint32(0x00010203),
					pad(4),
					uint64(0x00000000deadbeef),
					uint16(0x0a0b),
					pad(6),
					uint64(0xbadf00dbadf00d00),
					uint64(0x0000000031323334),
				),
			},
			settings: compiler.Settings{
				StorageABI: device.AndroidARMv7a,
			},
		},
		{
			name: "StorageMemoryLayout.StructWithStruct",
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
				name:   "Read",
				data:   D(ptrA),
				extras: read(ptrA, 26, resArmStructInStruct),
			}},
			expected: expected{
				data: D(
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					pad(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					pad(6),
					uint16(0x4253),
					pad(6),
				),
			},
			settings: compiler.Settings{
				StorageABI: device.AndroidARMv7a,
			},
		},
		{
			name: "StorageMemoryLayout.Slice.StructWithStruct",
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
				name:   "Read",
				data:   D(ptrA),
				extras: read(ptrA, 26, resArmStructInStruct).read(ptrA+32, 26, resArmStructInStruct),
			}},
			expected: expected{
				data: D(
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					pad(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					pad(6),
					uint16(0x4253),
					pad(6),
					uint64(0x00000000aabbccdd),
					uint16(0xfefe),
					pad(6),
					uint64(0xdadadadadabcdabc),
					uint16(0xaabb),
					pad(6),
					uint16(0x4253),
					pad(6),
				),
			},
			settings: compiler.Settings{
				StorageABI: device.AndroidARMv7a,
			},
		},
		{
			name: "StorageMemoryLayout.Write.Struct",
			src: `
class PodStruct {
	u32 a
	void* b
	u16 c
	u64 d
	size  e
}

cmd void Write(PodStruct* input, void* ptr) {
	p := PodStruct(0x00010203, ptr, 0x0a0b, 0xbadf00dbadf00d00, as!size(0x31323334))
	input[0] = p
}
`,
			cmds: []cmd{{
				name: "Write",
				data: D(ptrA, uint64(0xdeadbeef)),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x00010203), uint32(0xdeadbeef), uint16(0x0a0b)),
					ptrA + 16: D(uint64(0xbadf00dbadf00d00), uint32(0x31323334)),
				},
			},
			settings: compiler.Settings{
				StorageABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
		{
			name: "StorageMemoryLayout.WriteSlice.Struct",
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
				name: "Write",
				data: D(ptrA, uint64(0xdeadbeef)),
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
				StorageABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
		{
			name: "StorageMemoryLayout.Write.StructWithStruct",
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
				name: "Read",
				data: D(ptrA),
			}},
			expected: expected{
				buffers: buffers{
					ptrA:      D(uint32(0x3abbccdd), uint16(0xfefe)),
					ptrA + 8:  D(uint64(0xdadadadadabcdabc), uint16(0xaabb)),
					ptrA + 24: D(uint16(0x4253)),
				},
			},
			settings: compiler.Settings{
				StorageABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
		{
			name: "StorageMemoryLayout.Write.Slice.StructWithStruct",
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
				name: "Read",
				data: D(ptrA),
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
				StorageABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
		{
			name: "StorageMemoryLayout.StructWithPointer",
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
				name:   "Read",
				data:   D(ptrA),
				extras: read(ptrA, 32, resPointerTo500).read(ptrA+500, 10, resPointee),
			}},
			expected: expected{
				data: D(
					uint64(0xdeadbeefdeadbeef),
					uint16(0xffee),
					pad(6),
				),
			},
			settings: compiler.Settings{
				StorageABI:             device.AndroidARMv7a,
				WriteToApplicationPool: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := log.PutHandler(ctx, log.TestHandler(t, log.Normal))
			test.run(ctx, c)
		})
	}
}

type cmd struct {
	name   string
	data   []byte
	extras *extras
	thread uint64
}

var _ api.Cmd = &cmd{}

func (c *cmd) API() api.API                                                       { return nil }
func (c *cmd) Caller() api.CmdID                                                  { return 0 }
func (c *cmd) SetCaller(api.CmdID)                                                {}
func (c *cmd) Thread() uint64                                                     { return c.thread }
func (c *cmd) SetThread(thread uint64)                                            { c.thread = thread }
func (c *cmd) CmdName() string                                                    { return c.name }
func (c *cmd) CmdParams() api.Properties                                          { return nil }
func (c *cmd) CmdResult() *api.Property                                           { return nil }
func (c *cmd) CmdFlags(context.Context, api.CmdID, *api.GlobalState) api.CmdFlags { return 0 }
func (c *cmd) Extras() *api.CmdExtras {
	if c.extras == nil {
		return &api.CmdExtras{}
	}
	return c.extras.e
}
func (c *cmd) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder) error { return nil }

type extras struct {
	e *api.CmdExtras
}

func (e *extras) read(base uint64, size uint64, id id.ID) *extras {
	if e.e == nil {
		e.e = &api.CmdExtras{}
	}
	e.e.GetOrAppendObservations().AddRead(memory.Range{Base: base, Size: size}, id)
	return e
}

func (e *extras) write(base uint64, size uint64, id id.ID) *extras {
	if e.e == nil {
		e.e = &api.CmdExtras{}
	}
	e.e.GetOrAppendObservations().AddWrite(memory.Range{Base: base, Size: size}, id)
	return e
}

func read(base uint64, size uint64, id id.ID) *extras {
	e := &extras{}
	return e.read(base, size, id)
}

func write(base uint64, size uint64, id id.ID) *extras {
	e := &extras{}
	return e.write(base, size, id)
}

func (c cmd) Encode(out []byte) bool {
	w := endian.Writer(bytes.NewBuffer(out), device.LittleEndian)
	w.Uint64(c.thread)
	copy(out[8:], c.data)
	return true
}

func D(vals ...interface{}) []byte {
	buf := &bytes.Buffer{}
	w := endian.Writer(buf, device.LittleEndian)
	for _, d := range vals {
		binary.Write(w, d)
	}
	return buf.Bytes()
}

func pad(bytes int) []byte { return make([]byte, bytes) }

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

func (t test) run(ctx context.Context, c *capture.Capture) (succeeded bool) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic in test '%v':\n%v", t.name, r))
		}
	}()

	fmt.Printf("--- %s ---\n", t.name)

	processor := gapil.NewProcessor()
	processor.Loader = gapil.NewDataLoader([]byte(t.src))
	api, errs := processor.Resolve(t.name + ".api")
	if !assert.For(ctx, "Resolve").ThatSlice(errs).Equals(parse.ErrorList{}) {
		return false
	}

	t.settings.EmitExec = true

	program, err := compiler.Compile(api, processor.Mappings, t.settings)
	if !assert.For(ctx, "Compile").ThatError(err).Succeeded() {
		return false
	}

	exec := executor.New(program, false)
	env := exec.NewEnv(ctx, c)
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
		fmt.Printf("    > %s\n", cmd.name)
		testexterns.ExternA = func(env *executor.Env, i uint64, f float32, b bool) uint64 {
			externCalls = append(externCalls, externA{i, f, b})
			return i + uint64(f)
		}
		testexterns.ExternB = func(env *executor.Env, s string) bool {
			externCalls = append(externCalls, externB{s})
			return s == "meow"
		}
		err = env.Execute(ctx, &cmd)
		if !assert.For(ctx, "Execute(%v, %v)", i, cmd.name).ThatError(err).Equals(t.expected.err) {
			return false
		}
	}

	if t.expected.data != nil {
		if !assert.For(ctx, "Globals").ThatSlice(env.Globals).Equals(t.expected.data) {
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
	numContextAllocs := 1               // next_pool_id allocated in init_context
	numMemBlocks := c.Observed.Length() // observation allocations
	numOtherAllocs := numMemBlocks + numContextAllocs
	if !assert.For(ctx, "Allocations").That(stats.NumAllocations - numOtherAllocs).Equals(t.expected.numAllocs) {
		return false
	}
	return true
}
