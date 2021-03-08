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
	"github.com/google/gapid/core/text/parse/cst"
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
	API             []*semantic.API
	Abort           []*semantic.Abort
	Annotation      []*semantic.Annotation
	ArrayAssign     []*semantic.ArrayAssign
	ArrayIndex      []*semantic.ArrayIndex
	ArrayInit       []*semantic.ArrayInitializer
	Assert          []*semantic.Assert
	Assign          []*semantic.Assign
	BinaryOp        []*semantic.BinaryOp
	BitTest         []*semantic.BitTest
	Block           []*semantic.Block
	Branch          []*semantic.Branch
	Call            []*semantic.Call
	Callable        []*semantic.Callable
	Case            []*semantic.Case
	Cast            []*semantic.Cast
	Choice          []*semantic.Choice
	Class           []*semantic.Class
	ClassInit       []*semantic.ClassInitializer
	Clone           []*semantic.Clone
	Copy            []*semantic.Copy
	Create          []*semantic.Create
	DeclareLocal    []*semantic.DeclareLocal
	Definition      []*semantic.Definition
	DefinitionUsage []*semantic.DefinitionUsage
	Enum            []*semantic.Enum
	EnumEntry       []*semantic.EnumEntry
	Fence           []*semantic.Fence
	Field           []*semantic.Field
	FieldInit       []*semantic.FieldInitializer
	Function        []*semantic.Function
	Global          []*semantic.Global
	Ignore          []*semantic.Ignore
	Iteration       []*semantic.Iteration
	Length          []*semantic.Length
	Local           []*semantic.Local
	Make            []*semantic.Make
	Map             []*semantic.Map
	MapAssign       []*semantic.MapAssign
	MapContains     []*semantic.MapContains
	MapIndex        []*semantic.MapIndex
	MapIteration    []*semantic.MapIteration
	MapRemove       []*semantic.MapRemove
	MapClear        []*semantic.MapClear
	Member          []*semantic.Member
	MessageValue    []*semantic.MessageValue
	Observed        []*semantic.Observed
	Parameter       []*semantic.Parameter
	Pointer         []*semantic.Pointer
	PointerRange    []*semantic.PointerRange
	Print           []*semantic.Print
	Pseudonym       []*semantic.Pseudonym
	Read            []*semantic.Read
	Reference       []*semantic.Reference
	Return          []*semantic.Return
	Select          []*semantic.Select
	Signature       []*semantic.Signature
	Slice           []*semantic.Slice
	SliceAssign     []*semantic.SliceAssign
	SliceIndex      []*semantic.SliceIndex
	SliceRange      []*semantic.SliceRange
	Statement       []semantic.Statement
	StaticArray     []*semantic.StaticArray
	StringValue     []semantic.StringValue
	Switch          []*semantic.Switch
	UnaryOp         []*semantic.UnaryOp
	Unknown         []*semantic.Unknown
	Write           []*semantic.Write

	ASTAnnotation   []*ast.Annotation
	ASTAbort        []*ast.Abort
	ASTAPI          []*ast.API
	ASTAssign       []*ast.Assign
	ASTBinaryOp     []*ast.BinaryOp
	ASTBlock        []*ast.Block
	ASTBool         []*ast.Bool
	ASTBranch       []*ast.Branch
	ASTCall         []*ast.Call
	ASTCase         []*ast.Case
	ASTClass        []*ast.Class
	ASTClear        []*ast.Clear
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
	ASTImport       []*ast.Import
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

	CSTBranch []*cst.Branch
	CSTLeaf   []*cst.Leaf
	CSTSource []*cst.Source
}

