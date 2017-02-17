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

// Package binary implements encoding and decoding of various primitive data
// types to and from a binary stream. The package holds BitStream for packing
// and unpacking sequences of bits, Float16 for dealing with 16 bit floating-
// point values and Reader/Writer for encoding and decoding various value types
// to a binary stream. There is also the higher level Encoder/Decoder that can
// be used for serializing object hierarchies.
//
// The binary package defines a POD type as something with fixed size and
// layout. This allows all primitive types, fixed size arrays of POD types, and
// structures that contain only POD types.
// A Simple type is allowed more structure than a POD type, it may vary in
// size, which allows slices and maps, but must be of known type and not be
// able to contain cycles (thus no pointers, interfaces or slices of slices).
//
// pod.Reader and pod.Writer provide a symmetrical pair of methods for
// encoding and decoding various data types to a binary stream.
// They can encode all POD and Simple types, but not complex structures.
// For performance reasons, each data type has a separate method for encoding
// and decoding rather than having a single pair of methods encoding and
// decoding boxed values in an interface{}.
//
// binary.Encoder and binary.Decoder extend the pod.Reader and pod.Writer
// interfaces by also providing a symmetrical pair of methods for encoding and
// decoding object types.
//
package binary

// binary: Schema = false
