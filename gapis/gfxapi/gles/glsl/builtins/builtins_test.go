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

package builtins

import (
	"math"
	"testing"

	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

const fudge = 2e-5

// Compares values, just like ast.ValueEquals, but for testing, we use a fudge factor when
// comparing floating point values
func valueEqualsErr(left, right ast.Value) bool {
	switch left := left.(type) {
	case ast.IntValue, ast.UintValue, ast.BoolValue:
		return left == right

	case ast.FloatValue:
		lfloat, rfloat := float64(left), float64(right.(ast.FloatValue))
		return math.Abs(lfloat-rfloat) <= fudge

	case ast.VectorValue:
		right, ok := right.(ast.VectorValue)
		if !ok || len(left.Members) != len(right.Members) {
			return false
		}
		for i := range left.Members {
			if !valueEqualsErr(left.Members[i], right.Members[i]) {
				return false
			}
		}
		return true

	case ast.MatrixValue:
		right, ok := right.(ast.MatrixValue)
		if !ok || len(left.Members) != len(right.Members) {
			return false
		}
		for i := range left.Members {
			if len(left.Members[i]) != len(right.Members[i]) {
				return false
			}
			for j := range left.Members[i] {
				if !valueEqualsErr(left.Members[i][j], right.Members[i][j]) {
					return false
				}
			}
		}
		return true

	default:
		return false
	}
}

func findFunction(t *testing.T, name string, ret ast.BareType,
	signature []ast.BareType) func(v []ast.Value) ast.Value {

NEXT:
	for _, f := range BuiltinFunctions {
		if name != f.SymName || len(signature) != len(f.Params) {
			continue
		}
		for i := range signature {
			if signature[i] != f.Params[i].Type().(*ast.BuiltinType).Type {
				continue NEXT
			}
		}
		if ret != f.RetType.(*ast.BuiltinType).Type {
			t.Errorf("Function '%s%v' declared with unexpected result type. Expected: %q, got: %q.",
				name, signature, ret, f.RetType.(*ast.BuiltinType).Type)
			return nil
		}
		return f.Impl
	}
	t.Errorf("Function '%s%v' not found.", name, signature)
	return nil
}

var builtinFunctionTests = []struct {
	name      string
	arguments []ast.Value
	result    ast.Value
}{
	{"radians", []ast.Value{ast.FloatValue(180.0)}, ast.FloatValue(math.Pi)},

	{"radians",
		[]ast.Value{ast.NewVectorValue(ast.TVec4,
			func(i uint8) ast.Value { return ast.FloatValue(i) * 180.0 })},
		ast.NewVectorValue(ast.TVec4,
			func(i uint8) ast.Value { return ast.FloatValue(i) * math.Pi }),
	},

	{"degrees", []ast.Value{ast.FloatValue(math.Pi)}, ast.FloatValue(180.0)},
	{"sin", []ast.Value{ast.FloatValue(math.Pi)}, ast.FloatValue(0.0)},
	{"cos", []ast.Value{ast.FloatValue(math.Pi)}, ast.FloatValue(-1.0)},
	{"tan", []ast.Value{ast.FloatValue(math.Pi)}, ast.FloatValue(0.0)},
	{"asin", []ast.Value{ast.FloatValue(1.0)}, ast.FloatValue(math.Pi / 2.0)},
	{"acos", []ast.Value{ast.FloatValue(1.0)}, ast.FloatValue(0.0)},
	{"atan", []ast.Value{ast.FloatValue(1.0)}, ast.FloatValue(math.Pi / 4.0)},

	{"atan", []ast.Value{ast.FloatValue(-math.Pi), ast.FloatValue(-math.Pi)},
		ast.FloatValue(math.Pi/4.0 - math.Pi)},

	{"sinh", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(0.0)},
	{"cosh", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(1.0)},
	{"tanh", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(0.0)},
	{"asinh", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(0.0)},
	{"acosh", []ast.Value{ast.FloatValue(1.0)}, ast.FloatValue(0.0)},
	{"atanh", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(0.0)},

	{"pow", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0)}, ast.FloatValue(8.0)},
	{"exp", []ast.Value{ast.FloatValue(1.0)}, ast.FloatValue(math.E)},
	{"log", []ast.Value{ast.FloatValue(math.E)}, ast.FloatValue(1.0)},
	{"exp2", []ast.Value{ast.FloatValue(3.0)}, ast.FloatValue(8.0)},
	{"log2", []ast.Value{ast.FloatValue(8.0)}, ast.FloatValue(3.0)},
	{"sqrt", []ast.Value{ast.FloatValue(4.0)}, ast.FloatValue(2.0)},
	{"inversesqrt", []ast.Value{ast.FloatValue(4.0)}, ast.FloatValue(0.5)},

	{"abs", []ast.Value{ast.FloatValue(-1.0)}, ast.FloatValue(1.0)},
	{"abs", []ast.Value{ast.IntValue(-1)}, ast.IntValue(1)},
	{"abs", []ast.Value{ast.IntValue(1)}, ast.IntValue(1)},
	{"sign", []ast.Value{ast.FloatValue(-2.0)}, ast.FloatValue(-1.0)},
	{"sign", []ast.Value{ast.FloatValue(2.0)}, ast.FloatValue(1.0)},
	{"sign", []ast.Value{ast.FloatValue(0.0)}, ast.FloatValue(0.0)},
	{"sign", []ast.Value{ast.IntValue(-2)}, ast.IntValue(-1)},
	{"sign", []ast.Value{ast.IntValue(2)}, ast.IntValue(1)},
	{"sign", []ast.Value{ast.IntValue(0)}, ast.IntValue(0)},
	{"floor", []ast.Value{ast.FloatValue(-2.5)}, ast.FloatValue(-3.0)},
	{"trunc", []ast.Value{ast.FloatValue(-2.9)}, ast.FloatValue(-2)},
	{"round", []ast.Value{ast.FloatValue(-1.4)}, ast.FloatValue(-1.0)},
	{"roundEven", []ast.Value{ast.FloatValue(-1.5)}, ast.FloatValue(-2.0)},
	{"roundEven", []ast.Value{ast.FloatValue(-2.5)}, ast.FloatValue(-2.0)},
	{"roundEven", []ast.Value{ast.FloatValue(-2.4)}, ast.FloatValue(-2.0)},
	{"ceil", []ast.Value{ast.FloatValue(-1.5)}, ast.FloatValue(-1.0)},
	{"fract", []ast.Value{ast.FloatValue(-1.5)}, ast.FloatValue(0.5)},

	{"mod", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0)}, ast.FloatValue(2.0)},
	{"min", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0)}, ast.FloatValue(2.0)},
	{"min", []ast.Value{ast.IntValue(2.0), ast.IntValue(3)}, ast.IntValue(2)},
	{"min", []ast.Value{ast.UintValue(2.0), ast.UintValue(3)}, ast.UintValue(2)},
	{"max", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0)}, ast.FloatValue(3.0)},
	{"max", []ast.Value{ast.IntValue(2.0), ast.IntValue(3)}, ast.IntValue(3)},
	{"max", []ast.Value{ast.UintValue(2.0), ast.UintValue(3)}, ast.UintValue(3)},

	{"clamp", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.FloatValue(5.0)},
		ast.FloatValue(3.0)},
	{"clamp", []ast.Value{ast.IntValue(6), ast.IntValue(3), ast.IntValue(5)}, ast.IntValue(5)},
	{"clamp", []ast.Value{ast.UintValue(6), ast.UintValue(3), ast.UintValue(5)}, ast.UintValue(5)},

	{"mix", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.FloatValue(0.5)},
		ast.FloatValue(2.5)},
	{"mix", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.BoolValue(false)},
		ast.FloatValue(2.0)},
	{"mix", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.BoolValue(true)},
		ast.FloatValue(3.0)},
	{"step", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(1.0)}, ast.FloatValue(0.0)},
	{"step", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0)}, ast.FloatValue(1.0)},
	{"smoothstep", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.FloatValue(1.5)},
		ast.FloatValue(0)},
	{"smoothstep", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.FloatValue(2.5)},
		ast.FloatValue(0.5)},
	{"smoothstep", []ast.Value{ast.FloatValue(2.0), ast.FloatValue(3.0), ast.FloatValue(3.5)},
		ast.FloatValue(1)},
	{"isnan", []ast.Value{ast.FloatValue(-1.5)}, ast.BoolValue(false)},
	{"isinf", []ast.Value{ast.FloatValue(-1.5)}, ast.BoolValue(false)},

	{"floatBitsToInt", []ast.Value{ast.FloatValue(1.0)}, ast.IntValue(0x3f800000)},
	{"floatBitsToUint", []ast.Value{ast.FloatValue(-1.0)}, ast.UintValue(0xbf800000)},
	{"intBitsToFloat", []ast.Value{ast.IntValue(0x3f800000)}, ast.FloatValue(1.0)},
	{"uintBitsToFloat", []ast.Value{ast.UintValue(0xbf800000)}, ast.FloatValue(-1.0)},

	{"packSnorm2x16",
		[]ast.Value{ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i+1) * 0.1 })},
		ast.UintValue(0x19990ccd)},
	{"unpackSnorm2x16",
		[]ast.Value{ast.UintValue(0x19990ccd)},
		ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i+1) * 0.1 })},
	{"packUnorm2x16",
		[]ast.Value{ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i+1) * 0.1 })},
		ast.UintValue(0x3333199a)},
	{"unpackUnorm2x16",
		[]ast.Value{ast.UintValue(0x3333199a)},
		ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i+1) * 0.1 })},
	{"packHalf2x16",
		[]ast.Value{ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i + 1) })},
		ast.UintValue(0x40003c00)},
	{"unpackHalf2x16",
		[]ast.Value{ast.UintValue(0x40003c00)},
		ast.NewVectorValue(ast.TVec2,
			func(i uint8) ast.Value { return ast.FloatValue(i + 1) })},

	{"length",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		},
		ast.FloatValue(math.Sqrt(5.0))},
	{"distance",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
		},
		ast.FloatValue(math.Sqrt(3.0))},
	{"dot",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
		},
		ast.FloatValue(8.0)},
	{"cross",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
		},
		ast.NewVectorValue(ast.TVec3,
			func(i uint8) ast.Value { return []ast.FloatValue{-1, 2, -1}[i] })},
	{"normalize",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		},
		ast.NewVectorValue(ast.TVec3,
			func(i uint8) ast.Value { return ast.FloatValue(i) / ast.FloatValue(math.Sqrt(5)) })},
	{"faceforward",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 2) }),
		},
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return -ast.FloatValue(i) })},
	{"faceforward",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return -ast.FloatValue(i + 2) }),
		},
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) })},
	{"reflect",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
		},
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return -16 - 15*ast.FloatValue(i) })},
	{"refract",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
			ast.FloatValue(0.5),
		},
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value {
			return -7.0/2.0*ast.FloatValue(i) - 4.0 -
				ast.FloatValue(math.Sqrt(16.75))*ast.FloatValue(i+1)
		})},
	{"refract",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value {
				return ast.FloatValue(i) / 10
			}),
			ast.FloatValue(5),
		},
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(0.0) })},

	{"matrixCompMult",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
				return ast.FloatValue(i + j)
			}),
			ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
				return ast.FloatValue(i * j)
			}),
		},
		ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
			return ast.FloatValue((i + j) * i * j)
		})},
	{"outerProduct",
		[]ast.Value{
			ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
			ast.NewVectorValue(ast.TVec2, func(i uint8) ast.Value { return ast.FloatValue(i + 1) }),
		},
		ast.NewMatrixValue(ast.TMat3x2, func(i, j uint8) ast.Value {
			return ast.FloatValue(i * (j + 1))
		})},
	{"transpose",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat3x4, func(i, j uint8) ast.Value {
				return ast.FloatValue(i * j)
			}),
		},
		ast.NewMatrixValue(ast.TMat4x3, func(i, j uint8) ast.Value {
			return ast.FloatValue(j * i)
		})},
	{"determinant",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat2, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{1, 2}, {3, 4},
				}[i][j]
			}),
		},
		ast.FloatValue(-2)},
	{"determinant",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{1, 2, 3}, {0, 1, 1}, {2, -2, 1},
				}[i][j]
			}),
		},
		ast.FloatValue(1)},
	{"determinant",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat4, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{0, 1, 2, 3}, {-1, 1, 3, 1}, {1, 0, 1, 0}, {1, -1, 1, -1},
				}[i][j]
			}),
		},
		ast.FloatValue(8)},
	{"inverse",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat2, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{1, 2}, {3, 4},
				}[i][j]
			}),
		},
		ast.NewMatrixValue(ast.TMat2, func(i, j uint8) ast.Value {
			return [][]ast.FloatValue{
				{-2, 1}, {3. / 2, -1. / 2},
			}[i][j]
		})},
	{"inverse",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{1, 2, 3}, {0, 1, 1}, {2, -2, 1},
				}[i][j]
			}),
		},
		ast.NewMatrixValue(ast.TMat3, func(i, j uint8) ast.Value {
			return [][]ast.FloatValue{
				{3, -8, -1}, {2, -5, -1}, {-2, 6, 1},
			}[i][j]
		})},
	{"inverse",
		[]ast.Value{
			ast.NewMatrixValue(ast.TMat4, func(i, j uint8) ast.Value {
				return [][]ast.FloatValue{
					{0, 1, 2, 3}, {-1, 1, 3, 1}, {1, 0, 1, 0}, {1, -1, 1, -1},
				}[i][j]
			}),
		},
		ast.NewMatrixValue(ast.TMat4, func(i, j uint8) ast.Value {
			return [][]ast.FloatValue{
				{0, -1, 4, -1}, {-2, 1, 6, -5}, {0, 1, 0, 1}, {2, -1, -2, 1},
			}[i][j] / 4
		})},

	{"lessThan", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) })},
	{"lessThan", []ast.Value{
		ast.NewVectorValue(ast.TUvec3, func(i uint8) ast.Value { return ast.UintValue(i) }),
		ast.NewVectorValue(ast.TUvec3, func(i uint8) ast.Value { return ast.UintValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) })},
	{"lessThan", []ast.Value{
		ast.NewVectorValue(ast.TIvec3, func(i uint8) ast.Value { return ast.IntValue(i) }),
		ast.NewVectorValue(ast.TIvec3, func(i uint8) ast.Value { return ast.IntValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) })},
	{"lessThanEqual", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(true) })},
	{"greaterThan", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) })},
	{"greaterThanEqual", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(true) })},
	{"equal", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(true) })},
	{"notEqual", []ast.Value{
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
		ast.NewVectorValue(ast.TVec3, func(i uint8) ast.Value { return ast.FloatValue(i) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) })},
	{"any", []ast.Value{
		ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(i != 0) }),
	}, ast.BoolValue(true)},
	{"any", []ast.Value{
		ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(false) }),
	}, ast.BoolValue(false)},
	{"all", []ast.Value{
		ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(i != 0) }),
	}, ast.BoolValue(false)},
	{"all", []ast.Value{
		ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(true) }),
	}, ast.BoolValue(true)},
	{"not", []ast.Value{
		ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(i != 0) }),
	}, ast.NewVectorValue(ast.TBvec3, func(i uint8) ast.Value { return ast.BoolValue(i == 0) })},
}

