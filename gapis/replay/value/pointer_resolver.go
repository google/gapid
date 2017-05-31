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

package value

import "github.com/google/gapid/gapis/replay/protocol"

// PointerResolver is used to translate pointers into the volatile address-space.
type PointerResolver interface {
	// ResolveTemporaryPointer returns the temporary address-space pointer ptr
	// translated to volatile address-space.
	ResolveTemporaryPointer(TemporaryPointer) VolatilePointer

	// ResolveObservedPointer returns the pointer translated to volatile or
	// absolute address-space.
	ResolveObservedPointer(ObservedPointer) (protocol.Type, uint64)

	// ResolvePointerIndex returns the pointer to the pointer with the specified
	// index.
	ResolvePointerIndex(PointerIndex) (protocol.Type, uint64)
}
