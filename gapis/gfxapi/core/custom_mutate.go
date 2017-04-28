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

package core

import (
	"context"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles"
	"github.com/google/gapid/gapis/replay/builder"
)

var _ atom.Atom = &Architecture{}

func (a *Architecture) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	bo := device.BigEndian
	if a.LittleEndian {
		bo = device.LittleEndian
	}

	U64Alignment := int32(8)

	for _, e := range *a.Extras() {
		if align, ok := e.(*atom.FieldAlignments); ok {
			U64Alignment = int32(align.U64Alignment)
		}
	}
	s.MemoryLayout = &device.MemoryLayout{
		Pointer: &device.DataTypeLayout{Size: int32(a.PointerAlignment), Alignment: int32(a.PointerAlignment)},
		Integer: &device.DataTypeLayout{Size: int32(a.IntegerSize), Alignment: int32(a.IntegerSize)},
		Size:    &device.DataTypeLayout{Size: int32(a.PointerAlignment), Alignment: int32(a.PointerAlignment)},
		Char:    &device.DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &device.DataTypeLayout{Size: 8, Alignment: U64Alignment},
		I32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &device.DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &device.DataTypeLayout{Size: 8, Alignment: U64Alignment},
		F32:     &device.DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &device.DataTypeLayout{Size: 2, Alignment: 2},
		Endian:  bo,
	}
	return nil
}

func (a *SwitchThread) Mutate(ctx context.Context, gs *gfxapi.State, b *builder.Builder) error {
	err := a.mutate(ctx, gs, nil)
	if b == nil || err != nil {
		return err
	}
	return gles.OnSwitchThread(ctx, gs, b)
}
