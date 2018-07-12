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

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

type encoder struct {
	instances *Instances
	maps      encoderInstances
}

type encoderInstances struct {
	API        map[*semantic.API]uint64
	Array      map[*semantic.StaticArray]uint64
	Block      map[*semantic.Block]uint64
	Callable   map[*semantic.Callable]uint64
	Class      map[*semantic.Class]uint64
	Definition map[*semantic.Definition]uint64
	Enum       map[*semantic.Enum]uint64
	EnumEntry  map[*semantic.EnumEntry]uint64
	Expression map[semantic.Expression]uint64
	Field      map[*semantic.Field]uint64
	Function   map[*semantic.Function]uint64
	Global     map[*semantic.Global]uint64
	Local      map[*semantic.Local]uint64
	Map        map[*semantic.Map]uint64
	Parameter  map[*semantic.Parameter]uint64
	Pointer    map[*semantic.Pointer]uint64
	Pseudonym  map[*semantic.Pseudonym]uint64
	Reference  map[*semantic.Reference]uint64
	Signature  map[*semantic.Signature]uint64
	Slice      map[*semantic.Slice]uint64
	Statement  map[semantic.Statement]uint64

	ASTAnnotation   map[*ast.Annotation]uint64
	ASTAbort        map[*ast.Abort]uint64
	ASTAssign       map[*ast.Assign]uint64
	ASTBinaryOp     map[*ast.BinaryOp]uint64
	ASTBlock        map[*ast.Block]uint64
	ASTBool         map[*ast.Bool]uint64
	ASTBranch       map[*ast.Branch]uint64
	ASTCall         map[*ast.Call]uint64
	ASTCase         map[*ast.Case]uint64
	ASTClass        map[*ast.Class]uint64
	ASTDeclareLocal map[*ast.DeclareLocal]uint64
	ASTDefault      map[*ast.Default]uint64
	ASTDefinition   map[*ast.Definition]uint64
	ASTDelete       map[*ast.Delete]uint64
	ASTEnum         map[*ast.Enum]uint64
	ASTEnumEntry    map[*ast.EnumEntry]uint64
	ASTFence        map[*ast.Fence]uint64
	ASTField        map[*ast.Field]uint64
	ASTFunction     map[*ast.Function]uint64
	ASTGeneric      map[*ast.Generic]uint64
	ASTGroup        map[*ast.Group]uint64
	ASTIdentifier   map[*ast.Identifier]uint64
	ASTIndex        map[*ast.Index]uint64
	ASTIndexedType  map[*ast.IndexedType]uint64
	ASTIteration    map[*ast.Iteration]uint64
	ASTMapIteration map[*ast.MapIteration]uint64
	ASTMember       map[*ast.Member]uint64
	ASTNamedArg     map[*ast.NamedArg]uint64
	ASTNull         map[*ast.Null]uint64
	ASTNumber       map[*ast.Number]uint64
	ASTParameter    map[*ast.Parameter]uint64
	ASTPointerType  map[*ast.PointerType]uint64
	ASTPreConst     map[*ast.PreConst]uint64
	ASTPseudonym    map[*ast.Pseudonym]uint64
	ASTReturn       map[*ast.Return]uint64
	ASTString       map[*ast.String]uint64
	ASTSwitch       map[*ast.Switch]uint64
	ASTUnaryOp      map[*ast.UnaryOp]uint64
	ASTUnknown      map[*ast.Unknown]uint64

	String map[string]uint64
}

func (i *encoderInstances) build() {
	str := reflect.ValueOf(i).Elem()
	for i, c := 0, str.NumField(); i < c; i++ {
		f := str.Field(i)
		f.Set(reflect.MakeMap(f.Type()))
	}
}

