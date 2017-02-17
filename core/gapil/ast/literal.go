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

// Number represents a typeless numeric constant.
type Number struct {
	Value string // the string representation of the constant
}

func (Number) isNode() {}

// Bool is used for the "true" and "false" keywords.
type Bool struct {
	Value bool // The value of the boolean
}

func (Bool) isNode() {}

// String represents a quoted string constant.
type String struct {
	Value string // The body of the string, not including the delimiters
}

func (String) isNode() {}

// Unknown represents the "?" construct. This is used in places where an
// expression takes a value that is implementation defined.
type Unknown struct {
	ignore bool // filed added so the instances get a unique address
}

func (Unknown) isNode() {}

// Null represents the null literal. This is the default value for the inferred
// type, and must be used in a context where the type can be inferred.
type Null struct {
	ignore bool // filed added so the instances get a unique address
}

func (Null) isNode() {}
