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

// Package builder contains the Builder type to build replay payloads.
package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/asm"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

// ErrReplayNotExecuted indicate the replay request was canceled.
var ErrReplayNotExecuted = errors.New("replay was canceled")

type stackItem struct {
	ty  protocol.Type // Type of the item.
	idx int           // Index of the op that generated this.
}

type marker struct {
	instruction int    // first instruction index for this marker
	cmd         uint64 // the command identifier
}

// NotificationReader reads a given copy of notification message received from
// the replay virtual machine.
type NotificationReader func(p gapir.Notification)

// NotificationHandler handles the original Notification messages from the
// replay virtual machine.
type NotificationHandler func(p *gapir.Notification)

type FenceReadyRequestCallback func(p *gapir.FenceReadyRequest)

// PostDataHandler handles the original PostData messages, which may contains
// multiple pieces of post back data issued by multiple POST instructions, from
// the replay virual machine.
type PostDataHandler func(p *gapir.PostData)

type postBackDecoder struct {
	expectedSize int
	decode       Postback
}

const (
	// IssuesNotificationID is the Notification ID reserved for issues report.
	// The ID needs to be kept in sync with |kIssuesNotificationId| defined in
	// `gapir/cc/grpc_replay_service.cpp`.
	IssuesNotificationID = uint64(0)
	// ReplayProgressNotificationID is the Notification ID reserved for replay
	// status information transfer.
	// The ID needs to be kept in sync with |kReplayProgressNotificationID|
	// defined in `gapir/cc/grpc_replay_service.cpp`.
	ReplayProgressNotificationID = uint64(1)
	// The value used to initialize nextNotificationID. Need to be increased as
	// well when more notification id are reserved for special use.
	InitialNextNotificationID = uint64(2)
)

// Postback decodes a single command's post. If err is nil, it means there is
// no error in prior check of the post data, otherwise err will hold the error
// from the prior check (like a length mismatch error).
type Postback func(d binary.Reader, err error)

// Builder is used to build the Payload to send to the replay virtual machine.
// The builder has a number of methods for mutating the virtual machine stack,
// invoking functions and posting back data.
type Builder struct {
	constantMemory      *constantEncoder
	heap, temp          allocator
	resourceIDToIdx     map[id.ID]uint32
	resourceIdxToID     map[uint32]id.ID
	threadIDToIdx       map[uint64]uint32
	currentThreadID     uint64
	pendingThreadID     uint64
	resources           []*gapir.ResourceInfo
	reservedMemory      memory.RangeList // Reserved memory ranges for regular data.
	pointerMemory       memory.RangeList // Reserved memory ranges for the pointer table.
	mappedMemory        MappedMemoryRangeList
	instructions        []asm.Instruction
	decoders            []postBackDecoder
	nextNotificationID  uint64
	notificationReaders map[uint64]NotificationReader
	fenceReadyCallbacks map[uint32]FenceReadyRequestCallback
	stack               []stackItem
	memoryLayout        *device.MemoryLayout
	inCmd               bool   // true if between BeginCommand and CommitCommand/RevertCommand
	cmdStart            int    // index of current commands's first instruction
	pendingLabel        uint64 // label passed to BeginCommand written
	lastLabel           uint64 // label of last CommitCommand written
	volatileSpace       uint64 // Amount of volatile space already used

	// Remappings is a map of a arbitrary keys to pointers. Typically, this is
	// used as a map of observed values to values that are only known at replay
	// execution time, such as driver generated handles.
	// The Remappings field is not accessed by the Builder and can be used in any
	// way the developer requires.
	Remappings map[interface{}]value.Pointer
}

// New returns a newly constructed Builder configured to replay on a target
// with the specified MemoryLayout.
func New(memoryLayout *device.MemoryLayout, dependent *Builder) *Builder {
	ptrAlignment := uint64(memoryLayout.GetPointer().GetAlignment())
	var dependentMemory uint64
	mappedMemory := MappedMemoryRangeList{}
	remappings := make(map[interface{}]value.Pointer)
	volatileSpace := uint64(0)
	if dependent != nil {
		dependentMemory = dependent.volatileSpace
		volatileSpace = dependent.volatileSpace
		mappedMemory = append(mappedMemory, dependent.mappedMemory...)
		for k, v := range dependent.Remappings {
			remappings[k] = v
		}
	}

	return &Builder{
		constantMemory: newConstantEncoder(memoryLayout),
		heap: allocator{
			alignment: ptrAlignment,
			size:      dependentMemory,
			head:      dependentMemory,
		},
		temp:                allocator{alignment: ptrAlignment},
		resourceIDToIdx:     map[id.ID]uint32{},
		resourceIdxToID:     map[uint32]id.ID{},
		threadIDToIdx:       map[uint64]uint32{},
		resources:           []*gapir.ResourceInfo{},
		reservedMemory:      memory.RangeList{},
		pointerMemory:       memory.RangeList{},
		mappedMemory:        mappedMemory,
		instructions:        []asm.Instruction{},
		nextNotificationID:  InitialNextNotificationID,
		notificationReaders: map[uint64]NotificationReader{},
		fenceReadyCallbacks: map[uint32]FenceReadyRequestCallback{},
		memoryLayout:        memoryLayout,
		lastLabel:           ^uint64(0),
		volatileSpace:       volatileSpace,
		Remappings:          remappings,
	}
}

