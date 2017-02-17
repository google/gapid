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

// Package builtins contains the definitions of all OpenGL ES Shading Language builtin functions.
//
// It is not expected to be used directly by the user, it just provides the other packages with
// the list of builtin functions, their signatures and implementations.
package builtins

import (
	"fmt"
	"math"

	"github.com/google/gapid/core/math/f16"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

// Data needed for vectorization of a function.
type vectorData struct {
	retFund ast.BareType // The fundamental type of the return value.

	// The fundamental type of the arguments. A type TVoid means that this particular
	// argument should not be processed member-by-member, but provided as-is to each scalar
	// computation.
	argTypes []ast.BareType
	impl     func(v []ast.Value) ast.Value // The scalar implementation of a function.
}

// Implements the vector operation by calling the scalar function successively.
func (d vectorData) evaluate(size uint8, v []ast.Value) ast.Value {
	if size == 1 {
		return d.impl(v)
	}

	return ast.NewVectorValue(ast.GetVectorType(d.retFund, size), func(i uint8) ast.Value {
		v := append(make([]ast.Value, 0, len(v)), v...)
		for j := range d.argTypes {
			if d.argTypes[j] != ast.TVoid {
				v[j] = v[j].(ast.VectorValue).Members[i]
			}
		}
		return d.impl(v)
	})
}

func toFloat(v ast.Value) float64 { return float64(v.(ast.FloatValue)) }
func toInt(v ast.Value) int32     { return int32(v.(ast.IntValue)) }
func toUint(v ast.Value) uint32   { return uint32(v.(ast.UintValue)) }

func round(x float64) float64  { return math.Floor(x + 0.5) }
func mod(x, y float64) float64 { return x - y*math.Floor(x/y) }

// wraps a float64->float64 function into a function operating on ast.Values
func wrapFloatFunc(f func(x float64) float64) func(v []ast.Value) ast.Value {
	return func(v []ast.Value) ast.Value {
		return ast.FloatValue(f(toFloat(v[0])))
	}
}

// wraps a float64,float64->float64 function into a function operating on ast.Values
func wrapFloat2Func(f func(x, y float64) float64) func(v []ast.Value) ast.Value {
	return func(v []ast.Value) ast.Value {
		return ast.FloatValue(f(toFloat(v[0]), toFloat(v[1])))
	}
}

// wraps a int32,int32->int32 function into a function operating on ast.Values
func wrapInt2Func(f func(x, y int32) int32) func(v []ast.Value) ast.Value {
	return func(v []ast.Value) ast.Value {
		return ast.IntValue(f(toInt(v[0]), toInt(v[1])))
	}
}

// wraps a uint32,uint32->uint32 function into a function operating on ast.Values
func wrapUint2Func(f func(x, y uint32) uint32) func(v []ast.Value) ast.Value {
	return func(v []ast.Value) ast.Value {
		return ast.UintValue(f(toUint(v[0]), toUint(v[1])))
	}
}

// given a list of types, generates a list of function parameter declarations
func makeParams(types []ast.Type) []*ast.FuncParameterSym {
	params := make([]*ast.FuncParameterSym, len(types))
	for j := range params {
		params[j] = &ast.FuncParameterSym{
			SymType: types[j],
			SymName: fmt.Sprintf("arg%d", j),
		}
	}
	return params
}

func makeFloatVector(size uint8, member func(i uint8) ast.Value) ast.VectorValue {
	return ast.NewVectorValue(ast.GetVectorType(ast.TFloat, size), member)
}

const (
	factorSPack = 32767.0
	factorUPack = 65535.0
)

// BuiltinFunctions contains the list of builtin functions defined by the package, in no
// particular order.
//
// PS: The list showed by godoc is not complete. More functions are added during package init().
// For a definitive list, see The OpenGL ES Shading Language specification, section 8.
var BuiltinFunctions = []ast.BuiltinFunction{
	// 8.4 Floating-Point Pack and Unpack Functions
	{
		RetType: &ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint},
		SymName: "packSnorm2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Type: ast.TVec2}}),
		Impl: func(v []ast.Value) ast.Value {
			vec := v[0].(ast.VectorValue)
			x, y := toFloat(vec.Members[0]), toFloat(vec.Members[1])
			x = round(math.Min(math.Max(x, -1.0), 1.0) * factorSPack)
			y = round(math.Min(math.Max(y, -1.0), 1.0) * factorSPack)
			return ast.UintValue(int16(x)) + ast.UintValue(int16(y))<<16
		},
	},
	{
		RetType: &ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint},
		SymName: "packUnorm2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Type: ast.TVec2}}),
		Impl: func(v []ast.Value) ast.Value {
			vec := v[0].(ast.VectorValue)
			x, y := toFloat(vec.Members[0]), toFloat(vec.Members[1])
			x = round(math.Min(math.Max(x, 0.0), 1.0) * factorUPack)
			y = round(math.Min(math.Max(y, 0.0), 1.0) * factorUPack)
			return ast.UintValue(uint16(x)) + ast.UintValue(uint16(y))<<16
		},
	},
	{
		RetType: &ast.BuiltinType{Precision: ast.HighP, Type: ast.TVec2},
		SymName: "unpackSnorm2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint}}),
		Impl: func(v []ast.Value) ast.Value {
			pack := toUint(v[0])
			vec := []int16{int16(pack), int16(pack >> 16)}
			return makeFloatVector(2, func(i uint8) ast.Value {
				return ast.FloatValue(math.Min(
					math.Max(float64(vec[i])/factorSPack, -1.0), 1.0))
			})
		},
	},
	{
		RetType: &ast.BuiltinType{Precision: ast.HighP, Type: ast.TVec2},
		SymName: "unpackUnorm2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint}}),
		Impl: func(v []ast.Value) ast.Value {
			pack := toUint(v[0])
			vec := []uint16{uint16(pack), uint16(pack >> 16)}
			return makeFloatVector(2, func(i uint8) ast.Value {
				return ast.FloatValue(math.Min(
					math.Max(float64(vec[i])/factorUPack, -1.0), 1.0))
			})
		},
	},
	{
		RetType: &ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint},
		SymName: "packHalf2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Precision: ast.MediumP, Type: ast.TVec2}}),
		Impl: func(v []ast.Value) ast.Value {
			vec := v[0].(ast.VectorValue)
			x := f16.From(float32(toFloat(vec.Members[0])))
			y := f16.From(float32(toFloat(vec.Members[1])))
			return ast.UintValue(x) + ast.UintValue(y)<<16
		},
	},
	{
		RetType: &ast.BuiltinType{Precision: ast.MediumP, Type: ast.TVec2},
		SymName: "unpackHalf2x16",
		Params:  makeParams([]ast.Type{&ast.BuiltinType{Precision: ast.HighP, Type: ast.TUint}}),
		Impl: func(v []ast.Value) ast.Value {
			pack := toUint(v[0])
			vec := []f16.Number{f16.Number(pack), f16.Number(pack >> 16)}
			return makeFloatVector(2,
				func(i uint8) ast.Value { return ast.FloatValue(vec[i].Float32()) })
		},
	},

	// 8.5 Geometric functions (cross only, the rest are in init())
	{
		RetType: &ast.BuiltinType{Type: ast.TVec3},
		SymName: "cross",
		Params: makeParams([]ast.Type{&ast.BuiltinType{Type: ast.TVec3},
			&ast.BuiltinType{Type: ast.TVec3}}),

		Impl: func(v []ast.Value) ast.Value {
			x, y := v[0].(ast.VectorValue).Members, v[1].(ast.VectorValue).Members

			return ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value {
				j, k := (i+1)%3, (i+2)%3
				return x[j].(ast.FloatValue)*y[k].(ast.FloatValue) -
					y[j].(ast.FloatValue)*x[k].(ast.FloatValue)
			})
		},
	},
}

