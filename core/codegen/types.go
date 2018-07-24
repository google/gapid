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

package codegen

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"llvm/bindings/go/llvm"
)

// SizeOf returns the size of the type in bytes as a uint64.
// If ty is void, a value of 1 is returned.
func (b *Builder) SizeOf(ty Type) *Value {
	if ty == b.m.Types.Void {
		return b.Scalar(uint64(1))
	}
	if bits := ty.sizeInBits(); bits > 0 {
		return b.Scalar(uint64((bits + 7) / 8)).
			SetName(fmt.Sprintf("sizeof(%v)", ty.TypeName()))
	}
	return b.val(b.m.Types.Uint64, llvm.SizeOf(ty.llvmTy())).
		SetName(fmt.Sprintf("sizeof(%v)", ty.TypeName()))
}

// AlignOf returns the alignment of the type in bytes.
func (b *Builder) AlignOf(ty Type) *Value {
	return b.val(b.m.Types.Uint64, llvm.AlignOf(ty.llvmTy())).
		SetName(fmt.Sprintf("alignof(%v)", ty.TypeName()))
}

// Types contains all the types for a module.
type Types struct {
	m *Module

	Void    Type // Void is the void type.
	Bool    Type // Bool is a one-bit integer type.
	Int     Type // Int is a signed 32-bit or 64-bit integer type.
	Int8    Type // Int8 is a signed 8-bit integer type.
	Int16   Type // Int16 is a signed 16-bit integer type.
	Int32   Type // Int32 is a signed 32-bit integer type.
	Int64   Type // Int64 is a signed 64-bit integer type.
	Uint    Type // Uint is an unsigned 32-bit or 64-bit integer type.
	Uint8   Type // Uint8 is an unsigned 8-bit integer type.
	Uint16  Type // Uint16 is an unigned 16-bit integer type.
	Uint32  Type // Uint32 is an unsigned 32-bit integer type.
	Uint64  Type // Uint64 is an unsigned 64-bit integer type.
	Uintptr Type // Uinptr is an unsigned integer type of the same width as a pointer.
	Size    Type // Size is an unsigned integer of native bit-width.
	Float32 Type // Float32 is a 32-bit floating-point number type.
	Float64 Type // Float64 is a 64-bit floating-point number type.

	ptrSizeInBits int
	pointers      map[Type]Pointer // T -> T*
	arrays        map[typeInt]*Array
	structs       map[string]*Struct
	funcs         map[string]*FunctionType
	enums         map[string]Enum
	aliases       map[string]Alias
	named         map[string]Type
}

type typeInt struct {
	Type
	int
}

// Type represents a codegen type.
type Type interface {
	String() string
	TypeName() string

	sizeInBits() int // 0 means target-dependent or aggregate type
	llvmTy() llvm.Type
}

// TypeList is a slice of types.
type TypeList []Type

func (l TypeList) String() string {
	parts := make([]string, len(l))
	for i, p := range l {
		parts[i] = p.TypeName()
	}
	return strings.Join(parts, ", ")
}

func (l TypeList) llvm() []llvm.Type {
	out := make([]llvm.Type, len(l))
	for i, t := range l {
		out[i] = t.llvmTy()
	}
	return out
}

type basicType struct {
	name string
	bits int
	llvm llvm.Type
}

func (t basicType) TypeName() string  { return t.name }
func (t basicType) String() string    { return t.name }
func (t basicType) llvmTy() llvm.Type { return t.llvm }
func (t basicType) sizeInBits() int   { return t.bits }

// Pointer represents a pointer type.
type Pointer struct {
	Element Type // The type of the element the pointer points to.

	basicType
}

func (t Pointer) TypeName() string { return fmt.Sprintf("*%v", t.Element.TypeName()) }
func (t Pointer) String() string   { return fmt.Sprintf("*%v", t.Element) }

// Pointer returns a pointer type of el.
func (t *Types) Pointer(el Type) Pointer {
	p, ok := t.pointers[el]
	if !ok {
		if el == t.Void {
			el = t.Uint8
		}
		p = Pointer{el, basicType{"", t.ptrSizeInBits, llvm.PointerType(el.llvmTy(), 0)}}
		t.pointers[el] = p
	}
	return p
}

// Array represents a static array type.
type Array struct {
	Element Type // The type of the element the pointer points to.
	Size    int  // Number of elements in the array.

	basicType
}

func (t *Array) TypeName() string { return fmt.Sprintf("%v[%d]", t.Element.TypeName(), t.Size) }
func (t *Array) String() string   { return t.TypeName() }

