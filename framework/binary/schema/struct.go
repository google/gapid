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

// Struct is the Type descriptor for an binary.Object typed value.
type Struct struct {
	Relative string         // The relative go import name of the type, only useful to codergen.
	Entity   *binary.Entity // The schema entity this is a field of.
}

func (s *Struct) Representation() string {
	return fmt.Sprintf("%r", s)
}

func (s *Struct) String() string {
	return fmt.Sprint(s)
}

// Format implements the fmt.Formatter interface
func (s *Struct) Format(f fmt.State, c rune) {
	switch c {
	case 'z': // Private format specifier, supports Entity.Signature
		fmt.Fprint(f, "$")
	case 'r': // Private format specifier, supports Type.Representation
		fmt.Fprint(f, s.Entity.Name())
	default:
		if s.Relative != "" {
			fmt.Fprint(f, s.Relative)
		} else {
			fmt.Fprint(f, s.Entity.Name())
		}
	}
}

func (s *Struct) EncodeValue(e binary.Encoder, value interface{}) {
	e.Struct(value.(binary.Object))
}

func (s *Struct) DecodeValue(d binary.Decoder) interface{} {
	u := d.Lookup(s.Entity)
	if u == nil {
		d.SetError(fmt.Errorf("Unknown type id %v for %s", s.Entity, s))
		return nil
	}
	o := u.New()
	d.Struct(o)
	return o
}

func (s *Struct) Subspace() *binary.Subspace {
	return s.Entity.Subspace()
}

func (s *Struct) HasSubspace() bool {
	return true
}

func (s *Struct) IsPOD() bool {
	return s.Entity.IsPOD()
}

func (s *Struct) IsSimple() bool {
	return s.Entity.IsSimple()
}

func factory(t reflect.Type, tag reflect.StructTag, makeType binary.MakeTypeFun, pkg string) binary.Type {
	obj := reflect.New(reflect.PtrTo(t)).Elem().Interface().(binary.Object)
	s := &Struct{Entity: obj.Class().Schema()}
	if t.PkgPath() == pkg {
		s.Relative = t.Name()
	}
	return s
}

func init() {
	registry.Factories.Add(reflect.Struct, factory)
}