func (b *Builder) MemoryRanges() MappedMemoryRangeList {
	return b.mappedMemory
}

func (b *Builder) pushStack(t protocol.Type) {
	b.stack = append(b.stack, stackItem{t, len(b.instructions)})
}

func (b *Builder) popStack() {
	if len(b.stack) == 0 {
		panic("Stack underflow")
	}
	b.stack = b.stack[:len(b.stack)-1]
}

func (b *Builder) popStackMulti(count int) {
	if len(b.stack) < count {
		panic("Stack underflow")
	}
	b.stack = b.stack[:len(b.stack)-count]
}

func (b *Builder) peekStack() stackItem {
	if len(b.stack) == 0 {
		panic("Stack underflow")
	}
	return b.stack[len(b.stack)-1]
}

func (b *Builder) removeInstruction(at int) {
	if at == len(b.instructions)-1 {
		b.instructions = b.instructions[:at]
	} else {
		b.instructions[at] = asm.Nop{}
	}
}

func (b *Builder) remap(ptr value.Pointer) value.Pointer {
	p, ok := ptr.(value.ObservedPointer)
	if !ok {
		return ptr
	}

	idx := interval.IndexOf(&b.mappedMemory, uint64(p))
	if idx < 0 {
		return ptr
	}

	m := b.mappedMemory[idx]
	b.instructions = append(b.instructions,
		// load target address
		asm.Load{DataType: protocol.Type_AbsolutePointer, Source: m.Target},
		// push relative offset from target address
		asm.Push{Value: value.AbsolutePointer(uint64(p) - m.Base)},
		// apply offset
		asm.Add{Count: 2},
	)

	return value.AbsoluteStackPointer{}
}

// MemoryLayout returns the memory layout for the target replay device.
func (b *Builder) MemoryLayout() *device.MemoryLayout {
	return b.memoryLayout
}

// AllocateMemory allocates and returns a pointer to a block of memory in the
// volatile address-space big enough to hold size bytes. The memory will be
// allocated for the entire replay duration and cannot be freed.
func (b *Builder) AllocateMemory(size uint64) value.Pointer {
	return value.VolatilePointer(b.heap.alloc(size))
}

// AllocateTemporaryMemory allocates and returns a pointer to a block of memory
// in the temporary volatile address-space big enough to hold size bytes. The
// memory block will be freed on the next call to CommitCommand/AbortCommand,
// upon which reading or writing to this memory will result in undefined
// behavior.
// TODO: REMOVE
func (b *Builder) AllocateTemporaryMemory(size uint64) value.Pointer {
	return value.TemporaryPointer(b.temp.alloc(size))
}

// BeginCommand should be called before building any replay instructions.
func (b *Builder) BeginCommand(cmdID, threadID uint64) {
	if b.inCmd {
		panic("BeginCommand called while already building a command")
	}
	b.inCmd = true
	b.cmdStart = len(b.instructions)

	cmdID &= 0x3ffffff // Labels have 26 bit values.
	if b.lastLabel != cmdID {
		b.instructions = append(b.instructions, asm.Label{Value: uint32(cmdID)})
		b.pendingLabel = cmdID
	}

	if b.currentThreadID != threadID {
		b.pendingThreadID = threadID
		index, ok := b.threadIDToIdx[threadID]
		if !ok {
			index = uint32(len(b.threadIDToIdx)) + 1
			b.threadIDToIdx[threadID] = index
		}
		b.instructions = append(b.instructions, asm.SwitchThread{Index: index})
	}
}

type regionCalculatingVolatileMemoryLayout struct {
	reservedMemoryAsList interval.List // Alias of reservedMemory to minimize interface conversions.
	pointerMemoryAsList  interval.List // Alias of pointerMemory to minimize interface conversions.
	memoryLayout         *device.MemoryLayout
}

func (l regionCalculatingVolatileMemoryLayout) ResolveObservedPointer(p value.ObservedPointer) (protocol.Type, uint64) {
	bufferIdx := interval.IndexOf(l.reservedMemoryAsList, uint64(p))
	if bufferIdx < 0 {
		return protocol.Type_AbsolutePointer, unobservedPointer
	}
	return protocol.Type_VolatilePointer, unobservedPointer
}

func getRegionCalculatingVolatileMemoryLayout(b Builder) regionCalculatingVolatileMemoryLayout {
	return regionCalculatingVolatileMemoryLayout{
		reservedMemoryAsList: &b.reservedMemory,
		pointerMemoryAsList:  &b.pointerMemory,
		memoryLayout:         device.Little32,
	}
}

func (l regionCalculatingVolatileMemoryLayout) ResolveTemporaryPointer(p value.TemporaryPointer) value.VolatilePointer {
	return value.VolatilePointer(uint64(p))
}

func (l regionCalculatingVolatileMemoryLayout) ResolvePointerIndex(i value.PointerIndex) (protocol.Type, uint64) {
	addr := uint64(i) * uint64(l.memoryLayout.GetPointer().GetSize())
	bufferIdx := interval.IndexOf(l.pointerMemoryAsList, addr)
	if bufferIdx < 0 {
		// Pointer is not observed.
		return protocol.Type_AbsolutePointer, unobservedPointer
	}
	return protocol.Type_VolatilePointer, unobservedPointer
}

