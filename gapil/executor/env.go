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
	"runtime"
	"runtime/debug"
	"sync"
	"unsafe"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/stringtable"
)

// #include "env.h"
//
// #include <string.h> // memset
// #include <stdlib.h> // free
import "C"

// Env is the go execution environment.
type Env struct {
	// Executor is the parent executor.
	Executor *Executor

	// State is the global state for the environment.
	State *api.GlobalState

	pools map[memory.PoolID]*C.pool

	// Arena to use for buffers
	bufferArena arena.Arena
	buffers     []unsafe.Pointer
	lastCmdID   api.CmdID

	id   envID
	cCtx *C.gapil_context // The gapil C context.

	boundState struct {
		goCtx   context.Context // The go context.
		cmds    []api.Cmd       // The currently executing commands.
		isBound bool
	}

	// dispose / finalizer fields
	disposeOnce        sync.Once
	autoDispose        bool // Should the env automatically call Dispose on GC?
	creationStacktrace string
}

// Dispose releases the memory used by the environment.
// Call after the env is no longer needed to avoid leaking memory.
func (e *Env) Dispose() {
	e.disposeOnce.Do(func() {
		C.destroy_context(e.Executor.module, e.cCtx)
		e.bufferArena.Dispose()
		e.State.Arena.Dispose()
	})
}

// AutoDispose automatically releases the memory used by this env when the env
// is garbage collected by the go runtime. As the GC can happen long after the
// last reference is dropped, use explicit calls to Dispose() whenever possible.
func (e *Env) AutoDispose() {
	e.autoDispose = true
}

func envFinalizer(e *Env) {
	if e.autoDispose {
		e.Dispose()
	} else {
		e.disposeOnce.Do(func() {
			panic("Env.Dispose() not called before GC, leaking memory.\n" +
				"Env created here:\n" +
				e.creationStacktrace)
		})
	}
}

type envID uint32

var (
	nextEnvID envID
	envMutex  sync.RWMutex
	envs      = map[envID]*Env{}
)

// env returns the environment for the given context c.
func env(c *C.gapil_context) *Env {
	return envFromID(envID(c.id))
}

// envFromID returns the environment for the given envID.
func envFromID(id envID) *Env {
	envMutex.RLock()
	out, ok := envs[id]
	envMutex.RUnlock()
	if !ok {
		panic(fmt.Errorf("Unknown envID %v. Did you forget to call Env.Bind()?", id))
	}
	return out
}

// EnvFromNative returns the environment for the given context c.
func EnvFromNative(c unsafe.Pointer) *Env {
	return env((*C.gapil_context)(c))
}

// NewEnv creates a new execution environment for the given capture.
func (e *Executor) NewEnv(ctx context.Context) *Env {
	var id envID
	var env *Env

	func() {
		envMutex.Lock()
		defer envMutex.Unlock()

		id = nextEnvID
		nextEnvID++

		env = &Env{
			Executor:           e,
			id:                 envID(id),
			pools:              map[memory.PoolID]*C.pool{},
			creationStacktrace: string(debug.Stack()),
		}
		runtime.SetFinalizer(env, envFinalizer)
		envs[id] = env
	}()

	env.State = &api.GlobalState{
		MemoryLayout: e.cfg.CaptureABI.GetMemoryLayout(),
		Arena:        arena.New(),
		APIs:         map[api.ID]api.State{},
		Memory:       memory.NewPools(),
	}

	// Hold a back-reference from the state to the env. This is done so that
	// functions that return a state object from an env keep the env alive.
	env.State.HoldReference(env)

	env.bufferArena = arena.New()

	// Create the context and initialize the globals.
	status.Do(ctx, "Create Context", func(ctx context.Context) {
		env.Bind(ctx, func() {
			env.cCtx = C.create_context(e.module, (*C.arena)(env.State.Arena.Pointer))
			env.cCtx.id = (C.uint32_t)(id)
		})
	})

	// Prime the state objects.
	if env.cCtx.globals != nil {
		globalsBase := uintptr(unsafe.Pointer(env.cCtx.globals))
		for _, api := range api.All() {
			if m := C.get_api_module(e.module, C.uint32_t(api.Index())); m != nil {
				addr := uintptr(m.globals_offset) + globalsBase
				state := api.State(env.State.Arena, unsafe.Pointer(addr))
				state.HoldReference(env)
				env.State.APIs[api.ID()] = state
			}
		}
	}

	return env
}

// Bind binds the environment to call out to native code.
// Failure to call Bind() before making a GAPIL runtime call will likely result
// in a panic.
func (e *Env) Bind(ctx context.Context, f func()) {
	if !e.boundState.isBound {
		envMutex.Lock()
		envs[e.id] = e
		envMutex.Unlock()
	}

	prevState := e.boundState
	e.boundState.goCtx = ctx
	e.boundState.isBound = true

	f()

	e.boundState = prevState

	if !e.boundState.isBound {
		envMutex.Lock()
		delete(envs, e.id)
		envMutex.Unlock()
	}
}

