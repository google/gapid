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
	"context"
	"fmt"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/gapis/replay/opcode"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

const (
	// Various bit-masks used by this function.
	// Many opcodes can fit values into the opcode itself.
	// These masks are used to determine which values fit.

	mask19 = uint64(0x7ffff)
	mask20 = uint64(0xfffff)
	mask26 = uint64(0x3ffffff)
	mask45 = uint64(0x1fffffffffff)
	mask46 = uint64(0x3fffffffffff)
	mask52 = uint64(0xfffffffffffff)

	//     ▏60       ▏50       ▏40       ▏30       ▏20       ▏10
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●● mask19
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●● mask20
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●● mask26
	// ○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask45
	// ○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask46
	// ○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask52
	//                                            ▕      PUSHI 20     ▕
	//                                      ▕         EXTEND 26       ▕
)

// Instruction is the interface of all instruction types.
//
// Encode writes the instruction's opcodes to the binary writer w, translating
// all pointers to their final, resolved addresses using the PointerResolver r.
// An instruction can produce zero, one or many opcodes.
type Instruction interface {
	Encode(r value.PointerResolver, w binary.Writer) error
}

func encodePush(t protocol.Type, v uint64, w binary.Writer) error {
	switch t {
	case protocol.Type_Float:
		push := opcode.PushI{DataType: t, Value: uint32(v >> 23)}
		if err := push.Encode(w); err != nil {
			return err
		}
		if v&0x7fffff != 0 {
			return opcode.Extend{Value: uint32(v & 0x7fffff)}.Encode(w)
		}
		return nil
	case protocol.Type_Double:
		push := opcode.PushI{DataType: t, Value: uint32(v >> 52)}
		if err := push.Encode(w); err != nil {
			return err
		}
		v &= mask52
		if v != 0 {
			ext := opcode.Extend{Value: uint32(v >> 26)}
			if err := ext.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		}
		return nil
	case protocol.Type_Int8, protocol.Type_Int16, protocol.Type_Int32, protocol.Type_Int64:
		// Signed PUSHI types are sign-extended
		switch {
		case v&^mask19 == 0:
			// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                                            ▕      PUSHI 20     ▕
			return opcode.PushI{DataType: t, Value: uint32(v)}.Encode(w)
		case v&^mask19 == ^mask19:
			// ●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                                            ▕      PUSHI 20     ▕
			return opcode.PushI{DataType: t, Value: uint32(v & mask20)}.Encode(w)
		case v&^mask45 == 0:
			// ○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
			push := opcode.PushI{DataType: t, Value: uint32(v >> 26)}
			if err := push.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		case v&^mask45 == ^mask45:
			// ●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
			push := opcode.PushI{DataType: t, Value: uint32((v >> 26) & mask20)}
			if err := push.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		default:
			// ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
			push := opcode.PushI{DataType: t, Value: uint32(v >> 52)}
			if err := push.Encode(w); err != nil {
				return err
			}
			ext := opcode.Extend{Value: uint32((v >> 26) & mask26)}
			if err := ext.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		}
	case protocol.Type_Bool,
		protocol.Type_Uint8, protocol.Type_Uint16, protocol.Type_Uint32, protocol.Type_Uint64,
		protocol.Type_AbsolutePointer, protocol.Type_ConstantPointer, protocol.Type_VolatilePointer:
		switch {
		case v&^mask20 == 0:
			// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                                            ▕      PUSHI 20     ▕
			return opcode.PushI{DataType: t, Value: uint32(v)}.Encode(w)
		case v&^mask46 == 0:
			// ○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
			push := opcode.PushI{DataType: t, Value: uint32(v >> 26)}
			if err := push.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		default:
			// ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
			//▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
			push := opcode.PushI{DataType: t, Value: uint32(v >> 52)}
			if err := push.Encode(w); err != nil {
				return err
			}
			ext := opcode.Extend{Value: uint32((v >> 26) & mask26)}
			if err := ext.Encode(w); err != nil {
				return err
			}
			return opcode.Extend{Value: uint32(v & mask26)}.Encode(w)
		}
	}
	return fmt.Errorf("Cannot push value type %s", t)
}

// Nop is a no-operation Instruction. Instructions of this type do nothing.
type Nop struct{}

func (Nop) Encode(r value.PointerResolver, w binary.Writer) error {
	return nil
}

