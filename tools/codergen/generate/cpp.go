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

package generate

import (
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/text/copyright"
)

const (
	RefSuffix     = "ʳ"
	SliceSuffix   = "ˢ"
	ConstSuffix   = "ᶜ"
	PointerSuffix = "ᵖ"
	ArraySuffix   = "ᵃ"
	MapSuffix     = "ᵐ"
	TypeInfix     = "ː"
)

// CppNamespace is the struct handed to the Cpp.File template.
type CppNamespace struct {
	*Module                        // The go module to generate cpp binary coders for.
	Namespace  string              // The name to use for the cpp namespace itself.
	Copyright  string              // The copyright header to put on the file.
	IncludeSet map[string]struct{} // Namespaces to include as set (from Include())
	Includes   []string            // Namespaces to include as slice.
}

// Called by codergen to prepare and generate cpp code for a given module.
func Cpp(m *Module, info copyright.Info, gen chan Generate, path string) {
	namespace := m.Directives["cpp"]

	// Remove all the structs tagged with `cpp:"disable"`
	m = m.ShallowClone()
	m.Structs = m.Structs.Filter(func(s *Struct) bool { return s.Tags.Get("cpp") != "disable" })

	gen <- Generate{
		Name: "Cpp.Header",
		Arg: &CppNamespace{
			Module:     m,
			Namespace:  namespace,
			Copyright:  strings.TrimSpace(copyright.Build("generated_by", info)),
			IncludeSet: make(map[string]struct{}),
		},
		Output: filepath.Join(path, namespace+".h"),
		Indent: "    ",
	}
	gen <- Generate{
		Name: "Cpp.File",
		Arg: &CppNamespace{
			Module:     m,
			Namespace:  namespace,
			Copyright:  strings.TrimSpace(copyright.Build("generated_by", info)),
			IncludeSet: make(map[string]struct{}),
		},
		Output: filepath.Join(path, namespace+".cpp"),
		Indent: "    ",
	}
}

// Converts a typename to cpp form by replacing the unicode characters.
func (*CppNamespace) TypeName(n string) string {
	n = strings.Replace(n, ".", "::", -1)
	n = strings.Replace(n, ConstSuffix+PointerSuffix, "__CP", -1)
	n = strings.Replace(n, PointerSuffix, "__P", -1)
	n = strings.Replace(n, SliceSuffix, "__S", -1)
	n = strings.Replace(n, ArraySuffix, "__A", -1)
	n = strings.Replace(n, TypeInfix, "__", -1)
	return n
}

func (n *CppNamespace) ModuleOf(v interface{}) *Module {
	m, _ := n.ModuleAndName(v)
	return m
}

// Add a namespace to the headers to include
func (n *CppNamespace) Include(namespace string) string {
	if namespace != n.Namespace {
		if _, ok := n.IncludeSet[namespace]; !ok {
			n.Includes = append(n.Includes, namespace)
			n.IncludeSet[namespace] = struct{}{}
		}
	}
	return ""
}