func (e *Env) assertBound() {
	if !e.boundState.isBound {
		panic("Env not bound")
	}
}

// Execute executes the all the commands in l.
func (e *Env) Execute(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) error {
	return e.ExecuteN(ctx, cmdID, []api.Cmd{cmd})[0]
}

// ExecuteN executes the all the commands in cmds.
func (e *Env) ExecuteN(ctx context.Context, firstID api.CmdID, cmds []api.Cmd) []error {
	ctx = status.Start(ctx, "Execute<%v>", len(cmds))
	defer status.Finish(ctx)

	dataBuf := e.State.Arena.Allocate(len(cmds)*int(unsafe.Sizeof(C.cmd_data{})), int(unsafe.Alignof(C.cmd_data{})))
	defer e.State.Arena.Free(dataBuf)

	data := (*(*[1 << 40]C.cmd_data)(dataBuf))[:len(cmds)]
	for i, cmd := range cmds {
		flags := C.uint64_t(0)
		if extras := cmd.Extras(); extras != nil {
			if o := extras.Observations(); o != nil {
				if len(o.Reads) > 0 {
					flags |= C.CMD_FLAGS_HAS_READS
				}
				if len(o.Writes) > 0 {
					flags |= C.CMD_FLAGS_HAS_WRITES
				}
			}
		}
		data[i] = C.cmd_data{
			api_idx: C.uint32_t(cmd.API().Index()),
			cmd_idx: C.uint32_t(cmd.CmdIndex()),
			args:    cmd.ExecData(),
			id:      C.uint64_t(firstID) + C.uint64_t(i),
			flags:   flags,
			thread:  C.uint64_t(cmd.Thread()),
		}
	}

	res := make([]C.uint64_t, len(cmds))

	e.Bind(ctx, func() {
		e.boundState.cmds = cmds
		e.call(
			&data[0],
			C.uint64_t(len(cmds)),
			&res[0],
		)
		e.boundState.cmds = nil
	})

	out := make([]error, len(cmds))
	for i, r := range res {
		switch r {
		case C.GAPIL_ERR_SUCCESS:
			// all okay
		case C.GAPIL_ERR_ABORTED:
			out[i] = &api.ErrCmdAborted{}
		}
	}
	return out
}

// CContext returns the pointer to the c context.
func (e *Env) CContext() unsafe.Pointer {
	return unsafe.Pointer(e.cCtx)
}

// Context returns the go context of the environment.
func (e *Env) Context() context.Context {
	e.assertBound()
	return e.boundState.goCtx
}

// Cmd returns the currently executing command.
func (e *Env) Cmd() api.Cmd {
	e.assertBound()
	return e.boundState.cmds[e.cCtx.cmd_idx]
}

// CmdID returns the currently executing command identifer.
func (e *Env) CmdID() api.CmdID {
	e.assertBound()
	return api.CmdID(e.cCtx.cmd_id)
}

// Globals returns the memory of the global state.
func (e *Env) Globals() []byte {
	return slice.Bytes(unsafe.Pointer(e.cCtx.globals), uint64(e.Executor.module.globals_size))
}

// Pool returns the pool pointer for the given pool identifier.
func (e *Env) Pool(id memory.PoolID) unsafe.Pointer {
	return unsafe.Pointer(e.pools[id])
}

// GetOrMakePool returns an unsafe pointer to the allocated C.pool for the given
// pool ID, constructing one if no previous one existed.
func (e *Env) GetOrMakePool(id memory.PoolID) unsafe.Pointer {
	if id == memory.ApplicationPool {
		return nil
	}
	if p, ok := e.pools[id]; ok {
		return unsafe.Pointer(p)
	}
	return unsafe.Pointer(e.makePoolAt(id))
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
		s := (*C.gapil_string)(a.value)
		return C.GoString((*C.char)(unsafe.Pointer(&s.data[0])))
	}
	return nil
}

func (e *Env) changedCommand() bool {
	cur := api.CmdID(e.cCtx.cmd_id)
	changed := cur != e.lastCmdID
	e.lastCmdID = cur
	return changed
}

