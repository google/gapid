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

package executor

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/stringtable"
)

// #include "env.h"
// #include <stdlib.h> // free
import "C"

// buffer is an allocation used to hold remapped memory.
type buffer struct {
	rng   memory.Range
	alloc unsafe.Pointer
}

// buffers is a list of buffers.
type buffers []buffer

// add adds a new buffer spanning the storage memory range rng that maps to the
// allocated buffer at alloc the into the list.
func (l *buffers) add(rng memory.Range, alloc unsafe.Pointer) {
	*l = append(*l, buffer{rng, alloc})
}

// lookup returns the buffer that overlaps the storage memory pointer ptr, or
// nil if there is no buffer for the given pointer.
func (l buffers) lookup(ptr uint64) *buffer {
	for i, b := range l {
		if b.rng.Contains(ptr) {
			return &l[i]
		}
	}
	return nil
}

// find returns the buffer that overlaps the storage memory pointer ptr.
// If there is no buffer for the given pointer, find panics.
func (l buffers) find(ptr uint64) buffer {
	if b := l.lookup(ptr); b != nil {
		return *b
	}
	panic(fmt.Errorf("%v is not allocated", ptr))
}

// remap returns the base address allocated buffer for the storage memory range
// rng. If there is no allocated memory range for rng (or the range overflows
// the buffer) then remap panics.
func (l buffers) remap(rng memory.Range) unsafe.Pointer {
	b := l.find(rng.Base)
	if rng.Base+rng.Size > b.rng.Base+b.rng.Size {
		panic(fmt.Errorf("%v overflows allocation %v", rng, b.rng))
	}
	offset := (uintptr)(rng.Base - b.rng.Base)
	return (unsafe.Pointer)((uintptr)(b.alloc) + offset)
}

// Env is the go execution environment.
type Env struct {
	// Arena is the memory arena used by this execution environment.
	Arena arena.Arena

	// Executor is the parent executor.
	Executor *Executor

	id      envID
	cCtx    *C.context      // The gapil C context.
	goCtx   context.Context // The go context.
	cmd     api.Cmd         // The currently executing command.
	buffers buffers
}

// Dispose releases the memory used by the environment.
// Call after the env is no longer needed to avoid leaking memory.
func (e *Env) Dispose() {
	C.destroy_context((*C.TDestroyContext)(e.Executor.destroyContext), e.cCtx)
	e.Arena.Dispose()
}

type envID uint32

var (
	envMutex  sync.RWMutex
	nextEnvID envID
	envs      = map[envID]*Env{}
)

// env returns the environment for the given context c.
func env(c *C.context) *Env {
	id := envID(c.id)
	envMutex.RLock()
	out, ok := envs[id]
	envMutex.RUnlock()
	if !ok {
		panic(fmt.Errorf("Unknown envID %v", id))
	}
	return out
}

// GetEnv returns the environment for the given context c.
func GetEnv(c unsafe.Pointer) *Env {
	return env((*C.context)(c))
}

// NewEnv creates a new execution environment for the given capture.
func (e *Executor) NewEnv(ctx context.Context, capture *capture.GraphicsCapture) *Env {
	var id envID
	var env *Env

	func() {
		envMutex.Lock()
		defer envMutex.Unlock()

		id = nextEnvID
		nextEnvID++

		env = &Env{
			Executor: e,
			id:       envID(id),
		}
		envs[id] = env
	}()

	// Create the context and initialize the globals.
	env.Arena = arena.New()
	env.goCtx = ctx
	env.cCtx = C.create_context((*C.TCreateContext)(e.createContext), (*C.arena)(env.Arena.Pointer))
	env.cCtx.id = (C.uint32_t)(id)
	env.goCtx = nil

	// Allocate buffers for all the observed memory ranges.
	for _, r := range capture.Observed {
		ptr := env.Arena.Allocate((int)(r.Count), 16)
		rng := memory.Range{Base: r.First, Size: r.Count}
		env.buffers.add(rng, ptr)
	}

	return env
}

