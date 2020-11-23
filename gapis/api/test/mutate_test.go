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
	"bytes"
	"context"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/opcode"
	"github.com/google/gapid/gapis/replay/protocol"
)

type write struct {
	at  memory.Pointer
	src memory.Data
}

type expected struct {
	opcodes   []interface{}
	resources []id.ID
	constants []byte
}

type test struct {
	writes   []write
	cmds     []api.Cmd
	expected expected
}

func (test test) check(ctx context.Context, ca, ra *device.MemoryLayout) {
	b := builder.New(ra, nil)
	s := api.NewStateWithEmptyAllocator(device.Little32)
	s.MemoryLayout = ca

	for _, w := range test.writes {
		s.Memory.ApplicationPool().Write(w.at.Address(), w.src)
	}

	api.ForeachCmd(ctx, test.cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		b.BeginCommand(uint64(id), 0)
		err := cmd.Mutate(ctx, id, s, b, nil)
		assert.For(ctx, "Mutate command").ThatError(err).Succeeded()
		b.CommitCommand(ctx, false)
		return nil
	})

	payload, _, _, _, err := b.Build(ctx)
	assert.For(ctx, "Build opcodes").ThatError(err).Succeeded()

	ops := bytes.NewBuffer(payload.Opcodes)
	gotOpcodes, err := opcode.Disassemble(ops, device.LittleEndian)
	assert.For(ctx, "Dissasemble opcodes").ThatError(err).Succeeded()
	assert.For(ctx, "Opcodes").ThatSlice(gotOpcodes).Equals(test.expected.opcodes)
	checkResource(ctx, payload.Resources, test.expected.resources)
	assert.For(ctx, "Constants").ThatSlice(payload.Constants).Equals(test.expected.constants)
}