// Given a function name, the types of its arguments and an implementation, this function
// generates BuiltinFunction entries for each vector size, starting with minSize. The impl
// function should return the correct result, given the list of arguments and vector size.
// The first entry of fund and types define the return type, the rest are for the function
// arguments. The final type is determined as follows:
//
// - if fund[i] is ast.TVoid, then the type is types[i]
//
// - if fund[i] is a fundamental type (TFloat, TBool, TInt, TUint), then the type is a vector
// with the same fundamental type and the correct number of elements. types[i] is unused and it
// can be nil.
//
// If the whole types arrays is unused (fund[i] != TVoid, for any i) then array itself can be nil.
func vectorize(name string, types []ast.Type, fund []ast.BareType, minSize uint8,
	impl func(size uint8, v []ast.Value) ast.Value) {

	for i := minSize; i <= 4; i++ {
		i := i
		scalarTypes := make([]ast.Type, len(fund))
		copy(scalarTypes, types)
		for j := range fund {
			if fund[j] != ast.TVoid {
				scalarTypes[j] = &ast.BuiltinType{Type: ast.GetVectorType(fund[j], i)}
			}
		}

		BuiltinFunctions = append(BuiltinFunctions, ast.BuiltinFunction{
			RetType: scalarTypes[0],
			SymName: name,
			Params:  makeParams(scalarTypes[1:]),
			Impl:    func(v []ast.Value) ast.Value { return impl(i, v) },
		})
	}
}