// Array returns an n-element array type of el.
func (t *Types) Array(el Type, n int) *Array {
	a, ok := t.arrays[typeInt{el, n}]
	if !ok {
		a = &Array{el, n, basicType{"", 0, llvm.ArrayType(el.llvmTy(), n)}}
		t.arrays[typeInt{el, n}] = a
	}
	return a
}

// IsPointer returns true if ty is a pointer type.
func IsPointer(ty Type) bool {
	_, ok := ty.(Pointer)
	return ok
}

// Vector represents a vector type.
type Vector struct {
	Element Type // The type of the vector elements.
	Count   int  // Number of elements in a vector.
	basicType
}

func (t Vector) TypeName() string {
	return fmt.Sprintf("vec<%v, %d>", t.Element.TypeName(), t.Count)
}
func (t Vector) String() string { return fmt.Sprintf("vec<%v, %d>", t.Element, t.Count) }

// Vector returns a pointer type of el.
func (t *Types) Vector(el Type, count int) Vector {
	return Vector{el, count, basicType{"", 0, llvm.VectorType(el.llvmTy(), count)}}
}

// IsVector returns true if ty is a vector type.
func IsVector(ty Type) bool {
	_, ok := ty.(Vector)
	return ok
}

// Scalar returns the element type if ty is a vector, otherwise it returns
// ty.
func Scalar(ty Type) Type {
	if vec, ok := ty.(Vector); ok {
		return vec.Element
	}
	return ty
}

// Integer represents an integer type.
type Integer struct {
	Signed bool // Is this integer type signed?

	basicType
}

// IsBool returns true if ty is the boolean type.
func IsBool(ty Type) bool {
	t, ok := ty.(basicType)
	return ok && t.llvm.IntTypeWidth() == 1
}

// IsInteger returns true if ty is an integer type.
func IsInteger(ty Type) bool {
	_, ok := ty.(Integer)
	return ok
}

// IsEnum returns true if ty is an enum type.
func IsEnum(ty Type) bool {
	_, ok := ty.(Enum)
	return ok
}

// IsSignedInteger returns true if ty is a signed integer type.
func IsSignedInteger(ty Type) bool {
	i, ok := ty.(Integer)
	return ok && i.Signed
}

// IsUnsignedInteger returns true if ty is an unsigned integer type.
func IsUnsignedInteger(ty Type) bool {
	i, ok := ty.(Integer)
	return ok && !i.Signed
}

// IsIntegerOrEnum returns true if ty is an integer or enum type.
func IsIntegerOrEnum(ty Type) bool { return IsInteger(ty) || IsEnum(ty) }

// IsSignedIntegerOrEnum returns true if ty is a signed integer or enum type.
func IsSignedIntegerOrEnum(ty Type) bool { return IsSignedInteger(ty) || IsEnum(ty) }

// Float represents a floating-point type.
type Float struct {
	basicType
}

func (t Float) TypeName() string { return t.name }
func (t Float) String() string   { return t.name }

// IsFloat returns true if ty is a float type.
func IsFloat(ty Type) bool {
	_, ok := ty.(Float)
	return ok
}

// FunctionType is the type of a function
type FunctionType struct {
	Signature Signature
	llvm      llvm.Type
}

func (t FunctionType) TypeName() string  { return t.Signature.string("") }
func (t FunctionType) String() string    { return t.Signature.string("") }
func (t FunctionType) sizeInBits() int   { return 0 }
func (t FunctionType) llvmTy() llvm.Type { return t.llvm }

// Signature holds signature information about a function.
type Signature struct {
	Parameters TypeList
	Result     Type
	Variadic   bool
}

func (s Signature) string(name string) string {
	return fmt.Sprintf("%v %v(%v)", s.Result, name, s.Parameters)
}

func (s Signature) key() string {
	parts := make([]string, len(s.Parameters))
	for i, p := range s.Parameters {
		parts[i] = fmt.Sprint(p)
	}
	if s.Variadic {
		return fmt.Sprintf("(%v, ...)%v", s.Parameters, s.Result)
	}
	return fmt.Sprintf("(%v)%v", s.Parameters, s.Result)
}

type variadicTy struct{}

func (variadicTy) String() string    { return "..." }
func (variadicTy) TypeName() string  { return "..." }
func (variadicTy) sizeInBits() int   { panic("Cannot use Variadic as a regular type") }
func (variadicTy) llvmTy() llvm.Type { panic("Cannot use Variadic as a regular type") }

// Variadic is a type that can be used as the last parameter of a function
// definition to indicate a variadic function.
var Variadic variadicTy

// Struct is the type of a structure.
type Struct struct {
	Name         string
	Fields       []Field
	fieldIndices map[string]int
	llvm         llvm.Type
}