func (i *decoderInstances) build(p *Instances) {
	toProtoName := func(s string) string {
		s = strings.Replace(s, "AST", "Ast", -1)
		s = strings.Replace(s, "API", "Api", -1)
		s = strings.Replace(s, "CST", "Cst", -1)
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
func Decode(data []byte) ([]*semantic.API, *semantic.Mappings, error) {
	d := decoder{}
	if err := proto.Unmarshal(data, &d.content); err != nil {
		return nil, nil, err
	}
	d.inst.build(d.content.Instances)

	apis := make([]*semantic.API, len(d.content.Apis))
	for i, id := range d.content.Apis {
		apis[i] = d.api(id)
		if d.err != nil {
			return nil, nil, d.err
		}
	}

	mappings := d.mappings()
	if d.err != nil {
		return nil, nil, d.err
	}

	for _, s := range d.toSort {
		s.SortMembers()
	}
	return apis, mappings, nil
}

func (d *decoder) mappings() *semantic.Mappings {
	out := semantic.Mappings{}

	for _, semToAST := range d.content.Mappings.SemToAst {
		ast := d.astNode(semToAST.Ast)
		sem := d.node(semToAST.Sem)
		out.Add(ast, sem)
	}

	for _, astToCST := range d.content.Mappings.AstToCst {
		ast := d.astNode(astToCST.Ast)
		cst := d.cstNode(astToCST.Cst)
		out.AST.Add(ast, cst)
	}

	return &out
}

func (d *decoder) abort(abortID uint64) (out *semantic.Abort) {
	d.build(d.content.Instances.Abort, &d.inst.Abort, abortID, &out,
		func(p *Abort, s *semantic.Abort) {
			s.AST = d.astAbort(p.Ast)
			s.Function = d.function(p.Function)
			s.Statement = d.stat(p.Statement)
		})
	return
}

func (d *decoder) annotation(annotationID uint64) (out *semantic.Annotation) {
	d.build(d.content.Instances.Annotation, &d.inst.Annotation, annotationID, &out,
		func(p *Annotation, s *semantic.Annotation) {
			s.AST = d.astAnnotation(p.Ast)
			s.Named = semantic.Named(d.str(p.Name))
			if len(p.Arguments) > 0 {
				foreach(p.Arguments, d.expr, &out.Arguments)
			}
		})
	return
}

func (d *decoder) annotations(p *Annotations) semantic.Annotations {
	if p == nil || len(p.Annotations) == 0 {
		return nil
	}
	out := semantic.Annotations{}
	foreach(p.Annotations, d.annotation, &out)
	return out
}

func (d *decoder) assert(assertID uint64) (out *semantic.Assert) {
	d.build(d.content.Instances.Assert, &d.inst.Assert, assertID, &out,
		func(p *Assert, s *semantic.Assert) {
			s.AST = d.astCall(p.Ast)
			s.Condition = d.expr(p.Condition)
			s.Message = d.str(p.Message)
		})
	return
}

func (d *decoder) api(apiID uint64) (out *semantic.API) {
	d.build(d.content.Instances.Api, &d.inst.API, apiID, &out,
		func(p *API, s *semantic.API) {
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
			s.Index = semantic.Uint8Value(p.Index)
			d.toSort = append(d.toSort, s)
		})
	return
}

func (d *decoder) array(arrayID uint64) (out *semantic.StaticArray) {
	d.build(d.content.Instances.StaticArray, &d.inst.StaticArray, arrayID, &out,
		func(p *StaticArray, s *semantic.StaticArray) {
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

func (d *decoder) arrayAssign(arrayAssignID uint64) (out *semantic.ArrayAssign) {
	d.build(d.content.Instances.ArrayAssign, &d.inst.ArrayAssign, arrayAssignID, &out,
		func(p *ArrayAssign, s *semantic.ArrayAssign) {
			s.AST = d.astAssign(p.Ast)
			s.To = d.expr(p.To).(*semantic.ArrayIndex)
			s.Operator = d.str(p.Operator)
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) assign(assignID uint64) (out *semantic.Assign) {
	d.build(d.content.Instances.Assign, &d.inst.Assign, assignID, &out,
		func(p *Assign, s *semantic.Assign) {
			s.AST = d.astAssign(p.Ast)
			s.LHS = d.expr(p.Lhs)
			s.Operator = d.str(p.Operator)
			s.RHS = d.expr(p.Rhs)
		})
	return
}

func (d *decoder) arrayInit(arrayInitID uint64) (out *semantic.ArrayInitializer) {
	d.build(d.content.Instances.ArrayInit, &d.inst.ArrayInit, arrayInitID, &out,
		func(p *ArrayInitializer, s *semantic.ArrayInitializer) {
			s.AST = d.astCall(p.Ast)
			s.Array = d.ty(p.Array)
			foreach(p.Values, d.expr, &s.Values)
		})
	return
}

func (d *decoder) arrayIndex(arrayIndexID uint64) (out *semantic.ArrayIndex) {
	d.build(d.content.Instances.ArrayIndex, &d.inst.ArrayIndex, arrayIndexID, &out,
		func(p *ArrayIndex, s *semantic.ArrayIndex) {
			s.AST = d.astIndex(p.Ast)
			s.Type = d.array(p.Type)
			s.Array = d.expr(p.Array)
			s.Index = d.expr(p.Index)
		})
	return
}

func (d *decoder) binaryOp(binaryOpID uint64) (out *semantic.BinaryOp) {
	d.build(d.content.Instances.BinaryOp, &d.inst.BinaryOp, binaryOpID, &out,
		func(p *BinaryOp, s *semantic.BinaryOp) {
			s.AST = d.astBinaryOp(p.Ast)
			s.Type = d.ty(p.Type)
			s.LHS = d.expr(p.Lhs)
			s.Operator = d.str(p.Operator)
			s.RHS = d.expr(p.Rhs)
		})
	return
}

func (d *decoder) bitTest(bitTestID uint64) (out *semantic.BitTest) {
	d.build(d.content.Instances.BitTest, &d.inst.BitTest, bitTestID, &out,
		func(p *BitTest, s *semantic.BitTest) {
			s.AST = d.astBinaryOp(p.Ast)
			s.Bitfield = d.expr(p.Bitfield)
			s.Bits = d.expr(p.Bits)
		})
	return
}

func (d *decoder) branch(branchID uint64) (out *semantic.Branch) {
	d.build(d.content.Instances.Branch, &d.inst.Branch, branchID, &out,
		func(p *Branch, s *semantic.Branch) {
			s.AST = d.astBranch(p.Ast)
			s.Condition = d.expr(p.Condition)
			s.True = d.block(p.True)
			s.False = d.block(p.False)
		})
	return
}

func (d *decoder) boolValue(boolValueID uint64) semantic.BoolValue {
	return semantic.BoolValue(d.content.Instances.BoolValue[boolValueID-1].Value)
}

func (d *decoder) block(blockID uint64) (out *semantic.Block) {
	d.build(d.content.Instances.Block, &d.inst.Block, blockID, &out,
		func(p *Block, s *semantic.Block) {
			s.AST = d.astBlock(p.Ast)
			foreach(p.Statements, d.stat, &s.Statements)
		})
	return
}

func (d *decoder) builtin(p Builtin) *semantic.Builtin {
	switch p {
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
		panic(fmt.Errorf("Unhandled builtin %v", p))
	}
}

func (d *decoder) call(callID uint64) (out *semantic.Call) {
	d.build(d.content.Instances.Call, &d.inst.Call, callID, &out,
		func(p *Call, s *semantic.Call) {
			s.AST = d.astCall(p.Ast)
			s.Target = d.callable(p.Target)
			s.Type = d.ty(p.Type)
			foreach(p.Arguments, d.expr, &s.Arguments)
		})
	return
}

func (d *decoder) callable(callableID uint64) (out *semantic.Callable) {
	d.build(d.content.Instances.Callable, &d.inst.Callable, callableID, &out,
		func(p *Callable, s *semantic.Callable) {
			s.Function = d.function(p.Function)
			s.Object = d.expr(p.Object)
		})
	return
}

func (d *decoder) case_(caseID uint64) (out *semantic.Case) {
	d.build(d.content.Instances.Case, &d.inst.Case, caseID, &out,
		func(p *Case, s *semantic.Case) {
			s.AST = d.astCase(p.Ast)
			s.Annotations = d.annotations(p.Annotations)
			s.Block = d.block(p.Block)
			foreach(p.Conditions, d.expr, &s.Conditions)
		})
	return
}

func (d *decoder) cast(castID uint64) (out *semantic.Cast) {
	d.build(d.content.Instances.Cast, &d.inst.Cast, castID, &out,
		func(p *Cast, s *semantic.Cast) {
			s.AST = d.astCall(p.Ast)
			s.Object = d.expr(p.Object)
			s.Type = d.ty(p.Type)
		})
	return
}

func (d *decoder) choice(choiceID uint64) (out *semantic.Choice) {
	d.build(d.content.Instances.Choice, &d.inst.Choice, choiceID, &out,
		func(p *Choice, s *semantic.Choice) {
			s.AST = d.astCase(p.Ast)
			s.Annotations = d.annotations(p.Annotations)
			s.Expression = d.expr(p.Expression)
			foreach(p.Conditions, d.expr, &s.Conditions)
		})
	return
}

func (d *decoder) class(classID uint64) (out *semantic.Class) {
	d.build(d.content.Instances.Class, &d.inst.Class, classID, &out,
		func(p *Class, s *semantic.Class) {
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

func (d *decoder) classInit(classInitID uint64) (out *semantic.ClassInitializer) {
	d.build(d.content.Instances.ClassInit, &d.inst.ClassInit, classInitID, &out,
		func(p *ClassInitializer, s *semantic.ClassInitializer) {
			s.AST = d.astCall(p.Ast)
			s.Class = d.class(p.Class)
			foreach(p.Fields, d.fieldInit, &s.Fields)
		})
	return
}

func (d *decoder) clone(cloneID uint64) (out *semantic.Clone) {
	d.build(d.content.Instances.Clone, &d.inst.Clone, cloneID, &out,
		func(p *Clone, s *semantic.Clone) {
			s.AST = d.astCall(p.Ast)
			s.Slice = d.expr(p.Slice)
			s.Type = d.slice(p.Type)
		})
	return
}

func (d *decoder) copy(copyID uint64) (out *semantic.Copy) {
	d.build(d.content.Instances.Copy, &d.inst.Copy, copyID, &out,
		func(p *Copy, s *semantic.Copy) {
			s.AST = d.astCall(p.Ast)
			s.Src = d.expr(p.Src)
			s.Dst = d.expr(p.Dst)
		})
	return
}

func (d *decoder) create(createID uint64) (out *semantic.Create) {
	d.build(d.content.Instances.Create, &d.inst.Create, createID, &out,
		func(p *Create, s *semantic.Create) {
			s.AST = d.astCall(p.Ast)
			s.Type = d.reference(p.Type)
			s.Initializer = d.classInit(p.Initializer)
		})
	return
}

func (d *decoder) declareLocal(declareLocalID uint64) (out *semantic.DeclareLocal) {
	d.build(d.content.Instances.DeclareLocal, &d.inst.DeclareLocal, declareLocalID, &out,
		func(p *DeclareLocal, s *semantic.DeclareLocal) {
			s.AST = d.astDeclareLocal(p.Ast)
			s.Local = d.local(p.Local)
		})
	return
}

func (d *decoder) definition(definitionID uint64) (out *semantic.Definition) {
	d.build(d.content.Instances.Definition, &d.inst.Definition, definitionID, &out,
		func(p *Definition, s *semantic.Definition) {
			s.AST = d.astDefinition(p.Ast)
			s.Annotations = d.annotations(p.Annotations)
			s.Named = semantic.Named(d.str(p.Name))
			s.Docs = d.docs(p.Docs)
			s.Expression = d.expr(p.Expression)
		})
	return
}

func (d *decoder) definitionUsage(definitionUsageID uint64) (out *semantic.DefinitionUsage) {
	d.build(d.content.Instances.DefinitionUsage, &d.inst.DefinitionUsage, definitionUsageID, &out,
		func(p *DefinitionUsage, s *semantic.DefinitionUsage) {
			s.Definition = d.definition(p.Definition)
			s.Expression = d.expr(p.Expression)
		})
	return
}

func (d *decoder) docs(p *Documentation) semantic.Documentation {
	s := semantic.Documentation{}
	foreach(p.Strings, d.str, &s)
	return s
}

func (d *decoder) enum(enumID uint64) (out *semantic.Enum) {
	d.build(d.content.Instances.Enum, &d.inst.Enum, enumID, &out,
		func(p *Enum, s *semantic.Enum) {
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
			d.toSort = append(d.toSort, s)
		})
	return
}

func (d *decoder) enumEntry(enumEntryID uint64) (out *semantic.EnumEntry) {
	d.build(d.content.Instances.EnumEntry, &d.inst.EnumEntry, enumEntryID, &out,
		func(p *EnumEntry, s *semantic.EnumEntry) {
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

func (d *decoder) fence(fenceID uint64) (out *semantic.Fence) {
	d.build(d.content.Instances.Fence, &d.inst.Fence, fenceID, &out,
		func(p *Fence, s *semantic.Fence) {
			s.AST = d.astFence(p.Ast)
			s.Statement = d.stat(p.Statement)
			s.Explicit = p.Explicit
		})
	return
}

func (d *decoder) float32Value(float32ValueID uint64) semantic.Float32Value {
	return semantic.Float32Value(d.content.Instances.Float32Value[float32ValueID-1].Value)
}

func (d *decoder) float64Value(float64ValueID uint64) semantic.Float64Value {
	return semantic.Float64Value(d.content.Instances.Float64Value[float64ValueID-1].Value)
}

func (d *decoder) expr(exprID uint64) semantic.Expression {
	if exprID == 0 {
		return nil
	}
	exprIdx := exprID - 1
	switch p := d.content.Instances.Expression[exprIdx].Ty.(type) {
	case *Expression_ArrayIndex:
		return d.arrayIndex(p.ArrayIndex)
	case *Expression_ArrayInitializer:
		return d.arrayInit(p.ArrayInitializer)
	case *Expression_BinaryOp:
		return d.binaryOp(p.BinaryOp)
	case *Expression_BitTest:
		return d.bitTest(p.BitTest)
	case *Expression_BoolValue:
		return d.boolValue(p.BoolValue)
	case *Expression_Call:
		return d.call(p.Call)
	case *Expression_Cast:
		return d.cast(p.Cast)
	case *Expression_ClassInit:
		return d.classInit(p.ClassInit)
	case *Expression_Clone:
		return d.clone(p.Clone)
	case *Expression_Create:
		return d.create(p.Create)
	case *Expression_Definition:
		return d.definition(p.Definition)
	case *Expression_DefinitionUsage:
		return d.definitionUsage(p.DefinitionUsage)
	case *Expression_EnumEntry:
		return d.enumEntry(p.EnumEntry)
	case *Expression_Field:
		return d.field(p.Field)
	case *Expression_Float32Value:
		return d.float32Value(p.Float32Value)
	case *Expression_Float64Value:
		return d.float64Value(p.Float64Value)
	case *Expression_Global:
		return d.global(p.Global)
	case *Expression_Ignore:
		return d.ignore(p.Ignore)
	case *Expression_Int8Value:
		return d.int8Value(p.Int8Value)
	case *Expression_Int16Value:
		return d.int16Value(p.Int16Value)
	case *Expression_Int32Value:
		return d.int32Value(p.Int32Value)
	case *Expression_Int64Value:
		return d.int64Value(p.Int64Value)
	case *Expression_Length:
		return d.length(p.Length)
	case *Expression_Local:
		return d.local(p.Local)
	case *Expression_Make:
		return d.make(p.Make)
	case *Expression_MapContains:
		return d.mapContains(p.MapContains)
	case *Expression_MapIndex:
		return d.mapIndex(p.MapIndex)
	case *Expression_Member:
		return d.member(p.Member)
	case *Expression_MessageValue:
		return d.messageValue(p.MessageValue)
	case *Expression_Null:
		return d.null(p.Null)
	case *Expression_Observed:
		return d.observed(p.Observed)
	case *Expression_Parameter:
		return d.param(p.Parameter)
	case *Expression_PointerRange:
		return d.pointerRange(p.PointerRange)
	case *Expression_Select:
		return d.select_(p.Select)
	case *Expression_SliceIndex:
		return d.sliceIndex(p.SliceIndex)
	case *Expression_SliceRange:
		return d.sliceRange(p.SliceRange)
	case *Expression_StringValue:
		return d.stringValue(p.StringValue)
	case *Expression_Uint8Value:
		return d.uint8Value(p.Uint8Value)
	case *Expression_Uint16Value:
		return d.uint16Value(p.Uint16Value)
	case *Expression_Uint32Value:
		return d.uint32Value(p.Uint32Value)
	case *Expression_Uint64Value:
		return d.uint64Value(p.Uint64Value)
	case *Expression_UnaryOp:
		return d.unaryOp(p.UnaryOp)
	case *Expression_Unknown:
		return d.unknown(p.Unknown)
	default:
		panic(fmt.Errorf("Unhandled expression type %T", p))
	}
}

func (d *decoder) field(fieldID uint64) (out *semantic.Field) {
	d.build(d.content.Instances.Field, &d.inst.Field, fieldID, &out,
		func(p *Field, s *semantic.Field) {
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

func (d *decoder) fieldInit(fieldInitID uint64) (out *semantic.FieldInitializer) {
	d.build(d.content.Instances.FieldInit, &d.inst.FieldInit, fieldInitID, &out,
		func(p *FieldInitializer, s *semantic.FieldInitializer) {
			s.AST = d.astNode(p.Ast)
			s.Field = d.field(p.Field)
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) function(funcID uint64) (out *semantic.Function) {
	d.build(d.content.Instances.Function, &d.inst.Function, funcID, &out,
		func(p *Function, s *semantic.Function) {
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
	d.build(d.content.Instances.Global, &d.inst.Global, globalID, &out,
		func(p *Global, s *semantic.Global) {
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

func (d *decoder) ignore(ignoreID uint64) (out *semantic.Ignore) {
	d.build(d.content.Instances.Ignore, &d.inst.Ignore, ignoreID, &out,
		func(p *Ignore, s *semantic.Ignore) {
			s.AST = d.astNode(p.Ast)
		})
	return
}

func (d *decoder) int8Value(int8ValueID uint64) semantic.Int8Value {
	return semantic.Int8Value(d.content.Instances.Int8Value[int8ValueID-1].Value)
}

func (d *decoder) int16Value(int16ValueID uint64) semantic.Int16Value {
	return semantic.Int16Value(d.content.Instances.Int16Value[int16ValueID-1].Value)
}

func (d *decoder) int32Value(int32ValueID uint64) semantic.Int32Value {
	return semantic.Int32Value(d.content.Instances.Int32Value[int32ValueID-1].Value)
}

func (d *decoder) int64Value(int64ValueID uint64) semantic.Int64Value {
	return semantic.Int64Value(d.content.Instances.Int64Value[int64ValueID-1].Value)
}

func (d *decoder) iteration(iterationID uint64) (out *semantic.Iteration) {

	d.build(d.content.Instances.Iteration, &d.inst.Iteration, iterationID, &out,
		func(p *Iteration, s *semantic.Iteration) {
			s.AST = d.astIteration(p.Ast)
			s.Iterator = d.local(p.Iterator)
			s.From = d.expr(p.From)
			s.To = d.expr(p.To)
			s.Block = d.block(p.Block)
		})
	return
}
func (d *decoder) length(lengthID uint64) (out *semantic.Length) {
	d.build(d.content.Instances.Length, &d.inst.Length, lengthID, &out,
		func(p *Length, s *semantic.Length) {
			s.AST = d.astCall(p.Ast)
			s.Object = d.expr(p.Object)
			s.Type = d.ty(p.Type)
		})
	return
}

func (d *decoder) local(localID uint64) (out *semantic.Local) {
	d.build(d.content.Instances.Local, &d.inst.Local, localID, &out,
		func(p *Local, s *semantic.Local) {
			if p.Declaration != 0 {
				s.Declaration = d.stat(p.Declaration).(*semantic.DeclareLocal)
			}
			s.Type = d.ty(p.Type)
			s.Named = semantic.Named(d.str(p.Name))
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) make(makeID uint64) (out *semantic.Make) {
	d.build(d.content.Instances.Make, &d.inst.Make, makeID, &out,
		func(p *Make, s *semantic.Make) {
			s.AST = d.astCall(p.Ast)
			s.Type = d.slice(p.Type)
			s.Size = d.expr(p.Size)
		})
	return
}

func (d *decoder) map_(mapID uint64) (out *semantic.Map) {
	d.build(d.content.Instances.Map, &d.inst.Map, mapID, &out,
		func(p *Map, s *semantic.Map) {
			s.Named = semantic.Named(d.str(p.Name))
			s.KeyType = d.ty(p.KeyType)
			s.ValueType = d.ty(p.ValueType)
			s.Dense = p.Dense
			if owner := d.node(p.Owner); owner != nil {
				semantic.Add(owner.(semantic.Owner), s)
			}
			d.toSort = append(d.toSort, s)
		})
	return
}

func (d *decoder) mapAssign(mapAssignID uint64) (out *semantic.MapAssign) {
	d.build(d.content.Instances.MapAssign, &d.inst.MapAssign, mapAssignID, &out,
		func(p *MapAssign, s *semantic.MapAssign) {
			s.AST = d.astAssign(p.Ast)
			s.Operator = d.str(p.Operator)
			s.To = d.expr(p.To).(*semantic.MapIndex)
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) mapContains(mapContainsID uint64) (out *semantic.MapContains) {
	d.build(d.content.Instances.MapContains, &d.inst.MapContains, mapContainsID, &out,
		func(p *MapContains, s *semantic.MapContains) {
			s.AST = d.astBinaryOp(p.Ast)
			s.Type = d.map_(p.Type)
			s.Map = d.expr(p.Map)
			s.Key = d.expr(p.Key)
		})
	return
}

func (d *decoder) mapIndex(mapIndexID uint64) (out *semantic.MapIndex) {
	d.build(d.content.Instances.MapIndex, &d.inst.MapIndex, mapIndexID, &out,
		func(p *MapIndex, s *semantic.MapIndex) {
			s.AST = d.astIndex(p.Ast)
			s.Type = d.map_(p.Type)
			s.Map = d.expr(p.Map)
			s.Index = d.expr(p.Index)
		})
	return
}

func (d *decoder) mapIteration(mapIterationID uint64) (out *semantic.MapIteration) {
	d.build(d.content.Instances.MapIteration, &d.inst.MapIteration, mapIterationID, &out,
		func(p *MapIteration, s *semantic.MapIteration) {
			s.AST = d.astMapIteration(p.Ast)
			s.IndexIterator = d.local(p.IndexIterator)
			s.KeyIterator = d.local(p.KeyIterator)
			s.ValueIterator = d.local(p.ValueIterator)
			s.Map = d.expr(p.Map)
			s.Block = d.block(p.Block)
		})
	return
}

func (d *decoder) mapRemove(mapRemoveID uint64) (out *semantic.MapRemove) {
	d.build(d.content.Instances.MapRemove, &d.inst.MapRemove, mapRemoveID, &out,
		func(p *MapRemove, s *semantic.MapRemove) {
			s.AST = d.astDelete(p.Ast)
			s.Type = d.map_(p.Type)
			s.Map = d.expr(p.Map)
			s.Key = d.expr(p.Key)
		})
	return
}

func (d *decoder) mapClear(mapClearID uint64) (out *semantic.MapClear) {
	d.build(d.content.Instances.MapClear, &d.inst.MapClear, mapClearID, &out,
		func(p *MapClear, s *semantic.MapClear) {
			s.AST = d.astClear(p.Ast)
			s.Type = d.map_(p.Type)
			s.Map = d.expr(p.Map)
		})
	return
}

func (d *decoder) member(memberID uint64) (out *semantic.Member) {
	d.build(d.content.Instances.Member, &d.inst.Member, memberID, &out,
		func(p *Member, s *semantic.Member) {
			s.AST = d.astMember(p.Ast)
			s.Field = d.field(p.Field)
			s.Object = d.expr(p.Object)
		})
	return
}

func (d *decoder) messageValue(messageValueID uint64) (out *semantic.MessageValue) {
	d.build(d.content.Instances.MessageValue, &d.inst.MessageValue, messageValueID, &out,
		func(p *MessageValue, s *semantic.MessageValue) {
			s.AST = d.astClass(p.Ast)
			foreach(p.Arguments, d.fieldInit, &s.Arguments)
		})
	return
}

func (d *decoder) null(nullID uint64) semantic.Null {
	p := d.content.Instances.Null[nullID-1]
	return semantic.Null{
		AST:  d.astNull(p.Ast),
		Type: d.ty(p.Type),
	}
}

func (d *decoder) observed(observedID uint64) (out *semantic.Observed) {
	d.build(d.content.Instances.Observed, &d.inst.Observed, observedID, &out,
		func(p *Observed, s *semantic.Observed) {
			s.Parameter = d.param(p.Parameter)
		})
	return
}

func (d *decoder) param(paramID uint64) (out *semantic.Parameter) {
	d.build(d.content.Instances.Parameter, &d.inst.Parameter, paramID, &out,
		func(p *Parameter, s *semantic.Parameter) {
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
	d.build(d.content.Instances.Pointer, &d.inst.Pointer, ptrID, &out,
		func(p *Pointer, s *semantic.Pointer) {
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

func (d *decoder) pointerRange(ptrRangeID uint64) (out *semantic.PointerRange) {
	d.build(d.content.Instances.PointerRange, &d.inst.PointerRange, ptrRangeID, &out,
		func(p *PointerRange, s *semantic.PointerRange) {
			s.AST = d.astIndex(p.Ast)
			s.Type = d.slice(p.Type)
			s.Pointer = d.expr(p.Pointer)
			s.Range = d.expr(p.Range).(*semantic.BinaryOp)
		})
	return
}

func (d *decoder) print(printID uint64) (out *semantic.Print) {
	d.build(d.content.Instances.Print, &d.inst.Print, printID, &out,
		func(p *Print, s *semantic.Print) {
			s.AST = d.astCall(p.Ast)
			foreach(p.Arguments, d.expr, &s.Arguments)
		})
	return
}

func (d *decoder) pseudonym(pseudID uint64) (out *semantic.Pseudonym) {
	d.build(d.content.Instances.Pseudonym, &d.inst.Pseudonym, pseudID, &out,
		func(p *Pseudonym, s *semantic.Pseudonym) {
			s.AST = d.astPseudonym(p.Ast)
			s.Annotations = d.annotations(p.Annotations)
			s.Named = semantic.Named(d.str(p.Name))
			s.Docs = d.docs(p.Docs)
			s.To = d.ty(p.To)
			foreach(p.Methods, d.function, &s.Methods)
			if owner := d.node(p.Owner); owner != nil {
				semantic.Add(owner.(semantic.Owner), s)
			}
			d.toSort = append(d.toSort, s)
		})
	return
}

func (d *decoder) read(readID uint64) (out *semantic.Read) {
	d.build(d.content.Instances.Read, &d.inst.Read, readID, &out,
		func(p *Read, s *semantic.Read) {
			s.AST = d.astCall(p.Ast)
			s.Slice = d.expr(p.Slice)
		})
	return
}

func (d *decoder) reference(refID uint64) (out *semantic.Reference) {
	d.build(d.content.Instances.Reference, &d.inst.Reference, refID, &out,
		func(p *Reference, s *semantic.Reference) {
			s.Named = semantic.Named(d.str(p.Name))
			s.To = d.ty(p.To)
			if owner := d.node(p.Owner); owner != nil {
				semantic.Add(owner.(semantic.Owner), s)
			}
		})
	return
}

func (d *decoder) return_(returnID uint64) (out *semantic.Return) {
	d.build(d.content.Instances.Return, &d.inst.Return, returnID, &out,
		func(p *Return, s *semantic.Return) {
			s.AST = d.astReturn(p.Ast)
			s.Function = d.function(p.Function)
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) select_(selectID uint64) (out *semantic.Select) {
	d.build(d.content.Instances.Select, &d.inst.Select, selectID, &out,
		func(p *Select, s *semantic.Select) {
			s.AST = d.astSwitch(p.Ast)
			s.Type = d.ty(p.Type)
			s.Value = d.expr(p.Value)
			s.Default = d.expr(p.Default)
			foreach(p.Choices, d.choice, &s.Choices)
		})
	return
}

func (d *decoder) sliceAssign(sliceAssignID uint64) (out *semantic.SliceAssign) {
	d.build(d.content.Instances.SliceAssign, &d.inst.SliceAssign, sliceAssignID, &out,
		func(p *SliceAssign, s *semantic.SliceAssign) {
			s.AST = d.astAssign(p.Ast)
			s.To = d.expr(p.To).(*semantic.SliceIndex)
			s.Operator = d.str(p.Operator)
			s.Value = d.expr(p.Value)
		})
	return
}

func (d *decoder) signature(sigID uint64) (out *semantic.Signature) {
	d.build(d.content.Instances.Signature, &d.inst.Signature, sigID, &out,
		func(p *Signature, s *semantic.Signature) {
			s.Named = semantic.Named(d.str(p.Name))
			s.Return = d.ty(p.Return)
			foreach(p.Arguments, d.ty, &s.Arguments)
			if owner := d.node(p.Owner); owner != nil {
				semantic.Add(owner.(semantic.Owner), s)
			}
		})
	return
}

func (d *decoder) slice(sliceID uint64) (out *semantic.Slice) {
	d.build(d.content.Instances.Slice, &d.inst.Slice, sliceID, &out,
		func(p *Slice, s *semantic.Slice) {
			s.Named = semantic.Named(d.str(p.Name))
			s.To = d.ty(p.To)
			s.Pointer = d.pointer(p.Pointer)
			if owner := d.node(p.Owner); owner != nil {
				semantic.Add(owner.(semantic.Owner), s)
			}
		})
	return
}

func (d *decoder) sliceIndex(sliceIndexID uint64) (out *semantic.SliceIndex) {
	d.build(d.content.Instances.SliceIndex, &d.inst.SliceIndex, sliceIndexID, &out,
		func(p *SliceIndex, s *semantic.SliceIndex) {
			s.AST = d.astIndex(p.Ast)
			s.Type = d.slice(p.Type)
			s.Slice = d.expr(p.Slice)
			s.Index = d.expr(p.Index)
		})
	return
}

func (d *decoder) sliceRange(sliID uint64) (out *semantic.SliceRange) {
	d.build(d.content.Instances.SliceRange, &d.inst.SliceRange, sliID, &out,
		func(p *SliceRange, s *semantic.SliceRange) {
			s.AST = d.astIndex(p.Ast)
			s.Type = d.slice(p.Type)
			s.Slice = d.expr(p.Slice)
			s.Range = d.expr(p.Range).(*semantic.BinaryOp)
		})
	return
}

func (d *decoder) stat(statID uint64) semantic.Statement {
	if statID == 0 {
		return nil
	}
	statIdx := statID - 1
	switch p := d.content.Instances.Statement[statIdx].Ty.(type) {
	case *Statement_Abort:
		return d.abort(p.Abort)
	case *Statement_Assert:
		return d.assert(p.Assert)
	case *Statement_ArrayAssign:
		return d.arrayAssign(p.ArrayAssign)
	case *Statement_Assign:
		return d.assign(p.Assign)
	case *Statement_Branch:
		return d.branch(p.Branch)
	case *Statement_Call:
		return d.call(p.Call)
	case *Statement_Copy:
		return d.copy(p.Copy)
	case *Statement_DeclareLocal:
		return d.declareLocal(p.DeclareLocal)
	case *Statement_Fence:
		return d.fence(p.Fence)
	case *Statement_Iteration:
		return d.iteration(p.Iteration)
	case *Statement_MapAssign:
		return d.mapAssign(p.MapAssign)
	case *Statement_MapIteration:
		return d.mapIteration(p.MapIteration)
	case *Statement_MapRemove:
		return d.mapRemove(p.MapRemove)
	case *Statement_MapClear:
		return d.mapClear(p.MapClear)
	case *Statement_Print:
		return d.print(p.Print)
	case *Statement_Read:
		return d.read(p.Read)
	case *Statement_Return:
		return d.return_(p.Return)
	case *Statement_SliceAssign:
		return d.sliceAssign(p.SliceAssign)
	case *Statement_Switch:
		return d.switch_(p.Switch)
	case *Statement_Write:
		return d.write(p.Write)
	default:
		panic(fmt.Errorf("Unhandled statement type %T", p))
	}
}

func (d *decoder) stringValue(stringValueID uint64) semantic.StringValue {
	return semantic.StringValue(d.str(d.content.Instances.StringValue[stringValueID-1].Value))
}

func (d *decoder) str(stringID uint64) string {
	if stringID == 0 {
		return ""
	}
	return d.content.Instances.Symbols[stringID-1]
}

func (d *decoder) switch_(switchID uint64) (out *semantic.Switch) {
	d.build(d.content.Instances.Switch, &d.inst.Switch, switchID, &out,
		func(p *Switch, s *semantic.Switch) {
			s.AST = d.astSwitch(p.Ast)
			s.Value = d.expr(p.Value)
			s.Default = d.block(p.Default)
			foreach(p.Cases, d.case_, &s.Cases)
		})
	return
}

func (d *decoder) write(writeID uint64) (out *semantic.Write) {
	d.build(d.content.Instances.Write, &d.inst.Write, writeID, &out,
		func(p *Write, s *semantic.Write) {
			s.AST = d.astCall(p.Ast)
			s.Slice = d.expr(p.Slice)
		})
	return
}

func (d *decoder) node(p *Node) semantic.Node {
	if p == nil {
		return nil
	}
	switch p := p.Ty.(type) {
	case *Node_Abort:
		return d.abort(p.Abort)
	case *Node_Annotation:
		return d.annotation(p.Annotation)
	case *Node_Api:
		return d.api(p.Api)
	case *Node_ArrayAssign:
		return d.arrayAssign(p.ArrayAssign)
	case *Node_ArrayIndex:
		return d.arrayIndex(p.ArrayIndex)
	case *Node_ArrayInit:
		return d.arrayInit(p.ArrayInit)
	case *Node_Assert:
		return d.assert(p.Assert)
	case *Node_Assign:
		return d.assign(p.Assign)
	case *Node_BinaryOp:
		return d.binaryOp(p.BinaryOp)
	case *Node_BitTest:
		return d.bitTest(p.BitTest)
	case *Node_Block:
		return d.block(p.Block)
	case *Node_BoolValue:
		return d.boolValue(p.BoolValue)
	case *Node_Branch:
		return d.branch(p.Branch)
	case *Node_Builtin:
		return d.builtin(p.Builtin)
	case *Node_Call:
		return d.call(p.Call)
	case *Node_Callable:
		return d.callable(p.Callable)
	case *Node_Case:
		return d.case_(p.Case)
	case *Node_Cast:
		return d.cast(p.Cast)
	case *Node_Choice:
		return d.choice(p.Choice)
	case *Node_Copy:
		return d.copy(p.Copy)
	case *Node_Class:
		return d.class(p.Class)
	case *Node_ClassInit:
		return d.classInit(p.ClassInit)
	case *Node_Clone:
		return d.clone(p.Clone)
	case *Node_Create:
		return d.create(p.Create)
	case *Node_DeclareLocal:
		return d.declareLocal(p.DeclareLocal)
	case *Node_Definition:
		return d.definition(p.Definition)
	case *Node_EnumEntry:
		return d.enumEntry(p.EnumEntry)
	case *Node_Enum:
		return d.enum(p.Enum)
	case *Node_Expression:
		return d.expr(p.Expression)
	case *Node_Fence:
		return d.fence(p.Fence)
	case *Node_Field:
		return d.field(p.Field)
	case *Node_FieldInit:
		return d.fieldInit(p.FieldInit)
	case *Node_Function:
		return d.function(p.Function)
	case *Node_Global:
		return d.global(p.Global)
	case *Node_Iteration:
		return d.iteration(p.Iteration)
	case *Node_Local:
		return d.local(p.Local)
	case *Node_Map:
		return d.map_(p.Map)
	case *Node_MapAssign:
		return d.mapAssign(p.MapAssign)
	case *Node_MapIteration:
		return d.mapIteration(p.MapIteration)
	case *Node_MapRemove:
		return d.mapRemove(p.MapRemove)
	case *Node_MapClear:
		return d.mapClear(p.MapClear)
	case *Node_Parameter:
		return d.param(p.Parameter)
	case *Node_Pointer:
		return d.pointer(p.Pointer)
	case *Node_Pseudonym:
		return d.pseudonym(p.Pseudonym)
	case *Node_Print:
		return d.print(p.Print)
	case *Node_Read:
		return d.read(p.Read)
	case *Node_Reference:
		return d.reference(p.Reference)
	case *Node_Return:
		return d.return_(p.Return)
	case *Node_Signature:
		return d.signature(p.Signature)
	case *Node_Slice:
		return d.slice(p.Slice)
	case *Node_SliceAssign:
		return d.sliceAssign(p.SliceAssign)
	case *Node_Statement:
		return d.stat(p.Statement)
	case *Node_StaticArray:
		return d.array(p.StaticArray)
	case *Node_UnaryOp:
		return d.unaryOp(p.UnaryOp)
	default:
		panic(fmt.Errorf("Unhandled node type %T", p))
	}
}

func (d *decoder) ty(p *Type) semantic.Type {
	if p == nil {
		return nil
	}
	switch p := p.Ty.(type) {
	case *Type_Builtin:
		return d.builtin(p.Builtin)
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

func (d *decoder) uint8Value(uint8ValueID uint64) semantic.Uint8Value {
	return semantic.Uint8Value(d.content.Instances.Uint8Value[uint8ValueID-1].Value)
}

func (d *decoder) uint16Value(uint16ValueID uint64) semantic.Uint16Value {
	return semantic.Uint16Value(d.content.Instances.Uint16Value[uint16ValueID-1].Value)
}

func (d *decoder) uint32Value(uint32ValueID uint64) semantic.Uint32Value {
	return semantic.Uint32Value(d.content.Instances.Uint32Value[uint32ValueID-1].Value)
}

func (d *decoder) uint64Value(uint64ValueID uint64) semantic.Uint64Value {
	return semantic.Uint64Value(d.content.Instances.Uint64Value[uint64ValueID-1].Value)
}

func (d *decoder) unaryOp(unaryOpID uint64) (out *semantic.UnaryOp) {
	d.build(d.content.Instances.UnaryOp, &d.inst.UnaryOp, unaryOpID, &out,
		func(p *UnaryOp, s *semantic.UnaryOp) {
			s.AST = d.astUnaryOp(p.Ast)
			s.Type = d.ty(p.Type)
			s.Operator = d.str(p.Operator)
			s.Expression = d.expr(p.Expression)
		})
	return
}

func (d *decoder) unknown(unknownID uint64) (out *semantic.Unknown) {
	d.build(d.content.Instances.Unknown, &d.inst.Unknown, unknownID, &out,
		func(p *Unknown, s *semantic.Unknown) {
			s.AST = d.astUnknown(p.Ast)
			s.Inferred = d.expr(p.Inferred)
		})
	return
}

func (d *decoder) astAbort(astAbortID uint64) (out *ast.Abort) {
	d.build(d.content.Instances.AstAbort, &d.inst.ASTAbort, astAbortID, &out,
		func(p *ASTAbort, s *ast.Abort) {})
	return
}

func (d *decoder) astAnnotation(astAnnotationID uint64) (out *ast.Annotation) {
	d.build(d.content.Instances.AstAnnotation, &d.inst.ASTAnnotation, astAnnotationID, &out,
		func(p *ASTAnnotation, s *ast.Annotation) {
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

func (d *decoder) astAPI(astAPIID uint64) (out *ast.API) {
	d.build(d.content.Instances.AstApi, &d.inst.ASTAPI, astAPIID, &out,
		func(p *ASTAPI, s *ast.API) {
			foreach(p.Imports, d.astImport, &s.Imports)
			foreach(p.Externs, d.astFunction, &s.Externs)
			foreach(p.Commands, d.astFunction, &s.Commands)
			foreach(p.Subroutines, d.astFunction, &s.Subroutines)
			foreach(p.Pseudonyms, d.astPseudonym, &s.Pseudonyms)
			foreach(p.Enums, d.astEnum, &s.Enums)
			foreach(p.Classes, d.astClass, &s.Classes)
			foreach(p.Fields, d.astField, &s.Fields)
			foreach(p.Definitions, d.astDefinition, &s.Definitions)
			s.Index = d.astNumber(p.Index)
		})
	return
}

func (d *decoder) astAssign(astAssignID uint64) (out *ast.Assign) {
	d.build(d.content.Instances.AstAssign, &d.inst.ASTAssign, astAssignID, &out,
		func(p *ASTAssign, s *ast.Assign) {
			s.LHS = d.astNode(p.Lhs)
			s.Operator = d.str(p.Operator)
			s.RHS = d.astNode(p.Rhs)
		})
	return
}

func (d *decoder) astBinaryOp(astBinaryOpID uint64) (out *ast.BinaryOp) {
	d.build(d.content.Instances.AstBinaryOp, &d.inst.ASTBinaryOp, astBinaryOpID, &out,
		func(p *ASTBinaryOp, s *ast.BinaryOp) {
			s.LHS = d.astNode(p.Lhs)
			s.Operator = d.str(p.Operator)
			s.RHS = d.astNode(p.Rhs)
		})
	return
}

func (d *decoder) astBool(astBoolID uint64) (out *ast.Bool) {
	d.build(d.content.Instances.AstBool, &d.inst.ASTBool, astBoolID, &out,
		func(p *ASTBool, s *ast.Bool) {
			s.Value = p.Value
		})
	return
}

func (d *decoder) astBranch(astBranchID uint64) (out *ast.Branch) {
	d.build(d.content.Instances.AstBranch, &d.inst.ASTBranch, astBranchID, &out,
		func(p *ASTBranch, s *ast.Branch) {
			s.Condition = d.astNode(p.Condition)
			s.True = d.astBlock(p.True)
			s.False = d.astBlock(p.False)
		})
	return
}

func (d *decoder) astCall(astCallID uint64) (out *ast.Call) {
	d.build(d.content.Instances.AstCall, &d.inst.ASTCall, astCallID, &out,
		func(p *ASTCall, s *ast.Call) {
			s.Target = d.astNode(p.Target)
			foreach(p.Arguments, d.astNode, &s.Arguments)
		})
	return
}

func (d *decoder) astCase(astCaseID uint64) (out *ast.Case) {
	d.build(d.content.Instances.AstCase, &d.inst.ASTCase, astCaseID, &out,
		func(p *ASTCase, s *ast.Case) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Block = d.astBlock(p.Block)
			foreach(p.Conditions, d.astNode, &s.Conditions)
		})
	return
}

func (d *decoder) astClass(astClassID uint64) (out *ast.Class) {
	d.build(d.content.Instances.AstClass, &d.inst.ASTClass, astClassID, &out,
		func(p *ASTClass, s *ast.Class) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Name = d.astIdentifier(p.Name)
			foreach(p.Fields, d.astField, &s.Fields)
		})
	return
}

func (d *decoder) astBlock(astBlockID uint64) (out *ast.Block) {
	d.build(d.content.Instances.AstBlock, &d.inst.ASTBlock, astBlockID, &out,
		func(p *ASTBlock, s *ast.Block) {
			foreach(p.Statements, d.astNode, &s.Statements)
		})
	return
}

func (d *decoder) astDeclareLocal(astDeclareLocalID uint64) (out *ast.DeclareLocal) {
	d.build(d.content.Instances.AstDeclareLocal, &d.inst.ASTDeclareLocal, astDeclareLocalID, &out,
		func(p *ASTDeclareLocal, s *ast.DeclareLocal) {
			s.Name = d.astIdentifier(p.Name)
			s.RHS = d.astNode(p.Rhs)
		})
	return
}

func (d *decoder) astDefault(astDefaultID uint64) (out *ast.Default) {
	d.build(d.content.Instances.AstDefault, &d.inst.ASTDefault, astDefaultID, &out,
		func(p *ASTDefault, s *ast.Default) {
			s.Block = d.astBlock(p.Block)
		})
	return
}

func (d *decoder) astDefinition(astDefinitionID uint64) (out *ast.Definition) {
	d.build(d.content.Instances.AstDefinition, &d.inst.ASTDefinition, astDefinitionID, &out,
		func(p *ASTDefinition, s *ast.Definition) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Name = d.astIdentifier(p.Name)
			s.Expression = d.astNode(p.Expression)
		})
	return
}

func (d *decoder) astDelete(astDeleteID uint64) (out *ast.Delete) {
	d.build(d.content.Instances.AstDelete, &d.inst.ASTDelete, astDeleteID, &out,
		func(p *ASTDelete, s *ast.Delete) {
			s.Key = d.astNode(p.Key)
			s.Map = d.astNode(p.Map)
		})
	return
}

func (d *decoder) astClear(astClearID uint64) (out *ast.Clear) {
	d.build(d.content.Instances.AstClear, &d.inst.ASTClear, astClearID, &out,
		func(p *ASTClear, s *ast.Clear) {
			s.Map = d.astNode(p.Map)
		})
	return
}

func (d *decoder) astEnum(astEnumID uint64) (out *ast.Enum) {
	d.build(d.content.Instances.AstEnum, &d.inst.ASTEnum, astEnumID, &out,
		func(p *ASTEnum, s *ast.Enum) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.NumberType = d.astNode(p.NumberType)
			s.Name = d.astIdentifier(p.Name)
			s.IsBitfield = p.IsBitfield
			foreach(p.Entries, d.astEnumEntry, &s.Entries)
		})
	return
}

func (d *decoder) astEnumEntry(astEnumEntryID uint64) (out *ast.EnumEntry) {
	d.build(d.content.Instances.AstEnumEntry, &d.inst.ASTEnumEntry, astEnumEntryID, &out,
		func(p *ASTEnumEntry, s *ast.EnumEntry) {
			s.Owner = d.astEnum(p.Owner)
			s.Name = d.astIdentifier(p.Name)
			s.Value = d.astNumber(p.Value)
		})
	return
}

func (d *decoder) astField(astFieldID uint64) (out *ast.Field) {
	d.build(d.content.Instances.AstField, &d.inst.ASTField, astFieldID, &out,
		func(p *ASTField, s *ast.Field) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Type = d.astNode(p.Type)
			s.Name = d.astIdentifier(p.Name)
			s.Default = d.astNode(p.Default)
		})
	return
}

func (d *decoder) astFence(astFenceID uint64) (out *ast.Fence) {
	d.build(d.content.Instances.AstFence, &d.inst.ASTFence, astFenceID, &out,
		func(p *ASTFence, s *ast.Fence) {})
	return
}

func (d *decoder) astFunction(astFunctionID uint64) (out *ast.Function) {
	d.build(d.content.Instances.AstFunction, &d.inst.ASTFunction, astFunctionID, &out,
		func(p *ASTFunction, s *ast.Function) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Generic = d.astGeneric(p.Generic)
			foreach(p.Parameters, d.astParameter, &s.Parameters)
			s.Block = d.astBlock(p.Block)
		})
	return
}

func (d *decoder) astGeneric(astGenericID uint64) (out *ast.Generic) {
	d.build(d.content.Instances.AstGeneric, &d.inst.ASTGeneric, astGenericID, &out,
		func(p *ASTGeneric, s *ast.Generic) {
			s.Name = d.astIdentifier(p.Name)
			foreach(p.Arguments, d.astNode, &s.Arguments)
		})
	return
}

func (d *decoder) astGroup(astGroupID uint64) (out *ast.Group) {
	d.build(d.content.Instances.AstGroup, &d.inst.ASTGroup, astGroupID, &out,
		func(p *ASTGroup, s *ast.Group) {
			s.Expression = d.astNode(p.Expression)
		})
	return
}

func (d *decoder) astIdentifier(astIdentifierID uint64) (out *ast.Identifier) {
	d.build(d.content.Instances.AstIdentifier, &d.inst.ASTIdentifier, astIdentifierID, &out,
		func(p *ASTIdentifier, s *ast.Identifier) {
			s.Value = d.str(p.Value)
		})
	return
}

func (d *decoder) astIndex(astIndexID uint64) (out *ast.Index) {
	d.build(d.content.Instances.AstIndex, &d.inst.ASTIndex, astIndexID, &out,
		func(p *ASTIndex, s *ast.Index) {
			s.Index = d.astNode(p.Index)
			s.Object = d.astNode(p.Object)
		})
	return
}

func (d *decoder) astIndexedType(astIndexedTypeID uint64) (out *ast.IndexedType) {
	d.build(d.content.Instances.AstIndexedType, &d.inst.ASTIndexedType, astIndexedTypeID, &out,
		func(p *ASTIndexedType, s *ast.IndexedType) {
			s.Index = d.astNode(p.Index)
			s.ValueType = d.astNode(p.ValueType)
		})
	return
}

func (d *decoder) astImport(astImportID uint64) (out *ast.Import) {
	d.build(d.content.Instances.AstImport, &d.inst.ASTImport, astImportID, &out,
		func(p *ASTImport, s *ast.Import) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Path = d.astString(p.Path)
		})
	return
}

func (d *decoder) astIteration(astIterationID uint64) (out *ast.Iteration) {
	d.build(d.content.Instances.AstIteration, &d.inst.ASTIteration, astIterationID, &out,
		func(p *ASTIteration, s *ast.Iteration) {
			s.Variable = d.astIdentifier(p.Variable)
			s.Iterable = d.astNode(p.Iterable)
			s.Block = d.astBlock(p.Block)
		})
	return
}

func (d *decoder) astMapIteration(astMapIterationID uint64) (out *ast.MapIteration) {
	d.build(d.content.Instances.AstMapIteration, &d.inst.ASTMapIteration, astMapIterationID, &out,
		func(p *ASTMapIteration, s *ast.MapIteration) {
			s.IndexVariable = d.astIdentifier(p.IndexVariable)
			s.KeyVariable = d.astIdentifier(p.KeyVariable)
			s.ValueVariable = d.astIdentifier(p.ValueVariable)
			s.Map = d.astNode(p.Map)
			s.Block = d.astBlock(p.Block)
		})
	return
}

func (d *decoder) astMember(astMemberID uint64) (out *ast.Member) {
	d.build(d.content.Instances.AstMember, &d.inst.ASTMember, astMemberID, &out,
		func(p *ASTMember, s *ast.Member) {
			s.Name = d.astIdentifier(p.Name)
			s.Object = d.astNode(p.Object)
		})
	return
}

func (d *decoder) astNamedArg(astNamedArgID uint64) (out *ast.NamedArg) {
	d.build(d.content.Instances.AstNamedArg, &d.inst.ASTNamedArg, astNamedArgID, &out,
		func(p *ASTNamedArg, s *ast.NamedArg) {
			s.Name = d.astIdentifier(p.Name)
			s.Value = d.astNode(p.Value)
		})
	return
}

func (d *decoder) astNode(p *ASTNode) ast.Node {
	if p == nil {
		return nil
	}
	switch p := p.Ty.(type) {
	case *ASTNode_Abort:
		return d.astAbort(p.Abort)
	case *ASTNode_Annotation:
		return d.astAnnotation(p.Annotation)
	case *ASTNode_Api:
		return d.astAPI(p.Api)
	case *ASTNode_Assign:
		return d.astAssign(p.Assign)
	case *ASTNode_BinaryOp:
		return d.astBinaryOp(p.BinaryOp)
	case *ASTNode_Block:
		return d.astBlock(p.Block)
	case *ASTNode_Bool:
		return d.astBool(p.Bool)
	case *ASTNode_Branch:
		return d.astBranch(p.Branch)
	case *ASTNode_Call:
		return d.astCall(p.Call)
	case *ASTNode_Case:
		return d.astCase(p.Case)
	case *ASTNode_Class:
		return d.astClass(p.Class)
	case *ASTNode_Clear:
		return d.astClear(p.Clear)
	case *ASTNode_DeclareLocal:
		return d.astDeclareLocal(p.DeclareLocal)
	case *ASTNode_Default:
		return d.astDefault(p.Default)
	case *ASTNode_Definition:
		return d.astDefinition(p.Definition)
	case *ASTNode_Delete:
		return d.astDelete(p.Delete)
	case *ASTNode_Enum:
		return d.astEnum(p.Enum)
	case *ASTNode_EnumEntry:
		return d.astEnumEntry(p.EnumEntry)
	case *ASTNode_Fence:
		return d.astFence(p.Fence)
	case *ASTNode_Field:
		return d.astField(p.Field)
	case *ASTNode_Function:
		return d.astFunction(p.Function)
	case *ASTNode_Generic:
		return d.astGeneric(p.Generic)
	case *ASTNode_Group:
		return d.astGroup(p.Group)
	case *ASTNode_Identifier:
		return d.astIdentifier(p.Identifier)
	case *ASTNode_Import:
		return d.astImport(p.Import)
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
	case *ASTNode_Parameter:
		return d.astParameter(p.Parameter)
	case *ASTNode_PointerType:
		return d.astPointerType(p.PointerType)
	case *ASTNode_PreConst:
		return d.astPreConst(p.PreConst)
	case *ASTNode_Pseudonym:
		return d.astPseudonym(p.Pseudonym)
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
	d.build(d.content.Instances.AstNull, &d.inst.ASTNull, astNullID, &out,
		func(p *ASTNull, s *ast.Null) {})
	return
}

func (d *decoder) astNumber(astNumberID uint64) (out *ast.Number) {
	d.build(d.content.Instances.AstNumber, &d.inst.ASTNumber, astNumberID, &out,
		func(p *ASTNumber, s *ast.Number) {
			s.Value = d.str(p.Value)
		})
	return
}

func (d *decoder) astParameter(astParameterID uint64) (out *ast.Parameter) {
	d.build(d.content.Instances.AstParameter, &d.inst.ASTParameter, astParameterID, &out,
		func(p *ASTParameter, s *ast.Parameter) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.This = p.This
			s.Type = d.astNode(p.Type)
			s.Name = d.astIdentifier(p.Name)
		})
	return
}

func (d *decoder) astPointerType(astPointerTypeID uint64) (out *ast.PointerType) {
	d.build(d.content.Instances.AstPointerType, &d.inst.ASTPointerType, astPointerTypeID, &out,
		func(p *ASTPointerType, s *ast.PointerType) {
			s.To = d.astNode(p.To)
			s.Const = p.Const
		})
	return
}

func (d *decoder) astPreConst(astPreConstID uint64) (out *ast.PreConst) {
	d.build(d.content.Instances.AstPreConst, &d.inst.ASTPreConst, astPreConstID, &out,
		func(p *ASTPreConst, s *ast.PreConst) {
			s.Type = d.astNode(p.Type)
		})
	return
}

func (d *decoder) astPseudonym(astPseudonymID uint64) (out *ast.Pseudonym) {
	d.build(d.content.Instances.AstPseudonym, &d.inst.ASTPseudonym, astPseudonymID, &out,
		func(p *ASTPseudonym, s *ast.Pseudonym) {
			s.Annotations = d.astAnnotations(p.Annotations)
			s.Name = d.astIdentifier(p.Name)
			s.To = d.astNode(p.To)
		})
	return
}

func (d *decoder) astReturn(astReturnID uint64) (out *ast.Return) {
	d.build(d.content.Instances.AstReturn, &d.inst.ASTReturn, astReturnID, &out,
		func(p *ASTReturn, s *ast.Return) {
			s.Value = d.astNode(p.Value)
		})
	return
}

func (d *decoder) astString(astStringID uint64) (out *ast.String) {
	d.build(d.content.Instances.AstString, &d.inst.ASTString, astStringID, &out,
		func(p *ASTString, s *ast.String) {
			s.Value = d.str(p.Value)
		})
	return
}

func (d *decoder) astSwitch(astSwitchID uint64) (out *ast.Switch) {
	d.build(d.content.Instances.AstSwitch, &d.inst.ASTSwitch, astSwitchID, &out,
		func(p *ASTSwitch, s *ast.Switch) {
			s.Value = d.astNode(p.Value)
			foreach(p.Cases, d.astCase, &s.Cases)
			s.Default = d.astDefault(p.Default)
		})
	return
}

func (d *decoder) astUnaryOp(astUnaryOpID uint64) (out *ast.UnaryOp) {
	d.build(d.content.Instances.AstUnaryOp, &d.inst.ASTUnaryOp, astUnaryOpID, &out,
		func(p *ASTUnaryOp, s *ast.UnaryOp) {
			s.Operator = d.str(p.Operator)
			s.Expression = d.astNode(p.Expression)
		})
	return
}

func (d *decoder) astUnknown(astUnknownID uint64) (out *ast.Unknown) {
	d.build(d.content.Instances.AstUnknown, &d.inst.ASTUnknown, astUnknownID, &out,
		func(p *ASTUnknown, s *ast.Unknown) {})
	return
}

func (d *decoder) cstBranch(cstBranchID uint64) (out *cst.Branch) {
	d.build(d.content.Instances.CstBranch, &d.inst.CSTBranch, cstBranchID, &out,
		func(p *CSTBranch, s *cst.Branch) {
			s.Branch = d.cstBranch(p.Branch)
			s.Pre = d.cstSeparator(p.Pre)
			s.Post = d.cstSeparator(p.Post)
			foreach(p.Children, d.cstNode, &s.Children)
		})
	return
}

func (d *decoder) cstFragment(p *CSTFragment) cst.Fragment {
	switch n := p.Ty.(type) {
	case *CSTFragment_Branch:
		return d.cstBranch(n.Branch)
	case *CSTFragment_Token:
		return d.cstToken(n.Token)
	default:
		panic(fmt.Errorf("Unhandled CST Fragment: %T", n))
	}
}

func (d *decoder) cstLeaf(cstLeafID uint64) (out *cst.Leaf) {
	d.build(d.content.Instances.CstLeaf, &d.inst.CSTLeaf, cstLeafID, &out,
		func(p *CSTLeaf, s *cst.Leaf) {
			s.Token = d.cstToken(p.Token)
			s.Branch = d.cstBranch(p.Branch)
			s.Pre = d.cstSeparator(p.Pre)
			s.Post = d.cstSeparator(p.Post)
		})
	return
}

func (d *decoder) cstNode(p *CSTNode) cst.Node {
	switch n := p.Ty.(type) {
	case *CSTNode_Branch:
		return d.cstBranch(n.Branch)
	case *CSTNode_Leaf:
		return d.cstLeaf(n.Leaf)
	default:
		panic(fmt.Errorf("Unhandled CST Node: %T", n))
	}
}

func (d *decoder) cstSeparator(p *CSTSeparator) cst.Separator {
	if p == nil {
		return nil
	}
	s := cst.Separator{}
	foreach(p.Fragments, d.cstFragment, &s)
	return s
}

func (d *decoder) cstSource(cstSourceID uint64) (out *cst.Source) {
	d.build(d.content.Instances.CstSource, &d.inst.CSTSource, cstSourceID, &out,
		func(p *CSTSource, s *cst.Source) {
			s.Filename = d.str(p.Filename)
			s.Runes = []rune(d.str(p.Content))
		})
	return
}

func (d *decoder) cstToken(p *CSTToken) cst.Token {
	return cst.Token{
		Source: d.cstSource(p.Source),
		Start:  int(p.Start),
		End:    int(p.End),
	}
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
