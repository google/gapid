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

import "fmt"

// Nullptr is a zero-address pointer in the application pool.
var Nullptr = BytePtr(0, ApplicationPool)

// Values smaller than this are not legal addresses.
const lowMem = uint64(1) << 16
const bits32 = uint64(1) << 32

// Pointer is the type representing a memory pointer.
type Pointer interface {
	// IsNullptr returns true if the address is 0 and the pool is memory.ApplicationPool.
	IsNullptr() bool

	// Address returns the pointer's memory address.
	Address() uint64

	// Pool returns the memory pool.
	Pool() PoolID

	// Offset returns the pointer offset by n bytes.
	Offset(n uint64) Pointer
}

// BytePtr returns a pointer to bytes.
func BytePtr(addr uint64, pool PoolID) Pointer { return ptr{addr, pool} }

// ptr is a pointer to a basic type.
type ptr struct {
	addr uint64
	pool PoolID
}

func (p ptr) String() string { return PointerToString(p) }

// IsNullptr returns true if the address is 0 and the pool is memory.ApplicationPool.
func (p ptr) IsNullptr() bool { return p.addr == 0 && p.pool == ApplicationPool }

// Address returns the pointer's memory address.
func (p ptr) Address() uint64 { return p.addr }

// Pool returns the memory pool.
func (p ptr) Pool() PoolID { return p.pool }

// Offset returns the pointer offset by n bytes.
func (p ptr) Offset(n uint64) Pointer { return ptr{p.addr + n, p.pool} }

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
