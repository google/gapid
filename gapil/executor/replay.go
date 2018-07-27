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

package executor

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"reflect"
	"unsafe"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/compiler/plugins/replay"
	gapir "github.com/google/gapid/gapir/client"
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

// #include "gapil/runtime/cc/replay/asm.h"
// #include "gapil/runtime/cc/replay/replay.h"
//
// typedef gapil_replay_data* (TGetReplayData) (gapil_context*);
// gapil_replay_data* get_replay_data(TGetReplayData* func, gapil_context* ctx) { return func(ctx); }
import "C"

const replayPointerNamespace = 1

// ReplayBuilder returns a replay builder.
func (e *Env) ReplayBuilder(ctx context.Context) (builder.Builder, error) {
	pfn := e.Executor.Symbol(replay.GetReplayData)
	if pfn == nil {
		return nil, fmt.Errorf("Program did not export the function to get the replay opcodes")
	}

	gro := (*C.TGetReplayData)(pfn)
	c := (*C.gapil_context)(e.CContext())

	d := C.get_replay_data(gro, c)

	return &replayBuilder{
		ctx:          ctx,
		e:            e,
		d:            d,
		memoryLayout: e.Executor.cfg.ReplayABI.MemoryLayout,
		remappings:   map[interface{}]value.Pointer{},
		tmpFree:      allocationsBySize{},
		tmpAlloc:     allocationsBySize{},
	}, nil
}

type allocationsBySize map[uint64][]value.Pointer

type replayBuilder struct {
	ctx                 context.Context // HACK!
	e                   *Env
	d                   *C.gapil_replay_data
	memoryLayout        *device.MemoryLayout
	remappings          map[interface{}]value.Pointer
	decoders            []postbackDecoder
	notificationReaders []builder.NotificationReader
	tmpFree             allocationsBySize
	tmpAlloc            allocationsBySize
}

type postbackDecoder struct {
	expectedSize int
	decode       builder.Postback
}

func (b *replayBuilder) MemoryLayout() *device.MemoryLayout {
	panic("MemoryLayout not implemented")
}

func (b *replayBuilder) AllocateMemory(size uint64) value.Pointer {
	const alignment = 8
	ptr := C.gapil_replay_allocate_memory(b.e.cCtx, b.d, C.uint64_t(size), alignment)
	return value.VolatilePointer(ptr)
}

func (b *replayBuilder) AllocateTemporaryMemory(size uint64) value.Pointer {
	size = u64.NextPOT(size)
	allocs := b.tmpFree[size]
	var alloc value.Pointer
	if len(allocs) > 0 {
		alloc = allocs[len(allocs)-1]
		b.tmpFree[size] = allocs[0 : len(allocs)-1]
	} else {
		alloc = b.AllocateMemory(size)
	}
	b.tmpAlloc[size] = append(b.tmpAlloc[size], alloc)
	return alloc
}

func (b *replayBuilder) freeTmpAllocs() {
	for size, allocs := range b.tmpAlloc {
		b.tmpFree[size] = append(b.tmpFree[size], allocs...)
	}
	b.tmpAlloc = allocationsBySize{}
}

func (b *replayBuilder) BeginCommand(cmdID, threadID uint64) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_BEGIN_COMMAND,
		&C.gapil_replay_asm_begincommand{C.uint64_t(cmdID)})
}

func (b *replayBuilder) CommitCommand() { b.freeTmpAllocs() }

func (b *replayBuilder) RevertCommand(err error) {
	panic("RevertCommand not implemented")
	b.freeTmpAllocs()
}

func (b *replayBuilder) Buffer(count int) value.Pointer {
	panic("Buffer not implemented")
}

func (b *replayBuilder) String(s string) value.Pointer {
	panic("String not implemented")
}

func (b *replayBuilder) Call(f builder.FunctionInfo) {
	push := C.uint8_t(0)
	if f.ReturnType != protocol.Type_Void {
		push = 1
	}
	b.emit(C.GAPIL_REPLAY_ASM_INST_CALL,
		&C.gapil_replay_asm_call{
			push_return: push,
			api_index:   C.uint8_t(f.ApiIndex),
			function_id: C.uint16_t(f.ID),
		})
}

func (b *replayBuilder) Copy(count uint64) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_COPY,
		&C.gapil_replay_asm_copy{C.uint64_t(count)})
}

func (b *replayBuilder) Clone(index int) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_CLONE,
		&C.gapil_replay_asm_clone{C.uint32_t(index)})
}

func (b *replayBuilder) Load(ty protocol.Type, addr value.Pointer) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_LOAD,
		&C.gapil_replay_asm_load{
			data_type: b.protocolTy(ty),
			source:    b.value(addr),
		})
}

func (b *replayBuilder) Store(addr value.Pointer) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_STORE,
		&C.gapil_replay_asm_store{b.value(addr)})
}

