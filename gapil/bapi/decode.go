// Copyright (C) 2018 Google Inc.
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

package bapi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

type decoder struct {
	content Content
	inst    decoderInstances
	toSort  []semantic.Owner
	err     error
}

type decoderInstances struct {
	API         []*semantic.API
	StaticArray []*semantic.StaticArray
	Block       []*semantic.Block
	Callable    []*semantic.Callable
	Class       []*semantic.Class
	Definition  []*semantic.Definition
	Enum        []*semantic.Enum
	EnumEntry   []*semantic.EnumEntry
	Expression  []semantic.Expression
	Field       []*semantic.Field
	Function    []*semantic.Function
	Global      []*semantic.Global
	Local       []*semantic.Local
	Map         []*semantic.Map
	Parameter   []*semantic.Parameter
	Pointer     []*semantic.Pointer
	Pseudonym   []*semantic.Pseudonym
	Reference   []*semantic.Reference
	Signature   []*semantic.Signature
	Slice       []*semantic.Slice
	Statement   []semantic.Statement

	ASTAnnotation   []*ast.Annotation
	ASTAbort        []*ast.Abort
	ASTAssign       []*ast.Assign
	ASTBinaryOp     []*ast.BinaryOp
	ASTBlock        []*ast.Block
	ASTBool         []*ast.Bool
	ASTBranch       []*ast.Branch
	ASTCall         []*ast.Call
	ASTCase         []*ast.Case
	ASTClass        []*ast.Class
	ASTDeclareLocal []*ast.DeclareLocal
	ASTDefault      []*ast.Default
	ASTDefinition   []*ast.Definition
	ASTDelete       []*ast.Delete
	ASTEnum         []*ast.Enum
	ASTEnumEntry    []*ast.EnumEntry
	ASTFence        []*ast.Fence
	ASTField        []*ast.Field
	ASTFunction     []*ast.Function
	ASTGeneric      []*ast.Generic
	ASTGroup        []*ast.Group
	ASTIdentifier   []*ast.Identifier
	ASTIndex        []*ast.Index
	ASTIndexedType  []*ast.IndexedType
	ASTIteration    []*ast.Iteration
	ASTMapIteration []*ast.MapIteration
	ASTMember       []*ast.Member
	ASTNamedArg     []*ast.NamedArg
	ASTNull         []*ast.Null
	ASTNumber       []*ast.Number
	ASTParameter    []*ast.Parameter
	ASTPointerType  []*ast.PointerType
	ASTPreConst     []*ast.PreConst
	ASTPseudonym    []*ast.Pseudonym
	ASTReturn       []*ast.Return
	ASTString       []*ast.String
	ASTSwitch       []*ast.Switch
	ASTUnaryOp      []*ast.UnaryOp
	ASTUnknown      []*ast.Unknown
}

func (i *decoderInstances) build(p *Instances) {
	toProtoName := func(s string) string {
		s = strings.Replace(s, "AST", "Ast", -1)
		s = strings.Replace(s, "API", "Api", -1)
		return s
	}
	iV, pV := reflect.ValueOf(i).Elem(), reflect.ValueOf(p).Elem()
	for i, c := 0, iV.NumField(); i < c; i++ {
		name := toProtoName(iV.Type().Field(i).Name)
		pf := pV.FieldByName(name)
		if !pf.IsValid() {
			panic(fmt.Errorf("Proto Instances did not have a field with name %v", name))
		}
		count := pf.Len()
		iV.Field(i).Set(reflect.MakeSlice(iV.Field(i).Type(), count, count))
	}
}

// Decode deserializes the APIs from data.
func Decode(data []byte) ([]*semantic.API, error) {
	d := decoder{}
	if err := proto.Unmarshal(data, &d.content); err != nil {
		return nil, err
	}
	d.inst.build(d.content.Instances)

	apis := make([]*semantic.API, len(d.content.Apis))
	for i, id := range d.content.Apis {
		apis[i] = d.api(id)
		if d.err != nil {
			return nil, d.err
		}
	}
	for _, s := range d.toSort {
		s.SortMembers()
	}
	return apis, nil
}

func (d *decoder) api(apiID uint64) (out *semantic.API) {
	d.build(d.content.Instances.Api, &d.inst.API, apiID, &out, func(p *API, s *semantic.API) {
		s.Named = semantic.Named(d.str(p.Name))
		foreach(p.Enums, d.enum, &s.Enums)
		foreach(p.Definitions, d.definition, &s.Definitions)
		foreach(p.Classes, d.class, &s.Classes)
		foreach(p.Pseudonyms, d.pseudonym, &s.Pseudonyms)
		foreach(p.Externs, d.function, &s.Externs)
		foreach(p.Subroutines, d.function, &s.Subroutines)
		foreach(p.Functions, d.function, &s.Functions)
		foreach(p.Globals, d.global, &s.Globals)
		foreach(p.StaticArrays, d.array, &s.StaticArrays)
		foreach(p.Maps, d.map_, &s.Maps)
		foreach(p.Pointers, d.pointer, &s.Pointers)
		foreach(p.Slices, d.slice, &s.Slices)
		foreach(p.References, d.reference, &s.References)
		foreach(p.Signatures, d.signature, &s.Signatures)
		idx := semantic.Uint8Value(p.Index)
		s.Index = &idx
		d.toSort = append(d.toSort, s)
	})
	return
}

