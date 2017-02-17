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

// Package cyclic implements binary.Encoder and binary.Decoder with fully
// support for cyclic graphs of objects.

// Object encoding details
//
// Objects are encoded in three different forms, depending on whether the object
// was nil, or was already encoded to the stream. If the object is non-nil and
// is encoded for the first time for a given Encoder then the object is encoded
// as:
//   key  uint32   // A unique identifier for the object instance
//   type [20]byte // A unique identifier for the type of the object
//   ...data...    // The object's data (length dependent on the object type)
//
// All subsequent encodings of the object are encoded as the 16 bit key
// identifier without any additional data:
//   key uint32 // An identifier of a previously encoded object instance
//
// If the object is nil, then the object is encoded as a uint32 of 0.
//
package cyclic
