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

package generate

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/tools/codergen/scan"
)

var (
	structType = &schema.Struct{Entity: &binary.Entity{Identity: "TestObject"}}
	fields     = []binary.Field{
		{Declared: "u8", Type: &schema.Primitive{Name: "uint8", Method: schema.Uint8}},
		{Declared: "u16", Type: &schema.Primitive{Name: "uint16", Method: schema.Uint16}},
		{Declared: "u32", Type: &schema.Primitive{Name: "uint32", Method: schema.Uint32}},
		{Declared: "u64", Type: &schema.Primitive{Name: "uint64", Method: schema.Uint64}},
		{Declared: "i8", Type: &schema.Primitive{Name: "int8", Method: schema.Int8}},
		{Declared: "i16", Type: &schema.Primitive{Name: "int16", Method: schema.Int16}},
		{Declared: "i32", Type: &schema.Primitive{Name: "int32", Method: schema.Int32}},
		{Declared: "i64", Type: &schema.Primitive{Name: "int64", Method: schema.Int64}},
		{Declared: "f32", Type: &schema.Primitive{Name: "float32", Method: schema.Float32}},
		{Declared: "f64", Type: &schema.Primitive{Name: "float64", Method: schema.Float64}},
		{Declared: "bool", Type: &schema.Primitive{Name: "bool", Method: schema.Bool}},
		{Declared: "byte", Type: &schema.Primitive{Name: "byte", Method: schema.Uint8}},
		{Declared: "int", Type: &schema.Primitive{Name: "int", Method: schema.Int32}},
		{Declared: "str", Type: &schema.Primitive{Name: "string", Method: schema.String}},
		{Declared: "codeable", Type: structType},
		{Declared: "pointer", Type: &schema.Pointer{Type: structType}},
		{Declared: "object", Type: &schema.Interface{Name: "binary.Object"}},
		{Declared: "slice", Type: &schema.Slice{ValueType: structType}},
		{Declared: "alias", Type: &schema.Slice{Alias: "Other", ValueType: &schema.Primitive{Name: "int", Method: schema.Int32}}},
		{Declared: "array", Type: &schema.Array{ValueType: &schema.Primitive{Name: "int", Method: schema.Int32}, Size: 10}},
		{Declared: "dict", Type: &schema.Map{KeyType: &schema.Primitive{Name: "string", Method: schema.String}, ValueType: structType}},
		{Declared: "data", Type: &schema.Slice{ValueType: &schema.Primitive{Name: "uint8", Method: schema.Uint8}}},
		{Declared: "", Type: structType},
	}
)
var testId int

func parseStructs(ctx context.Context, source string) []*Struct {
	testId++
	fakeFile := fmt.Sprintf(`
	package fake
	import "github.com/google/gapid/framework/binary"
	type TestObject struct{binary.Generate}

	%s`, source)
	name := fmt.Sprintf("fake_%d.go", testId)
	pwd, _ := filepath.Abs(".")
	scanner := scan.New(ctx, pwd, "")
	scanner.ScanFile(name, fakeFile)
	if err := scanner.Process(ctx); err != nil {
		log.E(ctx, "Process failed: %v", err)
	}
	modules, err := From(scanner)
	if err != nil {
		log.E(ctx, "Scan failed: %v", err)
	}
	for _, m := range modules {
		if m.Source.Directory.ImportPath == name && !m.IsTest {
			return m.Structs
		}
	}
	log.F(ctx, "Module find failed")
	return nil
}

func parseStruct(ctx context.Context, name string, source string) *Struct {
	s := parseStructs(ctx, source)
	for _, entry := range s {
		if entry.Name() == name {
			return entry
		}
	}
	log.F(ctx, "No structs match %v in %v candidates.", name, len(s))
	return nil
}

func TestEmpty(t *testing.T) {
	ctx := log.Testing(t)
	s := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate}")
	assert.For(ctx, "Fields").That(len(s.Fields)).Equals(0)
}

