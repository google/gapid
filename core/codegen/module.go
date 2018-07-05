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
	"regexp"
	"strings"

	"github.com/google/gapid/core/os/device"

	"llvm/bindings/go/llvm"
)

// Module is a JIT module.
type Module struct {
	Types   Types
	llvm    llvm.Module
	ctx     llvm.Context
	target  *device.ABI
	triple  triple
	name    string
	funcs   map[string]*Function
	llvmDbg *llvm.DIBuilder
	memcpy  *Function
	memset  *Function
}

// NewModule returns a new module with the specified name.
func NewModule(name string, target *device.ABI) *Module {
	layout := target.MemoryLayout
	intSize := 8 * int(layout.Integer.Size)
	ptrSize := 8 * int(layout.Pointer.Size)
	sizeSize := 8 * int(layout.Size.Size)

	triple := targetTriple(target)

	ctx := llvm.NewContext()

	module := ctx.NewModule(name)
	module.SetTarget(triple.String())

	m := &Module{
		Types: Types{
			Void:          basicType{"void", 0, ctx.VoidType()},
			Bool:          basicType{"bool", 1, ctx.Int1Type()},
			Int:           Integer{true, basicType{"int", intSize, ctx.IntType(intSize)}},
			Int8:          Integer{true, basicType{"int8", 8, ctx.Int8Type()}},
			Int16:         Integer{true, basicType{"int16", 16, ctx.Int16Type()}},
			Int32:         Integer{true, basicType{"int32", 32, ctx.Int32Type()}},
			Int64:         Integer{true, basicType{"int64", 64, ctx.Int64Type()}},
			Uint:          Integer{false, basicType{"uint", intSize, ctx.IntType(intSize)}},
			Uint8:         Integer{false, basicType{"uint8", 8, ctx.Int8Type()}},
			Uint16:        Integer{false, basicType{"uint16", 16, ctx.Int16Type()}},
			Uint32:        Integer{false, basicType{"uint32", 32, ctx.Int32Type()}},
			Uint64:        Integer{false, basicType{"uint64", 64, ctx.Int64Type()}},
			Uintptr:       Integer{false, basicType{"uintptr", ptrSize, ctx.IntType(ptrSize)}},
			Size:          Integer{false, basicType{"size", sizeSize, ctx.IntType(sizeSize)}},
			Float32:       Float{basicType{"float32", 32, ctx.FloatType()}},
			Float64:       Float{basicType{"float64", 64, ctx.DoubleType()}},
			ptrSizeInBits: ptrSize,
			pointers:      map[Type]Pointer{},
			arrays:        map[typeInt]*Array{},
			structs:       map[string]*Struct{},
			funcs:         map[string]*FunctionType{},
			enums:         map[string]Enum{},
			aliases:       map[string]Alias{},
			named:         map[string]Type{},
		},
		llvm:   module,
		ctx:    ctx,
		target: target,
		triple: triple,
		name:   name,
		funcs:  map[string]*Function{},
	}

	voidPtr := m.Types.Pointer(m.Types.Void)
	// void llvm.memcpy.p0i8.p0i8.i32(i8 * <dest>, i8 * <src>, i32 <len>, i32 <align>, i1 <isvolatile>)
	m.memcpy = m.Function(m.Types.Void, "llvm.memcpy.p0i8.p0i8.i32",
		voidPtr, voidPtr, m.Types.Uint32, m.Types.Bool)
	// void @llvm.memset.p0i8.i32(i8* <dest>, i8 <val>, i32 <len>, i1 <isvolatile>)
	m.memset = m.Function(m.Types.Void, "llvm.memset.p0i8.i32",
		voidPtr, m.Types.Uint8, m.Types.Uint32, m.Types.Bool)

	m.Types.m = m
	return m
}

// Verify checks correctness of the module.
func (m *Module) Verify() error {
	for f := m.llvm.FirstFunction(); !f.IsNil(); f = llvm.NextFunction(f) {
		if err := llvm.VerifyFunction(f, llvm.ReturnStatusAction); err != nil {
			f.Dump()
			return fmt.Errorf("Function '%s' verification failed:\n%v", f.Name(), err)
		}
	}

	if err := llvm.VerifyModule(m.llvm, llvm.ReturnStatusAction); err != nil {
		return fmt.Errorf("Module verification failed:\n%v\n%v", err, m.String())
	}

	return nil
}

