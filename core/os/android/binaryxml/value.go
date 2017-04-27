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

package binaryxml

import (
	"fmt"

	"github.com/google/gapid/core/data/binary"
)

const valueSize = 8

type typedValue interface {
	fmt.Stringer
	encode(w binary.Writer)
}

type valIntDec int32
type valIntHex uint32
type valReference uint32
type valStringID stringPoolRef
type valNull uint32
type valIntBoolean bool
type valFloat float32
type valFloatPx float32
type valFloatDp float32
type valFloatSp float32
type valFloatPt float32
type valFloatIn float32
type valFloatMm float32

func (v valIntDec) String() string    { return fmt.Sprintf("%d", int32(v)) }
func (v valIntHex) String() string    { return fmt.Sprintf("0x%x", uint32(v)) }
func (v valReference) String() string { return fmt.Sprintf("@0x%x", uint32(v)) }
func (v valStringID) String() string  { return fmt.Sprintf("@0x%x", stringPoolRef(v).stringPoolIndex()) }
func (v valFloat) String() string     { return fmt.Sprintf("%f", float32(v)) }
func (v valFloatPx) String() string   { return fmt.Sprintf("%fpx", float32(v)) }
func (v valFloatDp) String() string   { return fmt.Sprintf("%fdp", float32(v)) }
func (v valFloatSp) String() string   { return fmt.Sprintf("%fsp", float32(v)) }
func (v valFloatPt) String() string   { return fmt.Sprintf("%fpt", float32(v)) }
func (v valFloatIn) String() string   { return fmt.Sprintf("%fin", float32(v)) }
func (v valFloatMm) String() string   { return fmt.Sprintf("%fmm", float32(v)) }

func (v valIntBoolean) String() string { return fmt.Sprintf("%t", bool(v)) }
func (v valNull) String() string {
	return "null" /* Not actually sure about this: 0 -> undefined, !=0 -> empty */
}

// https://android.googlesource.com/platform/frameworks/base/+/master/libs/androidfw/ResourceTypes.cpp
func encodeDimension(w binary.Writer, f float32, unit uint8) {
	neg := f < 0
	if neg {
		f = -f
	}
	bits := uint64(f*(1<<23) + .5)
	radix := uint32(0)
	shift := uint32(0)
	if (bits & 0x7fffff) == 0 {
		radix = 0
		shift = 23
	} else if (bits & 0xffffffffff800000) == 0 {
		radix = 3
		shift = 0
	} else if (bits & 0xffffffff80000000) == 0 {
		radix = 2
		shift = 8
	} else if (bits & 0xffffff8000000000) == 0 {
		radix = 1
		shift = 16
	} else {
		radix = 0
		shift = 23
	}
	mantissa := (int32)(bits>>shift) & 0xFFFFFF
	if neg {
		mantissa = (-mantissa) & 0xFFFFFF
	}

	e := uint32((radix << 4) | uint32(mantissa<<8) | uint32(unit))
	w.Uint32(e)
}

func (v valIntDec) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeIntDec)
	w.Int32(int32(v))
}
func (v valIntHex) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeIntHex)
	w.Uint32(uint32(v))
}
func (v valReference) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeReference)
	w.Uint32(uint32(v))
}
func (v valStringID) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeString)
	w.Uint32(stringPoolRef(v).stringPoolIndex())
}
func (v valFloat) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeFloat)
	w.Float32(float32(v))
}
func (v valFloatPx) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 0)
}
func (v valFloatDp) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 1)
}
func (v valFloatSp) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 2)
}
func (v valFloatPt) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 3)
}
func (v valFloatIn) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 4)
}
func (v valFloatMm) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeDimension)
	encodeDimension(w, float32(v), 5)
}
func (v valIntBoolean) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeIntBoolean)
	if bool(v) {
		w.Uint32(0xFFFFFFFF)
	} else {
		w.Uint32(0)
	}
}
func (v valNull) encode(w binary.Writer) {
	writeTypedValueHeader(w, typeNull)
	w.Uint32(uint32(v))
}

func writeTypedValueHeader(w binary.Writer, ty valueType) {
	w.Uint16(8)
	w.Uint8(0)
	w.Uint8(uint8(ty))
}

func decodeDimension(r binary.Reader) (typedValue, error) {
	radixes := []float32{1.0 / (1 << 8),
		1.0 / (1 << 15),
		1.0 / (1 << 23),
		1.0 / (1 << 31)}
	v := r.Uint32()
	fval := float32(int32(v&0xffffff00)) * radixes[(v>>4)&0x3]
	switch v & 0xf {
	case 0:
		return valFloatPx(fval), nil
	case 1:
		return valFloatDp(fval), nil
	case 2:
		return valFloatSp(fval), nil
	case 3:
		return valFloatPt(fval), nil
	case 4:
		return valFloatIn(fval), nil
	case 5:
		return valFloatMm(fval), nil
	default:
		return nil, fmt.Errorf("Unknown dimension format %d", v&0xf)
	}
}

func decodeValue(r binary.Reader, xml *xmlTree) (typedValue, error) {
	size := r.Uint16()
	if size != valueSize {
		return nil, fmt.Errorf("Value size was not as expected. Got %d, expected %d",
			size, valueSize)
	}
	res0 := r.Uint8()
	if res0 != 0 {
		return nil, fmt.Errorf("res0 was %d, expected 0", res0)
	}
	ty := valueType(r.Uint8())
	switch ty {
	case typeIntDec:
		return valIntDec(r.Int32()), nil
	case typeIntHex:
		return valIntHex(r.Uint32()), nil
	case typeReference:
		return valReference(r.Uint32()), nil
	case typeString:
		return valStringID(xml.decodeString(r)), nil
	case typeFloat:
		return valFloat(r.Float32()), nil
	case typeIntBoolean:
		return valIntBoolean(r.Uint32() != 0), nil
	case typeNull:
		return valNull(r.Uint32()), nil
	case typeDimension:
		return decodeDimension(r)
	default:
		return nil, fmt.Errorf("Value type %v not implemented", ty)
	}
}

type valueType uint8

const (
	typeNull             valueType = 0x00
	typeReference        valueType = 0x01
	typeAttribute        valueType = 0x02
	typeString           valueType = 0x03
	typeFloat            valueType = 0x04
	typeDimension        valueType = 0x05
	typeFraction         valueType = 0x06
	typeDynamicReference valueType = 0x07
	typeIntDec           valueType = 0x10
	typeIntHex           valueType = 0x11
	typeIntBoolean       valueType = 0x12
	typeIntColorARGB8    valueType = 0x1c
	typeIntColorRGB8     valueType = 0x1d
	typeIntColorARGB4    valueType = 0x1e
	typeIntColorRGB4     valueType = 0x1f
)

func (t valueType) String() string {
	switch t {
	case typeNull:
		return "Null"
	case typeReference:
		return "Reference"
	case typeAttribute:
		return "Attribute"
	case typeString:
		return "String"
	case typeFloat:
		return "Float"
	case typeDimension:
		return "Dimension"
	case typeFraction:
		return "Fraction"
	case typeDynamicReference:
		return "DynamicReference"
	case typeIntDec:
		return "IntDec"
	case typeIntHex:
		return "IntHex"
	case typeIntBoolean:
		return "IntBoolean"
	case typeIntColorARGB8:
		return "IntColorARGB8"
	case typeIntColorRGB8:
		return "IntColorRGB8"
	case typeIntColorARGB4:
		return "IntColorARGB4"
	case typeIntColorRGB4:
		return "IntColorRGB4"
	default:
		return fmt.Sprintf("type<%d>", t)
	}
}
