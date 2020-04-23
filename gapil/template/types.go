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

package template

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/serialization"
)

var (
	// NodeTypeList exposes all the types valid as Is*name* tests
	NodeTypeList = []interface{}{
		// Primitive types
		"",    // string
		false, // bool
		// primitive value node types
		semantic.BoolValue(true),
		semantic.StringValue(""),
		semantic.Int8Value(0),
		semantic.Uint8Value(0),
		semantic.Int16Value(0),
		semantic.Uint16Value(0),
		semantic.Int32Value(0),
		semantic.Uint32Value(0),
		semantic.Int64Value(0),
		semantic.Uint64Value(0),
		semantic.Float32Value(0),
		semantic.Float64Value(0),
		// semantic node types
		semantic.Abort{},
		semantic.Annotation{},
		semantic.API{},
		semantic.ArrayAssign{},
		semantic.ArrayIndex{},
		semantic.ArrayInitializer{},
		semantic.Assert{},
		semantic.Assign{},
		semantic.BinaryOp{},
		semantic.BitTest{},
		semantic.Block{},
		semantic.Branch{},
		semantic.Builtin{},
		semantic.Call{},
		semantic.Callable{},
		semantic.Cast{},
		semantic.Choice{},
		semantic.ClassInitializer{},
		semantic.Class{},
		semantic.Clone{},
		semantic.Copy{},
		semantic.Create{},
		semantic.DeclareLocal{},
		semantic.Definition{},
		semantic.EnumEntry{},
		semantic.Enum{},
		semantic.Fence{},
		semantic.Field{},
		semantic.FieldInitializer{},
		semantic.Function{},
		semantic.Global{},
		semantic.Ignore{},
		semantic.Iteration{},
		semantic.MapIteration{},
		semantic.Length{},
		semantic.Local{},
		semantic.Make{},
		semantic.MapAssign{},
		semantic.MapContains{},
		semantic.MapIndex{},
		semantic.MapRemove{},
		semantic.MapClear{},
		semantic.Map{},
		semantic.Member{},
		semantic.MessageValue{},
		semantic.New{},
		semantic.Null{},
		semantic.Observed{},
		semantic.Parameter{},
		semantic.PointerRange{},
		semantic.Pointer{},
		semantic.Print{},
		semantic.Pseudonym{},
		semantic.Read{},
		semantic.Reference{},
		semantic.Return{},
		semantic.Select{},
		semantic.SliceAssign{},
		semantic.SliceContains{},
		semantic.SliceIndex{},
		semantic.SliceRange{},
		semantic.Slice{},
		semantic.Slice{},
		semantic.StaticArray{},
		semantic.Switch{},
		semantic.UnaryOp{},
		semantic.Unknown{},
		semantic.Write{},
		// node interface types
		(*semantic.Annotated)(nil),
		(*semantic.Expression)(nil),
		(*semantic.Type)(nil),
		(*semantic.Owned)(nil),
	}

	nodeTypes = map[string]reflect.Type{}
)

func init() {
	for _, n := range NodeTypeList {
		nt := baseType(n)
		name := nt.Name()
		nodeTypes[name] = nt
	}
}

func initNodeTypes(f *Functions) {
	for _, b := range semantic.BuiltinTypes {
		b := b
		name := "Is" + strings.Trim(strings.Title(b.Name()), "<>")
		f.funcs[name] = func(t semantic.Type) bool {
			return t == b
		}
	}
	for name, t := range nodeTypes {
		for _, r := range name {
			if unicode.IsUpper(r) {
				f.funcs["Is"+name] = isTypeTest(t)
			}
			break
		}
	}
}

// TypeOf returns the resolved semantic type of an expression node.
func (*Functions) TypeOf(v semantic.Node) (semantic.Type, error) {
	return semantic.TypeOf(v)
}

// IsStorageType returns true if ty can be used as a storage type.
func (*Functions) IsStorageType(ty semantic.Type) bool {
	return semantic.IsStorageType(ty)
}

// IsNumericValue returns true if v is one of the primitive numeric value types.
func (*Functions) IsNumericValue(v interface{}) bool {
	switch v.(type) {
	case semantic.Int8Value,
		semantic.Uint8Value,
		semantic.Int16Value,
		semantic.Uint16Value,
		semantic.Int32Value,
		semantic.Uint32Value,
		semantic.Int64Value,
		semantic.Uint64Value,
		semantic.Float32Value,
		semantic.Float64Value:
		return true
	default:
		return false
	}
}

// IsNumericType returns true if t is one of the primitive numeric types.
func (*Functions) IsNumericType(t interface{}) bool {
	ty, ok := t.(semantic.Type)
	return ok && semantic.IsNumeric(ty)
}

// UniqueEnumKeys returns the enum's list of EnumEntry with duplicate values
// removed. To remove duplicates the entry with the shortest name is picked.
func (*Functions) UniqueEnumKeys(e *semantic.Enum) ([]*semantic.EnumEntry, error) {
	keys := map[semantic.Expression]*semantic.EnumEntry{}
	for _, e := range e.Entries {
		if got, found := keys[e.Value]; !found || len(got.Name()) > len(e.Name()) {
			keys[e.Value] = e
		}
	}
	out := make([]*semantic.EnumEntry, 0, len(keys))
	for _, e := range e.Entries {
		if got := keys[e.Value]; got == e {
			out = append(out, e)
			delete(keys, e.Value)
		}
	}
	return out, nil
}

// Returns the base name of the type of v
func baseType(v interface{}) reflect.Type {
	ty := reflect.TypeOf(v)
	for ty != nil && ty.Kind() == reflect.Ptr {
		return ty.Elem()
	}
	return ty
}

func singleTypeTest(test reflect.Type, against reflect.Type) bool {
	if test == nil {
		if against == nil {
			return true
		}
		return false
	}
	if against == nil {
		return false
	}
	return test.AssignableTo(against)
}

func doTypeTest(v interface{}, against ...reflect.Type) bool {
	test := reflect.TypeOf(v)
	for {
		for _, t := range against {
			if singleTypeTest(test, t) {
				return true
			}
		}
		if test != nil && test.Kind() == reflect.Ptr {
			test = test.Elem()
		} else {
			return false
		}
	}
}

func isTypeTest(t reflect.Type) func(v interface{}) bool {
	return func(v interface{}) bool {
		return doTypeTest(v, t)
	}
}

// Asserts that the type of v is in the list of expected types
func (*Functions) AssertType(v interface{}, expected ...string) (string, error) {
	types := make([]reflect.Type, len(expected))
	for i, e := range expected {
		if e != "nil" {
			et, found := nodeTypes[e]
			if !found {
				return "", fmt.Errorf("%s is not a valid type", e)
			}
			types[i] = et
		}
	}
	if doTypeTest(v, types...) {
		return "", nil
	}

	msg := fmt.Sprintf("Type assertion. Got: %T, Expected: ", v)
	if c := len(expected); c > 1 {
		msg += strings.Join(expected[:c-1], ", ")
		msg += " or " + expected[c-1]
	} else {
		msg += expected[0]
	}
	return "", errors.New(msg)
}

// ProtoType returns the proto type name for the given type.
func (f *Functions) ProtoType(ty interface{}) string {
	return serialization.ProtoTypeName(ty.(semantic.Type))
}