func TestBuiltinFunctions(t *testing.T) {
	for _, test := range builtinFunctionTests {
		arguments := make([]ast.BareType, len(test.arguments))
		for i, arg := range test.arguments {
			arguments[i] = arg.Type().(*ast.BuiltinType).Type
		}
		result := test.result.Type().(*ast.BuiltinType).Type

		f := findFunction(t, test.name, result, arguments)
		if f == nil {
			continue
		}
		t.Logf("Calling %s%v", test.name, test.arguments)
		got := f(test.arguments)
		if !valueEqualsErr(got, test.result) {
			t.Errorf("Function %s%v returned unexpected result. Expected: %v, got: %v.",
				test.name, test.arguments, test.result, got)
		}
	}
}

func TestModf(t *testing.T) {
	argTypes := []ast.BareType{ast.TFloat, ast.TFloat}
	resType := ast.TFloat
	f := findFunction(t, "modf", resType, argTypes)
	if f == nil {
		return
	}
	arguments := []ast.Value{ast.FloatValue(1.25), ast.FloatValue(0.0)}
	result := f(arguments)
	if !valueEqualsErr(result, ast.FloatValue(0.25)) {
		t.Errorf("modf%v returned %v, but %v was expected.", arguments, result, ast.FloatValue(0.25))
	}
	if !valueEqualsErr(arguments[1], ast.FloatValue(1)) {
		t.Errorf("Out argument of modf%v was %v, but %v was expected.", arguments,
			arguments[1], ast.FloatValue(1))
	}
}

func TestModfVec(t *testing.T) {
	argTypes := []ast.BareType{ast.TVec4, ast.TVec4}
	resType := ast.TVec4
	f := findFunction(t, "modf", resType, argTypes)
	if f == nil {
		return
	}
	arguments := []ast.Value{
		ast.NewVectorValue(ast.TVec4, func(i uint8) ast.Value { return ast.FloatValue(i) * 1.1 }),
		ast.NewVectorValue(ast.TVec4, func(i uint8) ast.Value { return ast.FloatValue(0.0) }),
	}

	result := f(arguments)
	expectedResult := ast.NewVectorValue(ast.TVec4,
		func(i uint8) ast.Value { return ast.FloatValue(i) * 0.1 })
	if !valueEqualsErr(result, expectedResult) {
		t.Errorf("modf%v returned %v, but %v was expected.", arguments, result, expectedResult)
	}

	expectedOut := ast.NewVectorValue(ast.TVec4, func(i uint8) ast.Value { return ast.FloatValue(i) })
	if !valueEqualsErr(arguments[1], expectedOut) {
		t.Errorf("Out argument of modf%v was %v, but %v was expected.", arguments,
			arguments[1], expectedOut)
	}
}
