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

package ast

import (
	"github.com/google/gapid/core/text/parse"
)

type BareType struct {
	t *string
}

func (t BareType) String() string { return *t.t }

// All bare types of the language, in an array. Do not modify.
var BareTypes []BareType

func appendType(name string) BareType {
	t := BareType{&name}
	BareTypes = append(BareTypes, t)
	return t
}

// All types of the language.
var (
	TVoid                 = appendType("void")
	TFloat                = appendType("float")
	TInt                  = appendType("int")
	TUint                 = appendType("uint")
	TBool                 = appendType("bool")
	TMat2                 = appendType("mat2")
	TMat3                 = appendType("mat3")
	TMat4                 = appendType("mat4")
	TMat2x2               = appendType("mat2x2")
	TMat2x3               = appendType("mat2x3")
	TMat2x4               = appendType("mat2x4")
	TMat3x2               = appendType("mat3x2")
	TMat3x3               = appendType("mat3x3")
	TMat3x4               = appendType("mat3x4")
	TMat4x2               = appendType("mat4x2")
	TMat4x3               = appendType("mat4x3")
	TMat4x4               = appendType("mat4x4")
	TVec2                 = appendType("vec2")
	TVec3                 = appendType("vec3")
	TVec4                 = appendType("vec4")
	TIvec2                = appendType("ivec2")
	TIvec3                = appendType("ivec3")
	TIvec4                = appendType("ivec4")
	TBvec2                = appendType("bvec2")
	TBvec3                = appendType("bvec3")
	TBvec4                = appendType("bvec4")
	TUvec2                = appendType("uvec2")
	TUvec3                = appendType("uvec3")
	TUvec4                = appendType("uvec4")
	TSampler2D            = appendType("sampler2D")
	TSampler3D            = appendType("sampler3D")
	TSamplerCube          = appendType("samplerCube")
	TSampler2DShadow      = appendType("sampler2DShadow")
	TSamplerCubeShadow    = appendType("samplerCubeShadow")
	TSampler2DArray       = appendType("sampler2DArray")
	TSampler2DArrayShadow = appendType("sampler2DArrayShadow")
	TSamplerExternalOES   = appendType("samplerExternalOES")
	TIsampler2D           = appendType("isampler2D")
	TIsampler3D           = appendType("isampler3D")
	TIsamplerCube         = appendType("isamplerCube")
	TIsampler2DArray      = appendType("isampler2DArray")
	TUsampler2D           = appendType("usampler2D")
	TUsampler3D           = appendType("usampler3D")
	TUsamplerCube         = appendType("usamplerCube")
	TUsampler2DArray      = appendType("usampler2DArray")

	// ARB_texture_multisample
	TSampler2DMS        = appendType("sampler2DMS")
	TIsampler2DMS       = appendType("isampler2DMS")
	TUssampler2DMS      = appendType("ussampler2DMS")
	TSampler2DMSArray   = appendType("sampler2DMSArray")
	TIsampler2DMSArray  = appendType("isampler2DMSArray")
	TUssampler2DMSArray = appendType("ussampler2DMSArray")
)

// Canonicalize returns a canonical representation of the type. Specifically, it returns TMat2x2
// instead of TMat2, etc.
func (t BareType) Canonicalize() BareType {
	switch t {
	case TMat2:
		return TMat2x2
	case TMat3:
		return TMat3x3
	case TMat4:
		return TMat4x4
	default:
		return t
	}
}

// We store a value which is one less than the true dimensions, using the fact that unspecified
// types will then default to 1x1.
var typeDimensions = map[BareType]struct {
	col, row uint8
}{
	TVec2:   {0, 1},
	TVec3:   {0, 2},
	TVec4:   {0, 3},
	TIvec2:  {0, 1},
	TIvec3:  {0, 2},
	TIvec4:  {0, 3},
	TUvec2:  {0, 1},
	TUvec3:  {0, 2},
	TUvec4:  {0, 3},
	TBvec2:  {0, 1},
	TBvec3:  {0, 2},
	TBvec4:  {0, 3},
	TMat2:   {1, 1},
	TMat2x2: {1, 1},
	TMat2x3: {1, 2},
	TMat2x4: {1, 3},
	TMat3:   {2, 2},
	TMat3x2: {2, 1},
	TMat3x3: {2, 2},
	TMat3x4: {2, 3},
	TMat4:   {3, 3},
	TMat4x2: {3, 1},
	TMat4x3: {3, 2},
	TMat4x4: {3, 3},
}

