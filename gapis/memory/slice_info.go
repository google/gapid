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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// SliceInfo is the common data between all slice types.
type SliceInfo struct {
	binary.Generate `java:"MemorySliceInfo"`
	Root            uint64 // Original pointer this slice derives from.
	Base            uint64 // Address of first element.
	Count           uint64 // Number of elements in the slice.
	Pool            PoolID // The pool identifier.
}

// SliceMetadata is the meta information about a slice.
type SliceMetadata struct {
	binary.Generate `java:"MemorySliceMetadata"`
	ElementTypeName string // The name of the type that elements of the slice have.
}

func (s SliceInfo) ToProto() *memory_pb.Slice {
	return &memory_pb.Slice{
		Root:  s.Root,
		Base:  s.Base,
		Count: s.Count,
		Pool:  uint32(s.Pool),
	}
}

func (s *SliceInfo) Convert(ctx log.Context, out atom_pb.Handler) error {
	return out(ctx, s.ToProto())
}

func SliceInfoFrom(from *memory_pb.Slice) SliceInfo {
	return SliceInfo{
		Root:  from.Root,
		Base:  from.Base,
		Count: from.Count,
		Pool:  PoolID(from.Pool),
	}
}