func checkResource(ctx context.Context, gotInfos []*gapir.ResourceInfo, expectedIDs []id.ID) {
	var err error

	got := make([]interface{}, len(gotInfos))
	for i, g := range gotInfos {
		ctx := log.V{"id": g.Id}.Bind(ctx)
		id, err := id.Parse(g.Id)
		assert.For(ctx, "Parse resource ID").ThatError(err).Succeeded()
		got[i], err = database.Resolve(ctx, id)
		assert.For(ctx, "Get resource").ThatError(err).Succeeded()
	}

	expected := make([]interface{}, len(expectedIDs))
	for i, e := range expectedIDs {
		ctx := log.V{"id": e}.Bind(ctx)
		expected[i], err = database.Resolve(ctx, e)
		assert.For(ctx, "Get resource").ThatError(err).Succeeded()
	}

	assert.For(ctx, "Data").ThatSlice(got).DeepEquals(expected)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestOperationsOpCall_NoIn_NoOut(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdVoid(),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoid.ApiIndex, FunctionID: funcInfoCmdVoid.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_Clone(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	rng, rID := memory.Store(ctx, ml, p(0x100000), []uint8{5, 6, 7, 8, 9})

	test{
		cmds: []api.Cmd{
			cb.CmdClone(p(0x100000), 5).AddRead(rng, rID),
		},
		expected: expected{
			resources: []id.ID{rID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdClone.ApiIndex, FunctionID: funcInfoCmdClone.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_Make(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdMake(5),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdMake.ApiIndex, FunctionID: funcInfoCmdMake.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_Copy(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	rng, rID := memory.Store(ctx, ml, p(0x100000), []uint8{5, 6, 7, 8, 9})

	test{
		cmds: []api.Cmd{
			cb.CmdCopy(p(0x100000), 5).AddRead(rng, rID),
		},
		expected: expected{
			resources: []id.ID{rID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdCopy.ApiIndex, FunctionID: funcInfoCmdCopy.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_CharSliceToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	rng, rID := memory.Store(ctx, ml, p(0x100000), []uint8{5, 6, 0, 8, 9})

	test{
		cmds: []api.Cmd{
			cb.CmdCharsliceToString(p(0x100000), 5).AddRead(rng, rID),
		},
		expected: expected{
			resources: []id.ID{rID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdCharsliceToString.ApiIndex, FunctionID: funcInfoCmdCharsliceToString.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_CharPtrToString(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	_, rID := memory.Store(ctx, ml, p(0x100000), []uint8{'g', 'o', 'o', 'd', 0})

	test{
		cmds: []api.Cmd{
			cb.CmdCharptrToString(p(0x100000)).
				AddRead(memory.Store(ctx, ml, p(0x100000), []uint8{'g', 'o', 'o', 'd', 0, 'd', 'a', 'y'})),
		},
		expected: expected{
			resources: []id.ID{rID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdCharptrToString.ApiIndex, FunctionID: funcInfoCmdCharptrToString.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_Unknowns(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := &device.MemoryLayout{
		Endian:  device.LittleEndian,
		Pointer: &device.DataTypeLayout{Size: 4, Alignment: 4},
		Integer: &device.DataTypeLayout{Size: 8, Alignment: 4},
		Size:    &device.DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &device.DataTypeLayout{Size: 1, Alignment: 4},
		I64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &device.DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
	}

	test{
		cmds: []api.Cmd{
			cb.CmdUnknownRet(10),
			cb.CmdUnknownWritePtr(p(0x200000)).
				AddRead(memory.Store(ctx, ml, p(0x200000), int(100))).
				AddWrite(memory.Store(ctx, ml, p(0x200000), int(200))),
			cb.CmdUnknownWriteSlice(p(0x100000)).
				AddRead(memory.Store(ctx, ml, p(0x100000), []int{0, 1, 2, 3, 4})).
				AddWrite(memory.Store(ctx, ml, p(0x100000), []int{5, 6, 7, 8, 9})),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdUnknownRet.ApiIndex, FunctionID: funcInfoCmdUnknownRet.ID},

				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 8 * 5},
				opcode.Call{ApiIndex: funcInfoCmdUnknownWritePtr.ApiIndex, FunctionID: funcInfoCmdUnknownWritePtr.ID},

				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdUnknownWriteSlice.ApiIndex, FunctionID: funcInfoCmdUnknownWriteSlice.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_SingleInputArg(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdVoidU8(20),
			cb.CmdVoidS8(-20),
			cb.CmdVoidU16(200),
			cb.CmdVoidS16(-200),
			cb.CmdVoidF32(1.0),
			cb.CmdVoidU32(2000),
			cb.CmdVoidS32(-2000),
			cb.CmdVoidF64(1.0),
			cb.CmdVoidU64(20000),
			cb.CmdVoidS64(-20000),
			cb.CmdVoidBool(true),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_Uint8, Value: 20},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidU8.ApiIndex, FunctionID: funcInfoCmdVoidU8.ID},

				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_Int8, Value: 0xfffec},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidS8.ApiIndex, FunctionID: funcInfoCmdVoidS8.ID},

				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_Uint16, Value: 200},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidU16.ApiIndex, FunctionID: funcInfoCmdVoidU16.ID},

				opcode.Label{Value: 3},
				opcode.PushI{DataType: protocol.Type_Int16, Value: 0xfff38},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidS16.ApiIndex, FunctionID: funcInfoCmdVoidS16.ID},

				opcode.Label{Value: 4},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x7f},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidF32.ApiIndex, FunctionID: funcInfoCmdVoidF32.ID},

				opcode.Label{Value: 5},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 2000},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidU32.ApiIndex, FunctionID: funcInfoCmdVoidU32.ID},

				opcode.Label{Value: 6},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xff830},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidS32.ApiIndex, FunctionID: funcInfoCmdVoidS32.ID},

				opcode.Label{Value: 7},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0x3ff},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidF64.ApiIndex, FunctionID: funcInfoCmdVoidF64.ID},

				opcode.Label{Value: 8},
				opcode.PushI{DataType: protocol.Type_Uint64, Value: 20000},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidU64.ApiIndex, FunctionID: funcInfoCmdVoidU64.ID},

				opcode.Label{Value: 9},
				opcode.PushI{DataType: protocol.Type_Int64, Value: 0xfb1e0},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidS64.ApiIndex, FunctionID: funcInfoCmdVoidS64.ID},

				opcode.Label{Value: 10},
				opcode.PushI{DataType: protocol.Type_Bool, Value: 1},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoidBool.ApiIndex, FunctionID: funcInfoCmdVoidBool.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_3_In_Arrays(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little64

	aRng, aID := memory.Store(ctx, ml, p(0x40000+5* /* sizeof(u8)  */ 1), []uint8{
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	})
	bRng, bID := memory.Store(ctx, ml, p(0x50000+5* /* sizeof(u32) */ 4), []uint32{
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	})
	cRng, cID := memory.Store(ctx, ml, p(0x60000+5* /* sizeof(int) */ 8), []int{
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	})

	test{
		cmds: []api.Cmd{
			cb.CmdVoid3InArrays(p(0x40000), p(0x50000), p(0x60000)).
				AddRead(aRng, aID).
				AddRead(bRng, bID).
				AddRead(cRng, cID),
		},
		expected: expected{
			//   ┌────┬────┬────┬────┬────╔════╤════╤════╤════╤════╤════╤════╤════╤════╤════╗
			// b │0x10│0x14│0x18│0x1c│0x20║0x24│0x28│0x2c│0x30│0x34│0x38│0x3c│0x40│0x44│0x48║
			//   └────┴────┴────┴────┴────╚════╧════╧════╧════╧════╧════╧════╧════╧════╧════╝
			//   ┌────┬────┬────┬────┬────╔════╤════╤════╤════╤════╤════╤════╤════╤════╤════╗
			// c │0x50│0x58│0x60│0x68│0x70║0x78│0x80│0x88│0x90│0x98│0xa0│0xa8│0xb0│0xb8│0xc0║
			//   └────┴────┴────┴────┴────╚════╧════╧════╧════╧════╧════╧════╧════╧════╧════╝
			//   ┌────┬────┬────┬────┬────╔════╤════╤════╤════╤════╤════╤════╤════╤════╤════╗
			// a │0x00│0x01│0x02│0x03│0x04║0x05│0x06│0x07│0x08│0x09│0x0a│0x0b│0x0c│0x0d│0x0e║
			//   └────┴────┴────┴────┴────╚════╧════╧════╧════╧════╧════╧════╧════╧════╧════╝
			resources: []id.ID{bID, cID, aID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x24},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x78},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x5},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x50},
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoid3InArrays.ApiIndex, FunctionID: funcInfoCmdVoid3InArrays.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_3_In_StaticArrays(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little64

	u8s := NewU8ː3ᵃ(4, 5, 6)
	u32s := NewU32ː3ᵃ(7, 8, 9)

	test{
		cmds: []api.Cmd{
			cb.CmdVoid3InStaticArrays(u8s, u32s, u8s),
		},
		expected: expected{
			constants: []byte{
				4, 5, 6,
				0, 0, 0, 0, 0, // alignment padding
				7, 0, 0, 0, 8, 0, 0, 0, 9, 0, 0, 0,
			},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_ConstantPointer, Value: 0x0},
				opcode.PushI{DataType: protocol.Type_ConstantPointer, Value: 0x8},
				opcode.PushI{DataType: protocol.Type_ConstantPointer, Value: 0x0}, // Shared constant data (dedupe)
				opcode.Call{PushReturn: false, ApiIndex: funcInfoCmdVoid3InStaticArrays.ApiIndex, FunctionID: funcInfoCmdVoid3InStaticArrays.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_InArrayOfStrings(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32

	aRng, aID := memory.Store(ctx, ml, p(0x100000), "array")
	bRng, bID := memory.Store(ctx, ml, p(0x200000), "of")
	cRng, cID := memory.Store(ctx, ml, p(0x300000), "strings")

	pRng, pID := memory.Store(ctx, ml, p(0x500000), []memory.Pointer{
		p(0x300000), p(0x200000), p(0x100000), p(0x200000), p(0x300000),
	})

	test{
		cmds: []api.Cmd{
			// 0x100000: "array"
			// 0x200000: "of"
			// 0x300000: "strings"
			// 0x500000: 0x300000
			// 0x500004: 0x200000
			// 0x500008: 0x100000
			// 0x50000c: 0x200000
			// 0x500010: 0x300000
			cb.CmdVoidInArrayOfStrings(p(0x500000), 5).
				AddRead(aRng, aID).
				AddRead(bRng, bID).
				AddRead(cRng, cID).
				AddRead(pRng, pID),
		},
		expected: expected{
			// 0x00: "array"   (6 bytes)
			// 0x08: "of"      (3 bytes)
			// 0x0c: "strings" (8 bytes)
			// 0x14: 0x0c
			// 0x18: 0x08
			// 0x1c: 0x00
			// 0x20: 0x08
			// 0x24: 0x0c
			resources: []id.ID{cID, bID, aID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				// TODO: Collate sequential reads / writes to reduce 5 Resource opcodes
				// to one.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.StoreV{Address: 0x14},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x18},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.StoreV{Address: 0x1c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 2},

				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x20},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.StoreV{Address: 0x24},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdVoidInArrayOfStrings.ApiIndex, FunctionID: funcInfoCmdVoidInArrayOfStrings.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_InArrayOfStrings_32bitTo64Bit(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ca := device.Little32
	ra := device.Little64

	aRng, aID := memory.Store(ctx, ca, p(0x100000), "array")
	bRng, bID := memory.Store(ctx, ca, p(0x200000), "of")
	cRng, cID := memory.Store(ctx, ca, p(0x300000), "strings")

	pRng, pID := memory.Store(ctx, ca, p(0x500000), []memory.Pointer{
		p(0x300000), p(0x200000), p(0x100000), p(0x200000), p(0x300000),
	})

	test{
		cmds: []api.Cmd{
			// 0x100000: "array"
			// 0x200000: "of"
			// 0x300000: "strings"
			// 0x500000: 0x300000
			// 0x500004: 0x200000
			// 0x500008: 0x100000
			// 0x50000c: 0x200000
			// 0x500010: 0x300000
			cb.CmdVoidInArrayOfStrings(p(0x500000), 5).
				AddRead(aRng, aID).
				AddRead(bRng, bID).
				AddRead(cRng, cID).
				AddRead(pRng, pID),
		},
		expected: expected{
			// 0x00: "array"   (6 bytes)
			// 0x08: "of"      (3 bytes)
			// 0x10: "strings" (8 bytes)
			// 0x18: 0x10
			// 0x20: 0x08
			// 0x28: 0x00
			// 0x30: 0x08
			// 0x38: 0x10
			resources: []id.ID{cID, bID, aID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				// TODO: Collate sequential reads / writes to reduce 5 Resource opcodes
				// to one.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.StoreV{Address: 0x18},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x20},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.StoreV{Address: 0x28},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 2},

				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x30},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.StoreV{Address: 0x38},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x18},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdVoidInArrayOfStrings.ApiIndex, FunctionID: funcInfoCmdVoidInArrayOfStrings.ID},
			},
		},
	}.check(ctx, ca, ra)
}

func TestOperationsOpCall_SinglePointerElementRead(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	p := memory.Pointer(p(0x100000))
	rng1, id1 := memory.Store(ctx, ml, p, []byte{
		0x01,
	})
	rng2, id2 := memory.Store(ctx, ml, p, []byte{
		0x01, 0x23,
	})
	rng4, id4 := memory.Store(ctx, ml, p, []byte{
		0x01, 0x23, 0x45, 0x67,
	})
	rng8, id8 := memory.Store(ctx, ml, p, []byte{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
	})
	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadBool(p).AddRead(rng1, id1),
			cb.CmdVoidReadU8(p).AddRead(rng1, id1),
			cb.CmdVoidReadS8(p).AddRead(rng1, id1),
			cb.CmdVoidReadU16(p).AddRead(rng2, id2),
			cb.CmdVoidReadS16(p).AddRead(rng2, id2),
			cb.CmdVoidReadF32(p).AddRead(rng4, id4),
			cb.CmdVoidReadU32(p).AddRead(rng4, id4),
			cb.CmdVoidReadS32(p).AddRead(rng4, id4),
			cb.CmdVoidReadF64(p).AddRead(rng8, id8),
			cb.CmdVoidReadU64(p).AddRead(rng8, id8),
			cb.CmdVoidReadS64(p).AddRead(rng8, id8),

			cb.CmdVoidReadS32(p),  // Uses previous observations
			cb.CmdVoidReadBool(p), // Uses previous observations
		},
		expected: expected{
			resources: []id.ID{id1, id2, id4, id8},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadBool.ApiIndex, FunctionID: funcInfoCmdVoidReadBool.ID},

				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadU8.ApiIndex, FunctionID: funcInfoCmdVoidReadU8.ID},

				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadS8.ApiIndex, FunctionID: funcInfoCmdVoidReadS8.ID},

				opcode.Label{Value: 3},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadU16.ApiIndex, FunctionID: funcInfoCmdVoidReadU16.ID},

				opcode.Label{Value: 4},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadS16.ApiIndex, FunctionID: funcInfoCmdVoidReadS16.ID},

				opcode.Label{Value: 5},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadF32.ApiIndex, FunctionID: funcInfoCmdVoidReadF32.ID},

				opcode.Label{Value: 6},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadU32.ApiIndex, FunctionID: funcInfoCmdVoidReadU32.ID},

				opcode.Label{Value: 7},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadS32.ApiIndex, FunctionID: funcInfoCmdVoidReadS32.ID},

				opcode.Label{Value: 8},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 3},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadF64.ApiIndex, FunctionID: funcInfoCmdVoidReadF64.ID},

				opcode.Label{Value: 9},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 3},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadU64.ApiIndex, FunctionID: funcInfoCmdVoidReadU64.ID},

				opcode.Label{Value: 10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 3},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadS64.ApiIndex, FunctionID: funcInfoCmdVoidReadS64.ID},

				opcode.Label{Value: 11},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadS32.ApiIndex, FunctionID: funcInfoCmdVoidReadS32.ID},

				opcode.Label{Value: 12},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadBool.ApiIndex, FunctionID: funcInfoCmdVoidReadBool.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_MultiplePointerElementReads(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := &device.MemoryLayout{
		Endian:  device.LittleEndian,
		Pointer: &device.DataTypeLayout{Size: 4, Alignment: 16},
		Integer: &device.DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &device.DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &device.DataTypeLayout{Size: 1, Alignment: 4},
		I64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &device.DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
	}

	aRng, aID := memory.Store(ctx, ml, p(0x100000), float32(10))
	bRng, bID := memory.Store(ctx, ml, p(0x200000), uint16(20))
	cRng, cID := memory.Store(ctx, ml, p(0x300000), false)
	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadPtrs(p(0x100000), p(0x200000), p(0x300000)).
				AddRead(aRng, aID).
				AddRead(bRng, bID).
				AddRead(cRng, cID),
		},
		expected: expected{
			resources: []id.ID{aID, bID, cID},
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x20},
				opcode.Resource{ID: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x20},
				opcode.Call{ApiIndex: funcInfoCmdVoidReadPtrs.ApiIndex, FunctionID: funcInfoCmdVoidReadPtrs.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_SinglePointerElementWrite(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdVoidWriteU8(p(0x100000)).
				AddWrite(memory.Store(ctx, ml, p(0x100000), uint8(1))),
			cb.CmdVoidWriteS8(p(0x200000)).
				AddWrite(memory.Store(ctx, ml, p(0x200000), int8(1))),
			cb.CmdVoidWriteU16(p(0x300000)).
				AddWrite(memory.Store(ctx, ml, p(0x300000), uint16(1))),
			cb.CmdVoidWriteS16(p(0x400000)).
				AddWrite(memory.Store(ctx, ml, p(0x400000), int16(1))),
			cb.CmdVoidWriteF32(p(0x500000)).
				AddWrite(memory.Store(ctx, ml, p(0x500000), float32(1))),
			cb.CmdVoidWriteU32(p(0x600000)).
				AddWrite(memory.Store(ctx, ml, p(0x600000), uint32(1))),
			cb.CmdVoidWriteS32(p(0x700000)).
				AddWrite(memory.Store(ctx, ml, p(0x700000), int32(1))),
			cb.CmdVoidWriteF64(p(0x800000)).
				AddWrite(memory.Store(ctx, ml, p(0x800000), float64(1))),
			cb.CmdVoidWriteU64(p(0x900000)).
				AddWrite(memory.Store(ctx, ml, p(0x900000), uint64(1))),
			cb.CmdVoidWriteS64(p(0xa00000)).
				AddWrite(memory.Store(ctx, ml, p(0xa00000), int64(1))),
			cb.CmdVoidWriteBool(p(0xb00000)).
				AddWrite(memory.Store(ctx, ml, p(0xb00000), bool(true))),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteU8.ApiIndex, FunctionID: funcInfoCmdVoidWriteU8.ID},
				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteS8.ApiIndex, FunctionID: funcInfoCmdVoidWriteS8.ID},
				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteU16.ApiIndex, FunctionID: funcInfoCmdVoidWriteU16.ID},
				opcode.Label{Value: 3},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteS16.ApiIndex, FunctionID: funcInfoCmdVoidWriteS16.ID},
				opcode.Label{Value: 4},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteF32.ApiIndex, FunctionID: funcInfoCmdVoidWriteF32.ID},
				opcode.Label{Value: 5},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteU32.ApiIndex, FunctionID: funcInfoCmdVoidWriteU32.ID},
				opcode.Label{Value: 6},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x18},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteS32.ApiIndex, FunctionID: funcInfoCmdVoidWriteS32.ID},
				opcode.Label{Value: 7},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteF64.ApiIndex, FunctionID: funcInfoCmdVoidWriteF64.ID},
				opcode.Label{Value: 8},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x24},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteU64.ApiIndex, FunctionID: funcInfoCmdVoidWriteU64.ID},
				opcode.Label{Value: 9},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x2c},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteS64.ApiIndex, FunctionID: funcInfoCmdVoidWriteS64.ID},
				opcode.Label{Value: 10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x34},
				opcode.Call{ApiIndex: funcInfoCmdVoidWriteBool.ApiIndex, FunctionID: funcInfoCmdVoidWriteBool.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_MultiplePointerElementWrites(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := &device.MemoryLayout{
		Endian:  device.LittleEndian,
		Pointer: &device.DataTypeLayout{Size: 4, Alignment: 16},
		Integer: &device.DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &device.DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &device.DataTypeLayout{Size: 1, Alignment: 4},
		I64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &device.DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &device.DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
	}
	test{
		cmds: []api.Cmd{
			cb.CmdVoidWritePtrs(p(0x100000), p(0x200000), p(0x300000)),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x20},
				opcode.Call{ApiIndex: funcInfoCmdVoidWritePtrs.ApiIndex, FunctionID: funcInfoCmdVoidWritePtrs.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReturnValue(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdU8(20),
			cb.CmdS8(-20),
			cb.CmdU16(200),
			cb.CmdS16(-200),
			cb.CmdF32(1.0),
			cb.CmdU32(2000),
			cb.CmdS32(-2000),
			cb.CmdF64(1.0),
			cb.CmdU64(20000),
			cb.CmdS64(-20000),
			cb.CmdBool(true),
			cb.CmdPointer(p(0x10000)),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				opcode.Call{ApiIndex: funcInfoCmdU8.ApiIndex, FunctionID: funcInfoCmdU8.ID},
				opcode.Label{Value: 1},
				opcode.Call{ApiIndex: funcInfoCmdS8.ApiIndex, FunctionID: funcInfoCmdS8.ID},
				opcode.Label{Value: 2},
				opcode.Call{ApiIndex: funcInfoCmdU16.ApiIndex, FunctionID: funcInfoCmdU16.ID},
				opcode.Label{Value: 3},
				opcode.Call{ApiIndex: funcInfoCmdS16.ApiIndex, FunctionID: funcInfoCmdS16.ID},
				opcode.Label{Value: 4},
				opcode.Call{ApiIndex: funcInfoCmdF32.ApiIndex, FunctionID: funcInfoCmdF32.ID},
				opcode.Label{Value: 5},
				opcode.Call{ApiIndex: funcInfoCmdU32.ApiIndex, FunctionID: funcInfoCmdU32.ID},
				opcode.Label{Value: 6},
				opcode.Call{ApiIndex: funcInfoCmdS32.ApiIndex, FunctionID: funcInfoCmdS32.ID},
				opcode.Label{Value: 7},
				opcode.Call{ApiIndex: funcInfoCmdF64.ApiIndex, FunctionID: funcInfoCmdF64.ID},
				opcode.Label{Value: 8},
				opcode.Call{ApiIndex: funcInfoCmdU64.ApiIndex, FunctionID: funcInfoCmdU64.ID},
				opcode.Label{Value: 9},
				opcode.Call{ApiIndex: funcInfoCmdS64.ApiIndex, FunctionID: funcInfoCmdS64.ID},
				opcode.Label{Value: 10},
				opcode.Call{ApiIndex: funcInfoCmdBool.ApiIndex, FunctionID: funcInfoCmdBool.ID},
				opcode.Label{Value: 11},
				opcode.Call{ApiIndex: funcInfoCmdPointer.ApiIndex, FunctionID: funcInfoCmdPointer.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_3Remapped(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdVoid3Remapped(0x10, 0x20, 0x10),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},
				// First-seen values get an identical remapping value.
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x10},
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x0},

				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x20},
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x4},

				// Subsequently-seen values use the remapped value.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x00},

				opcode.Call{ApiIndex: funcInfoCmdVoid3Remapped.ApiIndex, FunctionID: funcInfoCmdVoid3Remapped.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_InArrayOfRemapped(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	rng, id := memory.Store(ctx, ml, p(0x100000), []Remapped{10, 20, 10, 30, 20})

	pbase := uint32(4 * 3) // parameter array base address
	tbase := uint32(0)     // remap table base address

	test{
		cmds: []api.Cmd{
			cb.CmdVoidInArrayOfRemapped(p(0x100000)).
				AddRead(rng, id),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				// 10 --> remap[0], param[0]
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 10},
				opcode.Clone{},
				opcode.StoreV{Address: tbase + 4*0},
				opcode.StoreV{Address: pbase + 4*0},

				// 20 --> remap[1], param[1]
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 20},
				opcode.Clone{},
				opcode.StoreV{Address: tbase + 4*1},
				opcode.StoreV{Address: pbase + 4*1},

				// remap[0] --> param[2]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: tbase + 4*0},
				opcode.StoreV{Address: pbase + 4*2},

				// 30 --> remap[2], param[3]
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 30},
				opcode.Clone{},
				opcode.StoreV{Address: tbase + 4*2},
				opcode.StoreV{Address: pbase + 4*3},

				// remap[1] --> param[4]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: tbase + 4*1},
				opcode.StoreV{Address: pbase + 4*4},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: pbase},
				opcode.Call{ApiIndex: funcInfoCmdVoidInArrayOfRemapped.ApiIndex, FunctionID: funcInfoCmdVoidInArrayOfRemapped.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_OutArrayOfRemapped(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	pbase := uint32(4 * 3) // parameter array base address
	tbase := uint32(0)     // remap table base address

	test{
		cmds: []api.Cmd{
			cb.CmdVoidOutArrayOfRemapped(p(0x100000)).
				AddWrite(memory.Store(ctx, ml, p(0x100000), []Remapped{10, 20, 10, 30, 20})),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: pbase},
				opcode.Call{ApiIndex: funcInfoCmdVoidOutArrayOfRemapped.ApiIndex, FunctionID: funcInfoCmdVoidOutArrayOfRemapped.ID},

				// param[0] --> remap[0]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*0},
				opcode.StoreV{Address: tbase + 4*0},

				// param[1] --> remap[1]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*1},
				opcode.StoreV{Address: tbase + 4*1},

				// param[2] --> remap[0]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*2},
				opcode.StoreV{Address: tbase + 4*0},

				// param[3] --> remap[2]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*3},
				opcode.StoreV{Address: tbase + 4*2},

				// param[4] --> remap[2]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*4},
				opcode.StoreV{Address: tbase + 4*1},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_OutArrayOfUnknownRemapped(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	pbase := uint32(4 * 3) // parameter array base address
	tbase := uint32(0)     // remap table base address

	test{
		cmds: []api.Cmd{
			cb.CmdVoidOutArrayOfUnknownRemapped(p(0x100000)).
				AddWrite(memory.Store(ctx, ml, p(0x100000), []Remapped{10, 20, 10, 30, 20})),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: pbase},
				opcode.Call{ApiIndex: funcInfoCmdVoidOutArrayOfUnknownRemapped.ApiIndex, FunctionID: funcInfoCmdVoidOutArrayOfUnknownRemapped.ID},

				// param[0] --> remap[0]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*0},
				opcode.StoreV{Address: tbase + 4*0},

				// param[1] --> remap[1]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*1},
				opcode.StoreV{Address: tbase + 4*1},

				// param[2] --> remap[0]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*2},
				opcode.StoreV{Address: tbase + 4*0},

				// param[3] --> remap[2]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*3},
				opcode.StoreV{Address: tbase + 4*2},

				// param[4] --> remap[1]
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: pbase + 4*4},
				opcode.StoreV{Address: tbase + 4*1},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_Remapped(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	test{
		cmds: []api.Cmd{
			cb.CmdRemapped(200),
			cb.CmdVoid3Remapped(100, 200, 300),
		},
		expected: expected{
			opcodes: []interface{}{
				opcode.Label{Value: 0},

				opcode.Call{PushReturn: true, ApiIndex: funcInfoCmdRemapped.ApiIndex, FunctionID: funcInfoCmdRemapped.ID},
				opcode.StoreV{Address: 0x0},

				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 100},
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x4},

				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x00},

				opcode.PushI{DataType: protocol.Type_Uint32, Value: 300},
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x8},

				opcode.Call{ApiIndex: funcInfoCmdVoid3Remapped.ApiIndex, FunctionID: funcInfoCmdVoid3Remapped.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReadRemappedStruct(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32

	aRng, aID := memory.Store(ctx, ml, p(0x100000), NewRemappedStruct(10, 20, 30))
	bRng, bID := memory.Store(ctx, ml, p(0x200000), NewRemappedStruct(40, 20, 50))

	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadRemappedStruct(p(0x100000)).
				AddRead(aRng, aID),
			cb.CmdVoidReadRemappedStruct(p(0x100000)).
				AddRead(aRng, aID), // reading the same struct
			cb.CmdVoidReadRemappedStruct(p(0x200000)).
				AddRead(bRng, bID), // reading a different struct with the same handle
		},
		expected: expected{
			resources: []id.ID{aID, bID},
			opcodes: []interface{}{
				// 0x00: remapped handle

				// 0x04: RemappedStruct::F1     (size: 8)
				// 0x0c: RemappedStruct::Handle (size: 4)
				// 0x10: RemappedStruct::F3     (size: 4)

				// 0x14: RemappedStruct::F1     (size: 8)
				// 0x1c: RemappedStruct::Handle (size: 4)
				// 0x20: RemappedStruct::F3     (size: 4)

				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 20},
				// First-seen values get an identical remapping value.
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x00},
				// Update the handle value in the struct.
				opcode.StoreV{Address: 0x0c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadRemappedStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadRemappedStruct.ID},

				opcode.Label{Value: 1},
				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Resource{ID: 0},
				// Subsequently-seen values use the remapped value.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x00},
				// Update the handle value in the struct.
				opcode.StoreV{Address: 0x0c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadRemappedStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadRemappedStruct.ID},

				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Resource{ID: 1},
				// Subsequently-seen values use the remapped value.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x00},
				// Update the handle value in the struct.
				opcode.StoreV{Address: 0x1c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadRemappedStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadRemappedStruct.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReadPointerStruct(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32

	aRng, aID := memory.Store(
		ctx, ml, p(0x100000),
		NewPointerStruct(0x23, 0x01, 0x200000))
	bRng, bID := memory.Store(ctx, ml, p(0x200000), uint32(0x45))
	cRng, cID := memory.Store(
		ctx, ml, p(0x300000),
		NewPointerStruct(0x89, 0x67, 0x200000))

	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadPointerStruct(p(0x100000)).
				AddRead(aRng, aID).
				AddRead(bRng, bID),
			cb.CmdVoidReadPointerStruct(p(0x100000)).
				AddRead(aRng, aID).
				AddRead(bRng, bID), // same PointerStruct
			cb.CmdVoidReadPointerStruct(p(0x300000)).
				AddRead(cRng, cID).
				AddRead(bRng, bID), // different PointerStruct
		},
		expected: expected{
			resources: []id.ID{aID, bID, cID},
			opcodes: []interface{}{
				// 0x00: PointerStruct::F2      (size: 8)
				// 0x08: PointerStruct::F1      (size: 4)
				// 0x0c: PointerStruct::Pointer (size: 4)

				// 0x10: uint32(0x45)

				// 0x14: PointerStruct::F2      (size: 8)
				// 0x0c: PointerStruct::F1      (size: 4)
				// 0x20: PointerStruct::Pointer (size: 4)

				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 0},
				// Update the pointer address in the struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.StoreV{Address: 0x0c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadPointerStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadPointerStruct.ID},

				// The second command generates the same opcode stream.
				opcode.Label{Value: 1},
				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 0},
				// Update the pointer address in the struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.StoreV{Address: 0x0c},
				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadPointerStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadPointerStruct.ID},

				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Resource{ID: 2},
				// Update the pointer address in the struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.StoreV{Address: 0x20},
				// TODO: Resource loads below are redundant
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadPointerStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadPointerStruct.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReadNestedStruct(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32

	nestedRng, nestedID := memory.Store(
		ctx, ml, p(0x100000),
		NewNestedStruct(0x200000, 0x300000))
	rsRng, rsID := memory.Store(
		ctx, ml, p(0x200000),
		NewRemappedStruct(0x01, 0x23, 0x45))
	psRng, psID := memory.Store(
		ctx, ml, p(0x300000),
		NewPointerStruct(0x67, 0x89, 0x400000))
	pRng, pID := memory.Store(ctx, ml, p(0x400000), uint32(0xab))

	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadNestedStruct(p(0x100000)).
				AddRead(nestedRng, nestedID).
				AddRead(rsRng, rsID).
				AddRead(psRng, psID).
				AddRead(pRng, pID),
			cb.CmdVoidReadRemappedStruct(p(0x200000)).
				AddRead(rsRng, rsID), // use the remapped struct again
			cb.CmdVoidReadPointerStruct(p(0x300000)).
				AddRead(psRng, psID).
				AddRead(pRng, pID), // use the pointer struct again
		},
		expected: expected{
			resources: []id.ID{nestedID, rsID, psID, pID},
			opcodes: []interface{}{
				// 0x00: remapped handle

				// 0x04: NestedStruct::RS       (size: 4) -> Points to 0x0c
				// 0x08: NestedStruct::PS       (size: 4) -> Points to 0x1c

				// 0x0c: RemappedStruct::F1     (size: 8)
				// 0x14: RemappedStruct::Handle (size: 4)
				// 0x18: RemappedStruct::F3     (size: 4)

				// 0x1c: PointerStruct::F1      (size: 4)
				// 0x20: PointerStruct::F2      (size: 8)
				// 0x28: PointerStruct::Pointer (size: 4)

				// 0x2c: uint32(0xab)

				opcode.Label{Value: 0},

				// Nested struct laid out at 0x04.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Resource{ID: 0},
				// Update the pointer addresses in nested struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.StoreV{Address: 0x04},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.StoreV{Address: 0x08},

				// Remapped struct laid out at 0x0c.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 1},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x23},
				// First-seen values get an identical remapping value.
				opcode.Clone{Index: 0},
				opcode.StoreV{Address: 0x00},
				// Update the handle in the remapped struct.
				opcode.StoreV{Address: 0x14},

				// Pointer struct laid out at 0x1c.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.Resource{ID: 2},
				// Update the address in pointer struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x2c},
				opcode.StoreV{Address: 0x28},

				// Resource for uint32.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x2c},
				opcode.Resource{ID: 3},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x04},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadNestedStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadNestedStruct.ID},

				opcode.Label{Value: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 1}, // TODO: redundant resource request
				// Subsequently-seen values use the remapped value.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x00},
				opcode.StoreV{Address: 0x14},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadRemappedStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadRemappedStruct.ID},

				opcode.Label{Value: 2},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.Resource{ID: 2}, // TODO: redundant resource request
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x2c},
				opcode.StoreV{Address: 0x28},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x2c},
				opcode.Resource{ID: 3}, // TODO: redundant resource request

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadPointerStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadPointerStruct.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReadStringStruct(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32

	aRng, aID := memory.Store(ctx, ml, p(0x100000), "array")
	bRng, bID := memory.Store(ctx, ml, p(0x200000), "of")
	cRng, cID := memory.Store(ctx, ml, p(0x300000), "strings")

	pRng, pID := memory.Store(ctx, ml, p(0x400000), []memory.Pointer{
		p(0x300000), p(0x200000), p(0x100000), p(0x200000), p(0x300000),
	})

	ssRng, ssID := memory.Store(ctx, ml, p(0x500000),
		NewStringStruct(5 /* size */, 0x400000 /* strings */))

	test{
		cmds: []api.Cmd{
			cb.CmdVoidReadStringStruct(p(0x500000)).
				AddRead(ssRng, ssID).
				AddRead(pRng, pID).
				AddRead(aRng, aID).
				AddRead(bRng, bID).
				AddRead(cRng, cID),
		},
		expected: expected{
			resources: []id.ID{ssID, cID, bID, aID},
			opcodes: []interface{}{

				// 0x00  "array"               (size: 6)
				// 0x08: "of"                  (size: 3)
				// 0x0c: "strings"             (size: 8)

				// 0x14: StringStruct::Count   (size: 4)
				// 0x18: StringStruct::Strings (size: 4) -> 0x1c

				// 0x1c: -> 0x0c "strings"
				// 0x20: -> 0x08 "of"
				// 0x24: -> 0x00 "array"
				// 0x28: -> 0x08 "of"
				// 0x2c: -> 0x0c "strings"

				opcode.Label{Value: 0},

				// String struct laid out at 0x14.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Resource{ID: 0},

				// Update the pointer addresses in pointer struct.
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x1c},
				opcode.StoreV{Address: 0x18},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.StoreV{Address: 0x1c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 1},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x20},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 2},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.StoreV{Address: 0x24},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x00},
				opcode.Resource{ID: 3},

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.StoreV{Address: 0x28},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x08},
				opcode.Resource{ID: 2}, // TODO: redundant resource request

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.StoreV{Address: 0x2c},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x0c},
				opcode.Resource{ID: 1}, // TODO: redundant resource request

				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadStringStruct.ApiIndex,
					FunctionID: funcInfoCmdVoidReadStringStruct.ID},
			},
		},
	}.check(ctx, ml, ml)
}

