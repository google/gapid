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

package schema

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
)

// Method denotes the encoding/decoding method a primitive type will use.
type Method uint8

const (
	Bool Method = iota
	Int8
	Uint8
	Int16
	Uint16
	Int32
	Uint32
	Int64
	Uint64
	Float32
	Float64
	String
)

var (
	methodToString = map[Method]string{
		Bool:    "Bool",
		Int8:    "Int8",
		Uint8:   "Uint8",
		Int16:   "Int16",
		Uint16:  "Uint16",
		Int32:   "Int32",
		Uint32:  "Uint32",
		Int64:   "Int64",
		Uint64:  "Uint64",
		Float32: "Float32",
		Float64: "Float64",
		String:  "String",
	}
	stringToMethod = map[string]Method{}
	methodToBase   = map[Method]string{
		Bool:    "bool",
		Int8:    "int8",
		Uint8:   "uint8",
		Int16:   "int16",
		Uint16:  "uint16",
		Int32:   "int32",
		Uint32:  "uint32",
		Int64:   "int64",
		Uint64:  "uint64",
		Float32: "float32",
		Float64: "float64",
		String:  "string",
	}
)

func init() {
	for m, s := range methodToString {
		stringToMethod[s] = m
	}
	f := registry.Factories
	f.Add(reflect.Bool, Bool.factory)
	f.Add(reflect.Int8, Int8.factory)
	f.Add(reflect.Uint8, Uint8.factory)
	f.Add(reflect.Int16, Int16.factory)
	f.Add(reflect.Uint16, Uint16.factory)
	f.Add(reflect.Int32, Int32.factory)
	f.Add(reflect.Uint32, Uint32.factory)
	f.Add(reflect.Int64, Int64.factory)
	f.Add(reflect.Uint64, Uint64.factory)
	f.Add(reflect.Float32, Float32.factory)
	f.Add(reflect.Float64, Float64.factory)
	f.Add(reflect.String, String.factory)
	// It is with great sadness that I write to inform you that some of your
	// bits may go missing. For compatibility with codergen.
	f.Add(reflect.Int, Int32.factory)
	f.Add(reflect.Uint, Uint32.factory)
}

func (m Method) factory(t reflect.Type, tag reflect.StructTag, f binary.MakeTypeFun, pkg string) binary.Type {
	return &Primitive{Name: binary.TypeName(t, pkg), Method: m}
}

// Primitive is the kind for primitive types with corresponding direct methods on
// Encoder and Decoder
type Primitive struct {
	Name   string // The simple name of the type.
	Method Method // The enocde/decode method to use.
}

// Native returns the go native type name for this primitive.
func (p *Primitive) Native() string {
	return strings.ToLower(p.Method.String())
}

func (p *Primitive) Representation() string {
	return fmt.Sprintf("%r", p)
}

func (p *Primitive) String() string {
	return fmt.Sprint(p)
}

// Format implements the fmt.Formatter interface
func (p *Primitive) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprint(f, p.Method)
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprint(f, methodToBase[p.Method])
	default:
		if p.Name == "" {
			fmt.Fprint(f, p.Method)
		} else {
			fmt.Fprint(f, p.Name)
		}
	}
}

func (p *Primitive) EncodeValue(e binary.Encoder, value interface{}) {
	switch p.Method {
	case Bool:
		e.Bool(value.(bool))
	case Int8:
		e.Int8(value.(int8))
	case Uint8:
		e.Uint8(value.(uint8))
	case Int16:
		e.Int16(value.(int16))
	case Uint16:
		e.Uint16(value.(uint16))
	case Int32:
		e.Int32(value.(int32))
	case Uint32:
		e.Uint32(value.(uint32))
	case Int64:
		e.Int64(value.(int64))
	case Uint64:
		e.Uint64(value.(uint64))
	case Float32:
		e.Float32(value.(float32))
	case Float64:
		e.Float64(value.(float64))
	case String:
		e.String(value.(string))
	default:
		e.SetError(fmt.Errorf("Unknown encode method %q", p.Method))
	}
}

func (p *Primitive) DecodeValue(d binary.Decoder) interface{} {
	switch p.Method {
	case Bool:
		return d.Bool()
	case Int8:
		return d.Int8()
	case Uint8:
		return d.Uint8()
	case Int16:
		return d.Int16()
	case Uint16:
		return d.Uint16()
	case Int32:
		return d.Int32()
	case Uint32:
		return d.Uint32()
	case Int64:
		return d.Int64()
	case Uint64:
		return d.Uint64()
	case Float32:
		return d.Float32()
	case Float64:
		return d.Float64()
	case String:
		return d.String()
	default:
		d.SetError(fmt.Errorf("Unknown decode method %q", p.Method))
		return nil
	}
}

func (p *Primitive) Subspace() *binary.Subspace {
	return nil
}

func (p *Primitive) HasSubspace() bool {
	return false
}

func (*Primitive) IsPOD() bool {
	return true
}

func (*Primitive) IsSimple() bool {
	return true
}

// This will convert a string to a Method, or return an error if the string was
// not a valid method name.
func ParseMethod(s string) (Method, error) {
	if m, ok := stringToMethod[s]; ok {
		return m, nil
	}
	return 0, fmt.Errorf("Invalid Method name %s", s)
}

func (m Method) String() string {
	if s, ok := methodToString[m]; ok {
		return s
	}
	return fmt.Sprintf("Method(%d)", m)
}
