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

// #include "context.h"
import "C"

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

type buffer struct {
	rng   memory.Range
	alloc unsafe.Pointer
}

type buffers []buffer

func (l *buffers) add(rng memory.Range, alloc unsafe.Pointer) {
	*l = append(*l, buffer{rng, alloc})
}

func (l buffers) remap(rng memory.Range) unsafe.Pointer {
	for _, b := range l {
		if b.rng.Contains(rng.Base) {
			if rng.Base+rng.Size > b.rng.Base+b.rng.Size {
				panic(fmt.Errorf("%v overflows allocation %v", rng, b.rng))
			}
			offset := (uintptr)(rng.Base - b.rng.Base)
			return (unsafe.Pointer)((uintptr)(b.alloc) + offset)
		}
	}
	panic(fmt.Errorf("%v is not allocated", rng))
}

// Env is the go execution environment.
type Env struct {
	Globals []byte
	id      envID
	exec    *Executor
	arena   arena.Arena
	cCtx    *C.exec_context // The gapil C context.
	goCtx   context.Context // The go context.
	cmd     api.Cmd         // The currently executing command.
	buffers buffers
}

func (e *Env) Dispose() {
	e.arena.Dispose()
}

type envID uint32

var (
	envMutex  sync.Mutex
	nextEnvID envID
	envs      = map[envID]*Env{}
)

func env(c *C.context) *Env {
	id := envID(c.id)
	envMutex.Lock()
	out, ok := envs[id]
	envMutex.Unlock()
	if !ok {
		panic(fmt.Errorf("Unknown envID %v", id))
	}
	return out
}

func (e *Executor) NewEnv(ctx context.Context, capture *capture.Capture) *Env {
	var id envID
	var env *Env

	func() {
		envMutex.Lock()
		defer envMutex.Unlock()

		id = nextEnvID
		nextEnvID++

		a := arena.New()
		env = &Env{
			Globals: make([]byte, e.exec.SizeOf(e.program.Globals.Type)),
			id:      envID(id),
			exec:    e,
			arena:   a,
		}
		envs[id] = env
	}()

	// Initialize the context structure. This allocates the state block.
	var globals unsafe.Pointer
	if len(env.Globals) > 0 {
		globals = (unsafe.Pointer)(&env.Globals[0])
	}
	env.goCtx = ctx
	env.cCtx = C.create_context(
		C.uint32_t(id),
		(*C.globals)(globals),
		(*C.arena)(env.arena.Pointer))

	C.init_context(env.cCtx, (*C.TInit)(e.initFunction))
	env.goCtx = nil

	for _, r := range capture.Observed {
		ptr := env.arena.Allocate((int)(r.Count), 16)
		rng := memory.Range{Base: r.First, Size: r.Count}
		log.I(ctx, "Allocated %v for %v", ptr, rng)
		env.buffers.add(rng, ptr)
	}

	return env
}

func (e *Env) Execute(ctx context.Context, cmd api.Cmd) error {
	name := cmd.CmdName()
	fptr, ok := e.exec.cmdFunctions[name]
	if !ok {
		return fmt.Errorf("Program has no command '%v'", name)
	}

	var buf [1024]byte
	encodeCommand(cmd, buf[:])

	e.cmd = cmd
	res := e.call(ctx, fptr, (unsafe.Pointer)(&buf[0]))
	e.cmd = nil

	return res
}

// ArenaStats return statistics about the env's arena.
func (e *Env) ArenaStats() arena.Stats {
	return e.arena.Stats()
}

func byteSlice(ptr unsafe.Pointer, size uint64) []byte {
	return ((*[1 << 30]byte)(ptr))[:size]
}

func (e *Env) call(ctx context.Context, fptr, args unsafe.Pointer) error {
	e.goCtx = ctx
	err := compiler.ErrorCode(C.call(e.cCtx, args, (*C.TFunc)(fptr)))
	e.goCtx = nil

	return err.Err()
}

func (e *Env) applyObservations(l []api.CmdObservation) {
	for _, o := range l {
		obj, err := database.Resolve(e.goCtx, o.ID)
		if err != nil {
			panic(err)
		}
		data := obj.([]byte)
		ptr := e.buffers.remap(o.Range)
		dst := byteSlice(ptr, o.Range.Size)
		copy(dst, data)
	}
}

//export gapil_apply_reads
func gapil_apply_reads(c *C.context) {
	e := env(c)
	if extras := e.cmd.Extras(); extras != nil {
		if observations := extras.Observations(); observations != nil {
			e.applyObservations(observations.Reads)
		}
	}
}

//export gapil_apply_writes
func gapil_apply_writes(c *C.context) {
	e := env(c)
	if extras := e.cmd.Extras(); extras != nil {
		if observations := extras.Observations(); observations != nil {
			e.applyObservations(observations.Writes)
		}
	}
}

//export gapil_remap_pointer
func gapil_remap_pointer(c *C.context, ptr, length uint64) unsafe.Pointer {
	e := env(c)
	return e.buffers.remap(memory.Range{Base: ptr, Size: length})
}

//export gapil_get_code_location
func gapil_get_code_location(c *C.context, file **C.char, line *C.uint32_t) {
	e := env(c)
	l := compiler.Location{File: "<unknown>"}
	if loc := int(e.cCtx.ctx.location); loc < len(e.exec.program.Locations) {
		l = e.exec.program.Locations[loc]
	}
	*file = C.CString(l.File)
	*line = (C.uint32_t)(l.Line)
}

/*
//export gapil_call_extern
func gapil_call_extern(c *C.context, cname *C.char, args, res unsafe.Pointer) {
	e := env(c)
	name := C.GoString(cname)
	extern, ok := e.Executor.Externs[name]
	if !ok {
		panic("callExtern called with unknown extern: " + name)
	}
	log := ctx.log()
	if e.binding == nil {
		log.Error().Logf("callExtern called unbound extern '%v'", name)
		return
	}
	extern.binding(log, ctx, args, res)
}
*/
