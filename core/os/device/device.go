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

package device

import (
	"github.com/golang/protobuf/proto"
)

const (
	UnknownArchitecture = Architecture_UnknownArchitecture
	ARMv7a              = Architecture_ARMv7a
	ARMv8a              = Architecture_ARMv8a
	X86                 = Architecture_X86
	X86_64              = Architecture_X86_64
	MIPS                = Architecture_MIPS
	MIPS64              = Architecture_MIPS64
)

const (
	UnknownEndian = Endian_UnknownEndian
	BigEndian     = Endian_BigEndian
	LittleEndian  = Endian_LittleEndian
)

const (
	UnknownOS = OSKind_UnknownOS
	Windows   = OSKind_Windows
	OSX       = OSKind_OSX
	Linux     = OSKind_Linux
	Android   = OSKind_Android
)

var (
	// ARMv7aLayout is the memory layout for the armv7a ABI.
	// http://infocenter.arm.com/help/topic/com.arm.doc.ihi0042f/IHI0042F_aapcs.pdf
	// 4 DATA TYPES AND ALIGNMENT
	ARMv7aLayout = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 4, Alignment: 4},
		Integer: &DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}

	// ARM64v8aLayout is the memory layout for the arm64v8a ABI.
	// http://infocenter.arm.com/help/topic/com.arm.doc.ihi0055b/IHI0055B_aapcs64.pdf
	// 4 DATA TYPES AND ALIGNMENT
	ARM64v8aLayout = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 8, Alignment: 8},
		Integer: &DataTypeLayout{Size: 8, Alignment: 8},
		Size:    &DataTypeLayout{Size: 8, Alignment: 8},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}

	// X86IA32Layout is the memory layout for the x86 IA-32 ABI.
	X86IA32Layout = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 4, Alignment: 4},
		Integer: &DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 4},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 4},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}

	// X86_64Layout is the memory layout for the x86_64 ABI.
	X86_64Layout = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 8, Alignment: 8},
		Integer: &DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &DataTypeLayout{Size: 8, Alignment: 8},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}

	Little32 = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 4, Alignment: 4},
		Integer: &DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}
	Little64 = &MemoryLayout{
		Endian:  LittleEndian,
		Pointer: &DataTypeLayout{Size: 8, Alignment: 8},
		Integer: &DataTypeLayout{Size: 8, Alignment: 8},
		Size:    &DataTypeLayout{Size: 8, Alignment: 8},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}
	Big32 = &MemoryLayout{
		Endian:  BigEndian,
		Pointer: &DataTypeLayout{Size: 4, Alignment: 4},
		Integer: &DataTypeLayout{Size: 4, Alignment: 4},
		Size:    &DataTypeLayout{Size: 4, Alignment: 4},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}
	Big64 = &MemoryLayout{
		Endian:  BigEndian,
		Pointer: &DataTypeLayout{Size: 8, Alignment: 8},
		Integer: &DataTypeLayout{Size: 8, Alignment: 8},
		Size:    &DataTypeLayout{Size: 8, Alignment: 8},
		Char:    &DataTypeLayout{Size: 1, Alignment: 1},
		I64:     &DataTypeLayout{Size: 8, Alignment: 8},
		I32:     &DataTypeLayout{Size: 4, Alignment: 4},
		I16:     &DataTypeLayout{Size: 2, Alignment: 2},
		I8:      &DataTypeLayout{Size: 1, Alignment: 1},
		F64:     &DataTypeLayout{Size: 8, Alignment: 8},
		F32:     &DataTypeLayout{Size: 4, Alignment: 4},
		F16:     &DataTypeLayout{Size: 2, Alignment: 2},
	}
)

// Clone returns a new MemoryLayout copied from m.
func (m *MemoryLayout) Clone() *MemoryLayout {
	var out MemoryLayout
	bytes, _ := proto.Marshal(m)
	proto.Unmarshal(bytes, &out)
	return &out
}

// SameAs returns true if the MemoryLayouts are equal.
func (m *MemoryLayout) SameAs(o *MemoryLayout) bool {
	return proto.Equal(m, o)
}