// CommitCommand should be called after emitting the commands to replay a single
// command.
// CommitCommand frees all temporary allocated memory and clears the stack.
func (b *Builder) CommitCommand(ctx context.Context, optimise bool) {

	if !b.inCmd {
		panic("CommitCommand called without a call to BeginCommand")
	}

	newInstructions := make([]asm.Instruction, 0)

	b.lastLabel, b.pendingLabel = b.pendingLabel, 0
	b.currentThreadID = b.pendingThreadID
	b.inCmd = false
	b.temp.reset()
	pop := uint32(len(b.stack))
	// Optimise the instructions.
	for si := len(b.stack) - 1; si >= 0; si-- {
		s := b.stack[si]
		switch i := b.instructions[s.idx].(type) {
		case asm.Call: // Change calls that push an unused return value to discard the value.
			if i.PushReturn {
				i.PushReturn = false
				b.instructions[s.idx] = i
				pop--
			}
		case asm.Clone, asm.Push, asm.Load: // Remove unused clones, pushes, loads
			b.instructions[s.idx] = asm.Nop{}
			pop--
		}
	}
	// Trim trailing no-ops
	for len(b.instructions) > 0 {
		if _, nop := b.instructions[len(b.instructions)-1].(asm.Nop); nop {
			b.instructions = b.instructions[:len(b.instructions)-1]
		} else {
			break
		}
	}
	// Pop any remaining stack values
	if pop > 0 {
		b.instructions = append(b.instructions, asm.Pop{Count: pop})
	}

	// Lets inine any small resources next...
	if optimise {
		inlineThreshold := 128
		for index := b.cmdStart; index < len(b.instructions); index++ {

			if resource, ok := b.instructions[index].(asm.Resource); ok {

				id := b.resourceIdxToID[resource.Index]

				obj, err := database.Resolve(ctx, id)
				if err != nil {
					panic(log.Err(ctx, ErrInvalidResource, "Couldn't resolve inline resource"))
				}
				data, ok := obj.([]byte)
				if !ok {
					panic(log.Err(ctx, ErrInvalidResource, "Inline resource didn't resolve to byte slice"))
				}

				if len(data) <= inlineThreshold {

					valuePatchUps := make([]asm.InlineResourceValuePatchUp, 0)
					pointerPatchUps := make([]asm.InlineResourcePointerPatchUp, 0)

					for donePatchups := false; donePatchups == false; {
						if index+2 < len(b.instructions) {
							push, pushOk := b.instructions[index+1].(asm.Push)
							load, loadOk := b.instructions[index+1].(asm.Load)
							store, storeOk := b.instructions[index+2].(asm.Store)
							if storeOk {
								if pushOk {
									resolver := getRegionCalculatingVolatileMemoryLayout(*b)
									ty, _, onStack := push.Value.Get(resolver)
									// Because the number of valuePatchUps is stored in the instruction bits, we need to stay before 64 of them here.
									if !onStack && ty == protocol.Type_VolatilePointer && len(valuePatchUps) < 64 {
										valuePatchUps = append(valuePatchUps, asm.InlineResourceValuePatchUp{Destination: store.Destination, Value: push.Value})
										index += 2
										continue
									}
								}

								if loadOk {
									resolver := getRegionCalculatingVolatileMemoryLayout(*b)
									ty1, _, onStack := load.Source.Get(resolver)
									if !onStack {
										ty2, _, onStack := store.Destination.Get(resolver)
										if !onStack && ty1 == protocol.Type_VolatilePointer && ty2 == protocol.Type_VolatilePointer {
											pointerPatchUps = append(pointerPatchUps, asm.InlineResourcePointerPatchUp{Destination: store.Destination, Source: load.Source})
											index += 2
											continue
										}
									}
								}
							}
						}

						donePatchups = true
					}

					inlineResource := asm.InlineResource{Ctx: ctx, Data: data, Destination: resource.Destination, ValuePatchUps: valuePatchUps, PointerPatchUps: pointerPatchUps}
					newInstructions = append(newInstructions, inlineResource)
				} else {
					newInstructions = append(newInstructions, b.instructions[index])
				}
			} else {
				newInstructions = append(newInstructions, b.instructions[index])
			}
		}

		b.instructions = append(b.instructions[:b.cmdStart], newInstructions...)
	}

	b.stack = b.stack[:0]
}

// RevertCommand reverts all the instructions since the last call to
// BeginCommand. Any postbacks issued since the last call to BeginCommand will
// be called with the error err and a nil decoder.
func (b *Builder) RevertCommand(err error) {
	if !b.inCmd {
		panic("RevertCommand called without a call to BeginCommand")
	}
	b.pendingLabel = 0
	b.pendingThreadID = b.currentThreadID
	b.inCmd = false
	// TODO: Revert calls to: AllocateMemory, Buffer, String, ReserveMemory, MapMemory, UnmapMemory, Write.
	b.temp.reset()
	b.stack = b.stack[:0]
	if len(b.instructions) > 0 {
		for i := len(b.instructions) - 1; i >= b.cmdStart; i-- {
			switch b.instructions[i].(type) {
			case asm.Post:
				idx := len(b.decoders) - 1
				b.decoders[idx].decode(nil, err)
				b.decoders = b.decoders[:idx]
			}
		}
		b.instructions = b.instructions[:b.cmdStart]
	}
}

