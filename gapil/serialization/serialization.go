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

// Package serialization contains constants and utility methods used to
// serialize .gfxtrace files.
package serialization

import (
	"fmt"

	"github.com/google/gapid/gapil/semantic"
)

// ProtoFieldID is a field identifier
type ProtoFieldID int

// These are the proto field identifiers for gapil proto messages.
const (
	StateStart      = ProtoFieldID(1)
	CmdThread       = ProtoFieldID(1)
	CmdResult       = ProtoFieldID(1)
	CmdFieldStart   = ProtoFieldID(8)
	ClassFieldStart = ProtoFieldID(1)
	MapRef          = ProtoFieldID(1)
	MapVal          = ProtoFieldID(2)
	MapKey          = ProtoFieldID(3)
	RefRef          = ProtoFieldID(1)
	RefVal          = ProtoFieldID(2)
	SliceRoot       = ProtoFieldID(1)
	SliceBase       = ProtoFieldID(2)
	SliceSize       = ProtoFieldID(3)
	SliceCount      = ProtoFieldID(4)
	SlicePool       = ProtoFieldID(5)
)

// IsEncodable returns true if the node is encodable.
func IsEncodable(n semantic.Node) bool {
	switch n := n.(type) {
	case *semantic.Global:
		return n.Annotations.GetAnnotation("serialize") != nil
	default:
		// Not special cased, presume true.
		return true
	}
}

// ProtoTypeName returns the proto type name for the given type.
func ProtoTypeName(ty semantic.Type) string {
	switch {
	case ty == semantic.BoolType:
		return "sint32"
	case ty == semantic.IntType:
		return "sint64"
	case ty == semantic.UintType:
		return "sint64"
	case ty == semantic.CharType:
		return "sint32"
	case ty == semantic.Float32Type:
		return "float"
	case ty == semantic.Float64Type:
		return "double"
	case ty == semantic.StringType:
		return "string"
	case semantic.IsNumeric(ty): // Must be after specializations above
		return "sint64"
	}

	switch ty := ty.(type) {
	case *semantic.Enum:
		return "sint64"
	case *semantic.Pointer:
		return "sint64"
	case *semantic.Reference:
		return ProtoTypeName(ty.To) + "_ref"
	case *semantic.Slice:
		return "memory.Slice"
	case *semantic.Class:
		return ty.Name()
	case *semantic.Map:
		return fmt.Sprintf("%v_to_%v_map", ProtoTypeName(ty.KeyType), ProtoTypeName(ty.ValueType))
	case *semantic.Pseudonym:
		return ProtoTypeName(ty.To)
	}

	panic(fmt.Sprintf("Unsupported type: %v %T", ty, ty))
}
