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

// Pointer is the Type descriptor for pointers.
type Pointer struct {
	Type binary.Type // The pointed to type.
}

func (p *Pointer) Representation() string {
	return fmt.Sprintf("%r", p)
}

func (p *Pointer) String() string {
	return fmt.Sprint(p)
}

// Format implements the fmt.Formatter interface
func (p *Pointer) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprintf(f, "*%z", p.Type)
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprintf(f, "*%r", p.Type)
	default:
		fmt.Fprintf(f, "*%v", p.Type)
	}
}

func (p *Pointer) EncodeValue(e binary.Encoder, value interface{}) {
	if value != nil { // TODO proper nil test needed?
		e.Object(value.(binary.Object))
	} else {
		e.Object(nil)
	}
}

func (p *Pointer) DecodeValue(d binary.Decoder) interface{} {
	return d.Object()
}

func (p *Pointer) Subspace() *binary.Subspace {
	return nil
}

func (p *Pointer) HasSubspace() bool {
	return false
}

func (*Pointer) IsPOD() bool {
	return false
}

func (*Pointer) IsSimple() bool {
	return false
}

func pointerFactory(t reflect.Type, tag reflect.StructTag, makeType binary.MakeTypeFun, pkg string) binary.Type {
	p := &Pointer{}
	p.Type = registry.PreventLoops(
		t, reflect.StructTag(""), p, makeType, t.Elem(), pkg)
	return p
}

func init() {
	registry.Factories.Add(reflect.Ptr, pointerFactory)
}