func (d *decoder) array(arrayID uint64) (out *semantic.StaticArray) {
	d.build(d.content.Instances.StaticArray, &d.inst.StaticArray, arrayID, &out, func(p *StaticArray, s *semantic.StaticArray) {
		s.Named = semantic.Named(d.str(p.Name))
		s.ValueType = d.ty(p.ValueType)
		s.Size = uint32(p.Size)
		s.SizeExpr = d.expr(p.SizeExpr)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) block(blockID uint64) (out *semantic.Block) {
	d.build(d.content.Instances.Block, &d.inst.Block, blockID, &out, func(p *Block, s *semantic.Block) {
		s.AST = d.astBlock(p.Ast)
		foreach(p.Statements, d.stat, &s.Statements)
	})
	return
}

func (d *decoder) callable(callableID uint64) (out *semantic.Callable) {
	d.build(d.content.Instances.Callable, &d.inst.Callable, callableID, &out, func(p *Callable, s *semantic.Callable) {
		s.Function = d.function(p.Function)
		s.Object = d.expr(p.Object)
	})
	return
}

func (d *decoder) case_(p *Case) *semantic.Case {
	s := &semantic.Case{
		AST:         d.astCase(p.Ast),
		Annotations: d.annotations(p.Annotations),
		Block:       d.block(p.Block),
	}
	foreach(p.Conditions, d.expr, &s.Conditions)
	return s
}

func (d *decoder) choice(p *Choice) *semantic.Choice {
	s := &semantic.Choice{}
	s.AST = d.astCase(p.Ast)
	s.Annotations = d.annotations(p.Annotations)
	s.Expression = d.expr(p.Expression)
	foreach(p.Conditions, d.expr, &s.Conditions)
	return s
}

func (d *decoder) class(classID uint64) (out *semantic.Class) {
	d.build(d.content.Instances.Class, &d.inst.Class, classID, &out, func(p *Class, s *semantic.Class) {
		s.AST = d.astClass(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		foreach(p.Fields, d.field, &s.Fields)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
		d.toSort = append(d.toSort, s)
	})
	return
}

func (d *decoder) definition(definitionID uint64) (out *semantic.Definition) {
	d.build(d.content.Instances.Definition, &d.inst.Definition, definitionID, &out, func(p *Definition, s *semantic.Definition) {
		s.AST = d.astDefinition(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.Expression = d.expr(p.Expression)
	})
	return
}

func (d *decoder) enum(enumID uint64) (out *semantic.Enum) {
	d.build(d.content.Instances.Enum, &d.inst.Enum, enumID, &out, func(p *Enum, s *semantic.Enum) {
		s.AST = d.astEnum(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.IsBitfield = p.IsBitfield
		s.NumberType = d.ty(p.NumberType)
		foreach(p.Entries, d.enumEntry, &s.Entries)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) enumEntry(enumEntryID uint64) (out *semantic.EnumEntry) {
	d.build(d.content.Instances.EnumEntry, &d.inst.EnumEntry, enumEntryID, &out, func(p *EnumEntry, s *semantic.EnumEntry) {
		s.AST = d.astEnumEntry(p.Ast)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.Value = d.expr(p.Value)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) expr(exprID uint64) (out semantic.Expression) {
	if exprID == 0 {
		return nil
	}
	exprIdx := exprID - 1
	if existing := d.inst.Expression[exprIdx]; existing != nil {
		return existing
	}

	// setOut assigns o to out and places o into the instances map.
	// This can be called before returning to break cyclic dependencies.
	setOut := func(o semantic.Expression) {
		out = o
		d.inst.Expression[exprIdx] = o
	}

	defer func() { setOut(out) }()

	switch p := protoutil.OneOf(d.content.Instances.Expression[exprIdx]).(type) {
	case *Expression_ArrayInitializer:
		s := &semantic.ArrayInitializer{}
		setOut(s)
		s.AST = d.astCall(p.ArrayInitializer.Ast)
		s.Array = d.ty(p.ArrayInitializer.Array)
		foreach(p.ArrayInitializer.Values, d.expr, &s.Values)
		return s
	case *Expression_ArrayIndex:
		s := &semantic.ArrayIndex{}
		setOut(s)
		s.AST = d.astIndex(p.ArrayIndex.Ast)
		s.Type = d.array(p.ArrayIndex.Type)
		s.Array = d.expr(p.ArrayIndex.Array)
		s.Index = d.expr(p.ArrayIndex.Index)
		return s
	case *Expression_BinaryOp:
		s := &semantic.BinaryOp{}
		setOut(s)
		s.AST = d.astBinaryOp(p.BinaryOp.Ast)
		s.Type = d.ty(p.BinaryOp.Type)
		s.LHS = d.expr(p.BinaryOp.Lhs)
		s.Operator = d.str(p.BinaryOp.Operator)
		s.RHS = d.expr(p.BinaryOp.Rhs)
		return s
	case *Expression_BitTest:
		s := &semantic.BitTest{}
		setOut(s)
		s.AST = d.astBinaryOp(p.BitTest.Ast)
		s.Bitfield = d.expr(p.BitTest.Bitfield)
		s.Bits = d.expr(p.BitTest.Bits)
		return s
	case *Expression_BoolValue:
		return semantic.BoolValue(p.BoolValue)
	case *Expression_Call:
		s := &semantic.Call{}
		setOut(s)
		s.AST = d.astCall(p.Call.Ast)
		s.Target = d.callable(p.Call.Target)
		s.Type = d.ty(p.Call.Type)
		foreach(p.Call.Arguments, d.expr, &s.Arguments)
		return s
	case *Expression_Cast:
		s := &semantic.Cast{}
		setOut(s)
		s.AST = d.astCall(p.Cast.Ast)
		s.Object = d.expr(p.Cast.Object)
		s.Type = d.ty(p.Cast.Type)
		return s
	case *Expression_ClassInitializer:
		s := &semantic.ClassInitializer{}
		setOut(s)
		s.AST = d.astCall(p.ClassInitializer.Ast)
		s.Class = d.class(p.ClassInitializer.Class)
		foreach(p.ClassInitializer.Fields, d.fieldInit, &s.Fields)
		return s
	case *Expression_Clone:
		s := &semantic.Clone{}
		setOut(s)
		s.AST = d.astCall(p.Clone.Ast)
		s.Slice = d.expr(p.Clone.Slice)
		s.Type = d.slice(p.Clone.Type)
		return s
	case *Expression_Create:
		s := &semantic.Create{}
		setOut(s)
		s.AST = d.astCall(p.Create.Ast)
		s.Type = d.reference(p.Create.Type)
		s.Initializer = d.expr(p.Create.Initializer).(*semantic.ClassInitializer)
		return s
	case *Expression_Definition:
		return d.definition(p.Definition)
	case *Expression_EnumEntry:
		return d.enumEntry(p.EnumEntry)
	case *Expression_Float32Value:
		return semantic.Float32Value(p.Float32Value)
	case *Expression_Float64Value:
		return semantic.Float64Value(p.Float64Value)
	case *Expression_Global:
		return d.global(p.Global)
	case *Expression_Ignore:
		return &semantic.Ignore{AST: d.astNode(p.Ignore.Ast)}
	case *Expression_Int8Value:
		return semantic.Int8Value(p.Int8Value)
	case *Expression_Int16Value:
		return semantic.Int16Value(p.Int16Value)
	case *Expression_Int32Value:
		return semantic.Int32Value(p.Int32Value)
	case *Expression_Int64Value:
		return semantic.Int64Value(p.Int64Value)
	case *Expression_Length:
		s := &semantic.Length{}
		setOut(s)
		s.AST = d.astCall(p.Length.Ast)
		s.Object = d.expr(p.Length.Object)
		s.Type = d.ty(p.Length.Type)
		return s
	case *Expression_Local:
		return d.local(p.Local)
	case *Expression_Make:
		s := &semantic.Make{}
		setOut(s)
		s.AST = d.astCall(p.Make.Ast)
		s.Type = d.slice(p.Make.Type)
		s.Size = d.expr(p.Make.Size)
		return s
	case *Expression_MapContains:
		s := &semantic.MapContains{}
		setOut(s)
		s.AST = d.astBinaryOp(p.MapContains.Ast)
		s.Type = d.map_(p.MapContains.Type)
		s.Map = d.expr(p.MapContains.Map)
		s.Key = d.expr(p.MapContains.Key)
		return s
	case *Expression_MapIndex:
		s := &semantic.MapIndex{}
		setOut(s)
		s.AST = d.astIndex(p.MapIndex.Ast)
		s.Type = d.map_(p.MapIndex.Type)
		s.Map = d.expr(p.MapIndex.Map)
		s.Index = d.expr(p.MapIndex.Index)
		return s
	case *Expression_Member:
		s := &semantic.Member{}
		setOut(s)
		s.AST = d.astMember(p.Member.Ast)
		s.Field = d.field(p.Member.Field)
		s.Object = d.expr(p.Member.Object)
		return s
	case *Expression_MessageValue:
		s := &semantic.MessageValue{}
		setOut(s)
		s.AST = d.astClass(p.MessageValue.Ast)
		foreach(p.MessageValue.Arguments, d.fieldInit, &s.Arguments)
		return s
	case *Expression_Null:
		s := semantic.Null{}
		setOut(s)
		s.AST = d.astNull(p.Null.Ast)
		s.Type = d.ty(p.Null.Type)
		return s
	case *Expression_Observed:
		return &semantic.Observed{Parameter: d.param(p.Observed.Parameter)}
	case *Expression_Parameter:
		return d.param(p.Parameter)
	case *Expression_PointerRange:
		s := &semantic.PointerRange{}
		setOut(s)
		s.AST = d.astIndex(p.PointerRange.Ast)
		s.Type = d.slice(p.PointerRange.Type)
		s.Pointer = d.expr(p.PointerRange.Pointer)
		s.Range = d.expr(p.PointerRange.Range).(*semantic.BinaryOp)
		return s
	case *Expression_Select:
		s := &semantic.Select{}
		setOut(s)
		s.AST = d.astSwitch(p.Select.Ast)
		s.Type = d.ty(p.Select.Type)
		s.Value = d.expr(p.Select.Value)
		s.Default = d.expr(p.Select.Default)
		foreach(p.Select.Choices, d.choice, &s.Choices)
		return s
	case *Expression_SliceIndex:
		s := &semantic.SliceIndex{}
		setOut(s)
		s.AST = d.astIndex(p.SliceIndex.Ast)
		s.Type = d.slice(p.SliceIndex.Type)
		s.Slice = d.expr(p.SliceIndex.Slice)
		s.Index = d.expr(p.SliceIndex.Index)
		return s
	case *Expression_SliceRange:
		s := &semantic.SliceRange{}
		setOut(s)
		s.AST = d.astIndex(p.SliceRange.Ast)
		s.Type = d.slice(p.SliceRange.Type)
		s.Slice = d.expr(p.SliceRange.Slice)
		s.Range = d.expr(p.SliceRange.Range).(*semantic.BinaryOp)
		return s
	case *Expression_StringValue:
		return semantic.StringValue(d.str(p.StringValue))
	case *Expression_Uint8Value:
		return semantic.Uint8Value(p.Uint8Value)
	case *Expression_Uint16Value:
		return semantic.Uint16Value(p.Uint16Value)
	case *Expression_Uint32Value:
		return semantic.Uint32Value(p.Uint32Value)
	case *Expression_Uint64Value:
		return semantic.Uint64Value(p.Uint64Value)
	case *Expression_UnaryOp:
		s := &semantic.UnaryOp{}
		setOut(s)
		s.AST = d.astUnaryOp(p.UnaryOp.Ast)
		s.Type = d.ty(p.UnaryOp.Type)
		s.Operator = d.str(p.UnaryOp.Operator)
		s.Expression = d.expr(p.UnaryOp.Expression)
		return s
	case *Expression_Unknown:
		s := &semantic.Unknown{}
		setOut(s)
		s.AST = d.astUnknown(p.Unknown.Ast)
		s.Inferred = d.expr(p.Unknown.Inferred)
		return s
	default:
		panic(fmt.Errorf("Unhandled expression type %T", p))
	}
}

func (d *decoder) field(fieldID uint64) (out *semantic.Field) {
	d.build(d.content.Instances.Field, &d.inst.Field, fieldID, &out, func(p *Field, s *semantic.Field) {
		s.AST = d.astField(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.Type = d.ty(p.Type)
		s.Default = d.expr(p.Default)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) fieldInit(p *FieldInitializer) *semantic.FieldInitializer {
	return &semantic.FieldInitializer{
		AST:   d.astNode(p.Ast),
		Field: d.field(p.Field),
		Value: d.expr(p.Value),
	}
}

func (d *decoder) function(funcID uint64) (out *semantic.Function) {
	d.build(d.content.Instances.Function, &d.inst.Function, funcID, &out, func(p *Function, s *semantic.Function) {
		s.AST = d.astFunction(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.Return = d.param(p.Return)
		s.This = d.param(p.This)
		foreach(p.FullParameters, d.param, &s.FullParameters)
		s.Block = d.block(p.Block)
		s.Signature = d.signature(p.Signature)
		s.Extern = p.Extern
		s.Subroutine = p.Subroutine
		s.Recursive = p.Recursive
		if p.Order.Resolved {
			s.Order |= semantic.Resolved
		}
		if p.Order.Pre {
			s.Order |= semantic.Pre
		}
		if p.Order.Post {
			s.Order |= semantic.Post
		}
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) global(globalID uint64) (out *semantic.Global) {
	d.build(d.content.Instances.Global, &d.inst.Global, globalID, &out, func(p *Global, s *semantic.Global) {
		s.AST = d.astField(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Type = d.ty(p.Type)
		s.Default = d.expr(p.Default)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) local(localID uint64) (out *semantic.Local) {
	d.build(d.content.Instances.Local, &d.inst.Local, localID, &out, func(p *Local, s *semantic.Local) {
		if p.Declaration != 0 {
			s.Declaration = d.stat(p.Declaration).(*semantic.DeclareLocal)
		}
		s.Type = d.ty(p.Type)
		s.Named = semantic.Named(d.str(p.Name))
		s.Value = d.expr(p.Value)
	})
	return
}

func (d *decoder) map_(mapID uint64) (out *semantic.Map) {
	d.build(d.content.Instances.Map, &d.inst.Map, mapID, &out, func(p *Map, s *semantic.Map) {
		s.Named = semantic.Named(d.str(p.Name))
		s.KeyType = d.ty(p.KeyType)
		s.ValueType = d.ty(p.ValueType)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) param(paramID uint64) (out *semantic.Parameter) {
	d.build(d.content.Instances.Parameter, &d.inst.Parameter, paramID, &out, func(p *Parameter, s *semantic.Parameter) {
		s.AST = d.astParameter(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Function = d.function(p.Function)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.Type = d.ty(p.Type)
	})
	return
}

func (d *decoder) pointer(ptrID uint64) (out *semantic.Pointer) {
	d.build(d.content.Instances.Pointer, &d.inst.Pointer, ptrID, &out, func(p *Pointer, s *semantic.Pointer) {
		s.Named = semantic.Named(d.str(p.Name))
		s.To = d.ty(p.To)
		s.Const = p.Const
		s.Slice = d.slice(p.Slice)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) pseudonym(pseudID uint64) (out *semantic.Pseudonym) {
	d.build(d.content.Instances.Pseudonym, &d.inst.Pseudonym, pseudID, &out, func(p *Pseudonym, s *semantic.Pseudonym) {
		s.AST = d.astPseudonym(p.Ast)
		s.Annotations = d.annotations(p.Annotations)
		s.Named = semantic.Named(d.str(p.Name))
		s.Docs = d.docs(p.Docs)
		s.To = d.ty(p.To)
		foreach(p.Methods, d.function, &s.Methods)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) reference(refID uint64) (out *semantic.Reference) {
	d.build(d.content.Instances.Reference, &d.inst.Reference, refID, &out, func(p *Reference, s *semantic.Reference) {
		s.Named = semantic.Named(d.str(p.Name))
		s.To = d.ty(p.To)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) signature(sigID uint64) (out *semantic.Signature) {
	d.build(d.content.Instances.Signature, &d.inst.Signature, sigID, &out, func(p *Signature, s *semantic.Signature) {
		s.Named = semantic.Named(d.str(p.Name))
		s.Return = d.ty(p.Return)
		foreach(p.Arguments, d.ty, &s.Arguments)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) slice(sliID uint64) (out *semantic.Slice) {
	d.build(d.content.Instances.Slice, &d.inst.Slice, sliID, &out, func(p *Slice, s *semantic.Slice) {
		s.Named = semantic.Named(d.str(p.Name))
		s.To = d.ty(p.To)
		s.Pointer = d.pointer(p.Pointer)
		if owner := d.node(p.Owner); owner != nil {
			semantic.Add(owner.(semantic.Owner), s)
		}
	})
	return
}

func (d *decoder) stat(statID uint64) (out semantic.Statement) {
	if statID == 0 {
		return nil
	}
	statIdx := statID - 1
	if existing := d.inst.Statement[statIdx]; existing != nil {
		return existing
	}

	// setOut assigns o to out and places o into the instances map.
	// This can be called before returning to break cyclic dependencies.
	setOut := func(o semantic.Statement) {
		out = o
		d.inst.Statement[statIdx] = o
	}

	defer func() { setOut(out) }()

	switch p := protoutil.OneOf(d.content.Instances.Statement[statIdx]).(type) {
	case *Statement_Abort:
		s := &semantic.Abort{}
		setOut(s)
		s.AST = d.astAbort(p.Abort.Ast)
		s.Function = d.function(p.Abort.Function)
		s.Statement = d.stat(p.Abort.Statement)
		return s
	case *Statement_Assert:
		s := &semantic.Assert{}
		setOut(s)
		s.AST = d.astCall(p.Assert.Ast)
		s.Condition = d.expr(p.Assert.Condition)
		return s
	case *Statement_ArrayAssign:
		s := &semantic.ArrayAssign{}
		setOut(s)
		s.AST = d.astAssign(p.ArrayAssign.Ast)
		s.To = d.expr(p.ArrayAssign.To).(*semantic.ArrayIndex)
		s.Operator = d.str(p.ArrayAssign.Operator)
		s.Value = d.expr(p.ArrayAssign.Value)
		return s
	case *Statement_Assign:
		s := &semantic.Assign{}
		setOut(s)
		s.AST = d.astAssign(p.Assign.Ast)
		s.LHS = d.expr(p.Assign.Lhs)
		s.Operator = d.str(p.Assign.Operator)
		s.RHS = d.expr(p.Assign.Rhs)
		return s
	case *Statement_Branch:
		s := &semantic.Branch{}
		setOut(s)
		s.AST = d.astBranch(p.Branch.Ast)
		s.Condition = d.expr(p.Branch.Condition)
		s.True = d.block(p.Branch.True)
		s.False = d.block(p.Branch.False)
		return s
	case *Statement_Copy:
		s := &semantic.Copy{}
		setOut(s)
		s.AST = d.astCall(p.Copy.Ast)
		s.Src = d.expr(p.Copy.Src)
		s.Dst = d.expr(p.Copy.Dst)
		return s
	case *Statement_DeclareLocal:
		s := &semantic.DeclareLocal{}
		setOut(s)
		s.AST = d.astDeclareLocal(p.DeclareLocal.Ast)
		s.Local = d.local(p.DeclareLocal.Local)
		return s
	case *Statement_Expression:
		return d.expr(p.Expression).(semantic.Statement)
	case *Statement_Fence:
		s := &semantic.Fence{}
		setOut(s)
		s.AST = d.astFence(p.Fence.Ast)
		s.Statement = d.stat(p.Fence.Statement)
		s.Explicit = p.Fence.Explicit
		return s
	case *Statement_Iteration:
		s := &semantic.Iteration{}
		setOut(s)
		s.AST = d.astIteration(p.Iteration.Ast)
		s.Iterator = d.local(p.Iteration.Iterator)
		s.From = d.expr(p.Iteration.From)
		s.To = d.expr(p.Iteration.To)
		s.Block = d.block(p.Iteration.Block)
		return s
	case *Statement_MapAssign:
		s := &semantic.MapAssign{}
		setOut(s)
		s.AST = d.astAssign(p.MapAssign.Ast)
		s.Operator = d.str(p.MapAssign.Operator)
		s.To = d.expr(p.MapAssign.To).(*semantic.MapIndex)
		s.Value = d.expr(p.MapAssign.Value)
		return s
	case *Statement_MapIteration:
		s := &semantic.MapIteration{}
		setOut(s)
		s.AST = d.astMapIteration(p.MapIteration.Ast)
		s.IndexIterator = d.local(p.MapIteration.IndexIterator)
		s.KeyIterator = d.local(p.MapIteration.KeyIterator)
		s.ValueIterator = d.local(p.MapIteration.ValueIterator)
		s.Map = d.expr(p.MapIteration.Map)
		s.Block = d.block(p.MapIteration.Block)
		return s
	case *Statement_MapRemove:
		s := &semantic.MapRemove{}
		setOut(s)
		s.AST = d.astDelete(p.MapRemove.Ast)
		s.Type = d.map_(p.MapRemove.Type)
		s.Map = d.expr(p.MapRemove.Map)
		s.Key = d.expr(p.MapRemove.Key)
		return s
	case *Statement_Read:
		s := &semantic.Read{}
		setOut(s)
		s.AST = d.astCall(p.Read.Ast)
		s.Slice = d.expr(p.Read.Slice)
		return s
	case *Statement_Return:
		s := &semantic.Return{}
		setOut(s)
		s.AST = d.astReturn(p.Return.Ast)
		s.Function = d.function(p.Return.Function)
		s.Value = d.expr(p.Return.Value)
		return s
	case *Statement_SliceAssign:
		s := &semantic.SliceAssign{}
		setOut(s)
		s.AST = d.astAssign(p.SliceAssign.Ast)
		s.To = d.expr(p.SliceAssign.To).(*semantic.SliceIndex)
		s.Operator = d.str(p.SliceAssign.Operator)
		s.Value = d.expr(p.SliceAssign.Value)
		return s
	case *Statement_Switch:
		s := &semantic.Switch{}
		setOut(s)
		s.AST = d.astSwitch(p.Switch.Ast)
		s.Value = d.expr(p.Switch.Value)
		s.Default = d.block(p.Switch.Default)
		foreach(p.Switch.Cases, d.case_, &s.Cases)
		return s
	case *Statement_Write:
		s := &semantic.Write{}
		setOut(s)
		s.AST = d.astCall(p.Write.Ast)
		s.Slice = d.expr(p.Write.Slice)
		return s
	default:
		panic(fmt.Errorf("Unhandled statement type %T", p))
	}
}

func (d *decoder) str(stringID uint64) string {
	if stringID == 0 {
		return ""
	}
	return d.content.Instances.Symbols[stringID-1]
}

func (d *decoder) annotation(p *Annotation) *semantic.Annotation {
	out := &semantic.Annotation{
		AST:   d.astAnnotation(p.Ast),
		Named: semantic.Named(d.str(p.Name)),
	}
	if len(p.Arguments) > 0 {
		foreach(p.Arguments, d.expr, &out.Arguments)
	}
	return out
}

func (d *decoder) annotations(p *Annotations) semantic.Annotations {
	if p == nil || len(p.Annotations) == 0 {
		return nil
	}
	out := semantic.Annotations{}
	foreach(p.Annotations, d.annotation, &out)
	return out
}

func (d *decoder) docs(p *Documentation) semantic.Documentation {
	s := semantic.Documentation{}
	foreach(p.Strings, d.str, &s)
	return s
}

func (d *decoder) node(p *Node) semantic.Node {
	if p == nil {
		return nil
	}
	switch p := protoutil.OneOf(p).(type) {
	case *Node_Api:
		return d.api(p.Api)
	case *Node_Block:
		return d.block(p.Block)
	case *Node_Class:
		return d.class(p.Class)
	case *Node_Definition:
		return d.definition(p.Definition)
	case *Node_EnumEntry:
		return d.enumEntry(p.EnumEntry)
	case *Node_Enum:
		return d.enum(p.Enum)
	case *Node_Expression:
		return d.expr(p.Expression)
	case *Node_Field:
		return d.field(p.Field)
	case *Node_Function:
		return d.function(p.Function)
	case *Node_Global:
		return d.global(p.Global)
	case *Node_Local:
		return d.local(p.Local)
	case *Node_Map:
		return d.map_(p.Map)
	case *Node_Parameter:
		return d.param(p.Parameter)
	case *Node_Pointer:
		return d.pointer(p.Pointer)
	case *Node_Pseudonym:
		return d.pseudonym(p.Pseudonym)
	case *Node_Reference:
		return d.reference(p.Reference)
	case *Node_Signature:
		return d.signature(p.Signature)
	case *Node_Slice:
		return d.slice(p.Slice)
	case *Node_Statement:
		return d.stat(p.Statement)
	case *Node_StaticArray:
		return d.array(p.StaticArray)
	}
	return nil
}

func (d *decoder) ty(p *Type) semantic.Type {
	if p == nil {
		return nil
	}
	switch p := protoutil.OneOf(p).(type) {
	case *Type_Builtin:
		switch p.Builtin {
		case Builtin_AnyType:
			return semantic.AnyType
		case Builtin_BoolType:
			return semantic.BoolType
		case Builtin_CharType:
			return semantic.CharType
		case Builtin_Float32Type:
			return semantic.Float32Type
		case Builtin_Float64Type:
			return semantic.Float64Type
		case Builtin_IntType:
			return semantic.IntType
		case Builtin_Int8Type:
			return semantic.Int8Type
		case Builtin_Int16Type:
			return semantic.Int16Type
		case Builtin_Int32Type:
			return semantic.Int32Type
		case Builtin_Int64Type:
			return semantic.Int64Type
		case Builtin_MessageType:
			return semantic.MessageType
		case Builtin_SizeType:
			return semantic.SizeType
		case Builtin_StringType:
			return semantic.StringType
		case Builtin_UintType:
			return semantic.UintType
		case Builtin_Uint8Type:
			return semantic.Uint8Type
		case Builtin_Uint16Type:
			return semantic.Uint16Type
		case Builtin_Uint32Type:
			return semantic.Uint32Type
		case Builtin_Uint64Type:
			return semantic.Uint64Type
		case Builtin_VoidType:
			return semantic.VoidType
		default:
			panic(fmt.Errorf("Unhandled builtin %v", p.Builtin))
		}
	case *Type_Class:
		return d.class(p.Class)
	case *Type_Enum:
		return d.enum(p.Enum)
	case *Type_Map:
		return d.map_(p.Map)
	case *Type_Pointer:
		return d.pointer(p.Pointer)
	case *Type_Pseudonym:
		return d.pseudonym(p.Pseudonym)
	case *Type_Reference:
		return d.reference(p.Reference)
	case *Type_Slice:
		return d.slice(p.Slice)
	case *Type_StaticArray:
		return d.array(p.StaticArray)
	default:
		panic(fmt.Errorf("Unhandled type %T", p))
	}
}

func (d *decoder) astAbort(astAbortID uint64) (out *ast.Abort) {
	d.build(d.content.Instances.AstAbort, &d.inst.ASTAbort, astAbortID, &out, func(p *ASTAbort, s *ast.Abort) {})
	return
}

func (d *decoder) astAnnotation(astAnnotationID uint64) (out *ast.Annotation) {
	d.build(d.content.Instances.AstAnnotation, &d.inst.ASTAnnotation, astAnnotationID, &out, func(p *ASTAnnotation, s *ast.Annotation) {
		s.Name = d.astIdentifier(p.Name)
		if len(p.Arguments) > 0 {
			foreach(p.Arguments, d.astNode, &s.Arguments)
		}
	})
	return
}

func (d *decoder) astAnnotations(p *ASTAnnotations) ast.Annotations {
	if p == nil || len(p.Annotations) == 0 {
		return nil
	}
	out := ast.Annotations{}
	foreach(p.Annotations, d.astAnnotation, &out)
	return out
}

func (d *decoder) astAssign(astAssignID uint64) (out *ast.Assign) {
	d.build(d.content.Instances.AstAssign, &d.inst.ASTAssign, astAssignID, &out, func(p *ASTAssign, s *ast.Assign) {
		s.LHS = d.astNode(p.Lhs)
		s.Operator = d.str(p.Operator)
		s.RHS = d.astNode(p.Rhs)
	})
	return
}

func (d *decoder) astBinaryOp(astBinaryOpID uint64) (out *ast.BinaryOp) {
	d.build(d.content.Instances.AstBinaryOp, &d.inst.ASTBinaryOp, astBinaryOpID, &out, func(p *ASTBinaryOp, s *ast.BinaryOp) {
		s.LHS = d.astNode(p.Lhs)
		s.Operator = d.str(p.Operator)
		s.RHS = d.astNode(p.Rhs)
	})
	return
}

func (d *decoder) astBool(astBoolID uint64) (out *ast.Bool) {
	d.build(d.content.Instances.AstBool, &d.inst.ASTBool, astBoolID, &out, func(p *ASTBool, s *ast.Bool) {
		s.Value = p.Value
	})
	return
}

func (d *decoder) astBranch(astBranchID uint64) (out *ast.Branch) {
	d.build(d.content.Instances.AstBranch, &d.inst.ASTBranch, astBranchID, &out, func(p *ASTBranch, s *ast.Branch) {
		s.Condition = d.astNode(p.Condition)
		s.True = d.astBlock(p.True)
		s.False = d.astBlock(p.False)
	})
	return
}

func (d *decoder) astCall(astCallID uint64) (out *ast.Call) {
	d.build(d.content.Instances.AstCall, &d.inst.ASTCall, astCallID, &out, func(p *ASTCall, s *ast.Call) {
		s.Target = d.astNode(p.Target)
		foreach(p.Arguments, d.astNode, &s.Arguments)
	})
	return
}

func (d *decoder) astCase(astCaseID uint64) (out *ast.Case) {
	d.build(d.content.Instances.AstCase, &d.inst.ASTCase, astCaseID, &out, func(p *ASTCase, s *ast.Case) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Block = d.astBlock(p.Block)
		foreach(p.Conditions, d.astNode, &s.Conditions)
	})
	return
}

func (d *decoder) astClass(astClassID uint64) (out *ast.Class) {
	d.build(d.content.Instances.AstClass, &d.inst.ASTClass, astClassID, &out, func(p *ASTClass, s *ast.Class) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Name = d.astIdentifier(p.Name)
		foreach(p.Fields, d.astField, &s.Fields)
	})
	return
}

func (d *decoder) astBlock(astBlockID uint64) (out *ast.Block) {
	d.build(d.content.Instances.AstBlock, &d.inst.ASTBlock, astBlockID, &out, func(p *ASTBlock, s *ast.Block) {
		foreach(p.Statements, d.astNode, &s.Statements)
	})
	return
}

func (d *decoder) astDeclareLocal(astDeclareLocalID uint64) (out *ast.DeclareLocal) {
	d.build(d.content.Instances.AstDeclareLocal, &d.inst.ASTDeclareLocal, astDeclareLocalID, &out, func(p *ASTDeclareLocal, s *ast.DeclareLocal) {
		s.Name = d.astIdentifier(p.Name)
		s.RHS = d.astNode(p.Rhs)
	})
	return
}

func (d *decoder) astDefault(astDefaultID uint64) (out *ast.Default) {
	d.build(d.content.Instances.AstDefault, &d.inst.ASTDefault, astDefaultID, &out, func(p *ASTDefault, s *ast.Default) {
		s.Block = d.astBlock(p.Block)
	})
	return
}

func (d *decoder) astDefinition(astDefinitionID uint64) (out *ast.Definition) {
	d.build(d.content.Instances.AstDefinition, &d.inst.ASTDefinition, astDefinitionID, &out, func(p *ASTDefinition, s *ast.Definition) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Name = d.astIdentifier(p.Name)
		s.Expression = d.astNode(p.Expression)
	})
	return
}

func (d *decoder) astDelete(astDeleteID uint64) (out *ast.Delete) {
	d.build(d.content.Instances.AstDelete, &d.inst.ASTDelete, astDeleteID, &out, func(p *ASTDelete, s *ast.Delete) {
		s.Key = d.astNode(p.Key)
		s.Map = d.astNode(p.Map)
	})
	return
}

func (d *decoder) astEnum(astEnumID uint64) (out *ast.Enum) {
	d.build(d.content.Instances.AstEnum, &d.inst.ASTEnum, astEnumID, &out, func(p *ASTEnum, s *ast.Enum) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.NumberType = d.astNode(p.NumberType)
		s.Name = d.astIdentifier(p.Name)
		s.IsBitfield = p.IsBitfield
		foreach(p.Entries, d.astEnumEntry, &s.Entries)
	})
	return
}

func (d *decoder) astEnumEntry(astEnumEntryID uint64) (out *ast.EnumEntry) {
	d.build(d.content.Instances.AstEnumEntry, &d.inst.ASTEnumEntry, astEnumEntryID, &out, func(p *ASTEnumEntry, s *ast.EnumEntry) {
		s.Owner = d.astEnum(p.Owner)
		s.Name = d.astIdentifier(p.Name)
		s.Value = d.astNumber(p.Value)
	})
	return
}

func (d *decoder) astField(astFieldID uint64) (out *ast.Field) {
	d.build(d.content.Instances.AstField, &d.inst.ASTField, astFieldID, &out, func(p *ASTField, s *ast.Field) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Type = d.astNode(p.Type)
		s.Name = d.astIdentifier(p.Name)
		s.Default = d.astNode(p.Default)
	})
	return
}

func (d *decoder) astFence(astFenceID uint64) (out *ast.Fence) {
	d.build(d.content.Instances.AstFence, &d.inst.ASTFence, astFenceID, &out, func(p *ASTFence, s *ast.Fence) {})
	return
}

func (d *decoder) astFunction(astFunctionID uint64) (out *ast.Function) {
	d.build(d.content.Instances.AstFunction, &d.inst.ASTFunction, astFunctionID, &out, func(p *ASTFunction, s *ast.Function) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Generic = d.astGeneric(p.Generic)
		foreach(p.Parameters, d.astParameter, &s.Parameters)
		s.Block = d.astBlock(p.Block)
	})
	return
}

func (d *decoder) astGeneric(astGenericID uint64) (out *ast.Generic) {
	d.build(d.content.Instances.AstGeneric, &d.inst.ASTGeneric, astGenericID, &out, func(p *ASTGeneric, s *ast.Generic) {
		s.Name = d.astIdentifier(p.Name)
		foreach(p.Arguments, d.astNode, &s.Arguments)
	})
	return
}

func (d *decoder) astGroup(astGroupID uint64) (out *ast.Group) {
	d.build(d.content.Instances.AstGroup, &d.inst.ASTGroup, astGroupID, &out, func(p *ASTGroup, s *ast.Group) {
		s.Expression = d.astNode(p.Expression)
	})
	return
}

func (d *decoder) astIdentifier(astIdentifierID uint64) (out *ast.Identifier) {
	d.build(d.content.Instances.AstIdentifier, &d.inst.ASTIdentifier, astIdentifierID, &out, func(p *ASTIdentifier, s *ast.Identifier) {
		s.Value = d.str(p.Value)
	})
	return
}

func (d *decoder) astIndex(astIndexID uint64) (out *ast.Index) {
	d.build(d.content.Instances.AstIndex, &d.inst.ASTIndex, astIndexID, &out, func(p *ASTIndex, s *ast.Index) {
		s.Index = d.astNode(p.Index)
		s.Object = d.astNode(p.Object)
	})
	return
}

func (d *decoder) astIndexedType(astIndexedTypeID uint64) (out *ast.IndexedType) {
	d.build(d.content.Instances.AstIndexedType, &d.inst.ASTIndexedType, astIndexedTypeID, &out, func(p *ASTIndexedType, s *ast.IndexedType) {
		s.Index = d.astNode(p.Index)
		s.ValueType = d.astNode(p.ValueType)
	})
	return
}

func (d *decoder) astIteration(astIterationID uint64) (out *ast.Iteration) {
	d.build(d.content.Instances.AstIteration, &d.inst.ASTIteration, astIterationID, &out, func(p *ASTIteration, s *ast.Iteration) {
		s.Variable = d.astIdentifier(p.Variable)
		s.Iterable = d.astNode(p.Iterable)
		s.Block = d.astBlock(p.Block)
	})
	return
}

func (d *decoder) astMapIteration(astMapIterationID uint64) (out *ast.MapIteration) {
	d.build(d.content.Instances.AstMapIteration, &d.inst.ASTMapIteration, astMapIterationID, &out, func(p *ASTMapIteration, s *ast.MapIteration) {
		s.IndexVariable = d.astIdentifier(p.IndexVariable)
		s.KeyVariable = d.astIdentifier(p.KeyVariable)
		s.ValueVariable = d.astIdentifier(p.ValueVariable)
		s.Map = d.astNode(p.Map)
		s.Block = d.astBlock(p.Block)
	})
	return
}

func (d *decoder) astMember(astMemberID uint64) (out *ast.Member) {
	d.build(d.content.Instances.AstMember, &d.inst.ASTMember, astMemberID, &out, func(p *ASTMember, s *ast.Member) {
		s.Name = d.astIdentifier(p.Name)
		s.Object = d.astNode(p.Object)
	})
	return
}

func (d *decoder) astNamedArg(astNamedArgID uint64) (out *ast.NamedArg) {
	d.build(d.content.Instances.AstNamedArg, &d.inst.ASTNamedArg, astNamedArgID, &out, func(p *ASTNamedArg, s *ast.NamedArg) {
		s.Name = d.astIdentifier(p.Name)
		s.Value = d.astNode(p.Value)
	})
	return
}

func (d *decoder) astNode(p *ASTNode) ast.Node {
	if p == nil {
		return nil
	}
	switch p := protoutil.OneOf(p).(type) {
	case *ASTNode_Abort:
		return d.astAbort(p.Abort)
	case *ASTNode_Assign:
		return d.astAssign(p.Assign)
	case *ASTNode_BinaryOp:
		return d.astBinaryOp(p.BinaryOp)
	case *ASTNode_Bool:
		return d.astBool(p.Bool)
	case *ASTNode_Branch:
		return d.astBranch(p.Branch)
	case *ASTNode_Call:
		return d.astCall(p.Call)
	case *ASTNode_DeclareLocal:
		return d.astDeclareLocal(p.DeclareLocal)
	case *ASTNode_Default:
		return d.astDefault(p.Default)
	case *ASTNode_Delete:
		return d.astDelete(p.Delete)
	case *ASTNode_Fence:
		return d.astFence(p.Fence)
	case *ASTNode_Generic:
		return d.astGeneric(p.Generic)
	case *ASTNode_Group:
		return d.astGroup(p.Group)
	case *ASTNode_Identifier:
		return d.astIdentifier(p.Identifier)
	case *ASTNode_Index:
		return d.astIndex(p.Index)
	case *ASTNode_IndexedType:
		return d.astIndexedType(p.IndexedType)
	case *ASTNode_Iteration:
		return d.astIteration(p.Iteration)
	case *ASTNode_MapIteration:
		return d.astMapIteration(p.MapIteration)
	case *ASTNode_Member:
		return d.astMember(p.Member)
	case *ASTNode_NamedArg:
		return d.astNamedArg(p.NamedArg)
	case *ASTNode_Null:
		return d.astNull(p.Null)
	case *ASTNode_Number:
		return d.astNumber(p.Number)
	case *ASTNode_PointerType:
		return d.astPointerType(p.PointerType)
	case *ASTNode_PreConst:
		return d.astPreConst(p.PreConst)
	case *ASTNode_Return:
		return d.astReturn(p.Return)
	case *ASTNode_String_:
		return d.astString(p.String_)
	case *ASTNode_Switch:
		return d.astSwitch(p.Switch)
	case *ASTNode_UnaryOp:
		return d.astUnaryOp(p.UnaryOp)
	case *ASTNode_Unknown:
		return d.astUnknown(p.Unknown)
	default:
		panic(fmt.Errorf("Unhandled ASTNode type: %T", p))
	}
}

func (d *decoder) astNull(astNullID uint64) (out *ast.Null) {
	d.build(d.content.Instances.AstNull, &d.inst.ASTNull, astNullID, &out, func(p *ASTNull, s *ast.Null) {})
	return
}

func (d *decoder) astNumber(astNumberID uint64) (out *ast.Number) {
	d.build(d.content.Instances.AstNumber, &d.inst.ASTNumber, astNumberID, &out, func(p *ASTNumber, s *ast.Number) {
		s.Value = d.str(p.Value)
	})
	return
}

func (d *decoder) astParameter(astParameterID uint64) (out *ast.Parameter) {
	d.build(d.content.Instances.AstParameter, &d.inst.ASTParameter, astParameterID, &out, func(p *ASTParameter, s *ast.Parameter) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.This = p.This
		s.Type = d.astNode(p.Type)
		s.Name = d.astIdentifier(p.Name)
	})
	return
}

func (d *decoder) astPointerType(astPointerTypeID uint64) (out *ast.PointerType) {
	d.build(d.content.Instances.AstPointerType, &d.inst.ASTPointerType, astPointerTypeID, &out, func(p *ASTPointerType, s *ast.PointerType) {
		s.To = d.astNode(p.To)
		s.Const = p.Const
	})
	return
}

func (d *decoder) astPreConst(astPreConstID uint64) (out *ast.PreConst) {
	d.build(d.content.Instances.AstPreConst, &d.inst.ASTPreConst, astPreConstID, &out, func(p *ASTPreConst, s *ast.PreConst) {
		s.Type = d.astNode(p.Type)
	})
	return
}

func (d *decoder) astPseudonym(astPseudonymID uint64) (out *ast.Pseudonym) {
	d.build(d.content.Instances.AstPseudonym, &d.inst.ASTPseudonym, astPseudonymID, &out, func(p *ASTPseudonym, s *ast.Pseudonym) {
		s.Annotations = d.astAnnotations(p.Annotations)
		s.Name = d.astIdentifier(p.Name)
		s.To = d.astNode(p.To)
	})
	return
}

func (d *decoder) astReturn(astReturnID uint64) (out *ast.Return) {
	d.build(d.content.Instances.AstReturn, &d.inst.ASTReturn, astReturnID, &out, func(p *ASTReturn, s *ast.Return) {
		s.Value = d.astNode(p.Value)
	})
	return
}

func (d *decoder) astString(astStringID uint64) (out *ast.String) {
	d.build(d.content.Instances.AstString, &d.inst.ASTString, astStringID, &out, func(p *ASTString, s *ast.String) {
		s.Value = d.str(p.Value)
	})
	return
}

func (d *decoder) astSwitch(astSwitchID uint64) (out *ast.Switch) {
	d.build(d.content.Instances.AstSwitch, &d.inst.ASTSwitch, astSwitchID, &out, func(p *ASTSwitch, s *ast.Switch) {
		s.Value = d.astNode(p.Value)
		foreach(p.Cases, d.astCase, &s.Cases)
		s.Default = d.astDefault(p.Default)
	})
	return
}

func (d *decoder) astUnaryOp(astUnaryOpID uint64) (out *ast.UnaryOp) {
	d.build(d.content.Instances.AstUnaryOp, &d.inst.ASTUnaryOp, astUnaryOpID, &out, func(p *ASTUnaryOp, s *ast.UnaryOp) {
		s.Operator = d.str(p.Operator)
		s.Expression = d.astNode(p.Expression)
	})
	return
}

func (d *decoder) astUnknown(astUnknownID uint64) (out *ast.Unknown) {
	d.build(d.content.Instances.AstUnknown, &d.inst.ASTUnknown, astUnknownID, &out, func(p *ASTUnknown, s *ast.Unknown) {})
	return
}

// build calls cb to decode an object instance from a proto.
// The output of cb is cached into s so each instance is only built once.
//
// i is the proto instances slice (d.content.Instances.Foo)
// s is a pointer to the decoder instances slice (&d.inst.Foo)
// p is the object's proto ID
// outPtr is a pointer to the output value
// cb is a func with the signature: func(InputType, *OutputType)
func (*decoder) build(i, s, p, outPtr, cb interface{}) {
	sV := reflect.ValueOf(s)
	pV := reflect.ValueOf(p)

	outPtrV := reflect.ValueOf(outPtr)
	outV := outPtrV.Elem()

	id := int(pV.Interface().(uint64))
	if id == 0 {
		return // 0 means nil
	}

	idx := id - 1

	if existing := sV.Elem().Index(idx); !existing.IsNil() {
		// Already built this object.
		outV.Set(existing)
		return
	}

	in := reflect.ValueOf(i).Index(idx)
	out := reflect.New(outPtrV.Type().Elem().Elem())
	outV.Set(out)

	// Store the id into the decoder instances slice before building to handle
	// cyclic dependencies.
	sV.Elem().Index(idx).Set(out)

	// Build
	reflect.ValueOf(cb).Call([]reflect.Value{in, out})
}
