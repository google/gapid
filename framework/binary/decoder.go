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

package binary

import "github.com/google/gapid/core/data/pod"

// Decoder extends Reader with additional methods for decoding objects.
type Decoder interface {
	pod.Reader
	// Entity supports reading a binary.Entity from the stream.
	Entity() *Entity
	// Struct decodes an sub-structure from the stream. The type of the
	// sub-structure in the stream depends on the state of the decoder.
	// It must be compatible with the object passed or it will panic.
	Struct(Object)
	// Variant decodes and returns an Object from the stream. The Class in the
	// stream must have been previously registered with binary.registry.Add.
	Variant() Object
	// Object decodes and returns an Object from the stream. Object instances
	// that were encoded multiple times may be decoded and returned as a shared,
	// single instance. The Class in the stream must have been previously
	// registered with binary.registry.Add.
	Object() Object
	// Lookup the upgrade decoder for decoding this type of entity.
	Lookup(*Entity) UpgradeDecoder
	// GetMode gets the current mode of the decoder.
	GetMode() Mode
}