// Execute executes the command cmd.
func (e *Env) Execute(ctx context.Context, cmd api.Cmd, id api.CmdID) error {
	name := cmd.CmdName()
	fptr, ok := e.Executor.cmdFunctions[name]
	if !ok {
		return fmt.Errorf("Program has no command '%v'", name)
	}

	var buf [1024]byte
	encodeCommand(cmd, buf[:])

	e.cmd = cmd
	e.cCtx.thread = (C.uint64_t)(cmd.Thread())
	e.cCtx.cmd_id = (C.uint64_t)(id)
	res := e.call(ctx, fptr, (unsafe.Pointer)(&buf[0]))
	e.cmd = nil

	return res
}

// CContext returns the pointer to the c context.
func (e *Env) CContext() unsafe.Pointer {
	return (unsafe.Pointer)(e.cCtx)
}

// Context returns the go context of the environment.
func (e *Env) Context() context.Context {
	return e.goCtx
}

// Globals returns the memory of the global state.
func (e *Env) Globals() []byte {
	return slice.Bytes((unsafe.Pointer)(e.cCtx.globals), e.Executor.globalsSize)
}

// Message unpacks the gapil_msg at p, returning a stringtable Msg.
func (e *Env) Message(p unsafe.Pointer) *stringtable.Msg {
	m := (*C.gapil_msg)(p)
	args := (*[256]C.gapil_msg_arg)(unsafe.Pointer(m.args))
	out := &stringtable.Msg{
		Identifier: C.GoString((*C.char)(unsafe.Pointer(m.identifier))),
		Arguments:  map[string]*stringtable.Value{},
	}
	for _, arg := range args {
		if arg.name == nil {
			break
		}
		name := C.GoString((*C.char)(unsafe.Pointer(arg.name)))
		val := e.Any(unsafe.Pointer(arg.value))
		out.Arguments[name] = stringtable.ToValue(val)
	}
	return out
}

// Any unpacks and returns the value held by the gapil_any at p.
func (e *Env) Any(p unsafe.Pointer) interface{} {
	a := (*C.gapil_any)(p)
	switch a.rtti.kind {
	case C.GAPIL_KIND_BOOL:
		return *(*bool)(a.value)
	case C.GAPIL_KIND_U8:
		return *(*uint8)(a.value)
	case C.GAPIL_KIND_S8:
		return *(*int8)(a.value)
	case C.GAPIL_KIND_U16:
		return *(*uint16)(a.value)
	case C.GAPIL_KIND_S16:
		return *(*int16)(a.value)
	case C.GAPIL_KIND_F32:
		return *(*float32)(a.value)
	case C.GAPIL_KIND_U32:
		return *(*uint32)(a.value)
	case C.GAPIL_KIND_S32:
		return *(*int32)(a.value)
	case C.GAPIL_KIND_F64:
		return *(*float64)(a.value)
	case C.GAPIL_KIND_U64:
		return *(*uint64)(a.value)
	case C.GAPIL_KIND_S64:
		return *(*int64)(a.value)
	case C.GAPIL_KIND_INT:
		return *(*memory.Int)(a.value)
	case C.GAPIL_KIND_UINT:
		return *(*memory.Uint)(a.value)
	case C.GAPIL_KIND_SIZE:
		return *(*memory.Size)(a.value)
	case C.GAPIL_KIND_CHAR:
		return *(*memory.Char)(a.value)
	case C.GAPIL_KIND_ARRAY:
		panic("Unpacking Arrays boxed in anys not implemented")
	case C.GAPIL_KIND_CLASS:
		panic("Unpacking Classes boxed in anys not implemented")
	case C.GAPIL_KIND_ENUM:
		panic("Unpacking Enums boxed in anys not implemented")
	case C.GAPIL_KIND_MAP:
		panic("Unpacking Maps boxed in anys not implemented")
	case C.GAPIL_KIND_POINTER:
		panic("Unpacking Pointers boxed in anys not implemented")
	case C.GAPIL_KIND_REFERENCE:
		panic("Unpacking References boxed in anys not implemented")
	case C.GAPIL_KIND_SLICE:
		panic("Unpacking Slices boxed in anys not implemented")
	case C.GAPIL_KIND_STRING:
		s := (*C.string)(a.value)
		return C.GoString((*C.char)(unsafe.Pointer(&s.data[0])))
	}
	return nil
}