// TypeDimensions returns the number of columns and rows a given type contains. Vectors are
// treated as column vectors (cols == 1). Scalars have 1 column and 1 row.
func TypeDimensions(t BareType) (col uint8, row uint8) {
	dim := typeDimensions[t]
	return dim.col + 1, dim.row + 1
}

var matrixTypes = map[uint8]map[uint8]BareType{
	1: {1: TFloat, 2: TVec2, 3: TVec3, 4: TVec4},
	2: {1: TVec2, 2: TMat2x2, 3: TMat2x3, 4: TMat2x4},
	3: {1: TVec3, 2: TMat3x2, 3: TMat3x3, 4: TMat3x4},
	4: {1: TVec4, 2: TMat4x2, 3: TMat4x3, 4: TMat4x4},
}

// Returns a matrix type with the given number of columns and rows. In case the number of rows or
// columns is 1, returns a vector type. In case both are 1, returns TFloat.
func GetMatrixType(col, row uint8) BareType { return matrixTypes[col][row] }

// IsMatrixType returns true if t is a matrix type.
func IsMatrixType(t BareType) bool { col, _ := TypeDimensions(t); return col > 1 }

var fundMap = map[BareType]BareType{
	TBool:   TBool,
	TFloat:  TFloat,
	TInt:    TInt,
	TUint:   TUint,
	TMat2:   TFloat,
	TMat2x2: TFloat,
	TMat2x3: TFloat,
	TMat2x4: TFloat,
	TMat3:   TFloat,
	TMat3x2: TFloat,
	TMat3x3: TFloat,
	TMat3x4: TFloat,
	TMat4:   TFloat,
	TMat4x2: TFloat,
	TMat4x3: TFloat,
	TMat4x4: TFloat,
	TVec2:   TFloat,
	TVec3:   TFloat,
	TVec4:   TFloat,
	TIvec2:  TInt,
	TIvec3:  TInt,
	TIvec4:  TInt,
	TUvec2:  TUint,
	TUvec3:  TUint,
	TUvec4:  TUint,
	TBvec2:  TBool,
	TBvec3:  TBool,
	TBvec4:  TBool,
}

// GetFundamentalType returns the fundamental type of a given builtin type. E.g., the fundamental
// type of TVec2 is TFloat, since vec2 is a vector of two flots. For types which cannot be used
// in vectors or matrices (such as sampler types or void), the function returns TVoid as their
// fundamental type.
func GetFundamentalType(t BareType) BareType {
	if fund, present := fundMap[t]; present {
		return fund
	} else {
		return TVoid
	}
}

var vectorTypes = map[BareType]map[uint8]BareType{
	TFloat: {1: TFloat, 2: TVec2, 3: TVec3, 4: TVec4},
	TInt:   {1: TInt, 2: TIvec2, 3: TIvec3, 4: TIvec4},
	TUint:  {1: TUint, 2: TUvec2, 3: TUvec3, 4: TUvec4},
	TBool:  {1: TBool, 2: TBvec2, 3: TBvec3, 4: TBvec4},
}

// HasVectorExpansions returns true if the type t has corresponding vector counterparts. This is
// currently true only for TBool, TFloat, TInt and TUint.
func HasVectorExpansions(t BareType) bool { return vectorTypes[t] != nil }

// GetVectorType returns the builtin vector type with the given fundamental type (TFloat, TInt,
// TUint and TBool) and the given size. E.g., GetVectorType(TBool, 4) == TBvec4. In case size is
// 1 it returns the corresponding scalar type. It is illegal to call this function with any other
// type except the four types mentioned.
func GetVectorType(fund BareType, size uint8) BareType { return vectorTypes[fund][size] }

