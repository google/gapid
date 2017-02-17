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

package atom

import (
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
)

// FieldAlignments holds the natural alignments of POD types inside a struct.
// This is not captured by the existing architecture Atom, but rather than breaking
// compatibility, we add it as an extra here.
type FieldAlignments struct {
	binary.Generate  `java:"disable"`
	CharAlignment    uint32 // Alignment of char elements inside a struct.
	IntAlignment     uint32 // Alignment of int elements inside a struct.
	U32Alignment     uint32 // Alignment of U32 elements inside a struct.
	U64Alignment     uint32 // Alignment of U64 elements inside a struct.
	PointerAlignment uint32 // Alignment of pointers inside a struct.
	// PointerAlignment is duplicated here from the Architecture Atom
	// for consistency sake.
}

func (f *FieldAlignments) Convert(ctx log.Context, out atom_pb.Handler) error {
	return out(ctx, &atom_pb.FieldAlignments{
		CharAlignment:    f.CharAlignment,
		IntAlignment:     f.IntAlignment,
		U32Alignment:     f.U32Alignment,
		U64Alignment:     f.U64Alignment,
		PointerAlignment: f.PointerAlignment,
	})
}

func FieldAlignmentsFrom(from *atom_pb.FieldAlignments) FieldAlignments {
	return FieldAlignments{
		CharAlignment:    from.CharAlignment,
		IntAlignment:     from.IntAlignment,
		U32Alignment:     from.U32Alignment,
		U64Alignment:     from.U64Alignment,
		PointerAlignment: from.PointerAlignment,
	}
}
