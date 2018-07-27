// Copyright (C) 2018 Google Inc.
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

package test

import (
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

func primeBasicTypes(o BasicTypes) {
	o.SetA(10)    // u8
	o.SetB(20)    // s8
	o.SetC(30)    // u16
	o.SetD(40)    // s16
	o.SetE(50)    // f32
	o.SetF(60)    // u32
	o.SetG(70)    // s32
	o.SetH(80)    // f64
	o.SetI(90)    // u64
	o.SetJ(100)   // s64
	o.SetK(true)  // bool
	o.SetL(E_TWO) // E
}

func zeroBasicTypes(o BasicTypes) {
	o.SetA(0)     // u8
	o.SetB(0)     // s8
	o.SetC(0)     // u16
	o.SetD(0)     // s16
	o.SetE(0)     // f32
	o.SetF(0)     // u32
	o.SetG(0)     // s32
	o.SetH(0)     // f64
	o.SetI(0)     // u64
	o.SetJ(0)     // s64
	o.SetK(false) // bool
	o.SetL(0)     // E
}

func checkBasicTypesPrimed(ctx context.Context, o BasicTypes) {
	assert.For(ctx, "a").That(o.A()).Equals(uint8(10))
	assert.For(ctx, "b").That(o.B()).Equals(int8(20))
	assert.For(ctx, "c").That(o.C()).Equals(uint16(30))
	assert.For(ctx, "d").That(o.D()).Equals(int16(40))
	assert.For(ctx, "e").That(o.E()).Equals(float32(50))
	assert.For(ctx, "f").That(o.F()).Equals(uint32(60))
	assert.For(ctx, "g").That(o.G()).Equals(int32(70))
	assert.For(ctx, "h").That(o.H()).Equals(float64(80))
	assert.For(ctx, "i").That(o.I()).Equals(uint64(90))
	assert.For(ctx, "j").That(o.J()).Equals(int64(100))
	assert.For(ctx, "k").That(o.K()).Equals(true)
	assert.For(ctx, "l").That(o.L()).Equals(E_TWO)
}

func checkBasicTypesZero(ctx context.Context, o BasicTypes) {
	assert.For(ctx, "a").That(o.A()).Equals(uint8(0))
	assert.For(ctx, "b").That(o.B()).Equals(int8(0))
	assert.For(ctx, "c").That(o.C()).Equals(uint16(0))
	assert.For(ctx, "d").That(o.D()).Equals(int16(0))
	assert.For(ctx, "e").That(o.E()).Equals(float32(0))
	assert.For(ctx, "f").That(o.F()).Equals(uint32(0))
	assert.For(ctx, "g").That(o.G()).Equals(int32(0))
	assert.For(ctx, "h").That(o.H()).Equals(float64(0))
	assert.For(ctx, "i").That(o.I()).Equals(uint64(0))
	assert.For(ctx, "j").That(o.J()).Equals(int64(0))
	assert.For(ctx, "k").That(o.K()).Equals(false)
	assert.For(ctx, "l").That(o.L()).Equals(E(0))
}

func TestCloneBasicTypes(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := MakeBasicTypesʳ(a1)
	primeBasicTypes(orig.Get())
	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	checkBasicTypesPrimed(ctx, clone.Get())
	zeroBasicTypes(orig.Get())
	checkBasicTypesPrimed(ctx, clone.Get())
}

func TestCloneNested(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := MakeNestedʳ(a1)
	primeBasicTypes(orig.Class())
	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	checkBasicTypesPrimed(ctx, clone.Class())
	zeroBasicTypes(orig.Class())
	checkBasicTypesPrimed(ctx, clone.Class())
}

func TestCloneRefs(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := MakeRefsʳ(a1)
	orig.SetBasicRef(MakeBasicTypesʳ(a1))
	primeBasicTypes(orig.BasicRef().Get())

	orig.SetCyclicRef(MakeRefsʳ(a1))
	orig.CyclicRef().SetBasicRef(MakeBasicTypesʳ(a1))
	primeBasicTypes(orig.CyclicRef().BasicRef().Get())

	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	assert.For(ctx, "clone.basic_ref").That(clone.BasicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref").That(clone.CyclicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref.basic_ref").That(clone.CyclicRef().BasicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref.cyclic_ref").That(clone.CyclicRef().CyclicRef().IsNil()).Equals(true)

	zeroBasicTypes(orig.BasicRef().Get())
	zeroBasicTypes(orig.CyclicRef().BasicRef().Get())
	orig.BasicRef().SetNil()
	orig.CyclicRef().BasicRef().SetNil()

	checkBasicTypesPrimed(ctx, clone.BasicRef().Get())
	checkBasicTypesPrimed(ctx, clone.CyclicRef().BasicRef().Get())
}

func TestCloneCyclicRefs(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := MakeRefsʳ(a1)
	orig.SetBasicRef(MakeBasicTypesʳ(a1))
	orig.SetCyclicRef(MakeRefsʳ(a1))
	orig.CyclicRef().SetCyclicRef(orig)
	primeBasicTypes(orig.BasicRef().Get())

	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	assert.For(ctx, "clone.basic_ref").That(clone.BasicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref").That(clone.CyclicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref.basic_ref").That(clone.CyclicRef().BasicRef().IsNil()).Equals(true)
	assert.For(ctx, "clone.cyclic_ref.cyclic_ref").That(clone.CyclicRef().CyclicRef().IsNil()).Equals(false)
	assert.For(ctx, "clone.cyclic_ref.cyclic_ref.basic_ref").That(clone.CyclicRef().CyclicRef().BasicRef().IsNil()).Equals(false)

	checkBasicTypesPrimed(ctx, clone.BasicRef().Get())
	checkBasicTypesPrimed(ctx, clone.CyclicRef().CyclicRef().BasicRef().Get())

	zeroBasicTypes(clone.BasicRef().Get())

	checkBasicTypesZero(ctx, clone.BasicRef().Get())
	checkBasicTypesZero(ctx, clone.CyclicRef().CyclicRef().BasicRef().Get())
}

func TestCloneMaps(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := NewMapsʳ(a1,
		NewU8ːu8ᵐ(a1).Add(1, 10).Add(2, 20).Add(3, 30),             // A
		NewStringːBasicTypesʳᵐ(a1).Add("foo", MakeBasicTypesʳ(a1)), // B
	)
	primeBasicTypes(orig.B().Get("foo").Get())

	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	orig.A().Clear()
	zeroBasicTypes(orig.B().Get("foo").Get())

	assert.For(ctx, "a").That(clone.A().Len()).Equals(3)
	assert.For(ctx, "a[1]").That(clone.A().Get(1)).Equals(uint8(10))
	assert.For(ctx, "a[2]").That(clone.A().Get(2)).Equals(uint8(20))
	assert.For(ctx, "a[3]").That(clone.A().Get(3)).Equals(uint8(30))
	checkBasicTypesPrimed(ctx2, clone.B().Get("foo").Get())
}

func TestCloneSlices(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	ctx1, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	sliceA := MakeU8ˢ(ctx1, 5)
	sliceB := MakeF32ˢ(ctx1, 3)

	orig := NewSlicesʳ(a1,
		AsU8ˢ(ctx1, sliceA),
		AsS32ˢ(ctx1, sliceA),
		AsF32ˢ(ctx1, sliceB),
		AsIntˢ(ctx1, sliceB),
	)

	assert.For(ctx, "orig pool A").That(orig.S32s().Pool()).Equals(orig.U8s().Pool())
	assert.For(ctx, "orig pool B").That(orig.Ints().Pool()).Equals(orig.F32s().Pool())
	assert.For(ctx, "orig pool A != B").That(orig.S32s().Pool()).NotEquals(orig.Ints().Pool())

	clone := orig.Clone(ctx2)
	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	assert.For(ctx, "U8s Root").That(clone.U8s().Root()).Equals(orig.U8s().Root())
	assert.For(ctx, "S32s Root").That(clone.S32s().Root()).Equals(orig.S32s().Root())
	assert.For(ctx, "F32s Root").That(clone.F32s().Root()).Equals(orig.F32s().Root())
	assert.For(ctx, "Ints Root").That(clone.Ints().Root()).Equals(orig.Ints().Root())

	assert.For(ctx, "U8s Base").That(clone.U8s().Base()).Equals(orig.U8s().Base())
	assert.For(ctx, "S32s Base").That(clone.S32s().Base()).Equals(orig.S32s().Base())
	assert.For(ctx, "F32s Base").That(clone.F32s().Base()).Equals(orig.F32s().Base())
	assert.For(ctx, "Ints Base").That(clone.Ints().Base()).Equals(orig.Ints().Base())

	assert.For(ctx, "U8s Size").That(clone.U8s().Size()).Equals(orig.U8s().Size())
	assert.For(ctx, "S32s Size").That(clone.S32s().Size()).Equals(orig.S32s().Size())
	assert.For(ctx, "F32s Size").That(clone.F32s().Size()).Equals(orig.F32s().Size())
	assert.For(ctx, "Ints Size").That(clone.Ints().Size()).Equals(orig.Ints().Size())

	assert.For(ctx, "U8s Count").That(clone.U8s().Count()).Equals(orig.U8s().Count())
	assert.For(ctx, "S32s Count").That(clone.S32s().Count()).Equals(orig.S32s().Count())
	assert.For(ctx, "F32s Count").That(clone.F32s().Count()).Equals(orig.F32s().Count())
	assert.For(ctx, "Ints Count").That(clone.Ints().Count()).Equals(orig.Ints().Count())

	assert.For(ctx, "cloned pool A").That(clone.S32s().Pool()).Equals(clone.U8s().Pool())
	assert.For(ctx, "cloned pool B").That(clone.Ints().Pool()).Equals(clone.F32s().Pool())
	assert.For(ctx, "cloned pool A != B").That(clone.S32s().Pool()).NotEquals(clone.Ints().Pool())
}

func TestCloneArrays(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	tmp := arena.New()

	orig := NewArraysʳ(a1,
		NewU8ː4ᵃ(tmp, 10, 20, 30, 40),                                                                             // A
		NewBasicTypesː4ᵃ(tmp, MakeBasicTypes(tmp), MakeBasicTypes(tmp), MakeBasicTypes(tmp), MakeBasicTypes(tmp)), // B
	)
	primeBasicTypes(orig.B().Get(2))

	tmp.Dispose()

	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	orig.A().Set(0, 0)
	orig.A().Set(1, 0)
	orig.A().Set(2, 0)
	orig.A().Set(3, 0)
	zeroBasicTypes(orig.B().Get(2))

	assert.For(ctx, "a[0]").That(clone.A().Get(0)).Equals(uint8(10))
	assert.For(ctx, "a[1]").That(clone.A().Get(1)).Equals(uint8(20))
	assert.For(ctx, "a[2]").That(clone.A().Get(2)).Equals(uint8(30))
	assert.For(ctx, "a[3]").That(clone.A().Get(3)).Equals(uint8(40))

	checkBasicTypesZero(ctx, clone.B().Get(0))
	checkBasicTypesZero(ctx, clone.B().Get(1))
	checkBasicTypesPrimed(ctx, clone.B().Get(2))
	checkBasicTypesZero(ctx, clone.B().Get(3))
}

func TestCloneStrings(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	orig := NewStringsʳ(a1,
		"cat",  // A
		"says", // B
		"meow", // C
	)

	clone := orig.Clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	orig.SetA("dog")
	orig.SetB("says")
	orig.SetC("woof")

	assert.For(ctx, "a").That(clone.A()).Equals("cat")
	assert.For(ctx, "b").That(clone.B()).Equals("says")
	assert.For(ctx, "c").That(clone.C()).Equals("meow")
}

func TestCloneCmd(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	_, a1 := newEnv(ctx)
	ctx2, a2 := newEnv(ctx)

	cb := CommandBuilder{Arena: a1, Thread: 99}
	orig := cb.Foo(U8ᵖ(0x1234), 123.456, true, 42)

	clone := orig.clone(ctx2)

	assert.For(ctx, "arena").That(a2.Stats()).Equals(a1.Stats())

	orig.SetA(0)
	orig.SetB(0)
	orig.SetC(false)
	orig.SetResult(0)

	assert.For(ctx, "a").That(clone.A()).Equals(U8ᵖ(0x1234))
	assert.For(ctx, "b").That(clone.B()).Equals(float32(123.456))
	assert.For(ctx, "c").That(clone.C()).Equals(true)
	assert.For(ctx, "res").That(clone.Result()).Equals(memory.Int(42))
	assert.For(ctx, "thread").That(clone.Thread()).Equals(uint64(99))
}

func newEnv(ctx context.Context) (context.Context, arena.Arena) {
	p, err := capture.New(ctx, arena.New(), "cloned", nil, nil, nil)
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, p)

	c, err := capture.Resolve(ctx)
	if err != nil {
		panic(err)
	}

	env := c.Env().InitState().Build(ctx)
	env.AutoDispose()
	ctx = executor.PutEnv(ctx, env)

	return ctx, c.Arena
}
