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

package stream

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/protoutil"
)

var (
	// U1 represents a 1-bit unsigned integer.
	U1 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 1}}}
	// U2 represents a 2-bit unsigned integer.
	U2 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 2}}}
	// U4 represents a 4-bit unsigned integer.
	U4 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 4}}}
	// U5 represents a 5-bit unsigned integer.
	U5 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 5}}}
	// U6 represents a 6-bit unsigned integer.
	U6 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 6}}}
	// U8 represents a 8-bit unsigned integer.
	U8 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 8}}}
	// U9 represents a 9-bit unsigned integer.
	U9 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 9}}}
	// U10 represents a 10-bit unsigned integer.
	U10 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 10}}}
	// U11 represents a 11-bit unsigned integer.
	U11 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 11}}}
	// U16 represents a 16-bit unsigned integer.
	U16 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 16}}}
	// U24 represents a 24-bit unsigned integer.
	U24 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 24}}}
	// U32 represents a 32-bit unsigned integer.
	U32 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 32}}}
	// U64 represents a 64-bit unsigned integer.
	U64 = DataType{Signed: false, Kind: &DataType_Integer{&Integer{Bits: 64}}}
	// S2 represents a 2-bit signed integer.
	S2 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 1}}}
	// S8 represents a 8-bit signed integer.
	S8 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 7}}}
	// S10 represents a 10-bit signed integer.
	S10 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 9}}}
	// S11 represents a 11-bit signed integer.
	S11 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 10}}}
	// S16 represents a 16-bit signed integer.
	S16 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 15}}}
	// S32 represents a 32-bit signed integer.
	S32 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 31}}}
	// S64 represents a 64-bit signed integer.
	S64 = DataType{Signed: true, Kind: &DataType_Integer{&Integer{Bits: 63}}}
	// F10 represents a 10-bit unsigned floating-point number.
	F10 = DataType{Signed: false, Kind: &DataType_Float{&Float{ExponentBits: 5, MantissaBits: 5}}}
	// F11 represents a 11-bit unsigned floating-point number.
	F11 = DataType{Signed: false, Kind: &DataType_Float{&Float{ExponentBits: 5, MantissaBits: 6}}}
	// F16 represents a 16-bit signed, floating-point number.
	F16 = DataType{Signed: true, Kind: &DataType_Float{&Float{ExponentBits: 5, MantissaBits: 10}}}
	// F32 represents a 32-bit signed, floating-point number.
	F32 = DataType{Signed: true, Kind: &DataType_Float{&Float{ExponentBits: 8, MantissaBits: 23}}}
	// F64 represents a 64-bit signed, floating-point number.
	F64 = DataType{Signed: true, Kind: &DataType_Float{&Float{ExponentBits: 11, MantissaBits: 52}}}
	// S16_16 represents a 16.16 bit signed, fixed-point number.
	S16_16 = DataType{Signed: true, Kind: &DataType_Fixed{&Fixed{IntegerBits: 15, FractionalBits: 16}}}
)

// Format prints the DataType to f.
func (t DataType) Format(f fmt.State, r rune) {
	switch {
	case t.Is(F10):
		fmt.Fprintf(f, "F10")
	case t.Is(F11):
		fmt.Fprintf(f, "F11")
	case t.Is(F16):
		fmt.Fprintf(f, "F16")
	case t.Is(F32):
		fmt.Fprintf(f, "F32")
	case t.Is(F64):
		fmt.Fprintf(f, "F64")
	case t.IsFloat() && t.Signed:
		fmt.Fprintf(f, "F:s:%d:%d", t.GetFloat().ExponentBits, t.GetFloat().MantissaBits)
	case t.IsFloat() && !t.Signed:
		fmt.Fprintf(f, "F:u:%d:%d", t.GetFloat().ExponentBits, t.GetFloat().MantissaBits)
	case t.IsInteger() && t.Signed:
		fmt.Fprintf(f, "S%d", t.GetInteger().Bits+1)
	case t.IsInteger() && !t.Signed:
		fmt.Fprintf(f, "U%d", t.GetInteger().Bits)
	case t.IsFixed() && t.Signed:
		fmt.Fprintf(f, "S%d.%d", t.GetFixed().IntegerBits+1, t.GetFixed().FractionalBits)
	case t.IsFixed() && !t.Signed:
		fmt.Fprintf(f, "U%d.%d", t.GetFixed().IntegerBits, t.GetFixed().FractionalBits)
	default:
		fmt.Fprintf(f, "<unknown kind %T>", t.Kind)
	}
}

// Bits returns the size of the data type in bits.
func (t *DataType) Bits() uint32 {
	bits := uint32(0)
	switch k := protoutil.OneOf(t.Kind).(type) {
	case *Integer:
		bits = k.Bits
	case *Float:
		bits = k.ExponentBits + k.MantissaBits
	case *Fixed:
		bits = k.IntegerBits + k.FractionalBits
	default:
		panic(fmt.Errorf("Unknown data type kind %T", k))
	}
	if t.Signed {
		bits++
	}
	return bits
}

// IsInteger returns true if t is an integer.
func (t *DataType) IsInteger() bool { return t.GetInteger() != nil }

// IsFloat returns true if t is a float.
func (t *DataType) IsFloat() bool { return t.GetFloat() != nil }

// IsFixed returns true if the DataType is a fixed point number.
func (t *DataType) IsFixed() bool { return t.GetFixed() != nil }

// Is returns true if t is equivalent to o.
func (t DataType) Is(o DataType) bool { return proto.Equal(&t, &o) }