func (b *replayBuilder) StorePointer(idx value.PointerIndex, ptr value.Pointer) {
	size := uint64(b.memoryLayout.Pointer.Size)
	base := uint64(idx) * size
	b.reserveMemory(memory.Range{Base: base, Size: size}, replayPointerNamespace)

	b.Push(ptr)
	b.emit(C.GAPIL_REPLAY_ASM_INST_STORE,
		&C.gapil_replay_asm_store{b.value(idx)})
}

func (b *replayBuilder) Strcpy(maxCount uint64) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_STRCPY,
		&C.gapil_replay_asm_strcpy{C.uint64_t(maxCount)})
}

func (b *replayBuilder) Post(addr value.Pointer, size uint64, p builder.Postback) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_POST,
		&C.gapil_replay_asm_post{
			source: b.value(addr),
			size:   C.uint64_t(size),
		})
	b.decoders = append(b.decoders, postbackDecoder{
		expectedSize: int(size),
		decode:       p,
	})
}

func (b *replayBuilder) Push(val value.Value) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_PUSH,
		&C.gapil_replay_asm_push{b.value(val)})
}

func (b *replayBuilder) Pop(count uint32) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_POP,
		&C.gapil_replay_asm_pop{C.uint32_t(count)})
}

func (b *replayBuilder) ReserveMemory(rng memory.Range) {
	b.reserveMemory(rng, 0)
}

func (b *replayBuilder) reserveMemory(rng memory.Range, namespace int) {
	sli := C.gapil_slice{
		root:  C.uint64_t(rng.Base),
		base:  C.uint64_t(rng.Base),
		size:  C.uint64_t(rng.Size),
		count: C.uint64_t(rng.Size),
	}
	const alignment = 8
	C.gapil_replay_reserve_memory(b.e.cCtx, b.d, &sli, C.uint32_t(namespace), alignment)
}

func (b *replayBuilder) MapMemory(rng memory.Range) {
	panic("MapMemory not implemented")

	// Allocate memory to hold the target mapped base address that is on the
	// stack.
	target := b.AllocateMemory(uint64(b.memoryLayout.GetPointer().GetSize()))
	// Store the target mapped base address to the allocation.
	b.Store(target)

	b.emit(C.GAPIL_REPLAY_ASM_INST_MAPMEMORY,
		&C.gapil_replay_asm_mapmemory{
			dst_pp:   C.uint64_t(target.(value.VolatilePointer)),
			src_base: C.uint64_t(rng.Base),
			size:     C.uint64_t(rng.Size),
		})
}

func (b *replayBuilder) UnmapMemory(rng memory.Range) {
	b.emit(C.GAPIL_REPLAY_ASM_INST_UNMAPMEMORY,
		&C.gapil_replay_asm_unmapmemory{
			src_base: C.uint64_t(rng.Base),
			size:     C.uint64_t(rng.Size),
		})
}

func (b *replayBuilder) Write(rng memory.Range, resourceID id.ID) {
	b.e.Bind(b.ctx, func() {
		id := (*C.uint8_t)(unsafe.Pointer(&resourceID[0]))
		size := C.uint64_t(rng.Size)
		idx := C.gapil_replay_add_resource_by_id(b.e.cCtx, b.d, id, size)
		b.emit(C.GAPIL_REPLAY_ASM_INST_RESOURCE,
			&C.gapil_replay_asm_resource{
				index: idx,
				dest:  b.value(value.ObservedPointer(rng.Base)),
			})
	})
}

func (b *replayBuilder) Remappings() map[interface{}]value.Pointer {
	return b.remappings
}

func (b *replayBuilder) RegisterNotificationReader(reader builder.NotificationReader) {
	b.notificationReaders = append(b.notificationReaders, reader)
}

func (b *replayBuilder) Export(ctx context.Context) (gapir.Payload, error) {
	panic("Export not implemented")
}
func (b *replayBuilder) Build(ctx context.Context) (gapir.Payload, builder.PostDataHandler, builder.NotificationHandler, error) {
	p := C.gapil_replay_payload{}
	C.gapil_replay_build(b.e.cCtx, b.d, &p)
	defer C.gapil_replay_free_payload(&p)

	// copyBuf makes a copy of the buffer data. This is done as the payload is
	// disposed by the defer, freeing the payload memory.
	copyBuf := func(ptr unsafe.Pointer, size uint64) []byte {
		g := make([]byte, size)
		copy(g, slice.Bytes(ptr, size))
		return g
	}

	resources := slice.Cast(
		copyBuf(unsafe.Pointer(p.resources.data), uint64(p.resources.size)),
		reflect.TypeOf([]C.gapil_replay_resource_info_t{})).([]C.gapil_replay_resource_info_t)

	payload := replaysrv.Payload{
		StackSize:          uint32(p.stack_size),
		VolatileMemorySize: uint32(p.volatile_size),
		Opcodes:            copyBuf(unsafe.Pointer(p.opcodes.data), uint64(p.opcodes.size)),
		Resources:          make([]*replaysrv.ResourceInfo, len(resources)),
		Constants:          copyBuf(unsafe.Pointer(p.constants.data), uint64(p.constants.size)),
	}

	for i, r := range resources {
		var id id.ID
		copy(id[:], slice.Bytes(unsafe.Pointer(&r.id[0]), 20))
		payload.Resources[i] = &replaysrv.ResourceInfo{
			Id:   id.String(),
			Size: uint32(r.size),
		}
	}

	// Make a copy of the reference of the finished decoder list to cut off the
	// connection between the builder and furture uses of the decoders so that
	// the builder do not need to be kept alive when using these decoders.
	byteOrder := b.memoryLayout.GetEndian()
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

	// Make a copy of the reference of the finished notification reader list to
	// cut off the connection between the builder and future uses of the readers
	// so that the builder do not need to be kept alive when using these readers.
	readers := b.notificationReaders
	handleNotification := func(n *gapir.Notification) {
		ctx = log.Enter(ctx, "NotificationHandler")
		if n == nil {
			log.E(ctx, "Cannot handle nil Notification")
			return
		}
		crash.Go(func() {
			for _, r := range readers {
				r(*n)
			}
		})
	}

	return payload, handlePost, handleNotification, nil
}

