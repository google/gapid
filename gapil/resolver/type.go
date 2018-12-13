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

package resolver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/semantic/printer"
)

const (
	RefRune     = 'ʳ'
	SliceRune   = 'ˢ'
	ConstRune   = 'ᶜ'
	PointerRune = 'ᵖ'
	ArrayRune   = 'ᵃ'
	MapRune     = 'ᵐ'
	TypeRune    = 'ː'

	RefSuffix     = string(RefRune)
	SliceSuffix   = string(SliceRune)
	ConstSuffix   = string(ConstRune)
	PointerSuffix = string(PointerRune)
	ArraySuffix   = string(ArrayRune)
	MapSuffix     = string(MapRune)
	TypeInfix     = string(TypeRune)
)

func type_(rv *resolver, in interface{}) semantic.Type {
	switch in := in.(type) {
	case *ast.Generic:
		return genericType(rv, in)
	case *ast.IndexedType:
		of := type_(rv, in.ValueType)
		if in.Index == nil {
			return getSliceType(rv, in, of)
		}
		var size uint32
		var sizeExpr semantic.Expression
		rv.with(semantic.Uint32Type, func() {
			e := expression(rv, in.Index)
			switch ty := e.(type) {
			case *semantic.DefinitionUsage:
				sizeExpr = ty.Definition
				e = ty.Expression
				rv.mappings.Remove(e)
			case *semantic.EnumEntry:
				sizeExpr = e
				e = ty.Value
			default:
				sizeExpr = e
			}
			if n, ok := e.(semantic.Uint32Value); ok {
				size = uint32(n)
			} else {
				rv.errorf(in.Index, "Array dimension must be a constant number, got %T", e)
			}
		})
		return getStaticArrayType(rv, in, of, size, sizeExpr)
	case *ast.PreConst:
		if ptr, ok := in.Type.(*ast.PointerType); ok {
			if ptr.Const {
				rv.errorf(in.Type, "Pointer type declared const twice (pre-const and post-const)")
			}
			to := type_(rv, ptr.To)
			return getPointerType(rv, ptr, to, true)
		}
		rv.errorf(in.Type, "Type %T cannot be declared const", in.Type)
		return semantic.VoidType
	case *ast.PointerType:
		to := type_(rv, in.To)
		return getPointerType(rv, in, to, in.Const)
	case *ast.Imported:
		return importedType(rv, in)
	case ast.Node:
		rv.errorf(in, "Unhandled typeref %T found", in)
		return semantic.VoidType
	default:
		rv.icef(in, "Non-node (%T) typeref found", in)
		return semantic.VoidType
	}
}

func genericType(rv *resolver, in *ast.Generic) semantic.Type {
	switch in.Name.Value {
	case "map":
		if len(in.Arguments) != 2 {
			rv.errorf(in, "Map requires 2 args, got %d", len(in.Arguments))
			return semantic.VoidType
		}
		kt := type_(rv, in.Arguments[0])
		vt := type_(rv, in.Arguments[1])
		return getMapType(rv, in, kt, vt, false)
	case "dense_map":
		if len(in.Arguments) != 2 {
			rv.errorf(in, "Map requires 2 args, got %d", len(in.Arguments))
			return semantic.VoidType
		}
		kt := type_(rv, in.Arguments[0])
		vt := type_(rv, in.Arguments[1])
		return getMapType(rv, in, kt, vt, true)
	case "ref":
		if len(in.Arguments) != 1 {
			rv.errorf(in, "Ref requires 1 arg, got %d", len(in.Arguments))
			return semantic.VoidType
		}
		vt := type_(rv, in.Arguments[0])
		return getRefType(rv, in, vt)
	default:
		if len(in.Arguments) != 0 {
			rv.errorf(in, "Type %s is not parameterised, got %d", in.Name, len(in.Arguments))
			return semantic.VoidType
		}
		return getSimpleType(rv, in.Name)
	}
}

func getSimpleType(rv *resolver, in *ast.Identifier) semantic.Type {
	name := in.Value
	out := rv.findType(in, name)
	if out == nil {
		rv.errorf(in, "Type %s not found", name)
		return semantic.VoidType
	}
	rv.mappings.Add(in, out)
	return out
}

