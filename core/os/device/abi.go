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
	"strings"
)

var (
	UnknownABI = abi("unknown", UnknownOS, UnknownArchitecture, &MemoryLayout{})
	// Keep this one in, some applications incorrectly advertise arm, when they mean armv7
	AndroidARM      = abi("armeabi", Android, ARMv7a, ARMv7aLayout)
	AndroidARMv7a   = abi("armeabi-v7a", Android, ARMv7a, ARMv7aLayout)
	AndroidARM64v8a = abi("arm64-v8a", Android, ARMv8a, ARM64v8aLayout)
	AndroidX86      = abi("x86", Android, X86, X86IA32Layout)
	AndroidX86_64   = abi("x86-64", Android, X86_64, X86_64Layout)
	AndroidMIPS     = abi("mips", Android, MIPS, Little32)
	AndroidMIPS64   = abi("mips64", Android, MIPS64, Little64)

	LinuxX86_64   = abi("linux_x64", Linux, X86_64, Little64)
	OSXX86_64     = abi("osx_x64", OSX, X86_64, Little64)
	WindowsX86_64 = abi("windows_x64", Windows, X86_64, Little64)

	FuchsiaARM64 = abi("fuchsia_aarch64", Fuchsia, ARMv8a, ARM64v8aLayout)
)

var androidAbiByName = map[string]*ABI{}

// abi is the private helper constructor used by the constants above. It also
// registers the Android ABIs in a map for AndroidABIByName.
func abi(name string, os OSKind, arch Architecture, ml *MemoryLayout) *ABI {
	abi := &ABI{
		Name:         name,
		OS:           os,
		Architecture: arch,
		MemoryLayout: ml,
	}

	if os == Android {
		if _, ok := androidAbiByName[name]; ok {
			panic("Duplicate Android ABI name: " + name)
		}
		androidAbiByName[name] = abi
	}
	return abi
}

// AndroidABIByName returns the Android ABI that matches the provided human name.
// If there is no standard ABI that matches the name, then the returned ABI will have an UnknownOS and
// UnknownArchitecture.
func AndroidABIByName(name string) *ABI {
	abi, ok := androidAbiByName[name]
	if !ok {
		abi = &ABI{
			Name:         name,
			OS:           UnknownOS,
			Architecture: UnknownArchitecture,
		}
	}
	return abi
}

// SameAs returns true if the two abi objects are a match.
// This is intended for matching an executable binary against a target ABI, so
// ABI's are a match if both their os and architecture are the same, and the
// name and memory layout are not considered.
func (a *ABI) SameAs(o *ABI) bool {
	if a == nil {
		a = UnknownABI
	}
	if o == nil {
		o = UnknownABI
	}
	return (a.OS == o.OS) && (a.Architecture == o.Architecture)
}

// SupportsABI returns true if the configuration supports the specified ABI.
func (c *Configuration) SupportsABI(abi *ABI) bool {
	for _, t := range c.ABIs {
		if t.SameAs(abi) {
			return true
		}
	}
	return false
}

// PreferredABI returns the first device-compatible ABI found in abis.
// If the device has no ABI support then UnknownABI is returned.
// If there are no compatible ABIs found between the device and the list then
// the first device ABI is returned.
func (c *Configuration) PreferredABI(abis []*ABI) *ABI {
	for _, preferred := range c.ABIs {
		for _, abi := range abis {
			if preferred.SameAs(abi) {
				return preferred
			}
		}
	}

	// Return the first ABI that is not a hardware ASAN ABI.
	for _, abi := range c.ABIs {
		if !strings.HasSuffix(abi.Name, "-hwasan") {
			return abi
		}
	}
	return UnknownABI
}