func TestOperationsOpCall_ReadAndConditionalWrite(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	cb := CommandBuilder{Thread: 0}
	ml := device.Little32
	rRng, rID := memory.Store(ctx, ml, p(0x100000), uint32(3))                  // read for all cases
	awcRng, awcID := memory.Store(ctx, ml, p(0x100000), uint32(2))              // write to count for Case 1
	bwcRng, bwcID := memory.Store(ctx, ml, p(0x100000), uint32(3))              // write to count for Case 2
	bwhRng, bwhID := memory.Store(ctx, ml, p(0x200000), []Remapped{10, 20, 30}) // write to handles for Case 2
	cwcRng, cwcID := memory.Store(ctx, ml, p(0x100000), uint32(1))              // write to count for Case 3
	cwhRng, cwhID := memory.Store(ctx, ml, p(0x300000), []Remapped{40})         // write to handles for Case 3

	test{
		cmds: []api.Cmd{
			// Case 1: only update the count. So we pass null to pHandles.
			cb.CmdVoidReadAndConditionalWrite(p(0x100000), p(0x0)).
				AddRead(rRng, rID).
				AddWrite(awcRng, awcID),
			// Case 2: update handles up to the count provided. pHandles is at 0x200000.
			cb.CmdVoidReadAndConditionalWrite(p(0x100000), p(0x200000)).
				AddRead(rRng, rID).
				AddWrite(bwcRng, bwcID).
				AddWrite(bwhRng, bwhID),
			// Case 3: update handles less than the count provided. pHandles is at 0x300000.
			cb.CmdVoidReadAndConditionalWrite(p(0x100000), p(0x300000)).
				AddRead(rRng, rID).
				AddWrite(cwcRng, cwcID).
				AddWrite(cwhRng, cwhID),
			// Let's use some of the handles generated.
			cb.CmdVoid3Remapped(10, 20, 40),
		},
		expected: expected{
			resources: []id.ID{rID},
			opcodes: []interface{}{

				// 0x00:  remapped pHandles[0] for Case 2
				// 0x04:  remapped pHandles[1] for Case 2
				// 0x08:  remapped pHandles[2] for Case 2

				// 0x0c:  remapped pHandles[0] for Case 3

				// 0x10:  uint32(3)

				// 0x14:  pHandles[0] for Case 2 -> 0x00
				// 0x18:  pHandles[1] for Case 2 -> 0x04
				// 0x1c:  pHandles[2] for Case 2 -> 0x08

				// 0x20:  pHandles[0] for Case 3 -> 0x0c

				// Case 1
				opcode.Label{Value: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 0},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_AbsolutePointer, Value: 0x00},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadAndConditionalWrite.ApiIndex,
					FunctionID: funcInfoCmdVoidReadAndConditionalWrite.ID},

				// Case 2
				opcode.Label{Value: 1},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 0}, // TODO: redundant resource request
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x14},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadAndConditionalWrite.ApiIndex,
					FunctionID: funcInfoCmdVoidReadAndConditionalWrite.ID},
				// Update the remap table.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x14},
				opcode.StoreV{Address: 0x0},
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x18},
				opcode.StoreV{Address: 0x4},
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x1c},
				opcode.StoreV{Address: 0x8},

				// Case 3
				opcode.Label{Value: 2},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.Resource{ID: 0}, // TODO: redundant resource request
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x10},
				opcode.PushI{DataType: protocol.Type_VolatilePointer, Value: 0x20},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoidReadAndConditionalWrite.ApiIndex,
					FunctionID: funcInfoCmdVoidReadAndConditionalWrite.ID},
				// Update the remap table.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x20},
				opcode.StoreV{Address: 0xc},

				opcode.Label{Value: 3},
				// Subsequently-seen values use the remapped value.
				// We are getting the remapped pHandles[0] & pHandles[1] from Case 2,
				// and pHandles[0] from Case 3.
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x0},
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0x4},
				opcode.LoadV{DataType: protocol.Type_Uint32, Address: 0xc},
				opcode.Call{
					ApiIndex:   funcInfoCmdVoid3Remapped.ApiIndex,
					FunctionID: funcInfoCmdVoid3Remapped.ID},
			},
		},
	}.check(ctx, ml, ml)
}