// IsVectorType returns true if the given type is a vector type.
func IsVectorType(t BareType) bool { col, row := TypeDimensions(t); return col == 1 && row > 1 }

// A map used for default precision inheritance.
var precisionTypeMap = map[BareType]BareType{
	TInt:                  TInt,
	TUint:                 TInt,
	TIvec2:                TInt,
	TIvec3:                TInt,
	TIvec4:                TInt,
	TUvec2:                TInt,
	TUvec3:                TInt,
	TUvec4:                TInt,
	TFloat:                TFloat,
	TVec2:                 TFloat,
	TVec3:                 TFloat,
	TVec4:                 TFloat,
	TMat2:                 TFloat,
	TMat3:                 TFloat,
	TMat4:                 TFloat,
	TMat2x2:               TFloat,
	TMat2x3:               TFloat,
	TMat2x4:               TFloat,
	TMat3x2:               TFloat,
	TMat3x3:               TFloat,
	TMat3x4:               TFloat,
	TMat4x2:               TFloat,
	TMat4x3:               TFloat,
	TMat4x4:               TFloat,
	TSampler2D:            TSampler2D,
	TSampler3D:            TSampler3D,
	TSamplerCube:          TSamplerCube,
	TSampler2DShadow:      TSampler2DShadow,
	TSamplerCubeShadow:    TSamplerCubeShadow,
	TSampler2DArray:       TSampler2DArray,
	TSampler2DArrayShadow: TSampler2DArrayShadow,
	TSamplerExternalOES:   TSamplerExternalOES,
	TIsampler2D:           TIsampler2D,
	TIsampler3D:           TIsampler3D,
	TIsamplerCube:         TIsamplerCube,
	TIsampler2DArray:      TIsampler2DArray,
	TUsampler2D:           TUsampler2D,
	TUsampler3D:           TUsampler3D,
	TUsampler2DArray:      TUsampler2DArray,
	TSampler2DMS:          TSampler2DMS,
	TIsampler2DMS:         TIsampler2DMS,
	TUssampler2DMS:        TUssampler2DMS,
	TSampler2DMSArray:     TSampler2DMSArray,
	TIsampler2DMSArray:    TIsampler2DMSArray,
	TUssampler2DMSArray:   TUssampler2DMSArray,
}

// GetPrecisionType returns the type from which the provided type inherits is precision. E.g.,
// vec2 inherits its type from float as they have the same basic type. If the type cannot be
// qualified with a precision the second result is false.
func GetPrecisionType(t BareType) (prect BareType, hasPrecision bool) {
	prect, hasPrecision = precisionTypeMap[t]
	return
}

// Type represents a language type along with its precision.
type Type interface {
	// Whether two objects represent the same type, according to the language semantics.
	Equal(other Type) bool
}

// Precision of a type
type Precision uint8

const (
	NoneP Precision = iota
	LowP
	MediumP
	HighP
)

func (p Precision) String() string {
	switch p {
	case HighP:
		return "highp"
	case MediumP:
		return "mediump"
	case LowP:
		return "lowp"
	}
	return ""
}

// BuiltinType represents a builtin type of the language: int, float, vec2, etc. It also stores
// the precision of the type.
type BuiltinType struct {
	Precision Precision
	Type      BareType

	PrecisionCst *parse.Leaf
	TypeCst      *parse.Leaf
}

// Two builtin types are equal if their Type members are equal. Precision is ignored.
func (bt *BuiltinType) Equal(other Type) bool {
	if other, ok := other.(*BuiltinType); ok {
		return bt.Type.Canonicalize() == other.Type.Canonicalize()
	}
	return false
}

// ArrayType represents array types. Size is represented as an expression. Unspecified size is
// represented by a nil expression. After semantic analysis, the ComputedSize field will store
// the size of the array as an integer. This size will be determined by evaluating the size
// expression, or from context. Precision getters and setters use the precision of the Base type.
type ArrayType struct {
	Base Type
	Size Expression

	LBracketCst *parse.Leaf
	RBracketCst *parse.Leaf

	ComputedSize UintValue // The size determined by semantic analysis.
}