// Call is an Instruction to call a VM registered function.
// This instruction will pop the parameters from the VM stack starting with the
// first parameter. If PushReturn is true, then the return value of the function
// call will be pushed to the top of the VM stack.
type Call struct {
	PushReturn bool   // If true, the return value is pushed to the VM stack.
	ApiIndex   uint8  // The index of the API this call belongs to
	FunctionID uint16 // The function id registered with the VM to invoke.
}

func (a Call) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Call{
		PushReturn: a.PushReturn,
		ApiIndex:   a.ApiIndex,
		FunctionID: a.FunctionID,
	}.Encode(w)
}

// Push is an Instruction to push Value to the top of the VM stack.
type Push struct {
	Value value.Value // The value to push on to the VM stack.
}

func (a Push) Encode(r value.PointerResolver, w binary.Writer) error {
	if ty, val, onStack := a.Value.Get(r); onStack {
		return nil
	} else {
		return encodePush(ty, val, w)
	}
}

// Pop is an Instruction that discards Count values from the top of the VM
// stack.
type Pop struct {
	Count uint32 // Number of values to discard from the top of the VM stack.
}

func (a Pop) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Pop{Count: a.Count}.Encode(w)
}

// Copy is an Instruction that pops the target address and then the source
// address from the top of the VM stack, and then copies Count bytes from
// source to target.
type Copy struct {
	Count uint64 // Number of bytes to copy.
}

func (a Copy) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Copy{Count: uint32(a.Count)}.Encode(w)
}

// Clone is an Instruction that makes a copy of the the n-th element from the
// top of the VM stack and pushes the copy to the top of the VM stack.
type Clone struct {
	Index int
}

func (a Clone) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Clone{Index: uint32(a.Index)}.Encode(w)
}

// Load is an Instruction that loads the value of type DataType from pointer
// Source and pushes the loaded value to the top of the VM stack.
type Load struct {
	DataType protocol.Type
	Source   value.Pointer
}

func (a Load) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, addr, onStack := a.Source.Get(r)
	if !onStack {
		switch ty {
		case protocol.Type_ConstantPointer:
			if addr&^mask20 == 0 {
				return opcode.LoadC{DataType: a.DataType, Address: uint32(addr)}.Encode(w)
			}
		case protocol.Type_VolatilePointer:
			if addr&^mask20 == 0 {
				return opcode.LoadV{DataType: a.DataType, Address: uint32(addr)}.Encode(w)
			}
		default:
			return fmt.Errorf("Unsupported load source type %T", a.Source)
		}
		if err := encodePush(ty, addr, w); err != nil {
			return err
		}
	}
	return opcode.Load{DataType: a.DataType}.Encode(w)
}

// Store is an Instruction that pops the value from the top of the VM stack and
// writes the value to Destination.
type Store struct {
	Destination value.Pointer
}

func (a Store) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, addr, onStack := a.Destination.Get(r)
	if !onStack {
		if addr&^mask26 == 0 {
			return opcode.StoreV{Address: uint32(addr)}.Encode(w)
		} else {
			if err := encodePush(ty, addr, w); err != nil {
				return err
			}
		}
	}
	return opcode.Store{}.Encode(w)
}

// Strcpy is an Instruction that pops the target address then the source address
// from the top of the VM stack, and then copies at most MaxCount-1 bytes from
// source to target. If the MaxCount is greater than the source string length,
// then the target will be padded with 0s. The destination buffer will always be
// 0-terminated.
type Strcpy struct {
	MaxCount uint64
}

func (a Strcpy) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Strcpy{
		MaxSize: uint32(a.MaxCount),
	}.Encode(w)
}

// Resource is an Instruction that loads the resource with index Index of Size
// bytes and writes the resource to Destination.
type Resource struct {
	Index       uint32
	Destination value.Pointer
}

func (a Resource) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, val, onStack := a.Destination.Get(r)
	if !onStack {
		if err := encodePush(ty, val, w); err != nil {
			return err
		}
	}
	return opcode.Resource{
		ID: a.Index,
	}.Encode(w)
}

// InlineResource is an Instruction that loads the resource with index Index of Size
// bytes and writes the resource to Destination. Unlike the regular Resource instruction
// InlineResource packs the resource into the bytes following the initial 32 bit instruction
// In turn, this "inline" resource is followed by a pair of patch up tables. First some
// addresses to overwrite with constant values, then some pairs of addresses where the
// first address is an address to load from and the second is an address to store the loaded
// value.

type InlineResourceValuePatchUp struct {
	Destination value.Pointer
	Value       value.Value
}

type InlineResourcePointerPatchUp struct {
	Destination value.Pointer
	Source      value.Pointer
}

