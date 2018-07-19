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

package compiler

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/compiler/mangling"
)

func (c *C) mangleInt(size int32, signed bool, name string) mangling.Type {
	switch size {
	case c.T.targetABI.MemoryLayout.Integer.Size:
		if signed {
			return mangling.Int
		}
		return mangling.UInt
		// Below are fall-back assumptions
	case 8:
		if signed {
			return mangling.Long
		}
		return mangling.ULong
	case 4:
		if signed {
			return mangling.Int
		}
		return mangling.UInt
	case 2:
		if signed {
			return mangling.Short
		}
		return mangling.UShort
	case 1:
		if signed {
			return mangling.Char
		}
		return mangling.UChar
	default:
		fail("Don't know how to mangle %s (size: %d, signed: %v)", name, size, signed)
		return nil
	}
}

// Mangle returns the mangling type for the given codegen type.
func (c *C) Mangle(ty codegen.Type) mangling.Type {
	if ty, ok := c.T.mangled[ty]; ok {
		return ty
	}
	switch ty {
	case c.T.Str:
		return &mangling.Class{Parent: c.Root, Name: "string"}
	case c.T.Sli:
		return &mangling.Class{Parent: c.Root, Name: "slice"}
	case c.T.Bool:
		return mangling.Bool
	case c.T.Uint8:
		return mangling.UChar
	case c.T.Int8:
		return mangling.SChar
	case c.T.Uint16:
		return mangling.UShort
	case c.T.Int16:
		return mangling.Short
	case c.T.Float32:
		return mangling.Float
	case c.T.Uint32:
		return mangling.UInt
	case c.T.Int32:
		return mangling.Int
	case c.T.Float64:
		return mangling.Double
	case c.T.Uint64:
		return mangling.ULong
	case c.T.Int64:
		return mangling.Long
	case c.T.Uintptr:
		return c.mangleInt(c.T.targetABI.MemoryLayout.Pointer.Size, false, "uint")
	case c.T.Size:
		return c.mangleInt(c.T.targetABI.MemoryLayout.Size.Size, false, "size")
	case c.T.Int:
		return c.mangleInt(c.T.targetABI.MemoryLayout.Integer.Size, true, "int")
	case c.T.Uint:
		return c.mangleInt(c.T.targetABI.MemoryLayout.Integer.Size, false, "uint")
	}
	switch ty := ty.(type) {
	case codegen.Pointer:
		return c.Mangle(ty.Element)
	}
	fail("Don't know how to mangle %T %v", ty, ty)
	return nil
}

// Method declares a function as a member of owner using the compiler's mangler.
func (c *C) Method(
	isConst bool,
	owner codegen.Type,
	retTy codegen.Type,
	name string,
	params ...codegen.Type) *codegen.Function {

	name = c.Mangler(&mangling.Function{
		Name:   name,
		Parent: c.Mangle(owner).(mangling.Scope),
		Parameters: []mangling.Type{
			mangling.Pointer{To: mangling.Void},
			mangling.Bool,
		},
		Const: isConst,
	})

	return c.M.Function(retTy, name,
		append([]codegen.Type{c.T.Pointer(owner)}, params...)...)
}
