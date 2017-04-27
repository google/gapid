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

import (
	"fmt"

	"github.com/google/gapid/core/math/f16"
)

// Writer provides methods for encoding values.
type Writer interface {
	// Data writes the data bytes in their entirety.
	Data([]byte)
	// Bool encodes a boolean value to the Writer.
	Bool(bool)
	// Int8 encodes a signed, 8 bit integer value to the Writer.
	Int8(int8)
	// Uint8 encodes an unsigned, 8 bit integer value to the Writer.
	Uint8(uint8)
	// Int16 encodes a signed, 16 bit integer value to the Writer.
	Int16(int16)
	// Uint16 encodes an unsigned, 16 bit integer value to the Writer.
	Uint16(uint16)
	// Int32 encodes a signed, 32 bit integer value to the Writer.
	Int32(int32)
	// Uint32 encodes an usigned, 32 bit integer value to the Writer.
	Uint32(uint32)
	// Float16 encodes a 16 bit floating-point value to the Writer.
	Float16(f16.Number)
	// Float32 encodes a 32 bit floating-point value to the Writer.
	Float32(float32)
	// Int64 encodes a signed, 64 bit integer value to the Writer.
	Int64(int64)
	// Uint64 encodes an unsigned, 64 bit integer value to the Encoders's io.Writer.
	Uint64(uint64)
	// Float64 encodes a 64 bit floating-point value to the Writer.
	Float64(float64)
	// String encodes a string to the Writer.
	String(string)
	// Simple encodes a Writable type to the Writer.
	Simple(Writable)
	// If there is an error writing any output, all further writing becomes
	// a no-op. Error() returns the error which stopped writing to the stream.
	// If writing has not stopped it returns nil.
	Error() error
	// Set the error state and stop writing to the stream.
	SetError(error)
}

// WriteUint writes the unsigned integer v of either 8, 16, 32 or 64 bits to w.
func WriteUint(w Writer, bits int32, v uint64) {
	switch bits {
	case 8:
		w.Uint8(uint8(v))
	case 16:
		w.Uint16(uint16(v))
	case 32:
		w.Uint32(uint32(v))
	case 64:
		w.Uint64(uint64(v))
	default:
		w.SetError(fmt.Errorf("Unsupported integer bit count %v", bits))
	}
}

// WriteInt writes the signed integer v of either 8, 16, 32 or 64 bits to w.
func WriteInt(w Writer, bits int32, v int64) {
	switch bits {
	case 8:
		w.Int8(int8(v))
	case 16:
		w.Int16(int16(v))
	case 32:
		w.Int32(int32(v))
	case 64:
		w.Int64(int64(v))
	default:
		w.SetError(fmt.Errorf("Unsupported integer bit count %v", bits))
	}
}

// WriteBytes writes the given v for count times to writer w.
func WriteBytes(w Writer, v uint8, count int32) {
	for i := int32(0); i < count; i++ {
		w.Uint8(v)
	}
}
