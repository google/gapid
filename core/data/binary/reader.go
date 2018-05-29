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
	"io"

	"github.com/google/gapid/core/math/f16"
)

// Reader provides methods for decoding values.
type Reader interface {
	io.Reader
	// Data reads the data bytes in their entirety.
	Data([]byte)
	// Bool decodes and returns a boolean value from the Reader.
	Bool() bool
	// Int8 decodes and returns a signed, 8 bit integer value from the Reader.
	Int8() int8
	// Uint8 decodes and returns an unsigned, 8 bit integer value from the Reader.
	Uint8() uint8
	// Int16 decodes and returns a signed, 16 bit integer value from the Reader.
	Int16() int16
	// Uint16 decodes and returns an unsigned, 16 bit integer value from the Reader.
	Uint16() uint16
	// Int32 decodes and returns a signed, 32 bit integer value from the Reader.
	Int32() int32
	// Uint32 decodes and returns an unsigned, 32 bit integer value from the Reader.
	Uint32() uint32
	// Float16 decodes and returns a 16 bit floating-point value from the Reader.
	Float16() f16.Number
	// Float32 decodes and returns a 32 bit floating-point value from the Reader.
	Float32() float32
	// Int64 decodes and returns a signed, 64 bit integer value from the Reader.
	Int64() int64
	// Uint64 decodes and returns an unsigned, 64 bit integer value from the Reader.
	Uint64() uint64
	// Float64 decodes and returns a 64 bit floating-point value from the Reader.
	Float64() float64
	// String decodes and returns a string from the Reader.
	String() string
	// Decode a collection count from the stream.
	Count() uint32
	// If there is an error reading any input, all further reading returns the
	// zero value of the type read. Error() returns the error which stopped
	// reading from the stream. If reading has not stopped it returns nil.
	Error() error
	// Set the error state and stop reading from the stream.
	SetError(error)
}

// ReadUint reads an unsigned integer of either 8, 16, 32 or 64 bits from r,
// returning the result as a uint64.
func ReadUint(r Reader, bits int32) uint64 {
	switch bits {
	case 8:
		return uint64(r.Uint8())
	case 16:
		return uint64(r.Uint16())
	case 32:
		return uint64(r.Uint32())
	case 64:
		return r.Uint64()
	default:
		r.SetError(fmt.Errorf("Unsupported integer bit count %v", bits))
		return 0
	}
}

// ReadInt reads a signed integer of either 8, 16, 32 or 64 bits from r,
// returning the result as a int64.
func ReadInt(r Reader, bits int32) int64 {
	switch bits {
	case 8:
		return int64(r.Int8())
	case 16:
		return int64(r.Int16())
	case 32:
		return int64(r.Int32())
	case 64:
		return r.Int64()
	default:
		r.SetError(fmt.Errorf("Unsupported integer bit count %v", bits))
		return 0
	}
}

// ConsumeBytes reads and throws away a number of bytes from r, returning the
// number of bytes it consumed.
func ConsumeBytes(r Reader, bytes uint64) uint64 {
	for i := uint64(0); i < bytes; i++ {
		r.Uint8()
	}
	return bytes
}
