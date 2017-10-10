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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

func p(addr uint64) memory.Pointer {
	return memory.BytePtr(addr, memory.ApplicationPool)
}

func TestClone(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	expected := []byte{0x54, 0x33, 0x42, 0x43, 0x46, 0x34, 0x63, 0x24, 0x14, 0x24}
	api.MutateCmds(ctx, s, nil,
		cb.CmdClone(p(0x1234), 10).
			AddRead(memory.Store(ctx, s.MemoryLayout, p(0x1234), expected)),
	)
	got, err := GetState(s).U8s.Read(ctx, nil, s, nil)
	if assert.For(ctx, "err").ThatError(err).Succeeded() {
		assert.For(ctx, "got").ThatSlice(got).Equals(expected)
	}
}

func TestMake(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	idOfFirstPool, _ := s.Memory.New()
	assert.For(ctx, "initial NextPoolID").That(idOfFirstPool).Equals(memory.PoolID(1))
	api.MutateCmds(ctx, s, nil,
		cb.CmdMake(10),
	)
	assert.For(ctx, "buffer count").That(GetState(s).U8s.Count()).Equals(uint64(10))
	idOfOneAfterLastPool, _ := s.Memory.New()
	assert.For(ctx, "final NextPoolID").That(idOfOneAfterLastPool).Equals(memory.PoolID(3))
	assert.For(ctx, "pool count").That(s.Memory.Count()).Equals(4)
}

func TestCopy(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	expected := []byte{0x54, 0x33, 0x42, 0x43, 0x46, 0x34, 0x63, 0x24, 0x14, 0x24}
	api.MutateCmds(ctx, s, nil,
		cb.CmdMake(10),
		cb.CmdCopy(p(0x1234), 10).
			AddRead(memory.Store(ctx, s.MemoryLayout, p(0x1234), expected)),
	)
	got, err := GetState(s).U8s.Read(ctx, nil, s, nil)
	if assert.For(ctx, "err").ThatError(err).Succeeded() {
		assert.For(ctx, "got").ThatSlice(got).Equals(expected)
	}
}

func TestCharsliceToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	expected := "ħęľĺő ŵōřŀď"
	api.MutateCmds(ctx, s, nil,
		cb.CmdCharsliceToString(p(0x1234), uint32(len(expected))).
			AddRead(memory.Store(ctx, s.MemoryLayout, p(0x1234), expected)),
	)
	assert.For(ctx, "Data").That(GetState(s).Str).Equals(expected)
}

func TestCharptrToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	expected := "ħęľĺő ŵōřŀď"
	api.MutateCmds(ctx, s, nil,
		cb.CmdCharptrToString(p(0x1234)).
			AddRead(memory.Store(ctx, s.MemoryLayout, p(0x1234), expected)),
	)
	assert.For(ctx, "Data").That(GetState(s).Str).Equals(expected)
}

func TestSliceCasts(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	s := api.NewStateWithEmptyAllocator(device.Little32)
	l := s.MemoryLayout.Clone()
	l.Integer.Size = 6 // non-multiple of u16
	s.MemoryLayout = l
	api.MutateCmds(ctx, s, nil,
		cb.CmdSliceCasts(p(0x1234), 10),
	)
	assert.For(ctx, "U16[] -> U8[]").That(GetState(s).U8s).Equals(U8ᵖ{0x1234, 0}.Slice(0, 20, l))
	assert.For(ctx, "U16[] -> U16[]").That(GetState(s).U16s).Equals(U16ᵖ{0x1234, 0}.Slice(0, 10, l))
	assert.For(ctx, "U16[] -> U32[]").That(GetState(s).U32s).Equals(U32ᵖ{0x1234, 0}.Slice(0, 5, l))
	assert.For(ctx, "U16[] -> int[]").That(GetState(s).Ints).Equals(Intᵖ{0x1234, 0}.Slice(0, 3, l))
}