// Buffer returns a pointer to a block of memory in holding the count number of
// previously pushed values.
// If all the values are constant, then the buffer will be held in the constant
// address-space, otherwise the buffer will be built in the temporary
// address-space.
func (b *Builder) Buffer(count int) value.Pointer {
	pointerSize := b.memoryLayout.GetPointer().GetSize()
	dynamic := false
	size := 0

	// Examine the stack to see where these values came from
	for i := 0; i < count; i++ {
		e := b.stack[len(b.stack)-i-1]
		op := b.instructions[e.idx]
		ty := e.ty
		if _, isPush := op.(asm.Push); !isPush {
			// Values that have made their way on to the stack from non-constant
			// sources cannot be put in the constant buffer.
			dynamic = true
		}
		switch ty {
		case protocol.Type_ConstantPointer, protocol.Type_VolatilePointer:
			// Pointers cannot be put into the constant buffer as they are remapped
			// by the VM
			dynamic = true
		}

		size += ty.Size(pointerSize)
	}

	if dynamic {
		// At least one of the values was not from a Push()
		// Build the buffer in temporary memory at replay time.
		buf := b.AllocateTemporaryMemory(uint64(size))
		offset := size
		for i := 0; i < count; i++ {
			e := b.stack[len(b.stack)-1]
			offset -= e.ty.Size(pointerSize)
			b.Store(buf.Offset(uint64(offset)))
		}
		return buf
	}
	// All the values are constant.
	// Move the pushed values into a constant memory buffer.
	values := make([]value.Value, count)
	for i := 0; i < count; i++ {
		e := b.stack[len(b.stack)-1]
		values[count-i-1] = b.instructions[e.idx].(asm.Push).Value
		b.removeInstruction(e.idx)
		b.popStack()
	}
	return b.constantMemory.writeValues(values...)
}

// String returns a pointer to a block of memory in the constant address-space
// holding the string s. The string will be stored with a null-terminating byte.
func (b *Builder) String(s string) value.Pointer {
	return b.constantMemory.writeString(s)
}

// Call will invoke the function f, popping all parameter values previously
// pushed to the stack with Push, starting with the first parameter. If f has
// a non-void return type, after invoking the function the return value of the
// function will be pushed on to the stack.
func (b *Builder) Call(f FunctionInfo) {
	b.popStackMulti(f.Parameters)
	push := f.ReturnType != protocol.Type_Void
	if push {
		b.pushStack(f.ReturnType)
	}
	b.instructions = append(b.instructions, asm.Call{
		PushReturn: push,
		ApiIndex:   f.ApiIndex,
		FunctionID: f.ID,
	})
}

// Copy pops the target address and then the source address from the top of the
// stack, and then copies Count bytes from source to target.
func (b *Builder) Copy(size uint64) {
	b.popStackMulti(2)
	b.instructions = append(b.instructions, asm.Copy{
		Count: size,
	})
}

// Clone makes a copy of the n-th element from the top of the stack and pushes
// the copy to the top of the stack.
func (b *Builder) Clone(index int) {
	sidx := len(b.stack) - 1 - index
	// Change ownership of the top stack value to the clone instruction.
	b.stack[sidx].idx = len(b.instructions)
	b.pushStack(b.stack[sidx].ty)
	b.instructions = append(b.instructions, asm.Clone{
		Index: index,
	})
}

// Load loads the value of type ty from addr and then pushes the loaded value to
// the top of the stack.
func (b *Builder) Load(ty protocol.Type, addr value.Pointer) {
	if !addr.IsValid() {
		panic(fmt.Errorf("Pointer address %v is not valid", addr))
	}
	b.pushStack(ty)
	b.instructions = append(b.instructions, asm.Load{
		DataType: ty,
		Source:   b.remap(addr),
	})
}

// Store pops the value from the top of the stack and writes the value to addr.
func (b *Builder) Store(addr value.Pointer) {
	if !addr.IsValid() {
		panic(fmt.Errorf("Pointer address %v is not valid", addr))
	}
	b.popStack()
	b.instructions = append(b.instructions, asm.Store{
		Destination: b.remap(addr),
	})
}

// StorePointer writes ptr to the target pointer index.
// Pointers are stored in a separate address space and can only be loaded using
// PointerIndex values.
func (b *Builder) StorePointer(idx value.PointerIndex, ptr value.Pointer) {
	b.instructions = append(b.instructions,
		asm.Push{Value: ptr},
		asm.Store{Destination: idx},
	)
	rng := memory.Range{
		Base: uint64(idx) * uint64(b.memoryLayout.GetPointer().GetSize()),
		Size: uint64(b.memoryLayout.GetPointer().GetSize()),
	}
	interval.Merge(&b.pointerMemory, rng.Span(), true)
}

// Strcpy pops the source address then the target address from the top of the
// stack, and then copies at most maxCount-1 bytes from source to target. If
// maxCount is greater than the source string length, then the target will be
// padded with 0s. The destination buffer will always be 0-terminated.
func (b *Builder) Strcpy(maxCount uint64) {
	b.popStackMulti(2)
	b.instructions = append(b.instructions, asm.Strcpy{
		MaxCount: maxCount,
	})
}