func TestDisable(t *testing.T) {
	ctx := log.Testing(t)
	s := parseStructs(ctx, "type MyStruct struct {binary.Generate `disable:\"true\"`}")
	for _, entry := range s {
		assert.For(ctx, "Generated disabled struct").That(entry.Identity).NotEquals("MyStruct")
	}
}

func TestInterfaceVsAny(t *testing.T) {
	ctx := log.Testing(t)
	source := `
		type S struct {
			binary.Generate

			any0 interface{}
			any1 interface{ F() }
			any2 interface{ anyB; F2() }
			any3 anyA
			any4 anyB
			any5 anyC

			obj0 interface{ binary.Object }
			obj1 interface{ binary.Object; F() }
			obj2 interface{ objB; F3() }
			obj3 objA
			obj4 objB
			obj5 objC
		}

		type anyA interface{}
		type anyB interface{ F() }
		type anyC interface{ anyB; F2() }
		type objA interface{ binary.Object }
		type objB interface{ binary.Object; F() }
		type objC interface{ objB; F3() }
`

	s := parseStruct(ctx, "S", source)
	for _, f := range s.Fields {
		_, isAny := f.Type.(*schema.Any)
		_, isInt := f.Type.(*schema.Interface)
		if strings.HasPrefix(f.Name(), "any") && !isAny {
			log.E(ctx, "Field '%s' has unexpected type %T", f.Name(), f.Type)
		}
		if strings.HasPrefix(f.Name(), "obj") && !isInt {
			log.E(ctx, "Field '%s' has unexpected type %T", f.Name(), f.Type)
		}
	}
}
func TestStableSignature(t *testing.T) {
	ctx := log.Testing(t)
	source := "type MyStruct struct {binary.Generate}"
	a := parseStruct(ctx, "MyStruct", source)
	b := parseStruct(ctx, "MyStruct", source)
	assert.With(ctx).That(a.Signature()).Equals(b.Signature())
}

func TestNameAffectsSignature(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate}")
	b := parseStruct(ctx, "YourStruct", "type YourStruct struct {binary.Generate}")
	assert.With(ctx).That(a.Signature()).NotEquals(b.Signature())
}

func TestNameOveride(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate}")
	b := parseStruct(ctx, "YourStruct", "type YourStruct struct {binary.Generate `identity:\"MyStruct\"`}")
	assert.With(ctx).That(a.Signature()).Equals(b.Signature())
}

func TestVersionOveride(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate}")
	b := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate `version:\"1\"`}")
	assert.With(ctx).That(a.Signature()).NotEquals(b.Signature())
}

func TestFieldCountAffectsSignature(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate; a int}")
	b := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate}")
	assert.With(ctx).That(a.Signature()).NotEquals(b.Signature())
}

func TestFieldNameAffectsSignature(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate; a int}")
	b := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate; b int}")
	assert.With(ctx).That(a.Signature()).Equals(b.Signature())
}

func TestFieldTypeAffectsSignature(t *testing.T) {
	ctx := log.Testing(t)
	a := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate; a int}")
	b := parseStruct(ctx, "MyStruct", "type MyStruct struct {binary.Generate; a byte}")
	assert.With(ctx).That(a.Signature()).NotEquals(b.Signature())
}

func TestTypes(t *testing.T) {
	ctx := log.Testing(t)
	prefix := "type Other []int\n"
	source := &bytes.Buffer{}
	fmt.Fprintln(source, prefix)
	fmt.Fprint(source, "type MyStruct struct {binary.Generate;\n")
	for _, f := range fields {
		fmt.Fprintf(source, "  %s %s\n", f.Declared, f.Type)
	}
	fmt.Fprint(source, "}\n")
	s := parseStruct(ctx, "MyStruct", source.String())
	assert.For(ctx, "Field count").That(len(s.Fields)).Equals(len(fields))
	for i, got := range s.Fields {
		expected := fields[i]
		assert.For(ctx, "Declared").That(got.Declared).Equals(expected.Declared)
		assert.For(ctx, "Type").That(reflect.TypeOf(got.Type)).Equals(reflect.TypeOf(expected.Type))
		assert.For(ctx, "Typename").ThatString(got.Type.String()).Equals(expected.Type.String())
	}
}
