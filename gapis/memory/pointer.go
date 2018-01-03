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

package memory

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/os/device"
)

// Nullptr is a zero-address pointer in the application pool.
var Nullptr = BytePtr(0, ApplicationPool)

// Values smaller than this are not legal addresses.
const lowMem = uint64(1) << 16
const bits32 = uint64(1) << 32

// Pointer is the type representing a memory pointer.
type Pointer interface {
	ReflectPointer

	// IsNullptr returns true if the address is 0 and the pool is memory.ApplicationPool.
	IsNullptr() bool

	// Address returns the pointer's memory address.
	Address() uint64

	// Pool returns the memory pool.
	Pool() PoolID

	// Offset returns the pointer offset by n bytes.
	Offset(n uint64) Pointer

	// ElementSize returns the size in bytes of the element type.
	ElementSize(m *device.MemoryLayout) uint64

	// ElementType returns the type of the pointee.
	ElementType() reflect.Type

	// ISlice returns a new Slice of elements based from this pointer using
	// start and end indices.
	ISlice(start, end uint64, m *device.MemoryLayout) Slice
}

// This is a helper interface, if you want your pointer to be reflected
// then it must ALSO implement this interface. Since reflection is slow
// having a much smaller interface to check is significantly better
// We name this APointer since reflect.Implements checks functions
// in alphabetical order, meaning this should get hit first (or close to).
type ReflectPointer interface {
	APointer()
}

// NewPtr returns a new pointer.
func NewPtr(addr uint64, pool PoolID, elTy reflect.Type) Pointer {
	return ptr{addr, pool, elTy}
}

// BytePtr returns a pointer to bytes.
func BytePtr(addr uint64, pool PoolID) Pointer {
	return NewPtr(addr, pool, reflect.TypeOf(byte(0)))
}

// ptr is a pointer to a basic type.
type ptr struct {
	addr uint64
	pool PoolID
	elTy reflect.Type
}

func (p ptr) String() string                            { return PointerToString(p) }
func (p ptr) IsNullptr() bool                           { return p.addr == 0 && p.pool == ApplicationPool }
func (p ptr) Address() uint64                           { return p.addr }
func (p ptr) Pool() PoolID                              { return p.pool }
func (p ptr) Offset(n uint64) Pointer                   { return ptr{p.addr + n, p.pool, p.elTy} }
func (p ptr) ElementSize(m *device.MemoryLayout) uint64 { return SizeOf(p.elTy, m) }
func (p ptr) ElementType() reflect.Type                 { return p.elTy }
func (p ptr) APointer()                                 { return }
func (p ptr) ISlice(start, end uint64, m *device.MemoryLayout) Slice {
	if start > end {
		panic(fmt.Errorf("%v.Slice start (%d) is greater than the end (%d)", p, start, end))
	}
	return sli{root: p.addr, base: p.addr + start*p.ElementSize(m), count: end - start, pool: p.pool, elTy: p.elTy}
}

var _ data.Assignable = &ptr{}

func (p *ptr) Assign(o interface{}) bool {
	if o, ok := o.(Pointer); ok {
		*p = ptr{o.Address(), o.Pool(), p.elTy}
		return true
	}
	return false
}

// PointerToString returns a string representation of the pointer.
func PointerToString(p Pointer) string {
	addr, pool := p.Address(), p.Pool()
	if pool == PoolID(0) {
		if addr < lowMem {
			return fmt.Sprint(addr)
		}
		if addr < bits32 {
			return fmt.Sprintf("0x%.8x", addr)
		}
		return fmt.Sprintf("0x%.16x", addr)
	}
	if addr < bits32 {
		return fmt.Sprintf("0x%.8x@%d", addr, pool)
	}
	return fmt.Sprintf("0x%.16x@%d", addr, pool)
}