// Post posts size bytes from addr to the decoder d. The decoder d must consume
// all size bytes before returning; failure to do this will corrupt all
// subsequent postbacks.
func (b *Builder) Post(addr value.Pointer, size uint64, p Postback) {
	if !addr.IsValid() {
		panic(fmt.Errorf("Pointer address %v is not valid", addr))
	}
	b.instructions = append(b.instructions, asm.Post{
		Source: b.remap(addr),
		Size:   size,
	})
	b.decoders = append(b.decoders, postBackDecoder{
		expectedSize: int(size),
		decode:       p,
	})
}

// Push pushes val to the top of the stack.
func (b *Builder) Push(val value.Value) {
	if p, ok := val.(value.Pointer); ok {
		val = b.remap(p)
	}

	// HACK: ObservedPointers will use the trivialVolatileMemoryLayout to
	// decide the protocol type of the pointer. This will always be
	// 'unobserved' and therefor a TypeAbsolutePointer instead of a
	// TypeVolatilePointer. Nothing really cares at the moment though.
	ty, _, onStack := val.Get(trivialVolatileMemoryLayout)
	b.pushStack(ty)
	if !onStack {
		b.instructions = append(b.instructions, asm.Push{
			Value: val,
		})
	}
}

// Sub pops the value from top of stack, substracts by num and pushes back
// the result to the top of the stack.
func (b *Builder) Sub(num int32) {
	b.instructions = append(b.instructions, asm.Push{
		Value: value.S32(-num),
	}, asm.Add{
		Count: 2,
	})
	sidx := len(b.stack) - 1
	// Change ownership of the top stack value to the add instruction.
	b.stack[sidx].idx = len(b.instructions)
}

// JumpLabel adds a jump label to the instructions so that later can
// jump to that label and start execution from this label.
func (b *Builder) JumpLabel(label uint32) {
	b.instructions = append(b.instructions, asm.JumpLabel{
		Label: label,
	})
}

// JumpNZ jumps to the instruction specified label if the value
// on top of the stack is not zero. Otherwise it will be be a Nop.
func (b *Builder) JumpNZ(label uint32) {
	b.popStack()
	b.instructions = append(b.instructions, asm.JumpNZ{
		Label: label,
	})
}

// JumpZ jumps to the instruction specified label if the value
// on top of the stack is zero. Otherwise it will be be a Nop.
func (b *Builder) JumpZ(label uint32) {
	b.popStack()
	b.instructions = append(b.instructions, asm.JumpZ{
		Label: label,
	})
}

// Notification asks the GAPIR to stream back the size bytes from addr. The |ID| will
// be sent back as well to help identify which reader will process the notification.
// A Notification reader must be registered to read the data the from the stream.
func (b *Builder) Notification(ID uint64, addr value.Pointer, size uint64) {
	if !addr.IsValid() {
		panic(fmt.Errorf("Pointer address %v is not valid", addr))
	}
	b.instructions = append(b.instructions, asm.Notification{
		ID:     ID,
		Source: b.remap(addr),
		Size:   size,
	})
}

func (b *Builder) Wait(ID uint32) {
	b.instructions = append(b.instructions, asm.Wait{
		ID: ID,
	})
}

// Pop removes the top count values from the top of the stack.
func (b *Builder) Pop(count uint32) {
	b.popStackMulti(int(count))
	b.instructions = append(b.instructions, asm.Pop{
		Count: count,
	})
}

// ReserveMemory adds rng as a memory range that needs allocating for replay.
func (b *Builder) ReserveMemory(rng memory.Range) {
	interval.Merge(&b.reservedMemory, rng.Span(), true)
}

// GetMappedTarget gets current mapped memory's target address pointer in the builder.
// The input |ptr| is the start of the mapped memory range. Returns the target address
// on success and error otherwise.
func (b *Builder) GetMappedTarget(ptr value.Pointer) (value.Pointer, error) {
	p, ok := ptr.(value.ObservedPointer)
	if !ok {
		return ptr, fmt.Errorf("ptr %v is not type of value.ObservedPointer", ptr)
	}

	idx := interval.IndexOf(&b.mappedMemory, uint64(p))
	if idx < 0 {
		return ptr, fmt.Errorf("can not find ptr %v in mappedMemory", ptr)
	}

	if b.mappedMemory[idx].Range.Base != uint64(p) {
		return ptr, fmt.Errorf("the ptr %v is not the start of the mapped memory range", ptr)
	}

	return b.mappedMemory[idx].Target, nil
}

// MapMemory maps the memory range rng relative to the absolute pointer that is
// on the top of the stack. Any ObservedPointers that are used while the pointer
// is mapped will be automatically adjusted to the remapped address.
// The mapped memory range can be unmapped with a call to UnmapMemory.
func (b *Builder) MapMemory(rng memory.Range) {
	if ty := b.peekStack().ty; ty != protocol.Type_AbsolutePointer {
		panic(fmt.Errorf("MapMemory can only map to absolute pointers. Got type: %v", ty))
	}

	// Allocate memory to hold the target mapped base address.
	// Do not release the memory as it may be used during frame loop.
	target := b.AllocateMemory(uint64(b.memoryLayout.GetPointer().GetSize()))
	b.Store(target)

	s := rng.Span()
	i := interval.Merge(&b.mappedMemory, s, false)
	if b.mappedMemory[i].Span() != s {
		panic(fmt.Errorf("MapMemory range (%v) collides with existing mapped range (%v)",
			rng, b.mappedMemory[i]))
	}

	b.mappedMemory[i].Target = target
}