// GetBytes returns the bytes that are in the given memory range.
func (e *Env) GetBytes(rng memory.Range) []byte {
	basePtr := e.buffers.remap(rng)
	return slice.Bytes(basePtr, rng.Size)
}

func (e *Env) call(ctx context.Context, fptr, args unsafe.Pointer) error {
	e.goCtx = ctx
	e.cCtx.cmd_args = args
	err := compiler.ErrorCode(C.call(e.cCtx, (*C.TFunc)(fptr)))
	e.goCtx = nil

	return errorFromErrorCode(err)
}

func errorFromErrorCode(c compiler.ErrorCode) error {
	switch c {
	case compiler.ErrSuccess:
		return nil
	case compiler.ErrAborted:
		return api.ErrCmdAborted{}
	default:
		return fmt.Errorf("Unknown error code %v", c)
	}
}

func (e *Env) applyObservations(l []api.CmdObservation) {
	for _, o := range l {
		obj, err := database.Resolve(e.goCtx, o.ID)
		if err != nil {
			panic(err)
		}
		data := obj.([]byte)
		ptr := e.buffers.remap(o.Range)
		dst := slice.Bytes(ptr, o.Range.Size)
		copy(dst, data)
	}
}

func (e *Env) applyReads() {
	if extras := e.cmd.Extras(); extras != nil {
		if observations := extras.Observations(); observations != nil {
			e.applyObservations(observations.Reads)
		}
	}
}

func (e *Env) applyWrites() {
	if extras := e.cmd.Extras(); extras != nil {
		if observations := extras.Observations(); observations != nil {
			e.applyObservations(observations.Writes)
		}
	}
}

func (e *Env) resolvePoolData(pool *C.pool, ptr C.uint64_t, access C.gapil_data_access, len *C.uint64_t) unsafe.Pointer {
	if pool != nil {
		if ptr > pool.size {
			panic(fmt.Errorf("%v overflows pool buffer %v", ptr, pool.size))
		}
		if len != nil {
			*len = pool.size - ptr
		}
		return unsafe.Pointer(uintptr(pool.buffer) + uintptr(ptr))
	}

	// Application pool
	b := e.buffers.find(uint64(ptr))
	offset := uint64(ptr) - b.rng.Base
	if len != nil {
		*len = (C.uint64_t)(b.rng.Size - offset)
	}
	return (unsafe.Pointer)(uintptr(b.alloc) + uintptr(offset))
}

func (e *Env) storeInDatabase(ptr unsafe.Pointer, size C.uint64_t, idOut *C.uint8_t) {
	ctx := e.Context()
	sli := slice.Bytes(ptr, uint64(size))
	id, err := database.Store(ctx, sli)
	if err != nil {
		panic(err)
	}
	out := slice.Bytes((unsafe.Pointer)(idOut), 20)
	copy(out, id[:])
}

func init() {
	C.set_callbacks(callbacks())
}

func (e *Env) callExtern(name *C.uint8_t, args, res unsafe.Pointer) {
	n := C.GoString((*C.char)((unsafe.Pointer)(name)))
	f, ok := externs[n]
	if !ok {
		panic(fmt.Sprintf("No handler for extern '%v'", n))
	}
	f(e, args, res)
}

func init() {
	C.set_callbacks(callbacks())
}

func registerCExtern(name string, e unsafe.Pointer) {
	n := C.CString(name)
	C.register_c_extern(n, (*C.gapil_extern)(e))
	C.free(unsafe.Pointer(n))
}
