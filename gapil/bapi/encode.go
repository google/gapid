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
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

type encoder struct {
	instances *Instances
	maps      encoderInstances
}

type encoderInstances struct {
	Abort           map[*semantic.Abort]uint64
	Annotation      map[*semantic.Annotation]uint64
	API             map[*semantic.API]uint64
	ArrayAssign     map[*semantic.ArrayAssign]uint64
	ArrayIndex      map[*semantic.ArrayIndex]uint64
	ArrayInit       map[*semantic.ArrayInitializer]uint64
	Assert          map[*semantic.Assert]uint64
	Assign          map[*semantic.Assign]uint64
	BinaryOp        map[*semantic.BinaryOp]uint64
	BitTest         map[*semantic.BitTest]uint64
	Block           map[*semantic.Block]uint64
	BoolValue       map[semantic.BoolValue]uint64
	Branch          map[*semantic.Branch]uint64
	Call            map[*semantic.Call]uint64
	Callable        map[*semantic.Callable]uint64
	Case            map[*semantic.Case]uint64
	Cast            map[*semantic.Cast]uint64
	Choice          map[*semantic.Choice]uint64
	Class           map[*semantic.Class]uint64
	ClassInit       map[*semantic.ClassInitializer]uint64
	Clone           map[*semantic.Clone]uint64
	Copy            map[*semantic.Copy]uint64
	Create          map[*semantic.Create]uint64
	DeclareLocal    map[*semantic.DeclareLocal]uint64
	Definition      map[*semantic.Definition]uint64
	DefinitionUsage map[*semantic.DefinitionUsage]uint64
	Enum            map[*semantic.Enum]uint64
	EnumEntry       map[*semantic.EnumEntry]uint64
	Expression      map[semantic.Expression]uint64
	Fence           map[*semantic.Fence]uint64
	Field           map[*semantic.Field]uint64
	FieldInit       map[*semantic.FieldInitializer]uint64
	Float32Value    map[semantic.Float32Value]uint64
	Float64Value    map[semantic.Float64Value]uint64
	Function        map[*semantic.Function]uint64
	Global          map[*semantic.Global]uint64
	Ignore          map[*semantic.Ignore]uint64
	Int16Value      map[semantic.Int16Value]uint64
	Int32Value      map[semantic.Int32Value]uint64
	Int64Value      map[semantic.Int64Value]uint64
	Int8Value       map[semantic.Int8Value]uint64
	Iteration       map[*semantic.Iteration]uint64
	Length          map[*semantic.Length]uint64
	Local           map[*semantic.Local]uint64
	Make            map[*semantic.Make]uint64
	Map             map[*semantic.Map]uint64
	MapAssign       map[*semantic.MapAssign]uint64
	MapContains     map[*semantic.MapContains]uint64
	MapIndex        map[*semantic.MapIndex]uint64
	MapIteration    map[*semantic.MapIteration]uint64
	MapRemove       map[*semantic.MapRemove]uint64
	MapClear        map[*semantic.MapClear]uint64
	Member          map[*semantic.Member]uint64
	MessageValue    map[*semantic.MessageValue]uint64
	Null            map[semantic.Null]uint64
	Observed        map[*semantic.Observed]uint64
	Parameter       map[*semantic.Parameter]uint64
	Pointer         map[*semantic.Pointer]uint64
	PointerRange    map[*semantic.PointerRange]uint64
	Print           map[*semantic.Print]uint64
	Pseudonym       map[*semantic.Pseudonym]uint64
	Read            map[*semantic.Read]uint64
	Reference       map[*semantic.Reference]uint64
	Return          map[*semantic.Return]uint64
	Select          map[*semantic.Select]uint64
	Signature       map[*semantic.Signature]uint64
	Slice           map[*semantic.Slice]uint64
	SliceAssign     map[*semantic.SliceAssign]uint64
	SliceIndex      map[*semantic.SliceIndex]uint64
	SliceRange      map[*semantic.SliceRange]uint64
	Statement       map[semantic.Statement]uint64
	StaticArray     map[*semantic.StaticArray]uint64
	StringValue     map[semantic.StringValue]uint64
	Switch          map[*semantic.Switch]uint64
	Uint16Value     map[semantic.Uint16Value]uint64
	Uint32Value     map[semantic.Uint32Value]uint64
	Uint64Value     map[semantic.Uint64Value]uint64
	Uint8Value      map[semantic.Uint8Value]uint64
	UnaryOp         map[*semantic.UnaryOp]uint64
	Unknown         map[*semantic.Unknown]uint64
	Write           map[*semantic.Write]uint64

	ASTAnnotation   map[*ast.Annotation]uint64
	ASTAbort        map[*ast.Abort]uint64
	ASTAPI          map[*ast.API]uint64
	ASTAssign       map[*ast.Assign]uint64
	ASTBinaryOp     map[*ast.BinaryOp]uint64
	ASTBlock        map[*ast.Block]uint64
	ASTBool         map[*ast.Bool]uint64
	ASTBranch       map[*ast.Branch]uint64
	ASTCall         map[*ast.Call]uint64
	ASTCase         map[*ast.Case]uint64
	ASTClass        map[*ast.Class]uint64
	ASTClear        map[*ast.Clear]uint64
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
	ASTImport       map[*ast.Import]uint64
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

	CSTBranch map[*cst.Branch]uint64
	CSTLeaf   map[*cst.Leaf]uint64
	CSTSource map[*cst.Source]uint64

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
func Encode(apis []*semantic.API, mappings *semantic.Mappings) ([]byte, error) {
	e := &encoder{instances: &Instances{}}
	e.maps.build()
	content := &Content{
		Instances: e.instances,
		Apis:      make([]uint64, len(apis)),
	}

	// Serialize the APIs.
	for i, api := range apis {
		content.Apis[i] = e.api(api)
	}

	// Serialize the mappings.
	content.Mappings = e.mappings(mappings)

	data, err := proto.Marshal(content)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (e *encoder) mappings(m *semantic.Mappings) *Mappings {
	out := Mappings{}

	for sem, asts := range m.SemanticToAST {
		for _, ast := range asts {
			out.SemToAst = append(out.SemToAst, &SemanticToAST{
				Sem: e.node(sem),
				Ast: e.astNode(ast),
			})
		}
	}

	for ast, cst := range m.AST.ASTToCST {
		out.AstToCst = append(out.AstToCst, &ASTToCST{
			Ast: e.astNode(ast),
			Cst: e.cstNode(cst),
		})
	}

	return &out
}

func (e *encoder) node(n semantic.Node) *Node {
	switch n := n.(type) {
	case nil:
		return nil
	case *semantic.Abort:
		return &Node{Ty: &Node_Abort{Abort: e.abort(n)}}
	case *semantic.Annotation:
		return &Node{Ty: &Node_Annotation{Annotation: e.annotation(n)}}
	case *semantic.API:
		return &Node{Ty: &Node_Api{Api: e.api(n)}}
	case *semantic.ArrayAssign:
		return &Node{Ty: &Node_ArrayAssign{ArrayAssign: e.arrayAssign(n)}}
	case *semantic.ArrayIndex:
		return &Node{Ty: &Node_ArrayIndex{ArrayIndex: e.arrayIndex(n)}}
	case *semantic.ArrayInitializer:
		return &Node{Ty: &Node_ArrayInit{ArrayInit: e.arrayInit(n)}}
	case *semantic.Assert:
		return &Node{Ty: &Node_Assert{Assert: e.assert(n)}}
	case *semantic.Assign:
		return &Node{Ty: &Node_Assign{Assign: e.assign(n)}}
	case *semantic.BinaryOp:
		return &Node{Ty: &Node_BinaryOp{BinaryOp: e.binaryOp(n)}}
	case *semantic.BitTest:
		return &Node{Ty: &Node_BitTest{BitTest: e.bitTest(n)}}
	case *semantic.Block:
		return &Node{Ty: &Node_Block{Block: e.block(n)}}
	case semantic.BoolValue:
		return &Node{Ty: &Node_BoolValue{BoolValue: e.boolValue(n)}}
	case *semantic.Branch:
		return &Node{Ty: &Node_Branch{Branch: e.branch(n)}}
	case *semantic.Builtin:
		return &Node{Ty: &Node_Builtin{Builtin: e.builtin(n)}}
	case *semantic.Call:
		return &Node{Ty: &Node_Call{Call: e.call(n)}}
	case *semantic.Callable:
		return &Node{Ty: &Node_Callable{Callable: e.callable(n)}}
	case *semantic.Case:
		return &Node{Ty: &Node_Case{Case: e.case_(n)}}
	case *semantic.Cast:
		return &Node{Ty: &Node_Cast{Cast: e.cast(n)}}
	case *semantic.Choice:
		return &Node{Ty: &Node_Choice{Choice: e.choice(n)}}
	case *semantic.Class:
		return &Node{Ty: &Node_Class{Class: e.class(n)}}
	case *semantic.ClassInitializer:
		return &Node{Ty: &Node_ClassInit{ClassInit: e.classInit(n)}}
	case *semantic.Clone:
		return &Node{Ty: &Node_Clone{Clone: e.clone(n)}}
	case *semantic.Copy:
		return &Node{Ty: &Node_Copy{Copy: e.copy(n)}}
	case *semantic.Create:
		return &Node{Ty: &Node_Create{Create: e.create(n)}}
	case *semantic.DeclareLocal:
		return &Node{Ty: &Node_DeclareLocal{DeclareLocal: e.declareLocal(n)}}
	case *semantic.Definition:
		return &Node{Ty: &Node_Definition{Definition: e.definition(n)}}
	case *semantic.Enum:
		return &Node{Ty: &Node_Enum{Enum: e.enum(n)}}
	case *semantic.EnumEntry:
		return &Node{Ty: &Node_EnumEntry{EnumEntry: e.enumEntry(n)}}
	case semantic.Expression:
		return &Node{Ty: &Node_Expression{Expression: e.expr(n)}}
	case *semantic.Fence:
		return &Node{Ty: &Node_Fence{Fence: e.fence(n)}}
	case *semantic.Field:
		return &Node{Ty: &Node_Field{Field: e.field(n)}}
	case *semantic.FieldInitializer:
		return &Node{Ty: &Node_FieldInit{FieldInit: e.fieldInit(n)}}
	case semantic.Float32Value:
		return &Node{Ty: &Node_Float32Value{Float32Value: e.float32Value(n)}}
	case semantic.Float64Value:
		return &Node{Ty: &Node_Float64Value{Float64Value: e.float64Value(n)}}
	case *semantic.Function:
		return &Node{Ty: &Node_Function{Function: e.function(n)}}
	case *semantic.Global:
		return &Node{Ty: &Node_Global{Global: e.global(n)}}
	case *semantic.Ignore:
		return &Node{Ty: &Node_Ignore{Ignore: e.ignore(n)}}
	case semantic.Int16Value:
		return &Node{Ty: &Node_Int16Value{Int16Value: e.int16Value(n)}}
	case semantic.Int32Value:
		return &Node{Ty: &Node_Int32Value{Int32Value: e.int32Value(n)}}
	case semantic.Int64Value:
		return &Node{Ty: &Node_Int64Value{Int64Value: e.int64Value(n)}}
	case semantic.Int8Value:
		return &Node{Ty: &Node_Int8Value{Int8Value: e.int8Value(n)}}
	case *semantic.Iteration:
		return &Node{Ty: &Node_Iteration{Iteration: e.iteration(n)}}
	case *semantic.Length:
		return &Node{Ty: &Node_Length{Length: e.length(n)}}
	case *semantic.Local:
		return &Node{Ty: &Node_Local{Local: e.local(n)}}
	case *semantic.Make:
		return &Node{Ty: &Node_Make{Make: e.make(n)}}
	case *semantic.Map:
		return &Node{Ty: &Node_Map{Map: e.map_(n)}}
	case *semantic.MapAssign:
		return &Node{Ty: &Node_MapAssign{MapAssign: e.mapAssign(n)}}
	case *semantic.MapContains:
		return &Node{Ty: &Node_MapContains{MapContains: e.mapContains(n)}}
	case *semantic.MapIndex:
		return &Node{Ty: &Node_MapIndex{MapIndex: e.mapIndex(n)}}
	case *semantic.MapIteration:
		return &Node{Ty: &Node_MapIteration{MapIteration: e.mapIteration(n)}}
	case *semantic.MapRemove:
		return &Node{Ty: &Node_MapRemove{MapRemove: e.mapRemove(n)}}
	case *semantic.MapClear:
		return &Node{Ty: &Node_MapClear{MapClear: e.mapClear(n)}}
	case *semantic.Member:
		return &Node{Ty: &Node_Member{Member: e.member(n)}}
	case *semantic.MessageValue:
		return &Node{Ty: &Node_MessageValue{MessageValue: e.messageValue(n)}}
	case semantic.Null:
		return &Node{Ty: &Node_Null{Null: e.null(n)}}
	case *semantic.Observed:
		return &Node{Ty: &Node_Observed{Observed: e.observed(n)}}
	case *semantic.Parameter:
		return &Node{Ty: &Node_Parameter{Parameter: e.param(n)}}
	case *semantic.Pointer:
		return &Node{Ty: &Node_Pointer{Pointer: e.pointer(n)}}
	case *semantic.PointerRange:
		return &Node{Ty: &Node_PointerRange{PointerRange: e.pointerRange(n)}}
	case *semantic.Pseudonym:
		return &Node{Ty: &Node_Pseudonym{Pseudonym: e.pseudonym(n)}}
	case *semantic.Print:
		return &Node{Ty: &Node_Print{Print: e.print(n)}}
	case *semantic.Read:
		return &Node{Ty: &Node_Read{Read: e.read(n)}}
	case *semantic.Reference:
		return &Node{Ty: &Node_Reference{Reference: e.reference(n)}}
	case *semantic.Return:
		return &Node{Ty: &Node_Return{Return: e.return_(n)}}
	case *semantic.Select:
		return &Node{Ty: &Node_Select{Select: e.select_(n)}}
	case *semantic.Signature:
		return &Node{Ty: &Node_Signature{Signature: e.signature(n)}}
	case *semantic.Slice:
		return &Node{Ty: &Node_Slice{Slice: e.slice(n)}}
	case *semantic.SliceAssign:
		return &Node{Ty: &Node_SliceAssign{SliceAssign: e.sliceAssign(n)}}
	case *semantic.SliceIndex:
		return &Node{Ty: &Node_SliceIndex{SliceIndex: e.sliceIndex(n)}}
	case *semantic.SliceRange:
		return &Node{Ty: &Node_SliceRange{SliceRange: e.sliceRange(n)}}
	case semantic.Statement:
		return &Node{Ty: &Node_Statement{Statement: e.stat(n)}}
	case *semantic.StaticArray:
		return &Node{Ty: &Node_StaticArray{StaticArray: e.staticArray(n)}}
	case semantic.StringValue:
		return &Node{Ty: &Node_StringValue{StringValue: e.stringValue(n)}}
	case *semantic.Switch:
		return &Node{Ty: &Node_Switch{Switch: e.switch_(n)}}
	case semantic.Uint16Value:
		return &Node{Ty: &Node_Uint16Value{Uint16Value: e.uint16Value(n)}}
	case semantic.Uint32Value:
		return &Node{Ty: &Node_Uint32Value{Uint32Value: e.uint32Value(n)}}
	case semantic.Uint64Value:
		return &Node{Ty: &Node_Uint64Value{Uint64Value: e.uint64Value(n)}}
	case semantic.Uint8Value:
		return &Node{Ty: &Node_Uint8Value{Uint8Value: e.uint8Value(n)}}
	case *semantic.UnaryOp:
		return &Node{Ty: &Node_UnaryOp{UnaryOp: e.unaryOp(n)}}
	case *semantic.Unknown:
		return &Node{Ty: &Node_Unknown{Unknown: e.unknown(n)}}
	case *semantic.Write:
		return &Node{Ty: &Node_Write{Write: e.write(n)}}
	default:
		panic(fmt.Errorf("Unhandled node type %T", n))
	}
}

func (e *encoder) abort(n *semantic.Abort) (outID uint64) {
	e.build(&e.instances.Abort, e.maps.Abort, n, &outID, func() *Abort {
		return &Abort{
			Ast:       e.astAbort(n.AST),
			Function:  e.function(n.Function),
			Statement: e.stat(n.Statement),
		}
	})
	return
}

func (e *encoder) annotation(n *semantic.Annotation) (outID uint64) {
	e.build(&e.instances.Annotation, e.maps.Annotation, n, &outID, func() *Annotation {
		p := &Annotation{
			Ast:  e.astAnnotation(n.AST),
			Name: e.str(n.Name()),
		}
		foreach(n.Arguments, e.expr, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) annotations(n semantic.Annotations) *Annotations {
	p := &Annotations{}
	foreach(n, e.annotation, &p.Annotations)
	return p
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
		foreach(n.StaticArrays, e.staticArray, &p.StaticArrays)
		foreach(n.Maps, e.map_, &p.Maps)
		foreach(n.Pointers, e.pointer, &p.Pointers)
		foreach(n.Slices, e.slice, &p.Slices)
		foreach(n.References, e.reference, &p.References)
		foreach(n.Signatures, e.signature, &p.Signatures)
		p.Index = uint32(n.Index)
		return p
	})

	return
}

func (e *encoder) arrayAssign(n *semantic.ArrayAssign) (outID uint64) {
	e.build(&e.instances.ArrayAssign, e.maps.ArrayAssign, n, &outID, func() *ArrayAssign {
		return &ArrayAssign{
			Ast:      e.astAssign(n.AST),
			To:       e.expr(n.To),
			Operator: e.str(n.Operator),
			Value:    e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) arrayIndex(n *semantic.ArrayIndex) (outID uint64) {
	e.build(&e.instances.ArrayIndex, e.maps.ArrayIndex, n, &outID, func() *ArrayIndex {
		return &ArrayIndex{
			Ast:   e.astIndex(n.AST),
			Type:  e.staticArray(n.Type),
			Array: e.expr(n.Array),
			Index: e.expr(n.Index),
		}
	})
	return
}

func (e *encoder) arrayInit(n *semantic.ArrayInitializer) (outID uint64) {
	e.build(&e.instances.ArrayInit, e.maps.ArrayInit, n, &outID, func() *ArrayInitializer {
		p := &ArrayInitializer{
			Ast:   e.astCall(n.AST),
			Array: e.ty(n.Array),
		}
		foreach(n.Values, e.expr, &p.Values)
		return p
	})
	return
}

func (e *encoder) assert(n *semantic.Assert) (outID uint64) {
	e.build(&e.instances.Assert, e.maps.Assert, n, &outID, func() *Assert {
		return &Assert{
			Ast:       e.astCall(n.AST),
			Condition: e.expr(n.Condition),
			Message:   e.str(n.Message),
		}
	})
	return
}

func (e *encoder) assign(n *semantic.Assign) (outID uint64) {
	e.build(&e.instances.Assign, e.maps.Assign, n, &outID, func() *Assign {
		return &Assign{
			Ast:      e.astAssign(n.AST),
			Lhs:      e.expr(n.LHS),
			Operator: e.str(n.Operator),
			Rhs:      e.expr(n.RHS),
		}
	})
	return
}

func (e *encoder) binaryOp(n *semantic.BinaryOp) (outID uint64) {
	e.build(&e.instances.BinaryOp, e.maps.BinaryOp, n, &outID, func() *BinaryOp {
		return &BinaryOp{
			Ast:      e.astBinaryOp(n.AST),
			Type:     e.ty(n.Type),
			Lhs:      e.expr(n.LHS),
			Operator: e.str(n.Operator),
			Rhs:      e.expr(n.RHS)}
	})
	return
}

func (e *encoder) bitTest(n *semantic.BitTest) (outID uint64) {
	e.build(&e.instances.BitTest, e.maps.BitTest, n, &outID, func() *BitTest {
		return &BitTest{
			Ast:      e.astBinaryOp(n.AST),
			Bitfield: e.expr(n.Bitfield),
			Bits:     e.expr(n.Bits),
		}
	})
	return
}

func (e *encoder) boolValue(n semantic.BoolValue) (outID uint64) {
	e.build(&e.instances.BoolValue, e.maps.BoolValue, n, &outID, func() *BoolValue {
		return &BoolValue{
			Value: bool(n),
		}
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

func (e *encoder) branch(n *semantic.Branch) (outID uint64) {
	e.build(&e.instances.Branch, e.maps.Branch, n, &outID, func() *Branch {
		return &Branch{
			Ast:       e.astBranch(n.AST),
			Condition: e.expr(n.Condition),
			True:      e.block(n.True),
			False:     e.block(n.False),
		}
	})
	return
}

func (e *encoder) builtin(n *semantic.Builtin) Builtin {
	switch n {
	case semantic.VoidType:
		return Builtin_VoidType
	case semantic.AnyType:
		return Builtin_AnyType
	case semantic.StringType:
		return Builtin_StringType
	case semantic.MessageType:
		return Builtin_MessageType
	case semantic.BoolType:
		return Builtin_BoolType
	case semantic.IntType:
		return Builtin_IntType
	case semantic.UintType:
		return Builtin_UintType
	case semantic.SizeType:
		return Builtin_SizeType
	case semantic.CharType:
		return Builtin_CharType
	case semantic.Int8Type:
		return Builtin_Int8Type
	case semantic.Uint8Type:
		return Builtin_Uint8Type
	case semantic.Int16Type:
		return Builtin_Int16Type
	case semantic.Uint16Type:
		return Builtin_Uint16Type
	case semantic.Int32Type:
		return Builtin_Int32Type
	case semantic.Uint32Type:
		return Builtin_Uint32Type
	case semantic.Int64Type:
		return Builtin_Int64Type
	case semantic.Uint64Type:
		return Builtin_Uint64Type
	case semantic.Float32Type:
		return Builtin_Float32Type
	case semantic.Float64Type:
		return Builtin_Float64Type
	default:
		panic(fmt.Errorf("Unhandled builtin type %v", n))
	}
}

func (e *encoder) call(n *semantic.Call) (outID uint64) {
	e.build(&e.instances.Call, e.maps.Call, n, &outID, func() *Call {
		p := &Call{
			Ast:    e.astCall(n.AST),
			Target: e.callable(n.Target),
			Type:   e.ty(n.Type),
		}
		foreach(n.Arguments, e.expr, &p.Arguments)
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

func (e *encoder) case_(n *semantic.Case) (outID uint64) {
	e.build(&e.instances.Case, e.maps.Case, n, &outID, func() *Case {
		p := &Case{
			Ast:         e.astCase(n.AST),
			Annotations: e.annotations(n.Annotations),
			Block:       e.block(n.Block),
		}
		foreach(n.Conditions, e.expr, &p.Conditions)
		return p
	})
	return
}

func (e *encoder) cast(n *semantic.Cast) (outID uint64) {
	e.build(&e.instances.Cast, e.maps.Cast, n, &outID, func() *Cast {
		return &Cast{
			Ast:    e.astCall(n.AST),
			Object: e.expr(n.Object),
			Type:   e.ty(n.Type),
		}
	})
	return
}

func (e *encoder) choice(n *semantic.Choice) (outID uint64) {
	e.build(&e.instances.Choice, e.maps.Choice, n, &outID, func() *Choice {
		p := &Choice{
			Ast:         e.astCase(n.AST),
			Annotations: e.annotations(n.Annotations),
			Expression:  e.expr(n.Expression),
		}
		foreach(n.Conditions, e.expr, &p.Conditions)
		return p
	})
	return
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

func (e *encoder) classInit(n *semantic.ClassInitializer) (outID uint64) {
	e.build(&e.instances.ClassInit, e.maps.ClassInit, n, &outID, func() *ClassInitializer {
		p := &ClassInitializer{
			Ast:   e.astCall(n.AST),
			Class: e.class(n.Class),
		}
		foreach(n.Fields, e.fieldInit, &p.Fields)
		return p
	})
	return
}

func (e *encoder) clone(n *semantic.Clone) (outID uint64) {
	e.build(&e.instances.Clone, e.maps.Clone, n, &outID, func() *Clone {
		return &Clone{
			Ast:   e.astCall(n.AST),
			Slice: e.expr(n.Slice),
			Type:  e.slice(n.Type),
		}
	})
	return
}

func (e *encoder) copy(n *semantic.Copy) (outID uint64) {
	e.build(&e.instances.Copy, e.maps.Copy, n, &outID, func() *Copy {
		return &Copy{
			Ast: e.astCall(n.AST),
			Src: e.expr(n.Src),
			Dst: e.expr(n.Dst),
		}
	})
	return
}

func (e *encoder) create(n *semantic.Create) (outID uint64) {
	e.build(&e.instances.Create, e.maps.Create, n, &outID, func() *Create {
		return &Create{
			Ast:         e.astCall(n.AST),
			Type:        e.reference(n.Type),
			Initializer: e.classInit(n.Initializer),
		}
	})
	return
}

func (e *encoder) declareLocal(n *semantic.DeclareLocal) (outID uint64) {
	e.build(&e.instances.DeclareLocal, e.maps.DeclareLocal, n, &outID, func() *DeclareLocal {
		return &DeclareLocal{
			Ast:   e.astDeclareLocal(n.AST),
			Local: e.local(n.Local),
		}
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

func (e *encoder) definitionUsage(n *semantic.DefinitionUsage) (outID uint64) {
	e.build(&e.instances.DefinitionUsage, e.maps.DefinitionUsage, n, &outID, func() *DefinitionUsage {
		return &DefinitionUsage{
			Definition: e.definition(n.Definition),
			Expression: e.expr(n.Expression),
		}
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
		n.SortMembers()
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
			p.Ty = &Expression_ArrayIndex{e.arrayIndex(n)}
		case *semantic.ArrayInitializer:
			p.Ty = &Expression_ArrayInitializer{e.arrayInit(n)}
		case *semantic.BinaryOp:
			p.Ty = &Expression_BinaryOp{e.binaryOp(n)}
		case *semantic.BitTest:
			p.Ty = &Expression_BitTest{e.bitTest(n)}
		case semantic.BoolValue:
			p.Ty = &Expression_BoolValue{e.boolValue(n)}
		case *semantic.Call:
			p.Ty = &Expression_Call{e.call(n)}
		case *semantic.Cast:
			p.Ty = &Expression_Cast{e.cast(n)}
		case *semantic.ClassInitializer:
			p.Ty = &Expression_ClassInit{ClassInit: e.classInit(n)}
		case *semantic.Clone:
			p.Ty = &Expression_Clone{Clone: e.clone(n)}
		case *semantic.Create:
			p.Ty = &Expression_Create{Create: e.create(n)}
		case *semantic.Definition:
			p.Ty = &Expression_Definition{e.definition(n)}
		case *semantic.DefinitionUsage:
			p.Ty = &Expression_Definition{e.definitionUsage(n)}
		case *semantic.EnumEntry:
			p.Ty = &Expression_EnumEntry{e.enumEntry(n)}
		case *semantic.Field:
			p.Ty = &Expression_Field{e.field(n)}
		case semantic.Float32Value:
			p.Ty = &Expression_Float32Value{e.float32Value(n)}
		case semantic.Float64Value:
			p.Ty = &Expression_Float64Value{e.float64Value(n)}
		case *semantic.Global:
			p.Ty = &Expression_Global{e.global(n)}
		case *semantic.Ignore:
			p.Ty = &Expression_Ignore{e.ignore(n)}
		case semantic.Int8Value:
			p.Ty = &Expression_Int8Value{e.int8Value(n)}
		case semantic.Int16Value:
			p.Ty = &Expression_Int16Value{e.int16Value(n)}
		case semantic.Int32Value:
			p.Ty = &Expression_Int32Value{e.int32Value(n)}
		case semantic.Int64Value:
			p.Ty = &Expression_Int64Value{e.int64Value(n)}
		case *semantic.Length:
			p.Ty = &Expression_Length{e.length(n)}
		case *semantic.Local:
			p.Ty = &Expression_Local{e.local(n)}
		case *semantic.Make:
			p.Ty = &Expression_Make{e.make(n)}
		case *semantic.MapContains:
			p.Ty = &Expression_MapContains{e.mapContains(n)}
		case *semantic.MapIndex:
			p.Ty = &Expression_MapIndex{e.mapIndex(n)}
		case *semantic.Member:
			p.Ty = &Expression_Member{e.member(n)}
		case *semantic.MessageValue:
			p.Ty = &Expression_MessageValue{e.messageValue(n)}
		case semantic.Null:
			p.Ty = &Expression_Null{e.null(n)}
		case *semantic.Observed:
			p.Ty = &Expression_Observed{e.observed(n)}
		case *semantic.Parameter:
			p.Ty = &Expression_Parameter{e.param(n)}
		case *semantic.PointerRange:
			p.Ty = &Expression_PointerRange{e.pointerRange(n)}
		case *semantic.Select:
			p.Ty = &Expression_Select{e.select_(n)}
		case *semantic.SliceIndex:
			p.Ty = &Expression_SliceIndex{e.sliceIndex(n)}
		case *semantic.SliceRange:
			p.Ty = &Expression_SliceRange{e.sliceRange(n)}
		case semantic.StringValue:
			p.Ty = &Expression_StringValue{e.stringValue(n)}
		case semantic.Uint8Value:
			p.Ty = &Expression_Uint8Value{e.uint8Value(n)}
		case semantic.Uint16Value:
			p.Ty = &Expression_Uint16Value{e.uint16Value(n)}
		case semantic.Uint32Value:
			p.Ty = &Expression_Uint32Value{e.uint32Value(n)}
		case semantic.Uint64Value:
			p.Ty = &Expression_Uint64Value{e.uint64Value(n)}
		case *semantic.UnaryOp:
			p.Ty = &Expression_UnaryOp{e.unaryOp(n)}
		case *semantic.Unknown:
			p.Ty = &Expression_Unknown{e.unknown(n)}
		default:
			panic(fmt.Errorf("Unhandled expression type %T", n))
		}
		return p
	})
	return
}

func (e *encoder) fence(n *semantic.Fence) (outID uint64) {
	e.build(&e.instances.Fence, e.maps.Fence, n, &outID, func() *Fence {
		return &Fence{
			Ast:       e.astFence(n.AST),
			Statement: e.stat(n.Statement),
			Explicit:  n.Explicit,
		}
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

func (e *encoder) fieldInit(n *semantic.FieldInitializer) (outID uint64) {
	e.build(&e.instances.FieldInit, e.maps.FieldInit, n, &outID, func() *FieldInitializer {
		return &FieldInitializer{
			Ast:   e.astNode(n.AST),
			Field: e.field(n.Field),
			Value: e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) float32Value(n semantic.Float32Value) (outID uint64) {
	e.build(&e.instances.Float32Value, e.maps.Float32Value, n, &outID, func() *Float32Value {
		return &Float32Value{Value: float32(n)}
	})
	return
}

func (e *encoder) float64Value(n semantic.Float64Value) (outID uint64) {
	e.build(&e.instances.Float64Value, e.maps.Float64Value, n, &outID, func() *Float64Value {
		return &Float64Value{Value: float64(n)}
	})
	return
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

func (e *encoder) ignore(n *semantic.Ignore) (outID uint64) {
	e.build(&e.instances.Ignore, e.maps.Ignore, n, &outID, func() *Ignore {
		return &Ignore{
			Ast: e.astNode(n.AST),
		}
	})
	return
}

func (e *encoder) iteration(n *semantic.Iteration) (outID uint64) {
	e.build(&e.instances.Iteration, e.maps.Iteration, n, &outID, func() *Iteration {
		return &Iteration{
			Ast:      e.astIteration(n.AST),
			Iterator: e.local(n.Iterator),
			From:     e.expr(n.From),
			To:       e.expr(n.To),
			Block:    e.block(n.Block),
		}
	})
	return
}

func (e *encoder) int8Value(n semantic.Int8Value) (outID uint64) {
	e.build(&e.instances.Int8Value, e.maps.Int8Value, n, &outID, func() *Int8Value {
		return &Int8Value{
			Value: int32(n),
		}
	})
	return
}

func (e *encoder) int16Value(n semantic.Int16Value) (outID uint64) {
	e.build(&e.instances.Int16Value, e.maps.Int16Value, n, &outID, func() *Int16Value {
		return &Int16Value{
			Value: int32(n),
		}
	})
	return
}

func (e *encoder) int32Value(n semantic.Int32Value) (outID uint64) {
	e.build(&e.instances.Int32Value, e.maps.Int32Value, n, &outID, func() *Int32Value {
		return &Int32Value{
			Value: int32(n),
		}
	})
	return
}

func (e *encoder) int64Value(n semantic.Int64Value) (outID uint64) {
	e.build(&e.instances.Int64Value, e.maps.Int64Value, n, &outID, func() *Int64Value {
		return &Int64Value{
			Value: int64(n),
		}
	})
	return
}

func (e *encoder) length(n *semantic.Length) (outID uint64) {
	e.build(&e.instances.Length, e.maps.Length, n, &outID, func() *Length {
		return &Length{
			Ast:    e.astCall(n.AST),
			Object: e.expr(n.Object),
			Type:   e.ty(n.Type),
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

func (e *encoder) make(n *semantic.Make) (outID uint64) {
	e.build(&e.instances.Make, e.maps.Make, n, &outID, func() *Make {
		return &Make{
			Ast:  e.astCall(n.AST),
			Type: e.slice(n.Type),
			Size: e.expr(n.Size),
		}
	})
	return
}

func (e *encoder) map_(n *semantic.Map) (outID uint64) {
	e.build(&e.instances.Map, e.maps.Map, n, &outID, func() *Map {
		n.SortMembers()
		return &Map{
			Owner:     e.node(n.Owner()),
			Name:      e.str(n.Name()),
			KeyType:   e.ty(n.KeyType),
			ValueType: e.ty(n.ValueType),
			Dense:     n.Dense,
		}
	})
	return
}

func (e *encoder) mapAssign(n *semantic.MapAssign) (outID uint64) {
	e.build(&e.instances.MapAssign, e.maps.MapAssign, n, &outID, func() *MapAssign {
		return &MapAssign{
			Ast:      e.astAssign(n.AST),
			To:       e.expr(n.To),
			Operator: e.str(n.Operator),
			Value:    e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) mapContains(n *semantic.MapContains) (outID uint64) {
	e.build(&e.instances.MapContains, e.maps.MapContains, n, &outID, func() *MapContains {
		return &MapContains{
			Ast:  e.astBinaryOp(n.AST),
			Type: e.map_(n.Type),
			Map:  e.expr(n.Map),
			Key:  e.expr(n.Key),
		}
	})
	return
}

func (e *encoder) mapIndex(n *semantic.MapIndex) (outID uint64) {
	e.build(&e.instances.MapIndex, e.maps.MapIndex, n, &outID, func() *MapIndex {
		return &MapIndex{
			Ast:   e.astIndex(n.AST),
			Type:  e.map_(n.Type),
			Map:   e.expr(n.Map),
			Index: e.expr(n.Index),
		}
	})
	return
}

func (e *encoder) mapIteration(n *semantic.MapIteration) (outID uint64) {
	e.build(&e.instances.MapIteration, e.maps.MapIteration, n, &outID, func() *MapIteration {
		return &MapIteration{
			Ast:           e.astMapIteration(n.AST),
			IndexIterator: e.local(n.IndexIterator),
			KeyIterator:   e.local(n.KeyIterator),
			ValueIterator: e.local(n.ValueIterator),
			Map:           e.expr(n.Map),
			Block:         e.block(n.Block),
		}
	})
	return
}

func (e *encoder) mapRemove(n *semantic.MapRemove) (outID uint64) {
	e.build(&e.instances.MapRemove, e.maps.MapRemove, n, &outID, func() *MapRemove {
		return &MapRemove{
			Ast:  e.astDelete(n.AST),
			Type: e.map_(n.Type),
			Map:  e.expr(n.Map),
			Key:  e.expr(n.Key),
		}
	})
	return
}

func (e *encoder) mapClear(n *semantic.MapClear) (outID uint64) {
	e.build(&e.instances.MapClear, e.maps.MapClear, n, &outID, func() *MapClear {
		return &MapClear{
			Ast:  e.astClear(n.AST),
			Type: e.map_(n.Type),
			Map:  e.expr(n.Map),
		}
	})
	return
}

func (e *encoder) member(n *semantic.Member) (outID uint64) {
	e.build(&e.instances.Member, e.maps.Member, n, &outID, func() *Member {
		return &Member{
			Ast:    e.astMember(n.AST),
			Object: e.expr(n.Object),
			Field:  e.field(n.Field),
		}
	})
	return
}

func (e *encoder) messageValue(n *semantic.MessageValue) (outID uint64) {
	e.build(&e.instances.MessageValue, e.maps.MessageValue, n, &outID, func() *MessageValue {
		p := &MessageValue{
			Ast: e.astClass(n.AST),
		}
		foreach(n.Arguments, e.fieldInit, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) null(n semantic.Null) (outID uint64) {
	e.build(&e.instances.Null, e.maps.Null, n, &outID, func() *Null {
		return &Null{
			Ast:  e.astNull(n.AST),
			Type: e.ty(n.Type),
		}
	})
	return
}

func (e *encoder) observed(n *semantic.Observed) (outID uint64) {
	e.build(&e.instances.Observed, e.maps.Observed, n, &outID, func() *Observed {
		return &Observed{
			Parameter: e.param(n.Parameter),
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

func (e *encoder) pointerRange(n *semantic.PointerRange) (outID uint64) {
	e.build(&e.instances.PointerRange, e.maps.PointerRange, n, &outID, func() *PointerRange {
		return &PointerRange{
			Ast:     e.astIndex(n.AST),
			Type:    e.slice(n.Type),
			Pointer: e.expr(n.Pointer),
			Range:   e.expr(n.Range),
		}
	})
	return
}

func (e *encoder) print(n *semantic.Print) (outID uint64) {
	e.build(&e.instances.Print, e.maps.Print, n, &outID, func() *Print {
		p := &Print{
			Ast: e.astCall(n.AST),
		}
		foreach(n.Arguments, e.expr, &p.Arguments)
		return p
	})
	return
}

func (e *encoder) pseudonym(n *semantic.Pseudonym) (outID uint64) {
	e.build(&e.instances.Pseudonym, e.maps.Pseudonym, n, &outID, func() *Pseudonym {
		n.SortMembers()
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

func (e *encoder) read(n *semantic.Read) (outID uint64) {
	e.build(&e.instances.Read, e.maps.Read, n, &outID, func() *Read {
		return &Read{
			Ast:   e.astCall(n.AST),
			Slice: e.expr(n.Slice),
		}
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

func (e *encoder) return_(n *semantic.Return) (outID uint64) {
	e.build(&e.instances.Return, e.maps.Return, n, &outID, func() *Return {
		return &Return{
			Ast:      e.astReturn(n.AST),
			Function: e.function(n.Function),
			Value:    e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) select_(n *semantic.Select) (outID uint64) {
	e.build(&e.instances.Select, e.maps.Select, n, &outID, func() *Select {
		p := &Select{
			Ast:     e.astSwitch(n.AST),
			Type:    e.ty(n.Type),
			Value:   e.expr(n.Value),
			Default: e.expr(n.Default),
		}
		foreach(n.Choices, e.choice, &p.Choices)
		return p
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

func (e *encoder) sliceAssign(n *semantic.SliceAssign) (outID uint64) {
	e.build(&e.instances.SliceAssign, e.maps.SliceAssign, n, &outID, func() *SliceAssign {
		return &SliceAssign{
			Ast:      e.astAssign(n.AST),
			To:       e.expr(n.To),
			Operator: e.str(n.Operator),
			Value:    e.expr(n.Value),
		}
	})
	return
}

func (e *encoder) sliceIndex(n *semantic.SliceIndex) (outID uint64) {
	e.build(&e.instances.SliceIndex, e.maps.SliceIndex, n, &outID, func() *SliceIndex {
		return &SliceIndex{
			Ast:   e.astIndex(n.AST),
			Type:  e.slice(n.Type),
			Slice: e.expr(n.Slice),
			Index: e.expr(n.Index),
		}
	})
	return
}

func (e *encoder) sliceRange(n *semantic.SliceRange) (outID uint64) {
	e.build(&e.instances.SliceRange, e.maps.SliceRange, n, &outID, func() *SliceRange {
		return &SliceRange{
			Ast:   e.astIndex(n.AST),
			Type:  e.slice(n.Type),
			Slice: e.expr(n.Slice),
			Range: e.expr(n.Range),
		}
	})
	return
}

func (e *encoder) stat(n semantic.Statement) (outID uint64) {
	e.build(&e.instances.Statement, e.maps.Statement, n, &outID, func() *Statement {
		p := &Statement{}
		switch n := n.(type) {
		case *semantic.Abort:
			p.Ty = &Statement_Abort{e.abort(n)}
		case *semantic.ArrayAssign:
			p.Ty = &Statement_ArrayAssign{e.arrayAssign(n)}
		case *semantic.Assert:
			p.Ty = &Statement_Assert{e.assert(n)}
		case *semantic.Assign:
			p.Ty = &Statement_Assign{e.assign(n)}
		case *semantic.Branch:
			p.Ty = &Statement_Branch{e.branch(n)}
		case *semantic.Call:
			p.Ty = &Statement_Call{e.call(n)}
		case *semantic.Copy:
			p.Ty = &Statement_Copy{e.copy(n)}
		case *semantic.DeclareLocal:
			p.Ty = &Statement_DeclareLocal{e.declareLocal(n)}
		case *semantic.Fence:
			p.Ty = &Statement_Fence{e.fence(n)}
		case *semantic.Iteration:
			p.Ty = &Statement_Iteration{e.iteration(n)}
		case *semantic.MapAssign:
			p.Ty = &Statement_MapAssign{e.mapAssign(n)}
		case *semantic.MapIteration:
			p.Ty = &Statement_MapIteration{e.mapIteration(n)}
		case *semantic.MapRemove:
			p.Ty = &Statement_MapRemove{e.mapRemove(n)}
		case *semantic.MapClear:
			p.Ty = &Statement_MapClear{e.mapClear(n)}
		case *semantic.Print:
			p.Ty = &Statement_Print{e.print(n)}
		case *semantic.Read:
			p.Ty = &Statement_Read{e.read(n)}
		case *semantic.Return:
			p.Ty = &Statement_Return{e.return_(n)}
		case *semantic.SliceAssign:
			p.Ty = &Statement_SliceAssign{e.sliceAssign(n)}
		case *semantic.Switch:
			p.Ty = &Statement_Switch{e.switch_(n)}
		case *semantic.Write:
			p.Ty = &Statement_Write{e.write(n)}
		default:
			panic(fmt.Errorf("Unhandled statement type %T", n))
		}
		return p
	})
	return
}

func (e *encoder) staticArray(n *semantic.StaticArray) (outID uint64) {
	e.build(&e.instances.StaticArray, e.maps.StaticArray, n, &outID, func() *StaticArray {
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

func (e *encoder) stringValue(n semantic.StringValue) (outID uint64) {
	e.build(&e.instances.StringValue, e.maps.StringValue, n, &outID, func() *StringValue {
		return &StringValue{
			Value: e.str(string(n)),
		}
	})
	return
}

func (e *encoder) str(s string) (outID uint64) {
	e.build(&e.instances.Symbols, e.maps.String, s, &outID, func() string { return s })
	return
}

func (e *encoder) switch_(n *semantic.Switch) (outID uint64) {
	e.build(&e.instances.Switch, e.maps.Switch, n, &outID, func() *Switch {
		p := &Switch{
			Ast:     e.astSwitch(n.AST),
			Value:   e.expr(n.Value),
			Default: e.block(n.Default),
		}
		foreach(n.Cases, e.case_, &p.Cases)
		return p
	})
	return
}

func (e *encoder) ty(n semantic.Type) *Type {
	if isNil(n) {
		return nil
	}
	switch n := n.(type) {
	case *semantic.Builtin:
		return &Type{Ty: &Type_Builtin{e.builtin(n)}}
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
		return &Type{Ty: &Type_StaticArray{e.staticArray(n)}}
	default:
		panic(fmt.Errorf("Unhandled type %T", n))
	}
}

func (e *encoder) uint8Value(n semantic.Uint8Value) (outID uint64) {
	e.build(&e.instances.Uint8Value, e.maps.Uint8Value, n, &outID, func() *Uint8Value {
		return &Uint8Value{
			Value: uint32(n),
		}
	})
	return
}

func (e *encoder) uint16Value(n semantic.Uint16Value) (outID uint64) {
	e.build(&e.instances.Uint16Value, e.maps.Uint16Value, n, &outID, func() *Uint16Value {
		return &Uint16Value{
			Value: uint32(n),
		}
	})
	return
}

func (e *encoder) uint32Value(n semantic.Uint32Value) (outID uint64) {
	e.build(&e.instances.Uint32Value, e.maps.Uint32Value, n, &outID, func() *Uint32Value {
		return &Uint32Value{
			Value: uint32(n),
		}
	})
	return
}

func (e *encoder) uint64Value(n semantic.Uint64Value) (outID uint64) {
	e.build(&e.instances.Uint64Value, e.maps.Uint64Value, n, &outID, func() *Uint64Value {
		return &Uint64Value{
			Value: uint64(n),
		}
	})
	return
}

func (e *encoder) unaryOp(n *semantic.UnaryOp) (outID uint64) {
	e.build(&e.instances.UnaryOp, e.maps.UnaryOp, n, &outID, func() *UnaryOp {
		return &UnaryOp{
			Ast:        e.astUnaryOp(n.AST),
			Type:       e.ty(n.Type),
			Operator:   e.str(n.Operator),
			Expression: e.expr(n.Expression),
		}
	})
	return
}

func (e *encoder) unknown(n *semantic.Unknown) (outID uint64) {
	e.build(&e.instances.Unknown, e.maps.Unknown, n, &outID, func() *Unknown {
		return &Unknown{
			Ast:      e.astUnknown(n.AST),
			Inferred: e.expr(n.Inferred),
		}
	})
	return
}

func (e *encoder) write(n *semantic.Write) (outID uint64) {
	e.build(&e.instances.Write, e.maps.Write, n, &outID, func() *Write {
		return &Write{
			Ast:   e.astCall(n.AST),
			Slice: e.expr(n.Slice),
		}
	})
	return
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

func (e *encoder) astAPI(n *ast.API) (outID uint64) {
	e.build(&e.instances.AstApi, e.maps.ASTAPI, n, &outID, func() *ASTAPI {
		p := &ASTAPI{}
		foreach(n.Imports, e.astImport, &p.Imports)
		foreach(n.Externs, e.astFunction, &p.Externs)
		foreach(n.Commands, e.astFunction, &p.Commands)
		foreach(n.Subroutines, e.astFunction, &p.Subroutines)
		foreach(n.Pseudonyms, e.astPseudonym, &p.Pseudonyms)
		foreach(n.Enums, e.astEnum, &p.Enums)
		foreach(n.Classes, e.astClass, &p.Classes)
		foreach(n.Fields, e.astField, &p.Fields)
		foreach(n.Definitions, e.astDefinition, &p.Definitions)
		if n.Index != nil {
			p.Index = e.astNumber(n.Index)
		}
		return p
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

func (e *encoder) astClear(n *ast.Clear) (outID uint64) {
	e.build(&e.instances.AstClear, e.maps.ASTClear, n, &outID, func() *ASTClear {
		return &ASTClear{
			Map: e.astNode(n.Map),
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

func (e *encoder) astImport(n *ast.Import) (outID uint64) {
	e.build(&e.instances.AstImport, e.maps.ASTImport, n, &outID, func() *ASTImport {
		return &ASTImport{
			Annotations: e.astAnnotations(n.Annotations),
			Path:        e.astString(n.Path),
		}
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
	if n == nil {
		return nil
	}

	defer checkMessageEncodes(out)

	switch n := n.(type) {
	case *ast.Abort:
		return &ASTNode{Ty: &ASTNode_Abort{e.astAbort(n)}}
	case *ast.Annotation:
		return &ASTNode{Ty: &ASTNode_Annotation{e.astAnnotation(n)}}
	case *ast.API:
		return &ASTNode{Ty: &ASTNode_Api{e.astAPI(n)}}
	case *ast.Assign:
		return &ASTNode{Ty: &ASTNode_Assign{e.astAssign(n)}}
	case *ast.BinaryOp:
		return &ASTNode{Ty: &ASTNode_BinaryOp{e.astBinaryOp(n)}}
	case *ast.Block:
		return &ASTNode{Ty: &ASTNode_Block{e.astBlock(n)}}
	case *ast.Bool:
		return &ASTNode{Ty: &ASTNode_Bool{e.astBool(n)}}
	case *ast.Branch:
		return &ASTNode{Ty: &ASTNode_Branch{e.astBranch(n)}}
	case *ast.Call:
		return &ASTNode{Ty: &ASTNode_Call{e.astCall(n)}}
	case *ast.Case:
		return &ASTNode{Ty: &ASTNode_Case{e.astCase(n)}}
	case *ast.Class:
		return &ASTNode{Ty: &ASTNode_Class{e.astClass(n)}}
	case *ast.Clear:
		return &ASTNode{Ty: &ASTNode_Clear{e.astClear(n)}}
	case *ast.DeclareLocal:
		return &ASTNode{Ty: &ASTNode_DeclareLocal{e.astDeclareLocal(n)}}
	case *ast.Default:
		return &ASTNode{Ty: &ASTNode_Default{e.astDefault(n)}}
	case *ast.Definition:
		return &ASTNode{Ty: &ASTNode_Definition{e.astDefinition(n)}}
	case *ast.Delete:
		return &ASTNode{Ty: &ASTNode_Delete{e.astDelete(n)}}
	case *ast.Enum:
		return &ASTNode{Ty: &ASTNode_Enum{e.astEnum(n)}}
	case *ast.EnumEntry:
		return &ASTNode{Ty: &ASTNode_EnumEntry{e.astEnumEntry(n)}}
	case *ast.Fence:
		return &ASTNode{Ty: &ASTNode_Fence{e.astFence(n)}}
	case *ast.Field:
		return &ASTNode{Ty: &ASTNode_Field{e.astField(n)}}
	case *ast.Function:
		return &ASTNode{Ty: &ASTNode_Function{e.astFunction(n)}}
	case *ast.Generic:
		return &ASTNode{Ty: &ASTNode_Generic{e.astGeneric(n)}}
	case *ast.Group:
		return &ASTNode{Ty: &ASTNode_Group{e.astGroup(n)}}
	case *ast.Identifier:
		return &ASTNode{Ty: &ASTNode_Identifier{e.astIdentifier(n)}}
	case *ast.Import:
		return &ASTNode{Ty: &ASTNode_Import{e.astImport(n)}}
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
	case *ast.Parameter:
		return &ASTNode{Ty: &ASTNode_Parameter{e.astParameter(n)}}
	case *ast.PointerType:
		return &ASTNode{Ty: &ASTNode_PointerType{e.astPointerType(n)}}
	case *ast.PreConst:
		return &ASTNode{Ty: &ASTNode_PreConst{e.astPreConst(n)}}
	case *ast.Pseudonym:
		return &ASTNode{Ty: &ASTNode_Pseudonym{e.astPseudonym(n)}}
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

func (e *encoder) cstBranch(n *cst.Branch) (outID uint64) {
	e.build(&e.instances.CstBranch, e.maps.CSTBranch, n, &outID, func() *CSTBranch {
		p := &CSTBranch{
			Branch: e.cstBranch(n.Branch),
			Pre:    e.cstSeparator(n.Pre),
			Post:   e.cstSeparator(n.Post),
		}
		foreach(n.Children, e.cstNode, &p.Children)
		return p
	})
	return
}

func (e *encoder) cstFragment(n cst.Fragment) *CSTFragment {
	switch n := n.(type) {
	case *cst.Branch:
		return &CSTFragment{Ty: &CSTFragment_Branch{e.cstBranch(n)}}
	case cst.Token:
		return &CSTFragment{Ty: &CSTFragment_Token{e.cstToken(n)}}
	default:
		panic(fmt.Errorf("Unhandled CST Fragment: %T", n))
	}
}

func (e *encoder) cstLeaf(n *cst.Leaf) (outID uint64) {
	e.build(&e.instances.CstLeaf, e.maps.CSTLeaf, n, &outID, func() *CSTLeaf {
		return &CSTLeaf{
			Token:  e.cstToken(n.Token),
			Branch: e.cstBranch(n.Branch),
			Pre:    e.cstSeparator(n.Pre),
			Post:   e.cstSeparator(n.Post),
		}
	})
	return
}

func (e *encoder) cstNode(n cst.Node) (out *CSTNode) {
	switch n := n.(type) {
	case *cst.Branch:
		return &CSTNode{Ty: &CSTNode_Branch{e.cstBranch(n)}}
	case *cst.Leaf:
		return &CSTNode{Ty: &CSTNode_Leaf{e.cstLeaf(n)}}
	default:
		panic(fmt.Errorf("Unhandled CST Node: %T", n))
	}
}

func (e *encoder) cstSeparator(n cst.Separator) *CSTSeparator {
	if len(n) == 0 {
		return nil
	}
	p := &CSTSeparator{}
	foreach(n, e.cstFragment, &p.Fragments)
	return p
}

func (e *encoder) cstSource(n *cst.Source) (outID uint64) {
	e.build(&e.instances.CstSource, e.maps.CSTSource, n, &outID, func() *CSTSource {
		return &CSTSource{
			Filename: e.str(n.Filename),
			Content:  e.str(string(n.Runes)),
		}
	})
	return
}

func (e *encoder) cstToken(n cst.Token) *CSTToken {
	return &CSTToken{
		Source: e.cstSource(n.Source),
		Start:  uint64(n.Start),
		End:    uint64(n.End),
	}
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