// Given a function implementing a scalar operation, generates BuiltinFunction entries with the
// vector versions of the function.
func vectorizeReturn(name string, types []ast.Type, fund []ast.BareType, minSize uint8,
	impl func(v []ast.Value) ast.Value) {

	data := vectorData{fund[0], fund[1:], impl}
	vectorize(name, types, fund, minSize, data.evaluate)
}

func vectorizeFloat(name string, impl func(v float64) float64) {
	vectorizeReturn(name, nil, []ast.BareType{ast.TFloat, ast.TFloat}, 1, wrapFloatFunc(impl))
}

func vectorizeInt(name string, impl func(v int32) int32) {
	vectorizeReturn(name, nil, []ast.BareType{ast.TInt, ast.TInt}, 1,
		func(v []ast.Value) ast.Value { return ast.IntValue(impl(toInt(v[0]))) })
}

func vectorizeFloat2(name string, impl func(v1, v2 float64) float64) {
	vectorizeReturn(name, nil, []ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat}, 1,
		wrapFloat2Func(impl))
}

// scalar implementation of modf
func modfScalar(x ast.Value) (ast.Value, ast.Value) {
	whole, fract := math.Modf(toFloat(x))
	return ast.FloatValue(whole), ast.FloatValue(fract)
}

// vector version of modf
func modfImpl(v []ast.Value) ast.Value {
	if vec, ok := v[0].(ast.VectorValue); ok {
		t := ast.GetVectorType(ast.TFloat, uint8(len(vec.Members)))
		whole := ast.NewVectorValue(t, func(uint8) ast.Value { return nil })

		frac := ast.NewVectorValue(t, func(i uint8) (ret ast.Value) {
			whole.Members[i], ret = modfScalar(vec.Members[i])
			return
		})
		v[1] = whole
		return frac
	}
	var ret ast.Value
	v[1], ret = modfScalar(v[0])
	return ret
}

// generate modf entries
func genModf() {
	for i := uint8(1); i <= 4; i++ {
		floatType := &ast.BuiltinType{Type: ast.GetVectorType(ast.TFloat, i)}
		params := makeParams([]ast.Type{floatType, floatType})
		params[1].Direction = ast.DirOut

		BuiltinFunctions = append(BuiltinFunctions, ast.BuiltinFunction{
			RetType: floatType,
			SymName: "modf",
			Params:  params,
			Impl:    modfImpl,
		})
	}
}

func intMin(x, y int32) int32 {
	if x < y {
		return x
	} else {
		return y
	}
}

func uintMin(x, y uint32) uint32 {
	if x < y {
		return x
	} else {
		return y
	}
}

func intMax(x, y int32) int32 {
	if x > y {
		return x
	} else {
		return y
	}
}

func uintMax(x, y uint32) uint32 {
	if x > y {
		return x
	} else {
		return y
	}
}