func (b *replayBuilder) value(v value.Value) C.gapil_replay_asm_value {
	switch v := v.(type) {
	case value.Bool:
		out := C.gapil_replay_asm_value{
			data_type: C.GAPIL_REPLAY_ASM_TYPE_BOOL,
		}
		if v {
			out.data = 1
		}
		return out
	case value.U8:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_UINT8,
		}
	case value.S8:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_INT8,
		}
	case value.U16:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_UINT16,
		}
	case value.S16:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_INT16,
		}
	case value.F32:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(math.Float32bits(float32(v))),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_FLOAT,
		}
	case value.U32:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_UINT32,
		}
	case value.S32:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_INT32,
		}
	case value.F64:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(math.Float64bits(float64(v))),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_DOUBLE,
		}
	case value.U64:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_UINT64,
		}
	case value.S64:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_INT64,
		}
	case value.AbsolutePointer:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER,
		}
	case value.ConstantPointer:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER,
		}
	case value.VolatilePointer:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER,
		}
	case value.ObservedPointer:
		return C.gapil_replay_asm_value{
			data:      C.uint64_t(v),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0,
		}
	case value.PointerIndex:
		return C.gapil_replay_asm_value{
			data: C.uint64_t(v) * C.uint64_t(b.memoryLayout.Pointer.Size),
			data_type: C.GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0 +
				replayPointerNamespace,
		}
	default:
		panic(fmt.Errorf("Unhandled value type %T", v))
	}
}

func (b *replayBuilder) protocolTy(ty protocol.Type) C.gapil_replay_asm_type {
	switch ty {
	case protocol.Type_Bool:
		return C.GAPIL_REPLAY_ASM_TYPE_BOOL
	case protocol.Type_Int8:
		return C.GAPIL_REPLAY_ASM_TYPE_INT8
	case protocol.Type_Int16:
		return C.GAPIL_REPLAY_ASM_TYPE_INT16
	case protocol.Type_Int32:
		return C.GAPIL_REPLAY_ASM_TYPE_INT32
	case protocol.Type_Int64:
		return C.GAPIL_REPLAY_ASM_TYPE_INT64
	case protocol.Type_Uint8:
		return C.GAPIL_REPLAY_ASM_TYPE_UINT8
	case protocol.Type_Uint16:
		return C.GAPIL_REPLAY_ASM_TYPE_UINT16
	case protocol.Type_Uint32:
		return C.GAPIL_REPLAY_ASM_TYPE_UINT32
	case protocol.Type_Uint64:
		return C.GAPIL_REPLAY_ASM_TYPE_UINT64
	case protocol.Type_Float:
		return C.GAPIL_REPLAY_ASM_TYPE_FLOAT
	case protocol.Type_Double:
		return C.GAPIL_REPLAY_ASM_TYPE_DOUBLE
	case protocol.Type_AbsolutePointer:
		return C.GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER
	case protocol.Type_ConstantPointer:
		return C.GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER
	case protocol.Type_VolatilePointer:
		return C.GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER
	case protocol.Type_Void:
		return C.GAPIL_REPLAY_ASM_TYPE_VOID
	default:
		panic(fmt.Errorf("Unhandled protocol type %v", ty))
	}
}

func (b *replayBuilder) emit(ty C.gapil_replay_asm_inst, inst interface{}) {
	v := reflect.ValueOf(inst).Elem()
	ptr := unsafe.Pointer(v.UnsafeAddr())
	size := C.uint64_t(v.Type().Size())
	C.gapil_append_buffer(&b.d.instructions, unsafe.Pointer(&ty), 1)
	C.gapil_append_buffer(&b.d.instructions, ptr, size)
}