func hex(r rune) byte {
	switch {
	case r >= '0' && r <= '9':
		return byte(r - '0')
	case r >= 'A' && r <= 'Z':
		return byte(r-'A') + 10
	case r >= 'a' && r <= 'z':
		return byte(r-'a') + 10
	default:
		return 0
	}
}

func slashHexToRune(ir string) string {
	runes := ([]rune)(ir)
	out := bytes.Buffer{}
	for {
		for len(runes) >= 3 && runes[0] == '\\' {
			out.WriteByte(hex(runes[1])<<4 | hex(runes[2]))
			runes = runes[3:]
		}
		if len(runes) == 0 {
			break
		}
		out.WriteRune(runes[0])
		runes = runes[1:]
	}
	return out.String()
}

func (m *Module) String() string {
	return slashHexToRune(m.llvm.String())
}

var parseFuncRE = regexp.MustCompile(`(\w+\s*\**)\s*(\w+)\((.*)\)`)

// ParseFunctionSignature returns a function parsed from a C-style signature.
// Example:
//   "void* Foo(uint8_t i, bool b)"
func (m *Module) ParseFunctionSignature(sig string) *Function {
	parts := parseFuncRE.FindStringSubmatch(sig)
	if len(parts) != 4 {
		fail("'%v' is not a valid function signature", sig)
	}
	ret := m.parseType(parts[1])
	name := parts[2]
	args := m.parseTypes(strings.Split(parts[3], ","))
	return m.Function(ret, name, args...)
}

func (m *Module) parseTypes(l []string) []Type {
	out := make([]Type, len(l))
	for i, s := range l {
		out[i] = m.parseType(s)
	}
	return out
}

var parseTypeRE = regexp.MustCompile(`^\s*(\w+|\.\.\.)\s*([\*\s]*)`)

func (m *Module) parseType(s string) Type {
	parts := parseTypeRE.FindStringSubmatch(s)
	if len(parts) != 3 {
		fail("'%v' is not a valid type", s)
	}
	name := parts[1]
	ptrs := parts[2]

	ty := m.parseTypeName(name)
	for _, r := range ptrs {
		if r == '*' {
			ty = m.Types.Pointer(ty)
		}
	}
	return ty
}

func (m *Module) parseTypeName(name string) Type {
	switch name {
	case "void":
		return m.Types.Void
	case "bool":
		return m.Types.Bool
	case "int_t":
		return m.Types.Int
	case "int8_t":
		return m.Types.Int8
	case "int16_t":
		return m.Types.Int16
	case "int32_t":
		return m.Types.Int32
	case "int64_t":
		return m.Types.Int64
	case "uint_t":
		return m.Types.Uint
	case "uint8_t":
		return m.Types.Uint8
	case "uint16_t":
		return m.Types.Uint16
	case "uint32_t":
		return m.Types.Uint32
	case "uint64_t":
		return m.Types.Uint64
	case "float":
		return m.Types.Float32
	case "double":
		return m.Types.Float64
	case "uintptr_t":
		return m.Types.Uintptr
	case "...":
		return Variadic
	default:
		if ty, ok := m.Types.named[name]; ok {
			return ty
		}
		fail("'%v' is not a valid type name", name)
		return nil
	}
}

// Function creates a new function with the given name, result type and parameters.
func (m *Module) Function(resTy Type, name string, paramTys ...Type) *Function {
	ty := m.Types.Function(resTy, paramTys...)
	f := llvm.AddFunction(m.llvm, name, ty.llvm)
	out := &Function{Name: name, Type: ty, llvm: f, m: m}
	if name != "" {
		if _, existing := m.funcs[name]; existing {
			fail("Duplicate function with name: '%s'", name)
		}
		m.funcs[name] = out
	}
	return out
}

// Global is a global value.
type Global struct {
	Type Type
	llvm llvm.Value
}