func (t *Struct) TypeName() string { return t.Name }
func (t *Struct) String() string {
	if len(t.Fields) == 0 {
		return fmt.Sprintf("%v{}", t.Name)
	}
	b := bytes.Buffer{}
	b.WriteString(t.Name)
	b.WriteString(" {")
	for _, f := range t.Fields {
		b.WriteString("\n  ")
		b.WriteString(f.Name)
		b.WriteString(": ")
		b.WriteString(f.Type.TypeName())
	}
	b.WriteString("\n}")
	return b.String()
}
func (t *Struct) sizeInBits() int   { return 0 }
func (t *Struct) llvmTy() llvm.Type { return t.llvm }

// Field returns the field with the given name.
// Field panics if the struct does not contain the given field.
func (t *Struct) Field(name string) Field {
	f, ok := t.fieldIndices[name]
	if !ok {
		panic(fmt.Errorf("Struct '%v' does not have field with name '%v'", t.Name, name))
	}
	return t.Fields[f]
}

// FieldIndex returns the index of the field with the given name, or -1 if the
// struct does not have a field with the given name.
func (t *Struct) FieldIndex(name string) int {
	f, ok := t.fieldIndices[name]
	if !ok {
		return -1
	}
	return f
}

// IsStruct returns true if ty is a struct type.
func IsStruct(ty Type) bool {
	_, ok := ty.(*Struct)
	return ok
}

// Field is a single field in a struct.
type Field struct {
	Name string
	Type Type
}

// struct_ creates a new struct populated with the given fields.
// If packed is true then fields will be stored back-to-back.
func (t *Types) struct_(name string, packed bool, fields []Field) *Struct {
	name = sanitizeStructName(name)
	if s, ok := t.structs[name]; ok {
		if fields != nil {
			if !reflect.DeepEqual(fields, s.Fields) {
				panic(fmt.Errorf("Struct '%s' redeclared with different fields\nPrevious: %+v\nNew:      %+v",
					name, s.Fields, fields))
			}
			if packed != s.llvm.IsStructPacked() {
				panic(fmt.Errorf("Struct '%s' redeclared with different packed flags", name))
			}
		}
		return s
	}
	ty := t.m.ctx.StructCreateNamed(name)
	s := &Struct{Name: name, llvm: ty}
	t.registerNamed(s)
	if fields != nil {
		s.SetBody(packed, fields...)
	}
	t.structs[name] = s
	return s
}

func (t *Types) registerNamed(ty Type) {
	name := ty.TypeName()
	if _, dup := t.named[name]; dup {
		fail("Duplicate types with the name %v", name)
	}
	t.named[name] = ty
}

// SetBody sets the fields of the declared struct.
func (t *Struct) SetBody(packed bool, fields ...Field) *Struct {
	indices := map[string]int{}
	l := make([]llvm.Type, len(fields))
	for i, f := range fields {
		if f.Type == nil {
			fail("Field '%s' (%d) has nil type", f.Name, i)
		}
		l[i] = f.Type.llvmTy()
		indices[f.Name] = i
	}
	t.Fields = fields
	t.fieldIndices = indices
	t.llvm.StructSetBody(l, packed)
	return t
}

// Alias is a named type that aliases another type.
type Alias struct {
	name string
	to   Type
}

func (a Alias) String() string    { return fmt.Sprintf("%v (%v)", a.TypeName(), Underlying(a).TypeName()) }
func (a Alias) TypeName() string  { return a.name }
func (a Alias) sizeInBits() int   { return a.to.sizeInBits() }
func (a Alias) llvmTy() llvm.Type { return a.to.llvmTy() }

// Alias creates a new alias type.
func (t *Types) Alias(name string, to Type) Alias {
	ty := Alias{name: name, to: to}
	t.aliases[name] = ty
	t.registerNamed(ty)
	return ty
}

// Underlying returns the underlying non-aliased type for ty.
func Underlying(ty Type) Type {
	for {
		if a, ok := ty.(Alias); ok {
			ty = a.to
		} else {
			return ty
		}
	}
}

// Enum is an enumerator.
type Enum struct{ basicType }

// Enum creates a new enum type.
func (t *Types) Enum(name string) Enum {
	ty := Enum{
		basicType{
			name: name,
			bits: t.Int.sizeInBits(),
			llvm: t.Int.llvmTy(),
		},
	}
	t.enums[name] = ty
	t.registerNamed(ty)
	return ty
}

// DeclareStruct creates a new, empty struct type.
func (t *Types) DeclareStruct(name string) *Struct {
	return t.struct_(name, false, nil)
}

