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

// Bitness returns the natural bit width of the architecture.
// https://en.wiktionary.org/wiki/bitness
func (a Architecture) Bitness() int {
	switch a {
	case ARMv7a, X86, MIPS:
		return 32
	case ARMv8a, X86_64, MIPS64:
		return 64
	default:
		return 0
	}
}

// U64Alignment returns the alignment of 64-bit structs for the archiecture
func (a Architecture) U64Alignment() int32 {
	switch a {
	case X86:
		return 4
	case ARMv7a, MIPS, ARMv8a, X86_64, MIPS64:
		return 8
	default:
		return 0
	}
}

// MemoryLayout returns the default memory layout for this architecture.
// This may be overridden by the active ABI for any given device (32 bit on a 64 bit chip for instance)
func (a Architecture) MemoryLayout() *MemoryLayout {
	width := int32(a.Bitness() / 8)
	return &MemoryLayout{
		PointerAlignment: width,
		PointerSize:      width,
		IntegerSize:      width,
		SizeSize:         width,
		U64Alignment:     a.U64Alignment(),
		Endian:           LittleEndian,
	}
}

var architectureByName = map[string]Architecture{
	// possible values of runtime.GOARCH
	"386":   X86,
	"amd64": X86_64,
	"arm":   ARMv7a,
}

// ArchitectureByName returns the Architecture for the supplied human name.
// There is no guarantee that ArchitectureByName(name).String() == name, as multiple human names map to the same
// canonical Architecture.
// If the architecture name is not know, it returns UnknownArchitecture
func ArchitectureByName(name string) Architecture {
	if arch, ok := architectureByName[name]; ok {
		return arch
	}
	// fallback to enum name
	if arch, ok := Architecture_value[name]; ok {
		return Architecture(arch)
	}
	return UnknownArchitecture
}