// Value returns a Value for the global.
func (g Global) Value(b *Builder) *Value {
	return b.val(g.Type, g.llvm)
}

// LinkPrivate makes this global use private linkage.
func (g Global) LinkPrivate() Global {
	g.llvm.SetLinkage(llvm.PrivateLinkage)
	return g
}

// ZeroGlobal returns a zero-initialized new global variable with the specified
// name and type.
func (m *Module) ZeroGlobal(name string, ty Type) Global {
	v := llvm.AddGlobal(m.llvm, ty.llvmTy(), name)
	v.SetInitializer(llvm.ConstNull(ty.llvmTy()))
	v.SetLinkage(llvm.PrivateLinkage)
	return Global{m.Types.Pointer(ty), v}
}

// Global returns a new global variable intiailized with the specified constant
// value.
func (m *Module) Global(name string, val Const) Global {
	v := llvm.AddGlobal(m.llvm, val.Type.llvmTy(), name)
	v.SetInitializer(val.llvm)
	v.SetLinkage(llvm.PrivateLinkage)
	return Global{m.Types.Pointer(val.Type), v}
}

// Extern returns a global variable declared externally with the given name and
// type.
func (m *Module) Extern(name string, ty Type) Global {
	v := llvm.AddGlobal(m.llvm, ty.llvmTy(), name)
	return Global{m.Types.Pointer(ty), v}
}

// Const is an immutable value.
type Const struct {
	Type Type
	llvm llvm.Value
}

// Value returns a Value for the constant.
func (c Const) Value(b *Builder) *Value {
	return b.val(c.Type, c.llvm)
}

// Scalar returns a constant scalar with the value v.
func (m *Module) Scalar(v interface{}) Const {
	rty := reflect.TypeOf(v)
	ty := m.Types.TypeOf(v)
	var val llvm.Value
	switch {
	case rty.Kind() == reflect.Slice:
		rv := reflect.ValueOf(v)
		vals := make([]llvm.Value, rv.Len())
		for i := range vals {
			vals[i] = m.Scalar(rv.Index(i).Interface()).llvm
		}
		val = llvm.ConstArray(ty.llvmTy(), vals)
	case ty == m.Types.Bool:
		if reflect.ValueOf(v).Bool() {
			val = llvm.ConstInt(ty.llvmTy(), 1, false)
		} else {
			val = llvm.ConstInt(ty.llvmTy(), 0, false)
		}
	case IsSignedInteger(ty):
		val = llvm.ConstInt(ty.llvmTy(), uint64(reflect.ValueOf(v).Int()), false)
	case IsUnsignedInteger(ty):
		val = llvm.ConstInt(ty.llvmTy(), reflect.ValueOf(v).Uint(), false)
	case IsFloat(ty):
		val = llvm.ConstFloat(ty.llvmTy(), float64(reflect.ValueOf(v).Float()))
	case IsStruct(ty):
		ty := ty.(*Struct)
		r := reflect.ValueOf(v)
		fields := make([]llvm.Value, r.NumField())
		for i := range fields {
			fields[i] = m.Scalar(r.Field(i).Interface()).llvm
		}
		val = llvm.ConstNamedStruct(ty.llvm, fields)
	default:
		fail("Scalar does not support type %T", v)
	}

	if a, b := val.Type().String(), ty.llvmTy().String(); a != b {
		fail("Value type mismatch for %T: value: %v, type: %v", v, a, b)
	}

	val.SetName(fmt.Sprintf("%v", v))
	return Const{ty, val}
}

// ConstStruct returns a constant struct with the value v.
func (m *Module) ConstStruct(ty *Struct, fields map[string]interface{}) Const {
	vals := make([]llvm.Value, len(ty.Fields))
	for i, f := range ty.Fields {
		if v := fields[f.Name]; v == nil {
			vals[i] = llvm.ConstNull(f.Type.llvmTy())
		} else {
			vals[i] = m.Scalar(v).Cast(f.Type).llvm
		}
	}
	val := llvm.ConstNamedStruct(ty.llvm, vals)
	return Const{ty, val}
}