func getMapType(rv *resolver, at ast.Node, kt, vt semantic.Type, isDense bool) *semantic.Map {
	denseText := ""
	if isDense {
		denseText = "_dense_"
	}
	name := fmt.Sprintf("%s%s%s%s%s", strings.Title(kt.Name()), TypeInfix, vt.Name(), denseText, MapSuffix)
	for _, m := range rv.api.Maps {
		if equal(kt, m.KeyType) && equal(vt, m.ValueType) && isDense == m.Dense {
			rv.mappings.Add(at, m)
			return m
		}
	}
	out := &semantic.Map{
		Named:     semantic.Named(name),
		KeyType:   kt,
		ValueType: vt,
		Dense:     isDense,
	}
	rv.api.Maps = append(rv.api.Maps, out)
	rv.mappings.Add(at, out)
	return out
}

func getStaticArrayType(rv *resolver, at ast.Node, of semantic.Type, size uint32, sizeExpr semantic.Expression) *semantic.StaticArray {
	name := fmt.Sprintf("%s%s%d%s", strings.Title(of.Name()), TypeInfix, size, ArraySuffix)
	for _, a := range rv.api.StaticArrays {
		if equal(a.ValueType, of) && a.Size == size {
			rv.mappings.Add(at, a)
			return a
		}
	}
	out := &semantic.StaticArray{
		Named:     semantic.Named(name),
		ValueType: of,
		Size:      size,
		SizeExpr:  sizeExpr,
	}
	rv.api.StaticArrays = append(rv.api.StaticArrays, out)
	rv.mappings.Add(at, out)
	return out
}

func getRefType(rv *resolver, at ast.Node, to semantic.Type) *semantic.Reference {
	name := strings.Title(to.Name()) + RefSuffix
	for _, p := range rv.api.References {
		if equal(to, p.To) {
			rv.mappings.Add(at, p)
			return p
		}
	}
	out := &semantic.Reference{
		Named: semantic.Named(name),
		To:    to,
	}
	rv.api.References = append(rv.api.References, out)
	rv.mappings.Add(at, out)
	return out
}

func getPointerType(rv *resolver, at ast.Node, to semantic.Type, constant bool) *semantic.Pointer {
	name := strings.Title(to.Name())
	if constant {
		name += ConstSuffix
	}
	name += PointerSuffix
	for _, p := range rv.api.Pointers {
		if equal(to, p.To) && constant == p.Const {
			rv.mappings.Add(at, p)
			return p
		}
	}
	out := &semantic.Pointer{
		Named: semantic.Named(name),
		To:    to,
		Const: constant,
	}
	rv.api.Pointers = append(rv.api.Pointers, out)
	rv.mappings.Add(at, out)

	out.Slice = getSliceType(rv, at, to)

	return out
}

func getSliceType(rv *resolver, at ast.Node, to semantic.Type) *semantic.Slice {
	name := strings.Title(to.Name()) + SliceSuffix
	for _, s := range rv.api.Slices {
		if equal(to, s.To) {
			rv.mappings.Add(at, s)
			return s
		}
	}
	out := &semantic.Slice{
		Named: semantic.Named(name),
		To:    to,
	}
	rv.api.Slices = append(rv.api.Slices, out)
	rv.mappings.Add(at, out)

	out.Pointer = getPointerType(rv, at, to, false)

	return out
}

func definition(rv *resolver, out *semantic.Definition) {
	rv.defStack.push(out)
	defer rv.defStack.pop()

	in := out.AST
	out.Annotations = annotations(rv, in.Annotations)
	out.Docs = rv.findDocumentation(in)
	out.Expression = expression(rv, in.Expression)
	rv.mappings.Add(in, out)
}