// DeclarePackedStruct creates a new, packed empty struct type.
func (t *Types) DeclarePackedStruct(name string) *Struct {
	return t.struct_(name, true, nil)
}

// Struct creates a new unpacked struct type.
func (t *Types) Struct(name string, fields ...Field) *Struct {
	return t.struct_(name, false, fields)
}

// PackedStruct creates a new packed struct type.
func (t *Types) PackedStruct(name string, fields ...Field) *Struct {
	return t.struct_(name, true, fields)
}

// TypeOf returns the corresponding codegen type for the type of value v.
// TypeOf may also accept a reflect.Type.
func (t *Types) TypeOf(v interface{}) Type {
	ty, ok := v.(reflect.Type)
	if !ok {
		ty = reflect.TypeOf(v)
	}
	switch ty.Kind() {
	case reflect.Bool:
		return t.Bool
	case reflect.Int:
		return t.Int
	case reflect.Int8:
		return t.Int8
	case reflect.Int16:
		return t.Int16
	case reflect.Int32:
		return t.Int32
	case reflect.Int64:
		return t.Int64
	case reflect.Uint:
		return t.Uint
	case reflect.Uint8:
		return t.Uint8
	case reflect.Uint16:
		return t.Uint16
	case reflect.Uint32:
		return t.Uint32
	case reflect.Uint64:
		return t.Uint64
	case reflect.Float32:
		return t.Float32
	case reflect.Float64:
		return t.Float64
	case reflect.Ptr:
		return t.Pointer(t.TypeOf(ty.Elem()))
	case reflect.Interface:
		return t.TypeOf(ty.Elem())
	case reflect.UnsafePointer, reflect.Uintptr:
		return t.Pointer(t.Uint8)
	case reflect.Array:
		return t.Array(t.TypeOf(ty.Elem()), ty.Len())
	case reflect.Slice:
		return t.Array(t.TypeOf(ty.Elem()), reflect.ValueOf(v).Len())
	case reflect.String:
		return t.Pointer(t.Uint8)
	case reflect.Struct:
		name := sanitizeStructName(ty.Name())
		if s, ok := t.structs[name]; ok {
			return s // avoid stack overflow if type references itself.
		}
		s := t.DeclareStruct(name)
		fields := t.FieldsOf(ty)
		s.SetBody(false, fields...)
		return s
	default:
		panic(fmt.Errorf("Unsupported kind %v", ty.Kind()))
	}
}

// FieldsOf returns the codegen fields of the given struct type.
// FieldsOf may also accept a reflect.Type.
func (t *Types) FieldsOf(v interface{}) []Field {
	ty, ok := v.(reflect.Type)
	if !ok {
		ty = reflect.TypeOf(v)
	}
	if ty.Kind() != reflect.Struct {
		panic(fmt.Errorf("FieldsOf must be passed a struct type. Got %v", ty))
	}
	c := ty.NumField()
	fields := make([]Field, 0, c)
	for i := 0; i < c; i++ {
		f := ty.Field(i)
		if f.Name == "_" {
			continue // Cgo padding struct. No thanks.
		}
		ty := t.TypeOf(f.Type)
		if f.Type == reflect.TypeOf(unsafe.Pointer(nil)) {
			if name, ok := f.Tag.Lookup("ptr"); ok {
				s, ok := t.structs[name]
				if !ok {
					panic(fmt.Errorf("Unknown pointer type '%v'", name))
				}
				ty = t.Pointer(s)
			}
		}
		fields = append(fields, Field{Name: f.Name, Type: ty})
	}
	return fields
}

// sanitizeStructName removes cgo mangling from the struct name.
func sanitizeStructName(name string) string {
	if strings.HasPrefix(name, "_Ctype") {
		name = strings.TrimPrefix(name, "_Ctype_struct_") // Remove Cgo prefix...
		name = strings.TrimSuffix(name, "_t")             // ... and '_t'
	}
	return name
}

// Function returns a type representing the given function signature.
func (t *Types) Function(resTy Type, paramTys ...Type) *FunctionType {
	if resTy == nil {
		resTy = t.Void
	}
	params, variadic := TypeList(paramTys), false
	if len(params) > 0 && params[len(params)-1] == Variadic {
		params, variadic = params[:len(params)-1], true
	}
	sig := Signature{params, resTy, variadic}
	key := sig.key()
	ty, ok := t.funcs[key]
	if ok {
		return ty
	}
	ty = &FunctionType{
		sig,
		llvm.FunctionType(resTy.llvmTy(), params.llvm(), variadic),
	}
	t.funcs[key] = ty
	return ty
}
