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
	"github.com/google/gapid/core/os/device"
)

// Slice is the interface implemented by types that represent a slice on
// a memory pool.
type Slice interface {
	// Root returns the original pointer this slice derives from.
	Root() uint64

	// Base returns the address of first element.
	Base() uint64

	// Count returns the number of elements in the slice.
	Count() uint64

	// Pool returns the the pool identifier.
	Pool() PoolID

	// ElementSize returns the size in bytes of a single element in the slice.
	ElementSize(*device.MemoryLayout) uint64

	// Range returns the memory range this slice represents in the underlying pool.
	Range(*device.MemoryLayout) Range
}
