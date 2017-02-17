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

	"github.com/google/gapid/core/data/pod"
)

const valueSize = 8

type typedValue interface {
	fmt.Stringer
	encode(w pod.Writer)
}

type valIntDec int32
type valIntHex uint32
type valReference uint32
type valStringID stringPoolRef
type valNull uint32
type valIntBoolean bool
type valFloat float32

func (v valIntDec) String() string     { return fmt.Sprintf("%d", int32(v)) }
func (v valIntHex) String() string     { return fmt.Sprintf("0x%x", uint32(v)) }
func (v valReference) String() string  { return fmt.Sprintf("@0x%x", uint32(v)) }
func (v valStringID) String() string   { return fmt.Sprintf("@0x%x", stringPoolRef(v).stringPoolIndex()) }
func (v valFloat) String() string      { return fmt.Sprintf("%f", float32(v)) }
func (v valIntBoolean) String() string { return fmt.Sprintf("%t", bool(v)) }
func (v valNull) String() string {
	return "null" /* Not actually sure about this: 0 -> undefined, !=0 -> empty */
}

func (v valIntDec) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeIntDec)
	w.Int32(int32(v))
}
func (v valIntHex) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeIntHex)
	w.Uint32(uint32(v))
}
func (v valReference) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeReference)
	w.Uint32(uint32(v))
}
func (v valStringID) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeString)
	w.Uint32(stringPoolRef(v).stringPoolIndex())
}
func (v valFloat) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeFloat)
	w.Float32(float32(v))
}
func (v valIntBoolean) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeIntBoolean)
	if bool(v) {
		w.Uint32(0xFFFFFFFF)
	} else {
		w.Uint32(0)
	}
}
func (v valNull) encode(w pod.Writer) {
	writeTypedValueHeader(w, typeNull)
	w.Uint32(uint32(v))
}

func writeTypedValueHeader(w pod.Writer, ty valueType) {
	w.Uint16(8)
	w.Uint8(0)
	w.Uint8(uint8(ty))
}

func decodeValue(r pod.Reader, xml *xmlTree) (typedValue, error) {
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
