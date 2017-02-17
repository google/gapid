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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/memory/memory_pb"
	"github.com/google/gapid/gapis/service/path"
)

// Nullptr is a zero-address pointer in the application pool.
var Nullptr = Pointer{Pool: ApplicationPool}

// Values smaller than this are not legal addresses.
const lowMem = uint64(1) << 16
const bits32 = uint64(1) << 32

// Pointer is the type representing a memory pointer.
type Pointer struct {
	binary.Generate `java:"MemoryPointer"`
	Address         uint64 // The memory address.
	Pool            PoolID // The memory pool.
}

// Offset returns the pointer offset by n bytes.
func (p Pointer) Offset(n uint64) Pointer {
	return Pointer{Address: p.Address + n, Pool: p.Pool}
}

// Range returns a Range of size s with the base of this pointer.
func (p Pointer) Range(s uint64) Range {
	return Range{Base: p.Address, Size: s}
}

func (p Pointer) String() string {
	if p.Pool == PoolID(0) {
		if p.Address < lowMem {
			return fmt.Sprint(p.Address)
		}
		if p.Address < bits32 {
			return fmt.Sprintf("0x%.8x", p.Address)
		}
		return fmt.Sprintf("0x%.16x", p.Address)
	}
	if p.Address < bits32 {
		return fmt.Sprintf("0x%.8x@%d", p.Address, p.Pool)
	}
	return fmt.Sprintf("0x%.16x@%d", p.Address, p.Pool)
}

// Link return the path to the memory pointed-to by p.
func (p Pointer) Link(ctx log.Context, n path.Node) (path.Node, error) {
	if cmd := path.FindCommand(n); cmd != nil {
		return cmd.MemoryAfter(uint32(p.Pool), p.Address, 0), nil
	}
	return nil, nil
}

func (p Pointer) ToProto() *memory_pb.Pointer {
	return &memory_pb.Pointer{
		Address: p.Address,
		Pool:    uint32(p.Pool),
	}
}

func (p *Pointer) Convert(ctx log.Context, out atom_pb.Handler) error {
	return out(ctx, p.ToProto())
}

func PointerFrom(from *memory_pb.Pointer) Pointer {
	return Pointer{
		Address: from.Address,
		Pool:    PoolID(from.Pool),
	}
}