// UnmapMemory unmaps the memory range rng that was previously mapped with a
// call to MapMemory. If the memory range is not exactly a range previously
// mapped with a call to MapMemory then this function panics.
func (b *Builder) UnmapMemory(rng memory.Range) {
	i := interval.IndexOf(&b.mappedMemory, rng.Base)
	if i < 0 {
		panic(fmt.Errorf("Range (%v) was not mapped", rng))
	}
	if b.mappedMemory[i].Span() != rng.Span() {
		panic(fmt.Errorf("Range passed to UnmapMemory (%v) is not exactly the same range passed to MapMemory (%v)",
			rng, b.mappedMemory[i]))
	}
	interval.Remove(&b.mappedMemory, rng.Span())
}

// Write fills the memory range in capture address-space rng with the data
// of resourceID.
func (b *Builder) Write(rng memory.Range, resourceID id.ID) {
	if rng.Size > 0 {
		idx, found := b.resourceIDToIdx[resourceID]
		if !found {
			idx = uint32(len(b.resources))
			b.resourceIDToIdx[resourceID] = idx
			b.resourceIdxToID[idx] = resourceID
			b.resources = append(b.resources, &gapir.ResourceInfo{
				Id:   resourceID.String(),
				Size: uint32(rng.Size),
			})
		}
		b.instructions = append(b.instructions, asm.Resource{
			Index:       idx,
			Destination: b.remap(value.ObservedPointer(rng.Base)),
		})
	}
	b.ReserveMemory(rng)
}

// GetNotificationID returns the next available notification ID that identifies a notification.
func (b *Builder) GetNotificationID() uint64 {
	id := b.nextNotificationID
	b.nextNotificationID++
	return id
}

// RegisterNotificationReader registers a notification reader for a specific notificationID. Returns error
// if the notificationID has already been registered.
func (b *Builder) RegisterNotificationReader(notificationID uint64, reader NotificationReader) error {
	if _, ok := b.notificationReaders[notificationID]; ok {
		return fmt.Errorf("notificationID %d already registered", notificationID)
	}
	b.notificationReaders[notificationID] = reader
	return nil
}

// RegisterReplayStatusReader create and register a NotificationReader, which is used to decode and handle
// replay status information sent from GAPIR.
func (b *Builder) RegisterReplayStatusReader(ctx context.Context, r *status.Replay) error {
	reader := func(n gapir.Notification) {
		s := n.GetReplayStatus()
		label := s.GetLabel()
		totalInstrs := s.GetTotalInstrs()
		finishedInstrs := s.GetFinishedInstrs()

		log.D(ctx, "Replay status: Label: %v; Total instructions: %v; Finished percentage: %v.", label, totalInstrs, float32(finishedInstrs)/float32(totalInstrs))
		r.Progress(ctx, label, totalInstrs, finishedInstrs)
	}
	return b.RegisterNotificationReader(ReplayProgressNotificationID, reader)
}

func (b *Builder) RegisterFenceReadyRequestCallback(fenceID uint32, callback FenceReadyRequestCallback) error {
	if _, ok := b.fenceReadyCallbacks[fenceID]; ok {
		return fmt.Errorf("fenceID callback %d already registered", fenceID)
	}
	b.fenceReadyCallbacks[fenceID] = callback
	return nil
}

// Export compiles the replay instructions, returning a Payload that can be
// sent to the replay virtual-machine.
func (b *Builder) Export(ctx context.Context) (gapir.Payload, error) {
	ctx = status.Start(ctx, "Export")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Export")

	payload, _, _, _, err := b.Build(ctx)
	if err != nil {
		return payload, err
	}

	// Send phony postbacks.
	for _, decoder := range b.decoders {
		decoder.decode(nil, ErrReplayNotExecuted)
	}

	return payload, err
}

func (b *Builder) Memcpy(target value.Pointer, src value.Pointer, size uint64) {
	b.instructions = append(b.instructions,
		asm.Push{Value: b.remap(src)},
		asm.Push{Value: b.remap(target)},
		asm.Copy{Count: size},
	)
}