// Equal determines whether two objects represent the same type. Note that a semantic analysis
// pass over the AST (to populate ComputedSize) is needed to produce meaningful results.
func (at *ArrayType) Equal(other Type) bool {
	if other, ok := other.(*ArrayType); ok {
		return at.ComputedSize == other.ComputedSize && at.Base.Equal(other.Base)
	}
	return false
}

// Copy returns a (shallow) copy of the underlying object.
func (at *ArrayType) Copy() *ArrayType { copied := *at; return &copied }

// StructType represents structure types as a pointer to the corresponding structure symbol.
type StructType struct {
	Sym       *StructSym
	StructDef bool // if true, the structure was defined at this point in the AST.

	NameCst *parse.Leaf // Only used if StructDef is false
}

func (s *StructType) Equal(other Type) bool {
	if other, ok := other.(*StructType); ok {
		return s.Sym == other.Sym
	}
	return false
}

// A single name = value pair in the layout qualifier. If the qualifier is present just as a bare
// name (without the value) then Value is nil.
type LayoutQualifierID struct {
	Name  string
	Value Value

	NameCst  *parse.Leaf
	EqualCst *parse.Leaf
	ValueCst *parse.Leaf
}

// layout(...)
type LayoutQualifier struct {
	Ids []LayoutQualifierID

	LayoutCst *parse.Leaf
	LParenCst *parse.Leaf
	CommaCst  []*parse.Leaf
	RParenCst *parse.Leaf
}

type InterpolationQualifier uint8

const (
	IntNone InterpolationQualifier = iota
	IntSmooth
	IntFlat
)

func (i InterpolationQualifier) String() string {
	switch i {
	case IntSmooth:
		return "smooth"
	case IntFlat:
		return "flat"
	}
	return ""
}

type StorageQualifier uint8

const (
	StorNone StorageQualifier = iota
	StorConst
	StorUniform
	StorIn
	StorOut
	StorCentroidIn
	StorCentroidOut
	StorAttribute
	StorVarying
)

func (s StorageQualifier) String() string {
	switch s {
	case StorConst:
		return "const"
	case StorUniform:
		return "uniform"
	case StorIn:
		return "in"
	case StorOut:
		return "out"
	case StorAttribute:
		return "attribute"
	case StorVarying:
		return "varying"
	case StorCentroidIn:
		return "centroid in"
	case StorCentroidOut:
		return "centroid out"
	}
	return ""
}

// Invariant, interpolation, layout and storage qualifiers of declared variables.
type TypeQualifiers struct {
	Invariant     bool
	Interpolation InterpolationQualifier
	Layout        *LayoutQualifier
	Storage       StorageQualifier

	InvariantCst       *parse.Leaf
	InterpolationCst   *parse.Leaf
	CentroidStorageCst *parse.Leaf
	StorageCst         *parse.Leaf
}

// FunctionType represents function types. The GLES Shading Language does not specify the
// presence of function types, but we add it to be able to handle expressions uniformly. A
// function type can appear in the AST in the following manner:
//
// - VarRefExpr to a function declaration,
//
// - "array.length" expression,
//
// - the "type" part of the "type(expression)" type constructor syntax.
//
// In all valid programs a expression of a function type can be present only as a child of a
// call expression.
type FunctionType struct {
	ArgTypes []Type
	RetType  Type
}

// Two function types are equal if they have the same return and argument types.
func (t *FunctionType) Equal(other Type) bool {
	if other, ok := other.(*FunctionType); ok {
		if !t.RetType.Equal(other.RetType) {
			return false
		}
		if len(t.ArgTypes) != len(other.ArgTypes) {
			return false
		}
		for i := range t.ArgTypes {
			if !t.ArgTypes[i].Equal(other.ArgTypes[i]) {
				return false
			}
		}
		return true
	}
	return false
}