func enum(rv *resolver, out *semantic.Enum) {
	in := out.AST
	out.Docs = rv.findDocumentation(in)
	out.Annotations = annotations(rv, in.Annotations)
	out.IsBitfield = in.IsBitfield
	out.NumberType = semantic.Uint32Type
	if in.NumberType != nil {
		out.NumberType = type_(rv, in.NumberType)
	}
	if !semantic.IsInteger(out.NumberType) {
		rv.errorf(in.NumberType, "enum numerical type must be an integer, got: %v", out.NumberType)
	}
	for _, e := range in.Entries {
		var v semantic.Expression
		var err error
		intSize := semantic.IntegerSizeInBits(out.NumberType)
		if semantic.IsUnsigned(out.NumberType) {
			var i uint64
			i, err = strconv.ParseUint(e.Value.Value, 0, intSize)
			switch intSize {
			case 8:
				v = semantic.Uint8Value(i)
			case 16:
				v = semantic.Uint16Value(i)
			case 32:
				v = semantic.Uint32Value(i)
			case 64:
				v = semantic.Uint64Value(i)
			default:
				panic(fmt.Errorf("Unexpected enum entry unsigned integer size: %v", intSize))
			}
		} else {
			var i int64
			i, err = strconv.ParseInt(e.Value.Value, 0, intSize)
			switch intSize {
			case 8:
				v = semantic.Int8Value(i)
			case 16:
				v = semantic.Int16Value(i)
			case 32:
				v = semantic.Int32Value(i)
			case 64:
				v = semantic.Int64Value(i)
			default:
				panic(fmt.Errorf("Unexpected enum entry signed integer size: %v", intSize))
			}
		}
		if err != nil {
			rv.errorf(e, "could not parse %v as %v", e.Value, out.NumberType)
			continue
		}

		entry := &semantic.EnumEntry{
			AST:   e,
			Named: semantic.Named(e.Name.Value),
			Docs:  rv.findDocumentation(e),
			Value: v,
		}
		out.Entries = append(out.Entries, entry)
		rv.mappings.Add(e, entry)
		semantic.Add(out, entry)
		rv.addNamed(entry)
	}
	rv.mappings.Add(in, out)
}

func class(rv *resolver, out *semantic.Class) {
	in := out.AST
	out.Docs = rv.findDocumentation(in)
	out.Annotations = annotations(rv, in.Annotations)
	for _, f := range in.Fields {
		out.Fields = append(out.Fields, field(rv, f, out))
	}
	rv.mappings.Add(in, out)
	rv.mappings.Add(in.Name, out)
}

func field(rv *resolver, in *ast.Field, class *semantic.Class) *semantic.Field {
	out := &semantic.Field{AST: in, Named: semantic.Named(in.Name.Value)}
	semantic.Add(class, out)
	out.Docs = rv.findDocumentation(in)
	out.Annotations = annotations(rv, in.Annotations)
	out.Type = type_(rv, in.Type)
	if isVoid(out.Type) {
		rv.errorf(in, "void typed field %s on class %s", out.Name(), class.Name())
	}
	if in.Default != nil {
		rv.with(out.Type, func() {
			out.Default = expression(rv, in.Default)
		})
		dt := out.Default.ExpressionType()
		if !assignable(out.Type, dt) {
			rv.errorf(in, "cannot assign %s to %s", typename(dt), typename(out.Type))
		}
	}
	rv.mappings.Add(in, out)
	rv.mappings.Add(in.Name, out)
	return out
}

func pseudonym(rv *resolver, out *semantic.Pseudonym) {
	in := out.AST
	out.Docs = rv.findDocumentation(in)
	out.Annotations = annotations(rv, in.Annotations)
	out.To = type_(rv, in.To)
	rv.mappings.Add(in, out)
	// Check the pseudonym doesn't refer to itself
	path := stack{out}
	for ty := semantic.Type(out.To); ty != nil; {
		path.push(ty)
		if ty == out {
			rv.errorf(in, "cyclic type declaration: %v", path)
			out.To = semantic.VoidType
			return
		}
		switch t := ty.(type) {
		case *semantic.Pseudonym:
			ty = t.To
		case *semantic.Pointer:
			ty = t.To
		case *semantic.Slice:
			ty = t.To
		default:
			return // no cycles.
		}
	}
}

func importedType(rv *resolver, in *ast.Imported) semantic.Type {
	api, ok := rv.get(in, in.From.Value).(*semantic.API)
	if !ok {
		rv.errorf(in, "%s not an imported api", in.From.Value)
		return semantic.VoidType
	}
	t, ok := api.Member(in.Name.Value).(semantic.Type)
	if !ok {
		rv.errorf(in, "%s not a type in %s", in.Name.Value, in.From.Value)
		return semantic.VoidType
	}
	return t
}

func typename(t semantic.Type) string {
	return printer.New().WriteType(t).String()
}
