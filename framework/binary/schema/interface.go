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

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/registry"
)

// Interface is the Type descriptor for a field who's underlying type is dynamic.
type Interface struct {
	Name string // The simple name of the type.
}

func (i *Interface) Representation() string {
	return fmt.Sprintf("%r", i)
}

func (i *Interface) String() string {
	return fmt.Sprint(i)
}

// Format implements the fmt.Formatter interface
func (i *Interface) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprint(f, "?")
	default:
		fmt.Fprint(f, i.Name)
	}
}

func (i *Interface) EncodeValue(e binary.Encoder, value interface{}) {
	if value != nil { // TODO proper nil test needed?
		e.Object(value.(binary.Object))
	} else {
		e.Object(nil)
	}
}

func (i *Interface) DecodeValue(d binary.Decoder) interface{} {
	return d.Object()
}

func (i *Interface) Subspace() *binary.Subspace {
	return nil
}

func (i *Interface) HasSubspace() bool {
	return false
}

func (*Interface) IsPOD() bool {
	return false
}

func (*Interface) IsSimple() bool {
	return false
}

func interfaceFactory(t reflect.Type, tag reflect.StructTag, f binary.MakeTypeFun, pkg string) binary.Type {
	if !t.Implements(reflect.TypeOf((*binary.Object)(nil)).Elem()) {
		return &Any{}
	}
	return &Interface{Name: binary.TypeName(t, pkg)}
}

// Variant is the Type descriptor for a field who's underlying type is dynamic, but is encoded with Variant not Object.
type Variant struct {
	Name string // The simple name of the type.
}

func (i *Variant) Representation() string {
	return fmt.Sprintf("%r", i)
}

func (i *Variant) String() string {
	return fmt.Sprint(i)
}

// Format implements the fmt.Formatter interface
func (i *Variant) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprint(f, "&")
	default:
		fmt.Fprint(f, i.Name)
	}
}

func (i *Variant) EncodeValue(e binary.Encoder, value interface{}) {
	if value != nil { // TODO proper nil test needed?
		e.Variant(value.(binary.Object))
	} else {
		e.Variant(nil)
	}
}

func (i *Variant) DecodeValue(d binary.Decoder) interface{} {
	return d.Object()
}

func (i *Variant) Subspace() *binary.Subspace {
	return nil
}

func (i *Variant) HasSubspace() bool {
	return false
}

func (*Variant) IsPOD() bool {
	return false
}

func (*Variant) IsSimple() bool {
	return false
}

func variantFactory(t reflect.Type, tag reflect.StructTag, f binary.MakeTypeFun, pkg string) binary.Type {
	return &Variant{Name: binary.TypeName(t, pkg)}
}

func init() {
	registry.Factories.Add(reflect.Interface, interfaceFactory)
}