func genMinMaxClamp(t ast.BareType, min, max func(v []ast.Value) ast.Value) {
	scalar := &ast.BuiltinType{Type: ast.GetVectorType(t, 1)}
	vectorizeReturn("min", nil, []ast.BareType{t, t, t}, 1, min)
	vectorizeReturn("min", []ast.Type{nil, nil, scalar}, []ast.BareType{t, t, ast.TVoid}, 2, min)

	vectorizeReturn("max", nil, []ast.BareType{t, t, t}, 1, max)
	vectorizeReturn("max", []ast.Type{nil, nil, scalar}, []ast.BareType{t, t, ast.TVoid}, 2, max)

	clamp := func(v []ast.Value) ast.Value { return min([]ast.Value{max(v[0:2]), v[2]}) }
	vectorizeReturn("clamp", nil, []ast.BareType{t, t, t, t}, 1, clamp)
	vectorizeReturn("clamp", []ast.Type{nil, nil, scalar, scalar},
		[]ast.BareType{t, t, ast.TVoid, ast.TVoid}, 2, clamp)
}

func intAbs(x int32) int32 {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func sign(x float64) float64 {
	switch {
	case x < 0.0:
		return -1.0
	case x > 0.0:
		return 1.0
	default:
		return 0.0
	}
}

func intSign(x int32) int32 {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

func roundEven(x float64) float64 {
	i, frac := math.Modf(x)
	if math.Abs(frac) != 0.5 {
		return round(x)
	}
	if math.Mod(i, 2) == 0.0 {
		return i
	}
	return i + math.Copysign(1, x)
}

// func mix(x, y, a float) float
func mix(v []ast.Value) ast.Value {
	x, y, a := v[0].(ast.FloatValue), v[1].(ast.FloatValue), v[2].(ast.FloatValue)
	return x*(1-a) + y*a
}

// func mix(x, y float, a bool) float
func mixBool(v []ast.Value) ast.Value {
	if v[2].(ast.BoolValue) {
		return v[1]
	} else {
		return v[0]
	}
}

// func step(edge, x float) float
func step(v []ast.Value) ast.Value {
	edge, x := v[0].(ast.FloatValue), v[1].(ast.FloatValue)
	if x < edge {
		return ast.FloatValue(0.0)
	} else {
		return ast.FloatValue(1.0)
	}
}

// func smoothstep(edge0, edge1, x float) float
func smoothstep(v []ast.Value) ast.Value {
	edge0, edge1, x := v[0].(ast.FloatValue), v[1].(ast.FloatValue), v[2].(ast.FloatValue)
	switch {
	case x <= edge0:
		return ast.FloatValue(0.0)
	case x >= edge1:
		return ast.FloatValue(1.0)
	default:
		t := (x - edge0) / (edge1 - edge0)
		return t * t * (3 - 2*t)
	}
}

func isnan(v []ast.Value) ast.Value { return ast.BoolValue(math.IsNaN(toFloat(v[0]))) }
func isinf(v []ast.Value) ast.Value { return ast.BoolValue(math.IsInf(toFloat(v[0]), 0)) }

// func floatBitsToInt(value float) int
func floatBitsToInt(v []ast.Value) ast.Value {
	return ast.IntValue(math.Float32bits(float32(toFloat(v[0]))))
}

// func floatBitsToUint(value float) uint
func floatBitsToUint(v []ast.Value) ast.Value {
	return ast.UintValue(math.Float32bits(float32(toFloat(v[0]))))
}

// func intBitsToFloat(value int) float
func intBitsToFloat(v []ast.Value) ast.Value {
	return ast.FloatValue(math.Float32frombits(uint32(toInt(v[0]))))
}

// func uintBitsToFloat(value uint) float
func uintBitsToFloat(v []ast.Value) ast.Value {
	return ast.FloatValue(math.Float32frombits(toUint(v[0])))
}

// func length(x genType) float
func length(size uint8, v []ast.Value) ast.Value {
	vec := v[0].(ast.VectorValue)
	ret := ast.FloatValue(0.0)
	for i := uint8(0); i < size; i++ {
		f := vec.Members[i].(ast.FloatValue)
		ret += f * f
	}
	return ast.FloatValue(math.Sqrt(float64(ret)))
}

// func distance(p1, p2 genType) float
func distance(size uint8, v []ast.Value) ast.Value {
	p0, p1 := v[0].(ast.VectorValue), v[1].(ast.VectorValue)
	ret := ast.FloatValue(0.0)
	for i := uint8(0); i < size; i++ {
		f := p0.Members[i].(ast.FloatValue) - p1.Members[i].(ast.FloatValue)
		ret += f * f
	}
	return ast.FloatValue(math.Sqrt(float64(ret)))
}

// func dot(x, y genType) float
func dot(size uint8, v []ast.Value) ast.Value {
	p0, p1 := v[0].(ast.VectorValue), v[1].(ast.VectorValue)
	ret := ast.FloatValue(0.0)
	for i := uint8(0); i < size; i++ {
		f := p0.Members[i].(ast.FloatValue) * p1.Members[i].(ast.FloatValue)
		ret += f
	}
	return ret
}

// func normalize(genType x) genType
func normalize(size uint8, v []ast.Value) ast.Value {
	l := length(size, v).(ast.FloatValue)
	x := v[0].(ast.VectorValue).Members
	return makeFloatVector(size, func(i uint8) ast.Value { return x[i].(ast.FloatValue) / l })
}

// func faceforward(N, I, Nref genType) genType
func faceforward(size uint8, v []ast.Value) ast.Value {
	if dot(size, v[1:3]).(ast.FloatValue) < 0.0 {
		return v[0]
	} else {
		N := v[0].(ast.VectorValue).Members
		return makeFloatVector(size, func(i uint8) ast.Value { return -N[i].(ast.FloatValue) })
	}
}

// func reflect(I, N genType) genType
func reflect(size uint8, v []ast.Value) ast.Value {
	d := dot(size, v).(ast.FloatValue)
	I, N := v[0].(ast.VectorValue).Members, v[1].(ast.VectorValue).Members
	return makeFloatVector(size,
		func(i uint8) ast.Value { return I[i].(ast.FloatValue) - 2*d*N[i].(ast.FloatValue) })
}

// func refract(I, N genType, eta float) genType
func refract(size uint8, v []ast.Value) ast.Value {
	d := dot(size, v[0:2]).(ast.FloatValue)
	eta := v[2].(ast.FloatValue)
	k := 1.0 - eta*eta*(1.0-d*d)
	if k < 0.0 {
		return makeFloatVector(size, func(i uint8) ast.Value { return ast.FloatValue(0.0) })
	}

	t := eta*d + ast.FloatValue(math.Sqrt(float64(k)))
	I, N := v[0].(ast.VectorValue).Members, v[1].(ast.VectorValue).Members

	return makeFloatVector(size, func(i uint8) ast.Value {
		return eta*I[i].(ast.FloatValue) - t*N[i].(ast.FloatValue)
	})
}

type matrixType ast.BareType

// func matrixCompMult(x, y mat) mat
func (t matrixType) matrixCompMult(v []ast.Value) ast.Value {
	x, y := v[0].(ast.MatrixValue).Members, v[1].(ast.MatrixValue).Members
	return ast.NewMatrixValue(ast.BareType(t), func(i, j uint8) ast.Value {
		return x[i][j].(ast.FloatValue) * y[i][j].(ast.FloatValue)
	})
}

func genMatrixCompMult() {
	for i := uint8(2); i <= 4; i++ {
		for j := uint8(2); j <= 4; j++ {
			mat := &ast.BuiltinType{Type: ast.GetMatrixType(i, j)}
			types := []ast.Type{mat, mat}

			BuiltinFunctions = append(BuiltinFunctions, ast.BuiltinFunction{
				RetType: mat,
				SymName: "matrixCompMult",
				Params:  makeParams(types),
				Impl:    matrixType(mat.Type).matrixCompMult,
			})
		}
	}
}

// func outerProduct(c vecC, r vecR) matCxR
func (t matrixType) outerProduct(v []ast.Value) ast.Value {
	c, r := v[0].(ast.VectorValue).Members, v[1].(ast.VectorValue).Members
	return ast.NewMatrixValue(ast.BareType(t), func(i, j uint8) ast.Value {
		return c[i].(ast.FloatValue) * r[j].(ast.FloatValue)
	})
}

func genOuterProduct() {
	for i := uint8(2); i <= 4; i++ {
		for j := uint8(2); j <= 4; j++ {
			mat := &ast.BuiltinType{Type: ast.GetMatrixType(i, j)}
			types := []ast.Type{
				&ast.BuiltinType{Type: ast.GetVectorType(ast.TFloat, i)},
				&ast.BuiltinType{Type: ast.GetVectorType(ast.TFloat, j)},
			}

			BuiltinFunctions = append(BuiltinFunctions, ast.BuiltinFunction{
				RetType: mat,
				SymName: "outerProduct",
				Params:  makeParams(types),
				Impl:    matrixType(mat.Type).outerProduct,
			})
		}
	}
}

// func transpose(m matCxR) matRxC
func (t matrixType) transpose(v []ast.Value) ast.Value {
	c, r := ast.TypeDimensions(ast.BareType(t))
	m := v[0].(ast.MatrixValue).Members
	return ast.NewMatrixValueCR(c, r, func(i, j uint8) ast.Value { return m[j][i] })
}

func genTranspose() {
	for i := uint8(2); i <= 4; i++ {
		for j := uint8(2); j <= 4; j++ {
			mat := &ast.BuiltinType{Type: ast.GetMatrixType(j, i)}
			types := []ast.Type{&ast.BuiltinType{Type: ast.GetMatrixType(i, j)}}

			BuiltinFunctions = append(BuiltinFunctions, ast.BuiltinFunction{
				RetType: mat,
				SymName: "transpose",
				Params:  makeParams(types),
				Impl:    matrixType(mat.Type).transpose,
			})
		}
	}
}

// computes the (c,r) cofactor of matrix m
func cofactor(m [][]ast.Value, c, r uint8) ast.FloatValue {
	size := uint8(len(m))
	sign := ast.FloatValue(1.0)
	if (r+c)%2 == 1 {
		sign = -sign
	}

	subMat := ast.NewMatrixValueCR(size-1, size-1, func(i, j uint8) ast.Value {
		if i >= c {
			i++
		}
		if j >= r {
			j++
		}
		return m[i][j]
	})
	if size == 2 {
		return sign * subMat.Members[0][0].(ast.FloatValue)
	}

	subType := matrixType(ast.GetMatrixType(size-1, size-1))
	return sign * subType.determinant([]ast.Value{subMat}).(ast.FloatValue)
}

// func determinant(m matX) float
func (t matrixType) determinant(v []ast.Value) ast.Value {
	m := v[0].(ast.MatrixValue).Members
	ret := ast.FloatValue(0.0)
	c, _ := ast.TypeDimensions(ast.BareType(t))
	for k := uint8(0); k < c; k++ {
		ret += cofactor(m, 0, k) * m[0][k].(ast.FloatValue)
	}
	return ret
}

// func inverse(m matX) matX
func (t matrixType) inverse(v []ast.Value) ast.Value {
	det := 1.0 / t.determinant(v).(ast.FloatValue)
	m := v[0].(ast.MatrixValue).Members
	return ast.NewMatrixValue(ast.BareType(t), func(i, j uint8) ast.Value {
		return det * cofactor(m, j, i)
	})
}

func genDeterminantInverse() {
	float := &ast.BuiltinType{Type: ast.TFloat}
	for i := uint8(2); i <= 4; i++ {
		mat := &ast.BuiltinType{Type: ast.GetMatrixType(i, i)}
		types := []ast.Type{mat}
		params := makeParams(types)

		BuiltinFunctions = append(BuiltinFunctions,
			ast.BuiltinFunction{
				RetType: float,
				SymName: "determinant",
				Params:  params,
				Impl:    matrixType(mat.Type).determinant,
			},
			ast.BuiltinFunction{
				RetType: mat,
				SymName: "inverse",
				Params:  params,
				Impl:    matrixType(mat.Type).inverse,
			})
	}
}

// The "less" functions for different numeric types.
var vectorComparisonData = []struct {
	fund ast.BareType
	less func(x, y ast.Value) bool
}{
	{ast.TFloat, func(x, y ast.Value) bool { return x.(ast.FloatValue) < y.(ast.FloatValue) }},
	{ast.TInt, func(x, y ast.Value) bool { return x.(ast.IntValue) < y.(ast.IntValue) }},
	{ast.TUint, func(x, y ast.Value) bool { return x.(ast.UintValue) < y.(ast.UintValue) }},
}

func equal(v []ast.Value) ast.Value {
	return ast.BoolValue(ast.ValueEquals(v[0], v[1]))
}
func notEqual(v []ast.Value) ast.Value {
	return ast.BoolValue(!ast.ValueEquals(v[0], v[1]))
}

func genVectorComparison() {
	for _, d := range vectorComparisonData {
		d := d
		fund := []ast.BareType{ast.TBool, d.fund, d.fund}

		vectorizeReturn("lessThan", nil, fund, 2, func(v []ast.Value) ast.Value {
			return ast.BoolValue(d.less(v[0], v[1]))
		})

		vectorizeReturn("lessThanEqual", nil, fund, 2, func(v []ast.Value) ast.Value {
			return ast.BoolValue(!d.less(v[1], v[0]))
		})

		vectorizeReturn("greaterThan", nil, fund, 2, func(v []ast.Value) ast.Value {
			return ast.BoolValue(d.less(v[1], v[0]))
		})

		vectorizeReturn("greaterThanEqual", nil, fund, 2, func(v []ast.Value) ast.Value {
			return ast.BoolValue(!d.less(v[0], v[1]))
		})

		vectorizeReturn("equal", nil, fund, 2, equal)

		vectorizeReturn("notEqual", nil, fund, 2, notEqual)
	}

	fund := []ast.BareType{ast.TBool, ast.TBool}
	vectorizeReturn("equal", nil, fund, 2, equal)
	vectorizeReturn("notEqual", nil, fund, 2, notEqual)
}

// func any(x genBType) bool
func any(size uint8, x []ast.Value) ast.Value {
	vec := x[0].(ast.VectorValue).Members
	for i := uint8(0); i < size; i++ {
		if vec[i].(ast.BoolValue) == true {
			return ast.BoolValue(true)
		}
	}
	return ast.BoolValue(false)
}

// func all(x genBType) bool
func all(size uint8, x []ast.Value) ast.Value {
	vec := x[0].(ast.VectorValue).Members
	for i := uint8(0); i < size; i++ {
		if vec[i].(ast.BoolValue) == false {
			return ast.BoolValue(false)
		}
	}
	return ast.BoolValue(true)
}

func not(v []ast.Value) ast.Value { return !v[0].(ast.BoolValue) }

func init() {
	floatType := &ast.BuiltinType{Type: ast.TFloat}

	// 8.1 Angle and Trigonometry Functions
	vectorizeFloat("radians", func(degrees float64) float64 { return (math.Pi / 180.0) * degrees })
	vectorizeFloat("degrees", func(radians float64) float64 { return (180.0 / math.Pi) * radians })
	vectorizeFloat("sin", math.Sin)
	vectorizeFloat("cos", math.Cos)
	vectorizeFloat("tan", math.Tan)
	vectorizeFloat("asin", math.Asin)
	vectorizeFloat("acos", math.Acos)
	vectorizeFloat2("atan", math.Atan2)
	vectorizeFloat("atan", math.Atan)
	vectorizeFloat("sinh", math.Sinh)
	vectorizeFloat("cosh", math.Cosh)
	vectorizeFloat("tanh", math.Tanh)
	vectorizeFloat("asinh", math.Asinh)
	vectorizeFloat("acosh", math.Acosh)
	vectorizeFloat("atanh", math.Atanh)

	// 8.2 Exponential Functions
	vectorizeFloat2("pow", math.Pow)
	vectorizeFloat("exp", math.Exp)
	vectorizeFloat("log", math.Log)
	vectorizeFloat("exp2", math.Exp2)
	vectorizeFloat("log2", math.Log2)
	vectorizeFloat("sqrt", math.Sqrt)
	vectorizeFloat("inversesqrt", func(x float64) float64 { return 1 / math.Sqrt(x) })

	// 8.3 Common Functions
	vectorizeFloat("abs", math.Abs)
	vectorizeInt("abs", intAbs)
	vectorizeFloat("sign", sign)
	vectorizeInt("sign", intSign)
	vectorizeFloat("floor", math.Floor)
	vectorizeFloat("trunc", math.Trunc)
	vectorizeFloat("round", round)
	vectorizeFloat("roundEven", roundEven)
	vectorizeFloat("ceil", math.Ceil)
	vectorizeFloat("fract", func(x float64) float64 { return x - math.Floor(x) })

	vectorizeFloat2("mod", mod)
	vectorizeReturn("mod", []ast.Type{nil, nil, floatType},
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TVoid}, 2, wrapFloat2Func(mod))

	genModf()

	genMinMaxClamp(ast.TFloat, wrapFloat2Func(math.Min), wrapFloat2Func(math.Max))
	genMinMaxClamp(ast.TInt, wrapInt2Func(intMin), wrapInt2Func(intMax))
	genMinMaxClamp(ast.TUint, wrapUint2Func(uintMin), wrapUint2Func(uintMax))

	vectorizeReturn("mix", nil,
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TFloat}, 1, mix)
	vectorizeReturn("mix", []ast.Type{nil, nil, nil, floatType},
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TFloat}, 2, mix)
	vectorizeReturn("mix", nil,
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TBool}, 1, mixBool)

	vectorizeReturn("step", nil,
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat}, 1, step)
	vectorizeReturn("step", []ast.Type{nil, floatType, nil},
		[]ast.BareType{ast.TFloat, ast.TVoid, ast.TFloat}, 2, step)

	vectorizeReturn("smoothstep", nil,
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TFloat}, 1, smoothstep)
	vectorizeReturn("smoothstep", []ast.Type{nil, floatType, floatType, nil},
		[]ast.BareType{ast.TFloat, ast.TVoid, ast.TVoid, ast.TFloat}, 2, smoothstep)

	vectorizeReturn("isnan", nil, []ast.BareType{ast.TBool, ast.TFloat}, 1, isnan)
	vectorizeReturn("isinf", nil, []ast.BareType{ast.TBool, ast.TFloat}, 1, isinf)

	vectorizeReturn("floatBitsToInt", nil, []ast.BareType{ast.TInt, ast.TFloat}, 1, floatBitsToInt)
	vectorizeReturn("floatBitsToUint", nil, []ast.BareType{ast.TUint, ast.TFloat}, 1, floatBitsToUint)

	vectorizeReturn("intBitsToFloat", nil, []ast.BareType{ast.TFloat, ast.TInt}, 1, intBitsToFloat)
	vectorizeReturn("uintBitsToFloat", nil, []ast.BareType{ast.TFloat, ast.TUint}, 1, uintBitsToFloat)

	// 8.5 Geometric Functions (except cross, which is defined above)
	vectorize("length", []ast.Type{floatType, nil}, []ast.BareType{ast.TVoid, ast.TFloat}, 1, length)
	vectorize("distance", []ast.Type{floatType, nil, nil},
		[]ast.BareType{ast.TVoid, ast.TFloat, ast.TFloat}, 1, distance)
	vectorize("dot", []ast.Type{floatType, nil, nil},
		[]ast.BareType{ast.TVoid, ast.TFloat, ast.TFloat}, 1, dot)
	vectorize("normalize", []ast.Type{nil, nil}, []ast.BareType{ast.TFloat, ast.TFloat}, 1, normalize)
	vectorize("faceforward", []ast.Type{nil, nil, nil, nil},
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TFloat}, 1, faceforward)
	vectorize("reflect", []ast.Type{nil, nil, nil},
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat}, 1, reflect)
	vectorize("refract", []ast.Type{nil, nil, nil, floatType},
		[]ast.BareType{ast.TFloat, ast.TFloat, ast.TFloat, ast.TVoid}, 1, refract)

	// 8.6 Matrix Functions
	genMatrixCompMult()
	genOuterProduct()
	genTranspose()
	genDeterminantInverse()

	// 8.7 Vector Relational Functions
	genVectorComparison()
	vectorize("any", []ast.Type{&ast.BuiltinType{Type: ast.TBool}, nil},
		[]ast.BareType{ast.TVoid, ast.TBool}, 1, any)
	vectorize("all", []ast.Type{&ast.BuiltinType{Type: ast.TBool}, nil},
		[]ast.BareType{ast.TVoid, ast.TBool}, 1, all)
	vectorizeReturn("not", []ast.Type{nil, nil}, []ast.BareType{ast.TBool, ast.TBool}, 1, not)
}
