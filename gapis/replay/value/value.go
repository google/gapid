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

// Package value contains the value types used by the replay virtual machine.
//
// Each numerical and boolean value type is backed by a corresponding primitive
// Go type for convenience of construction and usage. Pointer values can belong
// to various different address spaces, and for compatibility with both 32 and
// 64 bit architectures, are all backed by uint64.
package value

import "github.com/google/gapid/gapis/replay/protocol"

// Value is the interface for all values to be passed either in opcodes or
// constant memory to the replay virtual machine.
type Value interface {
	// Get returns the protocol type and the bit-representation of the value.
	// For example a boolean value would either be 0 or 1, a uint32 value would be
	// zero-extended, a float64 would be the IEEE 754 representation
	// reinterpreted as a uint64.
	// If onStack returns true then the value is stored on the top of the VM
	// stack, and val should be ignored.
	Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool)
}

// Pointer is a pointer-typed Value.
type Pointer interface {
	Value

	// Offset returns the Pointer offset by v.
	Offset(v uint64) Pointer

	// IsValid returns true if the pointer is within acceptable ranges.
	IsValid() bool
}
