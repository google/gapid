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

import (
	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/test/robot/search"
)

// Replace substitues expr for match in the expression tree.
func (b Builder) Replace(match Builder, expr Builder) Builder {
	return Expression(replace(b.Expression(), match.Expression(), expr.Expression()))
}

// Set is a small helper on top of Replace for the common case of identifier substitution.
// name is a top level identifier to replace, and Value(value) is the expression to replace it with.
func (b Builder) Set(name string, value interface{}) Builder {
	return b.Replace(Name(name), Value(value))
}

// Using returns a function that when invoked returns a copy of b with Name(name) replace by Value(value).
// This is to make the common case of single value substitutions into precompiles queries easy.
// To use it
//    var myQuery = script.MustCompile("Object.Name == $V").Using("$V")
//    myService.Search(myQuery(nameToFind).Query())
func (b Builder) Using(name string) func(interface{}) Builder {
	return func(value interface{}) Builder { return b.Replace(Name(name), Value(value)) }
}

func replace(s, m, r *search.Expression) *search.Expression {
	if proto.Equal(s, m) {
		return r
	}
	switch e := s.Is.(type) {
	case *search.Expression_And:
		return exprAnd(replace(e.And.Lhs, m, r), replace(e.And.Rhs, m, r))
	case *search.Expression_Or:
		return exprOr(replace(e.Or.Lhs, m, r), replace(e.Or.Rhs, m, r))
	case *search.Expression_Equal:
		return exprEqual(replace(e.Equal.Lhs, m, r), replace(e.Equal.Rhs, m, r))
	case *search.Expression_Greater:
		return exprGreater(replace(e.Greater.Lhs, m, r), replace(e.Greater.Rhs, m, r))
	case *search.Expression_GreaterOrEqual:
		return exprGreaterOrEqual(replace(e.GreaterOrEqual.Lhs, m, r), replace(e.GreaterOrEqual.Rhs, m, r))
	case *search.Expression_Subscript:
		return exprSubscript(replace(e.Subscript.Container, m, r), replace(e.Subscript.Key, m, r))
	case *search.Expression_Regex:
		return exprRegex(replace(e.Regex.Value, m, r), e.Regex.Pattern)
	case *search.Expression_Member:
		return exprMember(replace(e.Member.Object, m, r), e.Member.Name)
	case *search.Expression_Not:
		return exprNot(replace(e.Not, m, r))
	default:
		return s
	}
}