// Build compiles the replay instructions, returning a Payload that can be
// sent to the replay virtual-machine and a PostDataHandler for interpreting
// the responses.
func (b *Builder) Build(ctx context.Context) (gapir.Payload, PostDataHandler, NotificationHandler, FenceReadyRequestCallback, error) {
	ctx = status.Start(ctx, "Build")
	defer status.Finish(ctx)
	ctx = log.Enter(ctx, "Build")

	if config.DebugReplayBuilder {
		log.I(ctx, "Instruction count: %d", len(b.instructions))
		b.assertResourceSizesAreAsExpected(ctx)
	}

	byteOrder := b.memoryLayout.GetEndian()

	opcodes := &bytes.Buffer{}
	w := endian.Writer(opcodes, byteOrder)
	id := uint32(0)

	vml := b.layoutVolatileMemory(ctx, w)

	for index, i := range b.instructions {
		if (index%10000 == 9999) || (index == len(b.instructions)-1) {
			status.UpdateProgress(ctx, uint64(index), uint64(len(b.instructions)))
		}
		if label, ok := i.(asm.Label); ok {
			id = label.Value
		}
		if err := i.Encode(vml, w); err != nil {
			err = fmt.Errorf("Encode %T failed for command with id %v: %v", i, id, err)
			return gapir.Payload{}, nil, nil, nil, err
		}
	}

	payload := gapir.Payload{
		StackSize:          uint32(512), // TODO: Calculate stack size
		VolatileMemorySize: uint32(vml.size),
		Constants:          b.constantMemory.data,
		Resources:          b.resources,
		Opcodes:            opcodes.Bytes(),
	}
	b.volatileSpace += vml.size

	if config.DebugReplayBuilder {
		log.E(ctx, "----------------------------------")
		log.E(ctx, "Stack size:           0x%x", payload.StackSize)
		log.E(ctx, "Volatile memory size: 0x%x", payload.VolatileMemorySize)
		log.E(ctx, "Constant memory size: 0x%x", len(payload.Constants))
		log.E(ctx, "Opcodes size:         0x%x", len(payload.Opcodes))
		log.E(ctx, "Resource count:         %d", len(payload.Resources))
		log.E(ctx, "Decoder count:         %d", len(b.decoders))
		log.E(ctx, "Readers count:         %d", len(b.notificationReaders))
		log.E(ctx, "----------------------------------")
	}

	// Make a copy of the reference of the finished decoder list to cut off the
	// connection between the builder and furture uses of the decoders so that
	// the builder do not need to be kept alive when using these decoders.
	decoders := b.decoders
	handlePost := func(pd *gapir.PostData) {
		// TODO: should we skip it instead of return error?
		ctx = log.Enter(ctx, "PostDataHandler")
		if pd == nil {
			log.E(ctx, "Cannot handle nil PostData")
		}
		crash.Go(func() {
			for _, p := range pd.GetPostDataPieces() {
				id := p.GetID()
				data := p.GetData()
				if id >= uint64(len(decoders)) {
					log.E(ctx, "No valid decoder found for %v'th post data", id)
				}
				// Check that each Postback consumes its expected number of bytes.
				var err error
				if len(data) != decoders[id].expectedSize {
					err = fmt.Errorf("%d'th post size mismatch, actual size: %d, expected size: %d", id, len(data), decoders[id].expectedSize)
				}
				r := endian.Reader(bytes.NewReader(data), byteOrder)
				decoders[id].decode(r, err)
			}
		})
	}

	// Make a copy of the reference of the finished notification reader map to
	// cut off the connection between the builder and future uses of the readers
	// so that the builder do not need to be kept alive when using these readers.
	readers := b.notificationReaders
	handleNotification := func(n *gapir.Notification) {
		if n == nil {
			log.E(ctx, "Cannot handle nil Notification")
			return
		}
		crash.Go(func() {
			id := n.GetId()
			if r, ok := readers[id]; ok {
				r(*n)
			} else {
				log.W(ctx, "Unknown notification received, ID is %d: %v", id, n)
			}
		})
	}

	callbacks := b.fenceReadyCallbacks
	fenceReadyCallback := func(n *gapir.FenceReadyRequest) {
		if n == nil {
			log.E(ctx, "Cannot handle nil FenceReadyRequest")
			return
		}
		id := n.GetId()
		if r, ok := callbacks[id]; ok {
			r(n)
		} else {
			log.W(ctx, "Unknown fence ready received, ID is %d: %v", id, n)
		}
	}

	return payload, handlePost, handleNotification, fenceReadyCallback, nil
}

const ErrInvalidResource = fault.Const("Invaid resource")

func (b *Builder) assertResourceSizesAreAsExpected(ctx context.Context) {
	for _, r := range b.resources {
		ctx := log.V{"resource-id": r.Id}.Bind(ctx)
		id, err := id.Parse(r.Id)
		if err != nil {
			panic(log.Err(ctx, ErrInvalidResource, "Couldn't parse identifier"))
		}
		obj, err := database.Resolve(ctx, id)
		if err != nil {
			panic(log.Err(ctx, ErrInvalidResource, "Couldn't resolve"))
		}
		data, ok := obj.([]byte)
		if !ok {
			panic(log.Err(ctx, ErrInvalidResource, "Didn't resolve to byte slice"))
		}
		if len(data) != int(r.Size) {
			panic(log.Errf(ctx, ErrInvalidResource, "Resource size mismatch. expected: %v, got: %v", r.Size, len(data)))
		}
	}
}

