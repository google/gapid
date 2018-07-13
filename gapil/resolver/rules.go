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

package resolver

import "github.com/google/gapid/gapil/semantic"

func implicit(lhs semantic.Type, rhs semantic.Type) bool {
	if lhs == semantic.AnyType {
		return true
	}
	return false
}

func assignable(lhs semantic.Type, rhs semantic.Type) bool {
	if isInvalid(lhs) || isInvalid(rhs) {
		return true // Don't snowball errors.
	}
	if isVoid(lhs) || isVoid(rhs) {
		return false
	}
	if equal(lhs, rhs) {
		return true
	}
	if implicit(lhs, rhs) {
		return true
	}
	if implicit(rhs, lhs) {
		return true
	}
	return false
}

func comparable(lhs semantic.Type, rhs semantic.Type) bool {
	if isVoid(lhs) || isVoid(rhs) {
		return false
	}
	if equal(lhs, rhs) {
		return true
	}
	if implicit(lhs, rhs) {
		return true
	}
	return implicit(rhs, lhs)
}

func equal(lhs semantic.Type, rhs semantic.Type) bool {
	return lhs == rhs
}

func isNumber(t semantic.Type) bool {
	switch t {
	case semantic.IntType, semantic.UintType,
		semantic.Int8Type, semantic.Uint8Type,
		semantic.Int16Type, semantic.Uint16Type,
		semantic.Int32Type, semantic.Uint32Type,
		semantic.Int64Type, semantic.Uint64Type,
		semantic.Float32Type, semantic.Float64Type,
		semantic.SizeType:
		return true
	default:
		return false
	}
}

func isInteger(t semantic.Type) bool {
	switch t {
	case semantic.IntType, semantic.UintType,
		semantic.Int8Type, semantic.Uint8Type,
		semantic.Int16Type, semantic.Uint16Type,
		semantic.Int32Type, semantic.Uint32Type,
		semantic.Int64Type, semantic.Uint64Type,
		semantic.SizeType:
		return true
	default:
		return false
	}
}

func isUnsignedInteger(t semantic.Type) bool {
	switch t {
	case semantic.UintType,
		semantic.Uint8Type,
		semantic.Uint16Type,
		semantic.Uint32Type,
		semantic.Uint64Type,
		semantic.SizeType:
		return true
	default:
		return false
	}
}

func castable(from semantic.Type, to semantic.Type) bool {
	fromBase := semantic.Underlying(from)
	toBase := semantic.Underlying(to)
	if assignable(toBase, fromBase) {
		return true
	}
	_, fromIsEnum := fromBase.(*semantic.Enum)
	_, toIsEnum := toBase.(*semantic.Enum)
	fromIsNumber, toIsNumber := isNumber(fromBase), isNumber(toBase)
	if fromIsEnum && toIsEnum {
		return true // enum -> enum
	}
	if fromIsEnum && toIsNumber {
		return true // enum -> number
	}
	if fromIsNumber && toIsEnum {
		return true // number -> enum
	}
	_, fromIsPointer := fromBase.(*semantic.Pointer)
	if fromIsPointer && toIsNumber {
		return true // pointer -> number
	}
	if fromIsNumber && toIsNumber {
		return true // any numeric conversion
	}
	fromIsBool, toIsBool := fromBase == semantic.BoolType, toBase == semantic.BoolType
	if fromIsBool && toIsNumber {
		return true // bool -> number
	}
	if fromIsNumber && toIsBool {
		return true // number -> bool
	}
	fromPointer, fromIsPointer := fromBase.(*semantic.Pointer)
	toPointer, toIsPointer := toBase.(*semantic.Pointer)
	if fromIsPointer && toIsPointer { // A* -> B*
		return true
	}
	fromSlice, fromIsSlice := semantic.Underlying(from).(*semantic.Slice)
	toSlice, toIsSlice := semantic.Underlying(to).(*semantic.Slice)
	if fromIsSlice && toIsSlice { // A[] -> B[]
		return true
	}
	if fromIsSlice && toIsPointer && fromSlice.To == toPointer.To { // T[] -> T*
		return equal(fromSlice.To, toPointer.To)
	}
	if fromIsPointer && fromPointer.To == semantic.CharType && to == semantic.StringType { // char* -> string
		return true
	}
	if fromIsSlice && fromSlice.To == semantic.CharType && to == semantic.StringType { // char[] -> string
		return true
	}
	if toIsSlice && toSlice.To == semantic.CharType && from == semantic.StringType { // string -> char[]
		return true
	}
	return false
}

func isInvalid(n semantic.Node) bool {
	if n == semantic.InvalidType {
		return true
	}
	_, invalid := n.(semantic.Invalid)
	return invalid
}

func isVoid(t semantic.Type) bool {
	return semantic.Underlying(t) == semantic.VoidType
}

func isLegalCommandParameterType(t semantic.Type) bool {
	switch semantic.Underlying(t) {
	case semantic.AnyType:
	case semantic.StringType:
		return false
	}
	switch t.(type) {
	case *semantic.Slice:
		return false
	}
	return true
}
