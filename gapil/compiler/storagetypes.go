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

package compiler

import (
	"fmt"

	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/semantic"
)

// StorageTypes are types that can be persisted in buffers, with possibly a
// different ABI to the target.
type StorageTypes struct {
	t       *Types
	layout  *device.MemoryLayout
	classes map[*semantic.Class]*codegen.Struct
}

// StorageTypes returns a StorageTypes built for the given ABI.
func (c *C) StorageTypes(layout *device.MemoryLayout, prefix string) *StorageTypes {
	key := memLayoutKey(layout.String())
	if existing, ok := c.T.storage[key]; ok {
		return existing
	}
	s := &StorageTypes{
		t:       &c.T,
		layout:  layout,
		classes: map[*semantic.Class]*codegen.Struct{},
	}
	c.T.storage[key] = s

	matchesTarget := c.T.targetABI.MemoryLayout.SameAs(layout)

	toBuild := []*semantic.Class{}

	for _, api := range c.APIs {
		for _, t := range api.Classes {
			if !semantic.IsStorageType(t) {
				continue
			}
			if matchesTarget {
				s.classes[t] = c.T.target[t].(*codegen.Struct)
			} else {
				s.classes[t] = c.T.DeclarePackedStruct(prefix + t.Name())
				toBuild = append(toBuild, t)
			}
		}
	}

	for _, t := range toBuild {
		offset := uint64(0)
		fields := make([]codegen.Field, 0, len(t.Fields))
		paddingFields := 0
		for _, f := range t.Fields {
			size := s.StrideOf(f.Type)
			alignment := s.AlignOf(f.Type)
			newOffset := (offset + (alignment - 1)) & ^(alignment - 1)
			if newOffset != offset {
				nm := fmt.Sprintf("__padding%d", paddingFields)
				paddingFields++
				fields = append(fields, codegen.Field{Name: nm, Type: c.T.Array(s.Get(semantic.Uint8Type), int(newOffset-offset))})
			}
			offset = newOffset + size
			fields = append(fields, codegen.Field{Name: f.Name(), Type: s.Get(f.Type)})
		}
		totalSize := s.StrideOf(t)
		if totalSize != offset {
			nm := fmt.Sprintf("__padding%d", paddingFields)
			fields = append(fields, codegen.Field{Name: nm, Type: c.T.Array(s.Get(semantic.Uint8Type), int(totalSize-offset))})
		}

		s.classes[t].SetBody(true, fields...)
	}

	return s
}

// Get returns the codegen type used to store ty in a buffer.
func (s *StorageTypes) Get(ty semantic.Type) codegen.Type {
	ty = semantic.Underlying(ty)
	switch ty := ty.(type) {
	case *semantic.Builtin:
		switch ty {
		case semantic.IntType:
			return s.t.basic(semantic.Integer(s.layout.Integer.Size))
		case semantic.SizeType:
			return s.t.basic(semantic.UnsignedInteger(s.layout.Size.Size))
		}
	case *semantic.StaticArray:
		return s.t.Array(s.Get(ty.ValueType), int(ty.Size))
	case *semantic.Pointer:
		return s.t.basic(semantic.UnsignedInteger(s.layout.Pointer.Size))
	case *semantic.Class:
		if out, ok := s.classes[ty]; ok {
			return out
		}
		fail("Storage class not registered: '%v'", ty.Name())
	case *semantic.Slice, *semantic.Reference, *semantic.Map:
		fail("Cannot store type '%v' (%T) in buffers", ty.Name(), ty)
	}
	return s.t.basic(ty)
}

// AlignOf returns the alignment of this type in bytes.
func (s *StorageTypes) AlignOf(ty semantic.Type) uint64 {
	return s.t.AlignOf(s.layout, ty)
}

// SizeOf returns size of the type.
func (s *StorageTypes) SizeOf(ty semantic.Type) uint64 {
	return s.t.SizeOf(s.layout, ty)
}

// StrideOf returns the number of bytes per element when held in an array.
func (s *StorageTypes) StrideOf(ty semantic.Type) uint64 {
	return s.t.StrideOf(s.layout, ty)
}