func (b *Builder) layoutVolatileMemory(ctx context.Context, w binary.Writer) *volatileMemoryLayout {
	// Volatile memory layout:
	//
	//  low ┌──────────────────┐
	//      │                  │
	//      │                  │
	//      │                  │
	//      │                  │
	//      │     dependent    │
	//      │                  │
	//      │                  │
	//      │                  │
	//      ├──────────────────┤
	//      │       heap       │
	//      ├──────────────────┤
	//      │       temp       │
	//      ├──────────────────┤
	//      │ reserved range 0 │
	//      ├──────────────────┤
	//      ├──────────────────┤
	//      │ reserved range N │
	//      ├──────────────────┤
	//      │ pointer  range 0 │
	//      ├──────────────────┤
	//      ├──────────────────┤
	//      │ pointer  range N │
	// high └──────────────────┘

	alloc := allocator{alignment: b.heap.alignment}

	// Allocate heap.
	heapStart := alloc.head
	alloc.alloc(b.heap.size)
	heapEnd := alloc.head - 1

	// Allocate temporary memory.
	tempStart := alloc.head
	alloc.alloc(b.temp.size)
	tempEnd := alloc.head - 1

	// Allocate blocks for the reserved memory regions.
	reservedStart := alloc.head
	reservedBases := make([]uint64, len(b.reservedMemory))
	for i, m := range b.reservedMemory {
		reservedBases[i] = alloc.alloc(m.Size)
	}
	reservedEnd := alloc.head - 1

	// Allocate blocks for the pointer table.
	pointerStart := alloc.head
	pointerBases := make([]uint64, len(b.pointerMemory))
	for i, m := range b.pointerMemory {
		pointerBases[i] = alloc.alloc(m.Size)
	}
	pointerEnd := alloc.head - 1

	size := alloc.head
	vml := &volatileMemoryLayout{
		tempBase:             tempStart,
		reservedBases:        reservedBases,
		reservedMemory:       b.reservedMemory,
		reservedMemoryAsList: &b.reservedMemory,
		pointerBases:         pointerBases,
		pointerMemory:        b.pointerMemory,
		pointerMemoryAsList:  &b.pointerMemory,
		size:                 size,
		memoryLayout:         b.memoryLayout,
	}

	if config.DebugReplayBuilder {
		log.I(ctx, "Volatile memory layout: [0x%x, 0x%x]", 0, size-1)
		log.I(ctx, "  Heap:      [0x%x, 0x%x]", heapStart, heapEnd)
		log.I(ctx, "  Temporary: [0x%x, 0x%x]", tempStart, tempEnd)
		log.I(ctx, "  Reserved:  [0x%x, 0x%x]", reservedStart, reservedEnd)
		for _, m := range b.reservedMemory {
			log.I(ctx, "    Block:   %v", m)
		}
		log.I(ctx, "  Pointers:  [0x%x, 0x%x]", pointerStart, pointerEnd)
		for _, m := range b.pointerMemory {
			log.I(ctx, "    Block:   %v", m)
		}
	}

	return vml
}

type volatileMemoryLayout struct {
	tempBase             uint64           // Base address of the temp space.
	reservedBases        []uint64         // Base address for each entry in reservedMemory.
	reservedMemory       memory.RangeList // Reserved memory ranges.
	reservedMemoryAsList interval.List    // Alias of reservedMemory to minimize interface conversions.
	pointerBases         []uint64         // Base address for each entry in pointerMemory.
	pointerMemory        memory.RangeList // Reserved memory ranges for pointer table.
	pointerMemoryAsList  interval.List    // Alias of pointerMemory to minimize interface conversions.
	size                 uint64           // Total size of volatile memory.
	memoryLayout         *device.MemoryLayout
}

var trivialVolatileMemoryLayout value.PointerResolver = volatileMemoryLayout{
	reservedMemoryAsList: &memory.RangeList{},
	pointerMemoryAsList:  &memory.RangeList{},
	memoryLayout:         device.Little32,
}

// Pointer value used when an unrecognised pointer is encountered, that cannot
// be remapped to a sensible location. In these situations we pass a pointer
// that should cause an access violation if it is dereferenced. We opt to not
// use 0x00 as this is often overloaded to mean something else.
// Must match value used in cc/gapir/memory_manager.h
const unobservedPointer = 0xBADF00D

// ResolveTemporaryPointer implements the PointerResolver interface method in
// the replay/value package.
// TODO: REMOVE
func (l volatileMemoryLayout) ResolveTemporaryPointer(p value.TemporaryPointer) value.VolatilePointer {
	return value.VolatilePointer(l.tempBase + uint64(p))
}

// ResolveObservedPointer implements the PointerResolver interface method in
// the replay/value package.
func (l volatileMemoryLayout) ResolveObservedPointer(p value.ObservedPointer) (protocol.Type, uint64) {
	bufferIdx := interval.IndexOf(l.reservedMemoryAsList, uint64(p))
	if bufferIdx < 0 {
		// Pointer is not observed. However, this can be legal - for example
		// glVertexAttribPointer may have been passed a pointer that was never
		// observed.
		return protocol.Type_AbsolutePointer, unobservedPointer
	}
	bufferStart := l.reservedMemory[bufferIdx].First()
	pointer := l.reservedBases[bufferIdx] + uint64(p) - uint64(bufferStart)
	return protocol.Type_VolatilePointer, pointer
}

func (l volatileMemoryLayout) ResolvePointerIndex(i value.PointerIndex) (protocol.Type, uint64) {
	addr := uint64(i) * uint64(l.memoryLayout.GetPointer().GetSize())
	bufferIdx := interval.IndexOf(l.pointerMemoryAsList, addr)
	if bufferIdx < 0 {
		// Pointer is not observed.
		return protocol.Type_AbsolutePointer, unobservedPointer
	}
	bufferStart := l.pointerMemory[bufferIdx].First()
	pointer := l.pointerBases[bufferIdx] + addr - uint64(bufferStart)
	return protocol.Type_VolatilePointer, pointer
}
