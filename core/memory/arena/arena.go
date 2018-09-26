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

// Package arena implements a memory arena.
package arena

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/google/gapid/core/data/compare"

	"github.com/google/gapid/core/context/keys"
)

// #include "core/memory/arena/cc/arena.h"
import "C"

// Arena is a native memory allocator that owns each of the allocations made by
// Allocate() and Reallocate(). If there are any outstanding allocations when
// the Arena is disposed then these allocations are automatically freed.
// Because the memory is allocated outside of the Go environment it is important
// to explicity free unused memory - either by calling Free() or calling
// Dispose() on the Arena.
// Failing to Dispose() the arena will leak memory.
type Arena struct{ Pointer unsafe.Pointer }

// New constructs a new arena.
// You must call Dispose to free the arena object and any arena-owned
// allocations.
func New() Arena {
	return Arena{Pointer: unsafe.Pointer(C.arena_create())}
}

// Dispose destructs and frees the arena and all arena-owned allocations.
func (a Arena) Dispose() {
	a.assertNotNil()
	C.arena_destroy((*C.arena)(a.Pointer))
}

// Allocate returns a pointer to a new arena-owned, contiguous block of memory
// of the specified size and alignment.
func (a Arena) Allocate(size, alignment int) unsafe.Pointer {
	a.assertNotNil()
	return C.arena_alloc((*C.arena)(a.Pointer), C.uint32_t(size), C.uint32_t(alignment))
}

// Reallocate reallocates the memory at ptr to the new size and alignment.
// ptr must have been allocated from this arena or be nil.
func (a Arena) Reallocate(ptr unsafe.Pointer, size, alignment int) unsafe.Pointer {
	a.assertNotNil()
	return C.arena_realloc((*C.arena)(a.Pointer), ptr, C.uint32_t(size), C.uint32_t(alignment))
}

// Free releases the memory at ptr, which must have been previously allocated by
// this arena.
func (a Arena) Free(ptr unsafe.Pointer) {
	a.assertNotNil()
	C.arena_free((*C.arena)(a.Pointer), ptr)
}

// Stats holds statistics of an Arena.
type Stats struct {
	NumAllocations    int
	NumBytesAllocated int
}

func (s Stats) String() string {
	return fmt.Sprintf("{allocs: %v, bytes: %v}", s.NumAllocations, s.NumBytesAllocated)
}

// Stats returns statistics of the current state of the Arena.
func (a Arena) Stats() Stats {
	var numAllocs, numBytes C.size_t
	a.assertNotNil()
	C.arena_stats((*C.arena)(a.Pointer), &numAllocs, &numBytes)
	return Stats{int(numAllocs), int(numBytes)}
}

func (a Arena) assertNotNil() {
	if a.Pointer == nil {
		panic("nil arena")
	}
}

type arenaKeyTy string

const arenaKey = arenaKeyTy("arena")

// Get returns the Arena attached to the given context.
func Get(ctx context.Context) Arena {
	if val := ctx.Value(arenaKey); val != nil {
		return val.(Arena)
	}
	panic("arena missing from context")
}

// Put amends a Context by attaching a Arena reference to it.
func Put(ctx context.Context, d Arena) context.Context {
	if val := ctx.Value(arenaKey); val != nil {
		panic("Context already holds an arena")
	}
	return keys.WithValue(ctx, arenaKey, d)
}

// Offsetable is used as an anonymous field of types that require a current
// offset value.
type Offsetable struct{ Offset int }

// AlignUp rounds-up the current offset so that is is a multiple of n.
func (o *Offsetable) AlignUp(n int) {
	pad := n - o.Offset%n
	if pad == n {
		return
	}
	o.Offset += pad
}

// Writer provides methods to help allocate and populate a native buffer with
// data. Use Arena.Writer() to construct.
type Writer struct {
	Offsetable // The current write-offset in bytes.
	arena      Arena
	size       int
	alignment  int
	base       unsafe.Pointer
	frozen     bool
}

// NewWriter returns a new Writer to a new arena allocated buffer of the initial
// size. The native buffer may grow if the writer exceeds the size of the
// buffer. The buffer will always be of the specified alignment in memory.
// The once the native buffer is no longer needed, the pointer returned by
// Pointer() should be passed to Arena.Free().
func (a Arena) NewWriter(size, alignment int) *Writer {
	base := a.Allocate(size, alignment)
	return &Writer{
		arena:     a,
		size:      size,
		alignment: alignment,
		base:      base,
	}
}

// Reset sets the write offset back to the start of the buffer and unfreezes
// the writer. This allows for efficient reuse of the writer's native buffer.
func (w *Writer) Reset() {
	w.Offset = 0
	w.frozen = false
}

// Pointer returns the base address of the native buffer for the writer.
// Calling Pointer() freezes the writer - once called no more writes to the
// buffer can be made, unless Reset() is called. Freezing attempts to reduce the
// chance of the stale pointer being used after a buffer reallocation.
func (w *Writer) Pointer() unsafe.Pointer {
	w.frozen = true
	return w.base
}

// Write copies size bytes from src to the current writer's write offset.
// If there is not enough space in the buffer for the write, then the buffer
// is grown via reallocation.
// Upon returning, the write offset is incremented by size bytes.
func (w *Writer) Write(src unsafe.Pointer, size int) {
	if w.frozen {
		panic("Cannot write to Writer after calling Pointer()")
	}
	if needed := w.Offset + size; needed > w.size {
		size := w.size
		for needed > size {
			size *= 2 // TODO: Snugger fit?
		}
		w.base = w.arena.Reallocate(w.base, size, w.alignment)
		w.size = size
	}
	dst := uintptr(w.base) + uintptr(w.Offset)
	for i := 0; i < size; i++ {
		dst := (*byte)(unsafe.Pointer(dst + uintptr(i)))
		src := (*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
		*dst = *src
	}
	w.Offset += size
}

// Reader provides the Read method to read native buffer data.
// Use NewReader() to construct.
type Reader struct {
	Offsetable // The current read-offset in bytes.
	base       unsafe.Pointer
}

// NewReader returns a new Reader to the native-buffer starting at ptr.
func NewReader(ptr unsafe.Pointer) *Reader {
	return &Reader{base: ptr}
}

// Read copies size bytes from the current read offset to dst.
// Upon returning, the read offset is incremented by size bytes.
func (r *Reader) Read(dst unsafe.Pointer, size int) {
	src := uintptr(r.base) + uintptr(r.Offset)
	for i := 0; i < size; i++ {
		src := (*byte)(unsafe.Pointer(src + uintptr(i)))
		dst := (*byte)(unsafe.Pointer(uintptr(dst) + uintptr(i)))
		*dst = *src
	}
	r.Offset += size
}

func init() {
	// Don't compare arenas.
	compare.Register(func(c compare.Comparator, a, b Arena) {
	})
}