type InlineResource struct {
	Data            []byte
	Destination     value.Pointer
	ValuePatchUps   []InlineResourceValuePatchUp
	PointerPatchUps []InlineResourcePointerPatchUp
	Ctx             context.Context
}

func (a InlineResource) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, val, onStack := a.Destination.Get(r)
	if !onStack {
		if err := encodePush(ty, val, w); err != nil {
			return err
		}
	}

	valuePatchUps := make([]opcode.InlineResourceValuePatchUp, 0)
	pointerPatchUps := make([]opcode.InlineResourcePointerPatchUp, 0)

	for _, valuePatchUp := range a.ValuePatchUps {
		valuePatchUps = append(valuePatchUps, opcode.InlineResourceValuePatchUp{Destination: valuePatchUp.Destination, Value: valuePatchUp.Value})
	}

	for _, pointerPatchUp := range a.PointerPatchUps {
		pointerPatchUps = append(pointerPatchUps, opcode.InlineResourcePointerPatchUp{Destination: pointerPatchUp.Destination, Source: pointerPatchUp.Source})
	}

	dataInstructions := len(a.Data) / 4
	if len(a.Data)%4 != 0 {
		dataInstructions = dataInstructions + 1
	}

	data := make([]uint32, dataInstructions)

	for i := 0; i < dataInstructions; i++ {

		v0 := uint32(0)
		v1 := uint32(0)
		v2 := uint32(0)
		v3 := uint32(0)

		v0 = uint32(a.Data[i*4+0])

		if i*4+1 < len(a.Data) {
			v1 = uint32(a.Data[i*4+1]) * 256
		}

		if i*4+2 < len(a.Data) {
			v2 = uint32(a.Data[i*4+2]) * 256 * 256
		}

		if i*4+3 < len(a.Data) {
			v3 = uint32(a.Data[i*4+3]) * 256 * 256 * 256
		}

		data[i] = v0 + v1 + v2 + v3
	}

	return opcode.InlineResource{
		Data:            data,
		DataSize:        uint32(len(a.Data)),
		ValuePatchUps:   valuePatchUps,
		PointerPatchUps: pointerPatchUps,
		Resolver:        r,
		Ctx:             a.Ctx,
	}.Encode(w)
}

// Post is an Instruction that posts Size bytes from Source to the server.
type Post struct {
	Source value.Pointer
	Size   uint64
}

func (a Post) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, val, onStack := a.Source.Get(r)
	if !onStack {
		if err := encodePush(ty, val, w); err != nil {
			return err
		}
	}
	if err := encodePush(protocol.Type_Uint32, a.Size, w); err != nil {
		return err
	}
	return opcode.Post{}.Encode(w)
}

// Add is an Instruction that pops and sums the top N stack values, pushing the
// result to the top of the stack. Each summed value must have the same type.
type Add struct {
	Count uint32
}

func (a Add) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Add{Count: a.Count}.Encode(w)
}

// Label is an Instruction that holds a marker value, used for debugging.
type Label struct {
	Value uint32
}

func (a Label) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Label{Value: a.Value}.Encode(w)
}

// SwitchThread is an Instruction that changes execution to a different thread.
type SwitchThread struct {
	Index uint32
}

func (a SwitchThread) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.SwitchThread{Index: a.Index}.Encode(w)
}

type JumpLabel struct {
	Label uint32
}

func (a JumpLabel) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.JumpLabel{Label: a.Label}.Encode(w)
}

type JumpNZ struct {
	Label uint32
}

func (a JumpNZ) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.JumpNZ{Label: a.Label}.Encode(w)
}

type JumpZ struct {
	Label uint32
}

func (a JumpZ) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.JumpZ{Label: a.Label}.Encode(w)
}

// Notification is an Instruction that sends Size bytes from Source to the server, with the ID returned as well.
type Notification struct {
	ID     uint64
	Source value.Pointer
	Size   uint64
}

func (a Notification) Encode(r value.PointerResolver, w binary.Writer) error {
	ty, val, onStack := a.Source.Get(r)
	if !onStack {
		if err := encodePush(ty, val, w); err != nil {
			return err
		}
	}
	if err := encodePush(protocol.Type_Uint32, a.ID, w); err != nil {
		return err
	}
	if err := encodePush(protocol.Type_Uint32, a.Size, w); err != nil {
		return err
	}
	return opcode.Notification{}.Encode(w)
}

type Wait struct {
	ID uint32
}

func (a Wait) Encode(r value.PointerResolver, w binary.Writer) error {
	return opcode.Wait{ID: a.ID}.Encode(w)
}