// Encode serializes apis to a byte slice.
func Encode(apis []*semantic.API) ([]byte, error) {
	e := &encoder{instances: &Instances{}}
	e.maps.build()
	content := &Content{
		Instances: e.instances,
		Apis:      make([]uint64, len(apis)),
	}
	for i, api := range apis {
		content.Apis[i] = e.api(api)
	}

	data, err := proto.Marshal(content)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (e *encoder) node(n semantic.Node) *Node {
	switch n := n.(type) {
	case nil:
		return nil
	case *semantic.API:
		return &Node{Ty: &Node_Api{Api: e.api(n)}}
	case *semantic.Class:
		return &Node{Ty: &Node_Class{Class: e.class(n)}}
	case *semantic.Enum:
		return &Node{Ty: &Node_Enum{Enum: e.enum(n)}}
	case *semantic.StaticArray:
		return &Node{Ty: &Node_StaticArray{StaticArray: e.array(n)}}
	default:
		panic(fmt.Errorf("Unhandled node type %T", n))
	}
}

func (e *encoder) annotation(n *semantic.Annotation) *Annotation {
	p := &Annotation{
		Ast:  e.astAnnotation(n.AST),
		Name: e.str(n.Name()),
	}
	foreach(n.Arguments, e.expr, &p.Arguments)
	return p
}

func (e *encoder) annotations(n semantic.Annotations) *Annotations {
	p := &Annotations{}
	foreach(n, e.annotation, &p.Annotations)
	return p
}

func (e *encoder) array(n *semantic.StaticArray) (outID uint64) {
	e.build(&e.instances.StaticArray, e.maps.Array, n, &outID, func() *StaticArray {
		p := &StaticArray{}
		p.Owner = e.node(n.Owner())
		p.Name = e.str(n.Name())
		p.ValueType = e.ty(n.ValueType)
		p.Size = n.Size
		p.SizeExpr = e.expr(n.SizeExpr)
		return p
	})
	return
}

func (e *encoder) api(n *semantic.API) (outID uint64) {
	e.build(&e.instances.Api, e.maps.API, n, &outID, func() *API {
		n.SortMembers()
		p := &API{}
		p.Name = e.str(n.Named.Name())
		foreach(n.Enums, e.enum, &p.Enums)
		foreach(n.Definitions, e.definition, &p.Definitions)
		foreach(n.Classes, e.class, &p.Classes)
		foreach(n.Pseudonyms, e.pseudonym, &p.Pseudonyms)
		foreach(n.Externs, e.function, &p.Externs)
		foreach(n.Subroutines, e.function, &p.Subroutines)
		foreach(n.Functions, e.function, &p.Functions)
		foreach(n.Methods, e.function, &p.Methods)
		foreach(n.Globals, e.global, &p.Globals)
		foreach(n.StaticArrays, e.array, &p.StaticArrays)
		foreach(n.Maps, e.map_, &p.Maps)
		foreach(n.Pointers, e.pointer, &p.Pointers)
		foreach(n.Slices, e.slice, &p.Slices)
		foreach(n.References, e.reference, &p.References)
		foreach(n.Signatures, e.signature, &p.Signatures)
		if n.Index != nil {
			p.Index = uint32(*n.Index)
		}
		return p
	})

	return
}

func (e *encoder) block(n *semantic.Block) (outID uint64) {
	e.build(&e.instances.Block, e.maps.Block, n, &outID, func() *Block {
		p := &Block{}
		p.Ast = e.astBlock(n.AST)
		foreach(n.Statements, e.stat, &p.Statements)
		return p
	})
	return
}

func (e *encoder) callable(n *semantic.Callable) (outID uint64) {
	e.build(&e.instances.Callable, e.maps.Callable, n, &outID, func() *Callable {
		return &Callable{
			Object:   e.expr(n.Object),
			Function: e.function(n.Function),
		}
	})
	return
}

func (e *encoder) case_(n *semantic.Case) *Case {
	p := &Case{
		Ast:         e.astCase(n.AST),
		Annotations: e.annotations(n.Annotations),
		Block:       e.block(n.Block),
	}
	foreach(n.Conditions, e.expr, &p.Conditions)
	return p
}

func (e *encoder) choice(n *semantic.Choice) *Choice {
	p := &Choice{
		Ast:         e.astCase(n.AST),
		Annotations: e.annotations(n.Annotations),
		Expression:  e.expr(n.Expression),
	}
	foreach(n.Conditions, e.expr, &p.Conditions)
	return p
}

func (e *encoder) class(n *semantic.Class) (outID uint64) {
	e.build(&e.instances.Class, e.maps.Class, n, &outID, func() *Class {
		n.SortMembers()
		p := &Class{}
		p.Owner = e.node(n.Owner())
		p.Ast = e.astClass(n.AST)
		p.Name = e.str(n.Name())
		p.Annotations = e.annotations(n.Annotations)
		p.Docs = e.docs(n.Docs)
		foreach(n.Fields, e.field, &p.Fields)
		foreach(n.Methods, e.function, &p.Methods)
		return p
	})
	return
}

func (e *encoder) definition(n *semantic.Definition) (outID uint64) {
	e.build(&e.instances.Definition, e.maps.Definition, n, &outID, func() *Definition {
		p := &Definition{}
		p.Name = e.str(n.Name())
		p.Ast = e.astDefinition(n.AST)
		p.Annotations = e.annotations(n.Annotations)
		p.Docs = e.docs(n.Docs)
		p.Expression = e.expr(n.Expression)
		return p
	})
	return
}

func (e *encoder) docs(n semantic.Documentation) *Documentation {
	p := &Documentation{}
	foreach(n, e.str, &p.Strings)
	return p
}

func (e *encoder) enum(n *semantic.Enum) (outID uint64) {
	e.build(&e.instances.Enum, e.maps.Enum, n, &outID, func() *Enum {
		p := &Enum{}
		p.Owner = e.node(n.Owner())
		p.Ast = e.astEnum(n.AST)
		p.Annotations = e.annotations(n.Annotations)
		p.Name = e.str(n.Name())
		p.Docs = e.docs(n.Docs)
		p.IsBitfield = n.IsBitfield
		p.NumberType = e.ty(n.NumberType)
		foreach(n.Entries, e.enumEntry, &p.Entries)
		return p
	})
	return
}

func (e *encoder) enumEntry(n *semantic.EnumEntry) (outID uint64) {
	e.build(&e.instances.EnumEntry, e.maps.EnumEntry, n, &outID, func() *EnumEntry {
		p := &EnumEntry{}
		p.Owner = e.node(n.Owner())
		p.Ast = e.astEnumEntry(n.AST)
		p.Name = e.str(n.Name())
		p.Docs = e.docs(n.Docs)
		p.Value = e.expr(n.Value)
		return p
	})
	return
}

func (e *encoder) expr(n semantic.Expression) (outID uint64) {
	e.build(&e.instances.Expression, e.maps.Expression, n, &outID, func() *Expression {
		p := &Expression{}
		switch n := n.(type) {
		case *semantic.ArrayIndex:
			p.Ty = &Expression_ArrayIndex{&ArrayIndex{
				Ast:   e.astIndex(n.AST),
				Type:  e.array(n.Type),
				Array: e.expr(n.Array),
				Index: e.expr(n.Index),
			}}
		case *semantic.ArrayInitializer:
			ai := &ArrayInitializer{
				Ast:   e.astCall(n.AST),
				Array: e.ty(n.Array),
			}
			foreach(n.Values, e.expr, &ai.Values)
			p.Ty = &Expression_ArrayInitializer{ai}
		case *semantic.BinaryOp:
			p.Ty = &Expression_BinaryOp{&BinaryOp{
				Ast:      e.astBinaryOp(n.AST),
				Type:     e.ty(n.Type),
				Lhs:      e.expr(n.LHS),
				Operator: e.str(n.Operator),
				Rhs:      e.expr(n.RHS),
			}}
		case *semantic.BitTest:
			p.Ty = &Expression_BitTest{&BitTest{
				Ast:      e.astBinaryOp(n.AST),
				Bitfield: e.expr(n.Bitfield),
				Bits:     e.expr(n.Bits),
			}}
		case semantic.BoolValue:
			p.Ty = &Expression_BoolValue{bool(n)}
		case *semantic.Call:
			c := &Call{
				Ast:    e.astCall(n.AST),
				Target: e.callable(n.Target),
				Type:   e.ty(n.Type),
			}
			foreach(n.Arguments, e.expr, &c.Arguments)
			p.Ty = &Expression_Call{c}
		case *semantic.Cast:
			p.Ty = &Expression_Cast{&Cast{
				Ast:    e.astCall(n.AST),
				Object: e.expr(n.Object),
				Type:   e.ty(n.Type),
			}}
		case *semantic.ClassInitializer:
			ci := &ClassInitializer{
				Ast:   e.astCall(n.AST),
				Class: e.class(n.Class),
			}
			foreach(n.Fields, e.fieldInit, &ci.Fields)
			p.Ty = &Expression_ClassInitializer{ci}
		case *semantic.Clone:
			p.Ty = &Expression_Clone{&Clone{
				Ast:   e.astCall(n.AST),
				Slice: e.expr(n.Slice),
				Type:  e.slice(n.Type),
			}}
		case *semantic.Create:
			p.Ty = &Expression_Create{&Create{
				Ast:         e.astCall(n.AST),
				Type:        e.reference(n.Type),
				Initializer: e.expr(n.Initializer),
			}}
		case *semantic.Definition:
			p.Ty = &Expression_Definition{e.definition(n)}
		case *semantic.EnumEntry:
			p.Ty = &Expression_EnumEntry{e.enumEntry(n)}
		case semantic.Float32Value:
			p.Ty = &Expression_Float32Value{float32(n)}
		case semantic.Float64Value:
			p.Ty = &Expression_Float64Value{float64(n)}
		case *semantic.Global:
			p.Ty = &Expression_Global{e.global(n)}
		case *semantic.Ignore:
			p.Ty = &Expression_Ignore{&Ignore{
				Ast: e.astNode(n.AST),
			}}
		case semantic.Int8Value:
			p.Ty = &Expression_Int8Value{int32(n)}
		case semantic.Int16Value:
			p.Ty = &Expression_Int16Value{int32(n)}
		case semantic.Int32Value:
			p.Ty = &Expression_Int32Value{int32(n)}
		case semantic.Int64Value:
			p.Ty = &Expression_Int64Value{int64(n)}
		case *semantic.Length:
			p.Ty = &Expression_Length{&Length{
				Ast:    e.astCall(n.AST),
				Object: e.expr(n.Object),
				Type:   e.ty(n.Type),
			}}
		case *semantic.Local:
			p.Ty = &Expression_Local{e.local(n)}
		case *semantic.Make:
			p.Ty = &Expression_Make{&Make{
				Ast:  e.astCall(n.AST),
				Type: e.slice(n.Type),
				Size: e.expr(n.Size),
			}}
		case *semantic.MapContains:
			p.Ty = &Expression_MapContains{&MapContains{
				Ast:  e.astBinaryOp(n.AST),
				Type: e.map_(n.Type),
				Map:  e.expr(n.Map),
				Key:  e.expr(n.Key),
			}}
		case *semantic.MapIndex:
			p.Ty = &Expression_MapIndex{&MapIndex{
				Ast:   e.astIndex(n.AST),
				Type:  e.map_(n.Type),
				Map:   e.expr(n.Map),
				Index: e.expr(n.Index),
			}}
		case *semantic.Member:
			p.Ty = &Expression_Member{&Member{
				Ast:    e.astMember(n.AST),
				Object: e.expr(n.Object),
				Field:  e.field(n.Field),
			}}
		case *semantic.MessageValue:
			mv := &MessageValue{
				Ast: e.astClass(n.AST),
			}
			foreach(n.Arguments, e.fieldInit, &mv.Arguments)
			p.Ty = &Expression_MessageValue{mv}
		case semantic.Null:
			p.Ty = &Expression_Null{&Null{
				Ast:  e.astNull(n.AST),
				Type: e.ty(n.Type),
			}}
		case *semantic.Observed:
			p.Ty = &Expression_Observed{&Observed{
				Parameter: e.param(n.Parameter),
			}}
		case *semantic.Parameter:
			p.Ty = &Expression_Parameter{e.param(n)}
		case *semantic.PointerRange:
			p.Ty = &Expression_PointerRange{&PointerRange{
				Ast:     e.astIndex(n.AST),
				Type:    e.slice(n.Type),
				Pointer: e.expr(n.Pointer),
				Range:   e.expr(n.Range),
			}}
		case *semantic.Select:
			s := &Select{
				Ast:     e.astSwitch(n.AST),
				Type:    e.ty(n.Type),
				Value:   e.expr(n.Value),
				Default: e.expr(n.Default),
			}
			foreach(n.Choices, e.choice, &s.Choices)
			p.Ty = &Expression_Select{s}
		case *semantic.SliceIndex:
			p.Ty = &Expression_SliceIndex{&SliceIndex{
				Ast:   e.astIndex(n.AST),
				Type:  e.slice(n.Type),
				Slice: e.expr(n.Slice),
				Index: e.expr(n.Index),
			}}
		case *semantic.SliceRange:
			p.Ty = &Expression_SliceRange{&SliceRange{
				Ast:   e.astIndex(n.AST),
				Type:  e.slice(n.Type),
				Slice: e.expr(n.Slice),
				Range: e.expr(n.Range),
			}}
		case semantic.StringValue:
			p.Ty = &Expression_StringValue{e.str(string(n))}
		case semantic.Uint8Value:
			p.Ty = &Expression_Uint8Value{uint32(n)}
		case semantic.Uint16Value:
			p.Ty = &Expression_Uint16Value{uint32(n)}
		case semantic.Uint32Value:
			p.Ty = &Expression_Uint32Value{uint32(n)}
		case semantic.Uint64Value:
			p.Ty = &Expression_Uint64Value{uint64(n)}
		case *semantic.UnaryOp:
			p.Ty = &Expression_UnaryOp{&UnaryOp{
				Ast:        e.astUnaryOp(n.AST),
				Type:       e.ty(n.Type),
				Operator:   e.str(n.Operator),
				Expression: e.expr(n.Expression),
			}}
		case *semantic.Unknown:
			p.Ty = &Expression_Unknown{&Unknown{
				Ast:      e.astUnknown(n.AST),
				Inferred: e.expr(n.Inferred),
			}}
		default:
			panic(fmt.Errorf("Unhandled expression type %T", n))
		}
		return p
	})
	return
}

func (e *encoder) field(n *semantic.Field) (outID uint64) {
	e.build(&e.instances.Field, e.maps.Field, n, &outID, func() *Field {
		p := &Field{}
		p.Owner = e.node(n.Owner())
		p.Ast = e.astField(n.AST)
		p.Annotations = e.annotations(n.Annotations)
		p.Type = e.ty(n.Type)
		p.Name = e.str(n.Name())
		p.Docs = e.docs(n.Docs)
		p.Default = e.expr(n.Default)
		return p
	})
	return
}

// TODO: Do we need to ID these?
func (e *encoder) fieldInit(n *semantic.FieldInitializer) *FieldInitializer {
	return &FieldInitializer{
		Ast:   e.astNode(n.AST),
		Field: e.field(n.Field),
		Value: e.expr(n.Value),
	}
}

func (e *encoder) function(n *semantic.Function) (outID uint64) {
	e.build(&e.instances.Function, e.maps.Function, n, &outID, func() *Function {
		p := &Function{}
		p.Owner = e.node(n.Owner())
		p.Ast = e.astFunction(n.AST)
		p.Annotations = e.annotations(n.Annotations)
		p.Name = e.str(n.Name())
		p.Docs = e.docs(n.Docs)
		p.Return = e.param(n.Return)
		p.This = e.param(n.This)
		foreach(n.FullParameters, e.param, &p.FullParameters)
		p.Block = e.block(n.Block)
		p.Signature = e.signature(n.Signature)
		p.Extern = n.Extern
		p.Subroutine = n.Subroutine
		p.Recursive = n.Recursive
		p.Order = &LogicalOrder{
			Resolved: n.Order.Resolved(),
			Pre:      n.Order.Pre(),
			Post:     n.Order.Post(),
		}
		return p
	})
	return
}

func (e *encoder) global(n *semantic.Global) (outID uint64) {
	e.build(&e.instances.Global, e.maps.Global, n, &outID, func() *Global {
		return &Global{
			Owner:       e.node(n.Owner()),
			Ast:         e.astField(n.AST),
			Annotations: e.annotations(n.Annotations),
			Type:        e.ty(n.Type),
			Name:        e.str(n.Name()),
			Default:     e.expr(n.Default),
		}
	})
	return
}

func (e *encoder) local(n *semantic.Local) (outID uint64) {
	e.build(&e.instances.Local, e.maps.Local, n, &outID, func() *Local {
		return &Local{
			Declaration: e.stat(n.Declaration),
			Type:        e.ty(n.Type),
			Name:        e.str(n.Name()),
			Value:       e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) map_(n *semantic.Map) (outID uint64) {
	e.build(&e.instances.Map, e.maps.Map, n, &outID, func() *Map {
		return &Map{
			Owner:     e.node(n.Owner()),
			Name:      e.str(n.Name()),
			KeyType:   e.ty(n.KeyType),
			ValueType: e.ty(n.ValueType),
		}
	})
	return
}

func (e *encoder) param(n *semantic.Parameter) (outID uint64) {
	e.build(&e.instances.Parameter, e.maps.Parameter, n, &outID, func() *Parameter {
		return &Parameter{
			Ast:         e.astParameter(n.AST),
			Annotations: e.annotations(n.Annotations),
			Function:    e.function(n.Function),
			Name:        e.str(n.Name()),
			Docs:        e.docs(n.Docs),
			Type:        e.ty(n.Type),
		}
	})
	return
}

func (e *encoder) pointer(n *semantic.Pointer) (outID uint64) {
	e.build(&e.instances.Pointer, e.maps.Pointer, n, &outID, func() *Pointer {
		return &Pointer{
			Owner: e.node(n.Owner()),
			Name:  e.str(n.Name()),
			To:    e.ty(n.To),
			Const: n.Const,
			Slice: e.slice(n.Slice),
		}
	})
	return
}

func (e *encoder) pseudonym(n *semantic.Pseudonym) (outID uint64) {
	e.build(&e.instances.Pseudonym, e.maps.Pseudonym, n, &outID, func() *Pseudonym {
		p := &Pseudonym{
			Owner:       e.node(n.Owner()),
			Ast:         e.astPseudonym(n.AST),
			Annotations: e.annotations(n.Annotations),
			Name:        e.str(n.Name()),
			Docs:        e.docs(n.Docs),
			To:          e.ty(n.To),
		}
		foreach(n.Methods, e.function, &p.Methods)
		return p
	})
	return
}

func (e *encoder) reference(n *semantic.Reference) (outID uint64) {
	e.build(&e.instances.Reference, e.maps.Reference, n, &outID, func() *Reference {
		return &Reference{
			Owner: e.node(n.Owner()),
			Name:  e.str(n.Name()),
			To:    e.ty(n.To),
		}
	})
	return
}

func (e *encoder) signature(n *semantic.Signature) (outID uint64) {
	e.build(&e.instances.Signature, e.maps.Signature, n, &outID, func() *Signature {
		p := &Signature{}
		p.Owner = e.node(n.Owner())
		p.Name = e.str(n.Name())
		p.Return = e.ty(n.Return)
		foreach(n.Arguments, e.ty, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) slice(n *semantic.Slice) (outID uint64) {
	e.build(&e.instances.Slice, e.maps.Slice, n, &outID, func() *Slice {
		return &Slice{
			Owner:   e.node(n.Owner()),
			Name:    e.str(n.Name()),
			To:      e.ty(n.To),
			Pointer: e.pointer(n.Pointer),
		}
	})
	return
}

func (e *encoder) stat(n semantic.Statement) (outID uint64) {
	e.build(&e.instances.Statement, e.maps.Statement, n, &outID, func() *Statement {
		p := &Statement{}
		switch n := n.(type) {
		case *semantic.Abort:
			p.Ty = &Statement_Abort{&Abort{
				Ast:       e.astAbort(n.AST),
				Function:  e.function(n.Function),
				Statement: e.stat(n.Statement),
			}}
		case *semantic.ArrayAssign:
			p.Ty = &Statement_ArrayAssign{&ArrayAssign{
				Ast:      e.astAssign(n.AST),
				To:       e.expr(n.To),
				Operator: e.str(n.Operator),
				Value:    e.expr(n.Value),
			}}
		case *semantic.Assert:
			p.Ty = &Statement_Assert{&Assert{
				Ast:       e.astCall(n.AST),
				Condition: e.expr(n.Condition),
			}}
		case *semantic.Assign:
			p.Ty = &Statement_Assign{&Assign{
				Ast:      e.astAssign(n.AST),
				Lhs:      e.expr(n.LHS),
				Operator: e.str(n.Operator),
				Rhs:      e.expr(n.RHS),
			}}
		case *semantic.Branch:
			p.Ty = &Statement_Branch{&Branch{
				Ast:       e.astBranch(n.AST),
				Condition: e.expr(n.Condition),
				True:      e.block(n.True),
				False:     e.block(n.False),
			}}
		case *semantic.Copy:
			p.Ty = &Statement_Copy{&Copy{
				Ast: e.astCall(n.AST),
				Src: e.expr(n.Src),
				Dst: e.expr(n.Dst),
			}}
		case *semantic.DeclareLocal:
			p.Ty = &Statement_DeclareLocal{&DeclareLocal{
				Ast:   e.astDeclareLocal(n.AST),
				Local: e.local(n.Local),
			}}
		case semantic.Expression:
			p.Ty = &Statement_Expression{e.expr(n)}
		case *semantic.Fence:
			p.Ty = &Statement_Fence{&Fence{
				Ast:       e.astFence(n.AST),
				Statement: e.stat(n.Statement),
				Explicit:  n.Explicit,
			}}
		case *semantic.Iteration:
			p.Ty = &Statement_Iteration{&Iteration{
				Ast:      e.astIteration(n.AST),
				Iterator: e.local(n.Iterator),
				From:     e.expr(n.From),
				To:       e.expr(n.To),
				Block:    e.block(n.Block),
			}}
		case *semantic.MapAssign:
			p.Ty = &Statement_MapAssign{&MapAssign{
				Ast:      e.astAssign(n.AST),
				To:       e.expr(n.To),
				Operator: e.str(n.Operator),
				Value:    e.expr(n.Value),
			}}
		case *semantic.MapIteration:
			p.Ty = &Statement_MapIteration{&MapIteration{
				Ast:           e.astMapIteration(n.AST),
				IndexIterator: e.local(n.IndexIterator),
				KeyIterator:   e.local(n.KeyIterator),
				ValueIterator: e.local(n.ValueIterator),
				Map:           e.expr(n.Map),
				Block:         e.block(n.Block),
			}}
		case *semantic.MapRemove:
			p.Ty = &Statement_MapRemove{&MapRemove{
				Ast:  e.astDelete(n.AST),
				Type: e.map_(n.Type),
				Map:  e.expr(n.Map),
				Key:  e.expr(n.Key),
			}}
		case *semantic.Read:
			p.Ty = &Statement_Read{&Read{
				Ast:   e.astCall(n.AST),
				Slice: e.expr(n.Slice),
			}}
		case *semantic.Return:
			p.Ty = &Statement_Return{&Return{
				Ast:      e.astReturn(n.AST),
				Function: e.function(n.Function),
				Value:    e.expr(n.Value),
			}}
		case *semantic.SliceAssign:
			p.Ty = &Statement_SliceAssign{&SliceAssign{
				Ast:      e.astAssign(n.AST),
				To:       e.expr(n.To),
				Operator: e.str(n.Operator),
				Value:    e.expr(n.Value),
			}}
		case *semantic.Switch:
			s := &Switch{
				Ast:     e.astSwitch(n.AST),
				Value:   e.expr(n.Value),
				Default: e.block(n.Default),
			}
			foreach(n.Cases, e.case_, &s.Cases)
			p.Ty = &Statement_Switch{s}
		case *semantic.Write:
			p.Ty = &Statement_Write{&Write{
				Ast:   e.astCall(n.AST),
				Slice: e.expr(n.Slice),
			}}
		default:
			panic(fmt.Errorf("Unhandled statement type %T", n))
		}
		return p
	})
	return
}

func (e *encoder) str(s string) (outID uint64) {
	e.build(&e.instances.Symbols, e.maps.String, s, &outID, func() string { return s })
	return
}

func (e *encoder) ty(n semantic.Type) *Type {
	if isNil(n) {
		return nil
	}
	switch n := n.(type) {
	case *semantic.Class:
		return &Type{Ty: &Type_Class{e.class(n)}}
	case *semantic.Enum:
		return &Type{Ty: &Type_Enum{e.enum(n)}}
	case *semantic.Map:
		return &Type{Ty: &Type_Map{e.map_(n)}}
	case *semantic.Pointer:
		return &Type{Ty: &Type_Pointer{e.pointer(n)}}
	case *semantic.Pseudonym:
		return &Type{Ty: &Type_Pseudonym{e.pseudonym(n)}}
	case *semantic.Reference:
		return &Type{Ty: &Type_Reference{e.reference(n)}}
	case *semantic.Slice:
		return &Type{Ty: &Type_Slice{e.slice(n)}}
	case *semantic.StaticArray:
		return &Type{Ty: &Type_StaticArray{e.array(n)}}
	case *semantic.Builtin:
		switch n {
		case semantic.VoidType:
			return &Type{Ty: &Type_Builtin{Builtin_VoidType}}
		case semantic.AnyType:
			return &Type{Ty: &Type_Builtin{Builtin_AnyType}}
		case semantic.StringType:
			return &Type{Ty: &Type_Builtin{Builtin_StringType}}
		case semantic.MessageType:
			return &Type{Ty: &Type_Builtin{Builtin_MessageType}}
		case semantic.BoolType:
			return &Type{Ty: &Type_Builtin{Builtin_BoolType}}
		case semantic.IntType:
			return &Type{Ty: &Type_Builtin{Builtin_IntType}}
		case semantic.UintType:
			return &Type{Ty: &Type_Builtin{Builtin_UintType}}
		case semantic.SizeType:
			return &Type{Ty: &Type_Builtin{Builtin_SizeType}}
		case semantic.CharType:
			return &Type{Ty: &Type_Builtin{Builtin_CharType}}
		case semantic.Int8Type:
			return &Type{Ty: &Type_Builtin{Builtin_Int8Type}}
		case semantic.Uint8Type:
			return &Type{Ty: &Type_Builtin{Builtin_Uint8Type}}
		case semantic.Int16Type:
			return &Type{Ty: &Type_Builtin{Builtin_Int16Type}}
		case semantic.Uint16Type:
			return &Type{Ty: &Type_Builtin{Builtin_Uint16Type}}
		case semantic.Int32Type:
			return &Type{Ty: &Type_Builtin{Builtin_Int32Type}}
		case semantic.Uint32Type:
			return &Type{Ty: &Type_Builtin{Builtin_Uint32Type}}
		case semantic.Int64Type:
			return &Type{Ty: &Type_Builtin{Builtin_Int64Type}}
		case semantic.Uint64Type:
			return &Type{Ty: &Type_Builtin{Builtin_Uint64Type}}
		case semantic.Float32Type:
			return &Type{Ty: &Type_Builtin{Builtin_Float32Type}}
		case semantic.Float64Type:
			return &Type{Ty: &Type_Builtin{Builtin_Float64Type}}
		default:
			panic(fmt.Errorf("Unhandled builtin type %v", n))
		}
	default:
		panic(fmt.Errorf("Unhandled type %T", n))
	}
}

func (e *encoder) astAnnotation(n *ast.Annotation) (outID uint64) {
	e.build(&e.instances.AstAnnotation, e.maps.ASTAnnotation, n, &outID, func() *ASTAnnotation {
		out := &ASTAnnotation{
			Name: e.astIdentifier(n.Name),
		}
		foreach(n.Arguments, e.astNode, &out.Arguments)
		return out
	})
	return
}

func (e *encoder) astAnnotations(n ast.Annotations) *ASTAnnotations {
	out := &ASTAnnotations{}
	foreach(n, e.astAnnotation, &out.Annotations)
	return out
}

func (e *encoder) astAbort(n *ast.Abort) (outID uint64) {
	e.build(&e.instances.AstAbort, e.maps.ASTAbort, n, &outID, func() *ASTAbort {
		return &ASTAbort{}
	})
	return
}

func (e *encoder) astAssign(n *ast.Assign) (outID uint64) {
	e.build(&e.instances.AstAssign, e.maps.ASTAssign, n, &outID, func() *ASTAssign {
		return &ASTAssign{
			Lhs:      e.astNode(n.LHS),
			Operator: e.str(n.Operator),
			Rhs:      e.astNode(n.RHS),
		}
	})
	return
}

func (e *encoder) astBinaryOp(n *ast.BinaryOp) (outID uint64) {
	e.build(&e.instances.AstBinaryOp, e.maps.ASTBinaryOp, n, &outID, func() *ASTBinaryOp {
		return &ASTBinaryOp{
			Lhs:      e.astNode(n.LHS),
			Operator: e.str(n.Operator),
			Rhs:      e.astNode(n.RHS),
		}
	})
	return
}

func (e *encoder) astBlock(n *ast.Block) (outID uint64) {
	e.build(&e.instances.AstBlock, e.maps.ASTBlock, n, &outID, func() *ASTBlock {
		p := &ASTBlock{}
		foreach(n.Statements, e.astNode, &p.Statements)
		return p
	})
	return
}

func (e *encoder) astBool(n *ast.Bool) (outID uint64) {
	e.build(&e.instances.AstBool, e.maps.ASTBool, n, &outID, func() *ASTBool {
		return &ASTBool{
			Value: n.Value,
		}
	})
	return
}

func (e *encoder) astBranch(n *ast.Branch) (outID uint64) {
	e.build(&e.instances.AstBranch, e.maps.ASTBranch, n, &outID, func() *ASTBranch {
		return &ASTBranch{
			Condition: e.astNode(n.Condition),
			True:      e.astBlock(n.True),
			False:     e.astBlock(n.False),
		}
	})
	return
}

func (e *encoder) astCall(n *ast.Call) (outID uint64) {
	e.build(&e.instances.AstCall, e.maps.ASTCall, n, &outID, func() *ASTCall {
		p := &ASTCall{
			Target: e.astNode(n.Target),
		}
		foreach(n.Arguments, e.astNode, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) astDeclareLocal(n *ast.DeclareLocal) (outID uint64) {
	e.build(&e.instances.AstDeclareLocal, e.maps.ASTDeclareLocal, n, &outID, func() *ASTDeclareLocal {
		return &ASTDeclareLocal{
			Name: e.astIdentifier(n.Name),
			Rhs:  e.astNode(n.RHS),
		}
	})
	return
}

func (e *encoder) astDefault(n *ast.Default) (outID uint64) {
	e.build(&e.instances.AstDefault, e.maps.ASTDefault, n, &outID, func() *ASTDefault {
		return &ASTDefault{
			Block: e.astBlock(n.Block),
		}
	})
	return
}

func (e *encoder) astDelete(n *ast.Delete) (outID uint64) {
	e.build(&e.instances.AstDelete, e.maps.ASTDelete, n, &outID, func() *ASTDelete {
		return &ASTDelete{
			Map: e.astNode(n.Map),
			Key: e.astNode(n.Key),
		}
	})
	return
}

func (e *encoder) astCase(n *ast.Case) (outID uint64) {
	e.build(&e.instances.AstCase, e.maps.ASTCase, n, &outID, func() *ASTCase {
		p := &ASTCase{
			Annotations: e.astAnnotations(n.Annotations),
			Block:       e.astBlock(n.Block),
		}
		foreach(n.Conditions, e.astNode, &p.Conditions)
		return p
	})
	return
}

func (e *encoder) astClass(n *ast.Class) (outID uint64) {
	e.build(&e.instances.AstClass, e.maps.ASTClass, n, &outID, func() *ASTClass {
		p := &ASTClass{
			Annotations: e.astAnnotations(n.Annotations),
			Name:        e.astIdentifier(n.Name),
		}
		foreach(n.Fields, e.astField, &p.Fields)
		return p
	})
	return
}

func (e *encoder) astDefinition(n *ast.Definition) (outID uint64) {
	e.build(&e.instances.AstDefinition, e.maps.ASTDefinition, n, &outID, func() *ASTDefinition {
		p := &ASTDefinition{
			Annotations: e.astAnnotations(n.Annotations),
			Name:        e.astIdentifier(n.Name),
			Expression:  e.astNode(n.Expression),
		}
		return p
	})
	return
}

func (e *encoder) astEnum(n *ast.Enum) (outID uint64) {
	e.build(&e.instances.AstEnum, e.maps.ASTEnum, n, &outID, func() *ASTEnum {
		p := &ASTEnum{
			Annotations: e.astAnnotations(n.Annotations),
			NumberType:  e.astNode(n.NumberType),
			Name:        e.astIdentifier(n.Name),
			IsBitfield:  n.IsBitfield,
		}
		foreach(n.Entries, e.astEnumEntry, &p.Entries)
		return p
	})
	return
}

func (e *encoder) astEnumEntry(n *ast.EnumEntry) (outID uint64) {
	e.build(&e.instances.AstEnumEntry, e.maps.ASTEnumEntry, n, &outID, func() *ASTEnumEntry {
		return &ASTEnumEntry{
			Owner: e.astEnum(n.Owner),
			Name:  e.astIdentifier(n.Name),
			Value: e.astNumber(n.Value),
		}
	})
	return
}

func (e *encoder) astFence(n *ast.Fence) (outID uint64) {
	e.build(&e.instances.AstFence, e.maps.ASTFence, n, &outID, func() *ASTFence {
		return &ASTFence{}
	})
	return
}

func (e *encoder) astField(n *ast.Field) (outID uint64) {
	e.build(&e.instances.AstField, e.maps.ASTField, n, &outID, func() *ASTField {
		return &ASTField{
			Annotations: e.astAnnotations(n.Annotations),
			Type:        e.astNode(n.Type),
			Name:        e.astIdentifier(n.Name),
			Default:     e.astNode(n.Default),
		}
	})
	return
}

func (e *encoder) astFunction(n *ast.Function) (outID uint64) {
	e.build(&e.instances.AstFunction, e.maps.ASTFunction, n, &outID, func() *ASTFunction {
		p := &ASTFunction{
			Annotations: e.astAnnotations(n.Annotations),
			Generic:     e.astGeneric(n.Generic),
			Block:       e.astBlock(n.Block),
		}
		foreach(n.Parameters, e.astParameter, &p.Parameters)
		return p
	})
	return
}

func (e *encoder) astGeneric(n *ast.Generic) (outID uint64) {
	e.build(&e.instances.AstGeneric, e.maps.ASTGeneric, n, &outID, func() *ASTGeneric {
		p := &ASTGeneric{
			Name: e.astIdentifier(n.Name),
		}
		foreach(n.Arguments, e.astNode, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) astGroup(n *ast.Group) (outID uint64) {
	e.build(&e.instances.AstGroup, e.maps.ASTGroup, n, &outID, func() *ASTGroup {
		return &ASTGroup{
			Expression: e.astNode(n.Expression),
		}
	})
	return
}

func (e *encoder) astIdentifier(n *ast.Identifier) (outID uint64) {
	e.build(&e.instances.AstIdentifier, e.maps.ASTIdentifier, n, &outID, func() *ASTIdentifier {
		return &ASTIdentifier{Value: e.str(n.Value)}
	})
	return
}

func (e *encoder) astIndex(n *ast.Index) (outID uint64) {
	e.build(&e.instances.AstIndex, e.maps.ASTIndex, n, &outID, func() *ASTIndex {
		return &ASTIndex{
			Object: e.astNode(n.Object),
			Index:  e.astNode(n.Index),
		}
	})
	return
}

func (e *encoder) astIndexedType(n *ast.IndexedType) (outID uint64) {
	e.build(&e.instances.AstIndexedType, e.maps.ASTIndexedType, n, &outID, func() *ASTIndexedType {
		return &ASTIndexedType{
			ValueType: e.astNode(n.ValueType),
			Index:     e.astNode(n.Index),
		}
	})
	return
}

func (e *encoder) astIteration(n *ast.Iteration) (outID uint64) {
	e.build(&e.instances.AstIteration, e.maps.ASTIteration, n, &outID, func() *ASTIteration {
		return &ASTIteration{
			Variable: e.astIdentifier(n.Variable),
			Iterable: e.astNode(n.Iterable),
			Block:    e.astBlock(n.Block),
		}
	})
	return
}

func (e *encoder) astMapIteration(n *ast.MapIteration) (outID uint64) {
	e.build(&e.instances.AstMapIteration, e.maps.ASTMapIteration, n, &outID, func() *ASTMapIteration {
		return &ASTMapIteration{
			IndexVariable: e.astIdentifier(n.IndexVariable),
			KeyVariable:   e.astIdentifier(n.KeyVariable),
			ValueVariable: e.astIdentifier(n.ValueVariable),
			Map:           e.astNode(n.Map),
			Block:         e.astBlock(n.Block),
		}
	})
	return
}

func (e *encoder) astMember(n *ast.Member) (outID uint64) {
	e.build(&e.instances.AstMember, e.maps.ASTMember, n, &outID, func() *ASTMember {
		return &ASTMember{
			Object: e.astNode(n.Object),
			Name:   e.astIdentifier(n.Name),
		}
	})
	return
}

func (e *encoder) astNamedArg(n *ast.NamedArg) (outID uint64) {
	e.build(&e.instances.AstNamedArg, e.maps.ASTNamedArg, n, &outID, func() *ASTNamedArg {
		return &ASTNamedArg{
			Name:  e.astIdentifier(n.Name),
			Value: e.astNode(n.Value),
		}
	})
	return
}

func (e *encoder) astNode(n ast.Node) (out *ASTNode) {
	if isNil(n) {
		return nil
	}

	defer checkMessageEncodes(out)

	switch n := n.(type) {
	case *ast.Abort:
		return &ASTNode{Ty: &ASTNode_Abort{e.astAbort(n)}}
	case *ast.Assign:
		return &ASTNode{Ty: &ASTNode_Assign{e.astAssign(n)}}
	case *ast.BinaryOp:
		return &ASTNode{Ty: &ASTNode_BinaryOp{e.astBinaryOp(n)}}
	case *ast.Bool:
		return &ASTNode{Ty: &ASTNode_Bool{e.astBool(n)}}
	case *ast.Branch:
		return &ASTNode{Ty: &ASTNode_Branch{e.astBranch(n)}}
	case *ast.Call:
		return &ASTNode{Ty: &ASTNode_Call{e.astCall(n)}}
	case *ast.DeclareLocal:
		return &ASTNode{Ty: &ASTNode_DeclareLocal{e.astDeclareLocal(n)}}
	case *ast.Default:
		return &ASTNode{Ty: &ASTNode_Default{e.astDefault(n)}}
	case *ast.Delete:
		return &ASTNode{Ty: &ASTNode_Delete{e.astDelete(n)}}
	case *ast.Fence:
		return &ASTNode{Ty: &ASTNode_Fence{e.astFence(n)}}
	case *ast.Generic:
		return &ASTNode{Ty: &ASTNode_Generic{e.astGeneric(n)}}
	case *ast.Group:
		return &ASTNode{Ty: &ASTNode_Group{e.astGroup(n)}}
	case *ast.Identifier:
		return &ASTNode{Ty: &ASTNode_Identifier{e.astIdentifier(n)}}
	case *ast.Index:
		return &ASTNode{Ty: &ASTNode_Index{e.astIndex(n)}}
	case *ast.IndexedType:
		return &ASTNode{Ty: &ASTNode_IndexedType{e.astIndexedType(n)}}
	case *ast.Iteration:
		return &ASTNode{Ty: &ASTNode_Iteration{e.astIteration(n)}}
	case *ast.MapIteration:
		return &ASTNode{Ty: &ASTNode_MapIteration{e.astMapIteration(n)}}
	case *ast.Member:
		return &ASTNode{Ty: &ASTNode_Member{e.astMember(n)}}
	case *ast.NamedArg:
		return &ASTNode{Ty: &ASTNode_NamedArg{e.astNamedArg(n)}}
	case *ast.Null:
		return &ASTNode{Ty: &ASTNode_Null{e.astNull(n)}}
	case *ast.Number:
		return &ASTNode{Ty: &ASTNode_Number{e.astNumber(n)}}
	case *ast.PointerType:
		return &ASTNode{Ty: &ASTNode_PointerType{e.astPointerType(n)}}
	case *ast.PreConst:
		return &ASTNode{Ty: &ASTNode_PreConst{e.astPreConst(n)}}
	case *ast.Return:
		return &ASTNode{Ty: &ASTNode_Return{e.astReturn(n)}}
	case *ast.String:
		return &ASTNode{Ty: &ASTNode_String_{e.astString(n)}}
	case *ast.Switch:
		return &ASTNode{Ty: &ASTNode_Switch{e.astSwitch(n)}}
	case *ast.UnaryOp:
		return &ASTNode{Ty: &ASTNode_UnaryOp{e.astUnaryOp(n)}}
	case *ast.Unknown:
		return &ASTNode{Ty: &ASTNode_Unknown{e.astUnknown(n)}}
	default:
		panic(fmt.Errorf("Unhandled AST node type: %T", n))
	}
}

func (e *encoder) astNull(n *ast.Null) (outID uint64) {
	e.build(&e.instances.AstNull, e.maps.ASTNull, n, &outID, func() *ASTNull {
		return &ASTNull{}
	})
	return
}

func (e *encoder) astNumber(n *ast.Number) (outID uint64) {
	e.build(&e.instances.AstNumber, e.maps.ASTNumber, n, &outID, func() *ASTNumber {
		return &ASTNumber{Value: e.str(n.Value)}
	})
	return
}

func (e *encoder) astParameter(n *ast.Parameter) (outID uint64) {
	e.build(&e.instances.AstParameter, e.maps.ASTParameter, n, &outID, func() *ASTParameter {
		return &ASTParameter{
			Annotations: e.astAnnotations(n.Annotations),
			This:        n.This,
			Type:        e.astNode(n.Type),
			Name:        e.astIdentifier(n.Name),
		}
	})
	return
}

func (e *encoder) astPointerType(n *ast.PointerType) (outID uint64) {
	e.build(&e.instances.AstPointerType, e.maps.ASTPointerType, n, &outID, func() *ASTPointerType {
		return &ASTPointerType{
			To:    e.astNode(n.To),
			Const: n.Const,
		}
	})
	return
}

func (e *encoder) astPreConst(n *ast.PreConst) (outID uint64) {
	e.build(&e.instances.AstPreConst, e.maps.ASTPreConst, n, &outID, func() *ASTPreConst {
		return &ASTPreConst{
			Type: e.astNode(n.Type),
		}
	})
	return
}

func (e *encoder) astPseudonym(n *ast.Pseudonym) (outID uint64) {
	e.build(&e.instances.AstPseudonym, e.maps.ASTPseudonym, n, &outID, func() *ASTPseudonym {
		return &ASTPseudonym{
			Annotations: e.astAnnotations(n.Annotations),
			Name:        e.astIdentifier(n.Name),
			To:          e.astNode(n.To),
		}
	})
	return
}

func (e *encoder) astReturn(n *ast.Return) (outID uint64) {
	e.build(&e.instances.AstReturn, e.maps.ASTReturn, n, &outID, func() *ASTReturn {
		return &ASTReturn{Value: e.astNode(n.Value)}
	})
	return
}

func (e *encoder) astString(n *ast.String) (outID uint64) {
	e.build(&e.instances.AstString, e.maps.ASTString, n, &outID, func() *ASTString {
		return &ASTString{Value: e.str(n.Value)}
	})
	return
}

func (e *encoder) astSwitch(n *ast.Switch) (outID uint64) {
	e.build(&e.instances.AstSwitch, e.maps.ASTSwitch, n, &outID, func() *ASTSwitch {
		p := &ASTSwitch{
			Value:   e.astNode(n.Value),
			Default: e.astDefault(n.Default),
		}
		foreach(n.Cases, e.astCase, &p.Cases)
		return p
	})
	return
}

func (e *encoder) astUnaryOp(n *ast.UnaryOp) (outID uint64) {
	e.build(&e.instances.AstUnaryOp, e.maps.ASTUnaryOp, n, &outID, func() *ASTUnaryOp {
		return &ASTUnaryOp{
			Operator:   e.str(n.Operator),
			Expression: e.astNode(n.Expression),
		}
	})
	return
}

func (e *encoder) astUnknown(n *ast.Unknown) (outID uint64) {
	e.build(&e.instances.AstUnknown, e.maps.ASTUnknown, n, &outID, func() *ASTUnknown {
		return &ASTUnknown{}
	})
	return
}

// build calls cb to encode an object instance to a proto.
// The return proto of cb is cached into i so each instance is only built once.
//
// i is a pointer to the proto instances slice (&e.instances.Foo)
// m is a pointer to the encoder instances map (e.maps.Foo)
// n is the object to encode
// out is a pointer to the output identifier
// cb is a func with the signature: func() *OutputType
func (e *encoder) build(i, m, n, out interface{}, cb interface{}) {
	iV := reflect.ValueOf(i)
	mV := reflect.ValueOf(m)
	nV := reflect.ValueOf(n)
	outV := reflect.ValueOf(out)
	cbV := reflect.ValueOf(cb)

	if !nV.IsValid() ||
		((nV.Kind() == reflect.Ptr || nV.Kind() == reflect.Interface) && nV.IsNil()) {
		// n is nil.
		// Don't assign to out to keep it at default 0 (representing nil)
		return
	}

	if existing := mV.MapIndex(nV); existing.IsValid() {
		// Already built this object.
		outV.Elem().Set(existing)
		return
	}

	idx := iV.Elem().Len()

	// The ID is the index + 1. And ID of 0 represents a nil.
	idV := reflect.ValueOf(uint64(idx + 1))
	outV.Elem().Set(idV)
	mV.SetMapIndex(nV, idV)

	// Grow the slice to hold the new entry. We do this before the call to cb
	// as this might call build() again, reordering the elements of the i slice.
	iV.Elem().Set(reflect.Append(iV.Elem(), reflect.Zero(iV.Type().Elem().Elem())))

	// Build
	pV := cbV.Call([]reflect.Value{})[0]

	// Assign the output proto value to the content instances for the given id.
	iV.Elem().Index(idx).Set(pV)

	checkMessageEncodes(pV.Interface())
}

// Help debug `proto: bapi.Content: illegal tag 0 (wire type 0)` errors
func checkMessageEncodes(v interface{}) {
	if false {
		if isNil(v) {
			return
		}
		if msg, ok := v.(proto.Message); ok {
			if _, err := proto.Marshal(msg); err != nil {
				panic(err)
			}
		}
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface {
		return r.IsNil()
	}
	return false
}
