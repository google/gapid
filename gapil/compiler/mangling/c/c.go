// Copyright (C) 2018 Google Inc.
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

// Package c implements a basic symbol mangling for that is compatible with C.
package c

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/gapil/compiler/mangling"
)

// Mangle returns the entity mangled conforming to the IA64 ABI.
func Mangle(s mangling.Entity) string {
	m := mangler{bytes.Buffer{}}
	m.mangle(s)
	return m.String()
}

type mangler struct {
	bytes.Buffer
}

func (m *mangler) mangle(v mangling.Entity) {
	if v, ok := v.(mangling.Scoped); ok {
		if s := v.Scope(); s != nil {
			m.mangle(s)
			m.WriteString("__")
		}
	}
	switch v := v.(type) {
	case mangling.Builtin:
		switch v {
		case mangling.WChar:
			m.WriteString("wchar")
		case mangling.Bool:
			m.WriteString("bool")
		case mangling.Char:
			m.WriteString("char")
		case mangling.SChar:
			m.WriteString("schar")
		case mangling.UChar:
			m.WriteString("uchar")
		case mangling.Short:
			m.WriteString("short")
		case mangling.UShort:
			m.WriteString("ushort")
		case mangling.Int:
			m.WriteString("int")
		case mangling.UInt:
			m.WriteString("uint")
		case mangling.Long:
			m.WriteString("long")
		case mangling.ULong:
			m.WriteString("ulong")
		case mangling.S64:
			m.WriteString("s64")
		case mangling.U64:
			m.WriteString("u64")
		case mangling.Float:
			m.WriteString("float")
		case mangling.Double:
			m.WriteString("double")
		default:
			panic(fmt.Errorf("Unhandled builtin type: %v", v))
		}

	case mangling.Pointer:
		m.mangle(v.To)
		m.WriteString("_ptr")

	case mangling.Named:
		m.WriteString(v.GetName())

	default:
		panic(fmt.Errorf("Unhandled type: %T", v))
	}

	if v, ok := v.(mangling.Templated); ok {
		for _, ty := range v.TemplateArguments() {
			m.WriteString("_")
			m.mangle(ty)
		}
	}
}
