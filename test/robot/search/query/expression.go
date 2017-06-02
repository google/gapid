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

package query

import "github.com/google/gapid/test/robot/search"

func exprBool(value bool) *search.Expression {
	return &search.Expression{Is: &search.Expression_Boolean{
		Boolean: value,
	}}
}

func exprString(value string) *search.Expression {
	return &search.Expression{Is: &search.Expression_String_{
		String_: value,
	}}
}

func exprSigned(value int64) *search.Expression {
	return &search.Expression{Is: &search.Expression_Signed{
		Signed: value,
	}}
}

func exprUnsigned(value uint64) *search.Expression {
	return &search.Expression{Is: &search.Expression_Unsigned{
		Unsigned: value,
	}}
}

func exprDouble(value float64) *search.Expression {
	return &search.Expression{Is: &search.Expression_Double{
		Double: value,
	}}
}

func exprName(name string) *search.Expression {
	return &search.Expression{Is: &search.Expression_Name{
		Name: name,
	}}
}

func exprAnd(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_And{
		And: &search.Binary{
			Lhs: lhs,
			Rhs: rhs,
		},
	}}
}

func exprOr(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Or{
		Or: &search.Binary{
			Lhs: lhs,
			Rhs: rhs,
		},
	}}
}

func exprEqual(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Equal{
		Equal: &search.Binary{
			Lhs: lhs,
			Rhs: rhs,
		},
	}}
}

func exprLess(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_GreaterOrEqual{
		GreaterOrEqual: &search.Binary{
			Lhs: rhs,
			Rhs: lhs,
		},
	}}
}

func exprLessOrEqual(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Greater{
		Greater: &search.Binary{
			Lhs: rhs,
			Rhs: lhs,
		},
	}}
}

func exprGreater(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Greater{
		Greater: &search.Binary{
			Lhs: lhs,
			Rhs: rhs,
		},
	}}
}

func exprGreaterOrEqual(lhs, rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_GreaterOrEqual{
		GreaterOrEqual: &search.Binary{
			Lhs: lhs,
			Rhs: rhs,
		},
	}}
}

func exprSubscript(value, key *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Subscript{
		Subscript: &search.Subscript{
			Container: value,
			Key:       key,
		},
	}}
}

func exprRegex(value *search.Expression, pattern string) *search.Expression {
	return &search.Expression{Is: &search.Expression_Regex{
		Regex: &search.Regex{
			Value:   value,
			Pattern: pattern,
		},
	}}
}

func exprMember(object *search.Expression, name string) *search.Expression {
	return &search.Expression{Is: &search.Expression_Member{
		Member: &search.Member{
			Object: object,
			Name:   name,
		},
	}}
}

func exprNot(rhs *search.Expression) *search.Expression {
	return &search.Expression{Is: &search.Expression_Not{
		Not: rhs,
	}}
}

func exprValue(value interface{}) *search.Expression {
	switch v := value.(type) {
	case bool:
		return exprBool(v)
	case int:
		return exprSigned((int64)(v))
	case int64:
		return exprSigned(v)
	case uint:
		return exprUnsigned((uint64)(v))
	case uint64:
		return exprUnsigned(v)
	case float32:
		return exprDouble((float64)(v))
	case float64:
		return exprDouble(v)
	case string:
		return exprString(v)
	case Builder:
		return v.Expression()
	case *search.Expression:
		return v
	default:
		return exprName("*Invalid*")
	}
}
