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

// Builder is the type used to allow fluent construction of search queries.
type Builder struct {
	e *search.Expression
}

// Expression creates a builder from a search expression.
func Expression(e *search.Expression) Builder {
	return Builder{e: e}
}

// Value creates a builder from a value.
// The type of expression will be inferred from the type of the value.
func Value(value interface{}) Builder {
	return Expression(exprValue(value))
}

// Expression returns the content of the builder as a search expression.
func (b Builder) Expression() *search.Expression {
	return b.e
}

// Query returns the content of the builder as a completed search query.
func (b Builder) Query() *search.Query {
	return &search.Query{Expression: b.Expression()}
}

// Bool builds a boolean literal search expression.
func Bool(value bool) Builder {
	return Expression(exprBool(value))
}

// String builds a string literal search expression.
func String(value string) Builder {
	return Expression(exprString(value))
}

// Signed builds a signed integer literal search expression.
func Signed(value int64) Builder {
	return Expression(exprSigned(value))
}

// Unsigned builds an unsigned integer literal search expression.
func Unsigned(value uint64) Builder {
	return Expression(exprUnsigned(value))
}

// Double builds an floating point literal search expression.
func Double(value float64) Builder {
	return Expression(exprDouble(value))
}

// Name builds a root name lookup search expression.
func Name(name string) Builder {
	return Expression(exprName(name))
}

// And builds a search expression that is the "and" of the lhs and rhs.
func (lhs Builder) And(rhs Builder) Builder {
	return Expression(exprAnd(lhs.Expression(), rhs.Expression()))
}

// Or builds a search expression that is the "or" of the lhs and rhs.
func (lhs Builder) Or(rhs Builder) Builder {
	return Expression(exprOr(lhs.Expression(), rhs.Expression()))
}

// Equal builds a search expression that compares the lhs and rhs for equality.
func (lhs Builder) Equal(rhs Builder) Builder {
	return Expression(exprEqual(lhs.Expression(), rhs.Expression()))
}

// Less builds a search expression that tests whether the lhs is less than the rhs.
func (lhs Builder) Less(rhs Builder) Builder {
	return Expression(exprLess(lhs.Expression(), rhs.Expression()))
}

// LessOrEqual builds a search expression that tests whether the lhs is less than or equal to the rhs.
func (lhs Builder) LessOrEqual(rhs Builder) Builder {
	return Expression(exprLessOrEqual(lhs.Expression(), rhs.Expression()))
}

// Greater builds a search expression that tests whether the lhs is greater than the rhs.
func (lhs Builder) Greater(rhs Builder) Builder {
	return Expression(exprGreater(lhs.Expression(), rhs.Expression()))
}

// GreaterOrEqual builds a search expression that tests whether the lhs is greater than or equal to the rhs.
func (lhs Builder) GreaterOrEqual(rhs Builder) Builder {
	return Expression(exprGreaterOrEqual(lhs.Expression(), rhs.Expression()))
}

// Subscript builds a search expression that applies the key as a subscript to the value.
func (value Builder) Subscript(key Builder) Builder {
	return Expression(exprSubscript(value.Expression(), key.Expression()))
}

// Regex builds a search expression that tests whether value matches the supplied regex pattern.
func (value Builder) Regex(pattern string) Builder {
	return Expression(exprRegex(value.Expression(), pattern))
}

// Member builds a search expression that looks up the named member of the object.
func (object Builder) Member(name string) Builder {
	return Expression(exprMember(object.Expression(), name))
}

// Not builds a search expression that applies a boolean not to the supplied rhs.
func Not(rhs Builder) Builder {
	return Expression(exprNot(rhs.Expression()))
}