func (e *Env) readPoolData(pool *memory.Pool, ptr, size uint64) unsafe.Pointer {
	if e.changedCommand() {
		for _, b := range e.buffers {
			e.bufferArena.Free(b)
		}
		e.buffers = e.buffers[:0]
	}

	ctx := e.boundState.goCtx

	rng := memory.Range{Base: ptr, Size: size}
	sli := pool.Slice(rng)

	switch sli := sli.(type) {
	case *memory.Native:
		return sli.Data()
	default:
		buf := e.bufferArena.Allocate(int(size), 1) // TODO: Free these!
		C.memset(buf, 0, C.size_t(size))            // TODO: Fix Get() to zero gaps
		if err := sli.Get(ctx, 0, slice.Bytes(buf, size)); err != nil {
			panic(err)
		}
		e.buffers = append(e.buffers, buf)
		return buf
	}
}

func (e *Env) writePoolData(pool *memory.Pool, ptr, size uint64) unsafe.Pointer {
	native := memory.NewNative(e.bufferArena, size)
	pool.Write(ptr, native)
	return native.Data()
}

func (e *Env) applyReads() {
	if extras := e.Cmd().Extras(); extras != nil {
		if o := extras.Observations(); o != nil {
			o.ApplyReads(e.State.Memory.ApplicationPool())
		}
	}
}

func (e *Env) applyWrites() {
	if extras := e.Cmd().Extras(); extras != nil {
		if o := extras.Observations(); o != nil {
			o.ApplyWrites(e.State.Memory.ApplicationPool())
		}
	}
}

func (e *Env) resolvePoolData(pool *C.pool, ptr C.uint64_t, access C.gapil_data_access, size C.uint64_t) unsafe.Pointer {
	switch access {
	case C.GAPIL_READ:
		return e.readPoolData(e.pool(pool), uint64(ptr), uint64(size))
	case C.GAPIL_WRITE:
		return e.writePoolData(e.pool(pool), uint64(ptr), uint64(size))
	default:
		panic(fmt.Errorf("Unexpected access: %v", access))
	}
}

func (e *Env) pool(pool *C.pool) *memory.Pool {
	if pool == nil {
		return e.State.Memory.ApplicationPool()
	}
	if e.id != envID(pool.env) {
		panic("Attempting to use pool from a different env")
	}
	id := memory.PoolID(pool.base.id)
	return e.State.Memory.MustGet(id)
}

func (e *Env) copySlice(dst, src *C.gapil_slice) {
	dstPool := e.pool((*C.pool)(unsafe.Pointer(dst.pool)))
	srcPool := e.pool((*C.pool)(unsafe.Pointer(src.pool)))
	size := u64.Min(uint64(dst.size), uint64(src.size))
	dstPool.Write(uint64(dst.base), srcPool.Slice(memory.Range{Base: uint64(src.base), Size: size}))
}

func (e *Env) cstringToSlice(ptr C.uint64_t, out *C.gapil_slice) {
	pool := e.State.Memory.ApplicationPool()
	size, err := pool.Strlen(e.boundState.goCtx, uint64(ptr))
	if err != nil {
		panic(err)
	}

	size++ // Include null terminator

	out.pool = nil
	out.root = C.uint64_t(ptr)
	out.base = C.uint64_t(ptr)
	out.size = C.uint64_t(size)
	out.count = C.uint64_t(size)
}

func (e *Env) storeInDatabase(ptr unsafe.Pointer, size C.uint64_t, idOut *C.uint8_t) {
	ctx := e.Context()
	sli := slice.Bytes(ptr, uint64(size))
	id, err := database.Store(ctx, sli)
	if err != nil {
		panic(err)
	}
	out := slice.Bytes(unsafe.Pointer(idOut), 20)
	copy(out, id[:])
}

func (e *Env) makePool() *C.pool {
	id, _ := e.State.Memory.New()
	return e.allocPool(id)
}

func (e *Env) makePoolAt(id memory.PoolID) *C.pool {
	e.State.Memory.NewAt(id)
	return e.allocPool(id)
}

func (e *Env) allocPool(id memory.PoolID) *C.pool {
	pool := (*C.pool)(e.State.Arena.Allocate(int(unsafe.Sizeof(C.pool{})), int(unsafe.Alignof(C.pool{}))))
	pool.base.ref_count = 1
	pool.base.id = C.uint32_t(id)
	pool.env = C.uint32_t(e.id)
	e.pools[id] = pool
	return pool
}

func (e *Env) cloneSlice(dst, src *C.gapil_slice) {
	*dst = *src
	if src.pool != nil {
		id := memory.PoolID(src.pool.id)
		if pool := e.pools[id]; pool != nil {
			dst.pool = &pool.base
			dst.pool.ref_count++
		} else {
			dst.pool = &e.makePoolAt(id).base
		}
	}
}

func (e *Env) freePool(pool *C.pool) {
	e.State.Arena.Free(unsafe.Pointer(pool))
}

func (e *Env) callExtern(name *C.uint8_t, args, res unsafe.Pointer) {
	n := C.GoString((*C.char)(unsafe.Pointer(name)))
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
