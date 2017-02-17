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
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"golang.org/x/tools/go/types"
)

const (
	binaryPackage  = "github.com/google/gapid/framework/binary"
	binaryGenerate = binaryPackage + ".Generate"
	schemaPackage  = "github.com/google/gapid/framework/binary/schema"
	binaryFrozen   = binaryPackage + ".Frozen"
)

func findType(pkg *types.Package, module string, typename string) types.Type {
	for _, p := range pkg.Imports() {
		if p.Path() == module {
			if o := p.Scope().Lookup(typename); o != nil {
				return o.Type().Underlying()
			}
		}
	}
	return nil
}

func findInterface(pkg *types.Package, module string, typename string) *types.Interface {
	t := findType(pkg, module, typename)
	if t == nil {
		return nil
	}
	return t.(*types.Interface)
}

func findStruct(pkg *types.Package, module string, typename string) *types.Struct {
	t := findType(pkg, module, typename)
	if t == nil {
		return nil
	}
	return t.(*types.Struct)
}

func spaceToUnderscore(r rune) rune {
	if unicode.IsSpace(r) {
		return '_'
	}
	return r
}

// fromType creates a appropriate binary.Type object from a types.Type.
func (m *Module) fromType(from types.Type, s *Struct, tags Tags) binary.Type {
	pkg := m.Source.Types
	alias := ""
	// When from is in a different package to pkg then fullname is the
	// fully-qualified name including full package path, otherwise it
	// is the relative name.
	fullname := types.TypeString(from, types.RelativeTo(pkg))
	name := strings.Map(spaceToUnderscore, path.Base(fullname))
	if named, isNamed := from.(*types.Named); isNamed {
		alias = name
		from = from.Underlying()
		p := named.Obj().Pkg()
		if p != nil && p != pkg {
			m.Imports.Add(Import{Name: p.Name(), Path: p.Path()})
		}
	}
	gotype := strings.Map(spaceToUnderscore, from.String())
	switch from := from.(type) {
	case *types.Basic:
		switch from.Kind() {
		case types.Int:
			return &schema.Primitive{Name: name, Method: schema.Int32}
		case types.Byte:
			return &schema.Primitive{Name: name, Method: schema.Uint8}
		case types.Rune:
			return &schema.Primitive{Name: name, Method: schema.Int32}
		default:
			m, err := schema.ParseMethod(strings.Title(gotype))
			if err != nil {
				return &schema.Primitive{Name: fmt.Sprintf("%s_bad_%s", name, gotype), Method: schema.String}
			}
			return &schema.Primitive{Name: name, Method: m}
		}
	case *types.Pointer:
		return &schema.Pointer{Type: m.fromType(from.Elem(), s, tags)}
	case *types.Interface:
		if m.binary != nil && !types.Implements(from, m.binary) {
			return &schema.Any{}
		} else {
			return &schema.Interface{Name: name}
		}
	case *types.Slice:
		vt := m.fromType(from.Elem(), s, "")
		if tags.Flag("variant") {
			if it, ok := vt.(*schema.Interface); ok {
				vt = &schema.Variant{Name: it.Name}
			}
		}
		return &schema.Slice{Alias: alias, ValueType: vt}
	case *types.Array:
		return &schema.Array{
			Alias:     alias,
			ValueType: m.fromType(from.Elem(), s, ""),
			Size:      uint32(from.Len()),
		}
	case *types.Map:
		return &schema.Map{
			Alias:     alias,
			KeyType:   m.fromType(from.Key(), s, ""),
			ValueType: m.fromType(from.Elem(), s, ""),
		}
	case *types.Struct:
		t := &schema.Struct{Relative: name}
		s.unresolved = append(s.unresolved, unresolved{t, from})
		return t
	default:
		panic(fmt.Errorf("fromType found '%v' as  %T\n", from.String(), from))
	}
}
