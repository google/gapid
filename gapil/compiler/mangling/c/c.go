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
	m.WriteString(v.(mangling.Named).GetName())
}
