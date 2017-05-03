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

package test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

func p(addr uint64) memory.Pointer {
	return memory.BytePtr(addr, memory.ApplicationPool)
}

func TestClone(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	expected := []byte{0x54, 0x33, 0x42, 0x43, 0x46, 0x34, 0x63, 0x24, 0x14, 0x24}
	for _, a := range []atom.Atom{
		NewCmdClone(p(0x1234), 10).
			AddRead(atom.Data(ctx, s.MemoryLayout, p(0x1234), expected)),
	} {
		a.Mutate(ctx, s, nil)
	}
	got := GetState(s).U8s.Read(ctx, nil, s, nil)
	assert.With(ctx).ThatSlice(got).Equals(expected)
}

func TestMake(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	assert.For(ctx, "initial NextPoolID").That(s.NextPoolID).Equals(memory.PoolID(1))
	NewCmdMake(10).Mutate(ctx, s, nil)
	assert.For(ctx, "buffer count").That(GetState(s).U8s.Count()).Equals(uint64(10))
	assert.For(ctx, "final NextPoolID").That(s.NextPoolID).Equals(memory.PoolID(2))
}

func TestCopy(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	expected := []byte{0x54, 0x33, 0x42, 0x43, 0x46, 0x34, 0x63, 0x24, 0x14, 0x24}
	for _, a := range []atom.Atom{
		NewCmdMake(10),
		NewCmdCopy(p(0x1234), 10).
			AddRead(atom.Data(ctx, s.MemoryLayout, p(0x1234), expected)),
	} {
		a.Mutate(ctx, s, nil)
	}
	got := GetState(s).U8s.Read(ctx, nil, s, nil)
	assert.With(ctx).ThatSlice(got).Equals(expected)
}

func TestCharsliceToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	expected := "ħęľĺő ŵōřŀď"
	NewCmdCharsliceToString(p(0x1234), uint32(len(expected))).
		AddRead(atom.Data(ctx, s.MemoryLayout, p(0x1234), expected)).
		Mutate(ctx, s, nil)
	assert.For(ctx, "Data").That(GetState(s).Str).Equals(expected)
}

func TestCharptrToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	expected := "ħęľĺő ŵōřŀď"
	NewCmdCharptrToString(p(0x1234)).
		AddRead(atom.Data(ctx, s.MemoryLayout, p(0x1234), expected)).
		Mutate(ctx, s, nil)
	assert.For(ctx, "Data").That(GetState(s).Str).Equals(expected)
}

func TestSliceCasts(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	s := gfxapi.NewStateWithEmptyAllocator(device.Little32)
	l := s.MemoryLayout.Clone()
	l.Integer.Size = 6 // non-multiple of u16
	s.MemoryLayout = l
	NewCmdSliceCasts(p(0x1234), 10).Mutate(ctx, s, nil)

	assert.For(ctx, "U16[] -> U8[]").That(GetState(s).U8s).Equals(U8ᵖ{0x1234, 0}.Slice(0, 20, l))
	assert.For(ctx, "U16[] -> U16[]").That(GetState(s).U16s).Equals(U16ᵖ{0x1234, 0}.Slice(0, 10, l))
	assert.For(ctx, "U16[] -> U32[]").That(GetState(s).U32s).Equals(U32ᵖ{0x1234, 0}.Slice(0, 5, l))
	assert.For(ctx, "U16[] -> int[]").That(GetState(s).Ints).Equals(Intᵖ{0x1234, 0}.Slice(0, 3, l))
}
