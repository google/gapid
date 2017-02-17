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
	"sort"
	"strings"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"golang.org/x/tools/go/types"
)

// Interface represents a named interface in a package.
type Interface struct {
	Name    string // Interface name.
	Package string // Interface package.
}

// Struct is a description of an encodable struct.
// Signature includes the package, name, and type of all the fields.
// Any change to the Signature will cause the ID to change.
type Struct struct {
	binary.Entity
	Tags       Tags // The tags associated with the type.
	Raw        *types.Struct
	Implements []Interface  // The list of interfaces implemented by this struct.
	unresolved []unresolved // The set of unresolved schema.Struct objects.
}

// StructList is a list of struct pointers
type StructList []*Struct

// Filter returns a new Struct list with all the elements that fail the
// predicate removed.
func (l StructList) Filter(predicate func(s *Struct) bool) StructList {
	out := make(StructList, 0, len(l))
	for _, s := range l {
		if predicate(s) {
			out = append(out, s)
		}
	}
	return out
}

type unresolved struct {
	s *schema.Struct
	t *types.Struct
}

func (m *Module) fromFieldType(f *binary.Field, invalidStruct *error, s *Struct, decl *types.Var, tags Tags) {
	defer func() {
		if r := recover(); r != nil && *invalidStruct == nil {
			*invalidStruct = fmt.Errorf("Handling %v.%v gave error %v", s.Name(), f.Name(), r)
		}
	}()
	f.Type = m.fromType(decl.Type(), s, tags)
}

func (m *Module) addStruct(n *types.TypeName) {
	t := n.Type().Underlying().(*types.Struct)
	s := &Struct{
		Entity: binary.Entity{
			Display:  n.Name(),
			Package:  m.Source.Types.Name(),
			Exported: n.Exported(),
		},
		Raw: t,
	}
	tagged := false
	var invalidStruct error
	frozen := false
	for i := 0; i < t.NumFields(); i++ {
		decl := t.Field(i)
		tags := Tags(t.Tag(i))
		if tags.Flag("disable") {
			continue
		}

		if decl.Anonymous() &&
			(decl.Type().String() == binaryGenerate ||
				decl.Type().String() == binaryFrozen) {
			if decl.Type().String() == binaryFrozen {
				frozen = true
			}
			tagged = true
			s.Tags = tags
			continue
		}
		f := binary.Field{}
		if !decl.Anonymous() {
			f.Declared = decl.Name()
		}
		m.fromFieldType(&f, &invalidStruct, s, decl, tags)
		s.Fields = append(s.Fields, f)
	}
	if tagged {
		if invalidStruct != nil {
			panic(invalidStruct)
		}
		s.Identity = s.Tag("identity", s.Display)
		if s.Display == s.Identity {
			s.Display = ""
		}
		s.Version = s.Tag("version", "")
		if frozen {
			m.Frozen = append(m.Frozen, s)
		} else {
			m.Structs = append(m.Structs, s)
		}
		implements := s.Tags.List("implements")
		s.Implements = make([]Interface, len(implements))
		for i := range implements {
			parts := strings.Split(implements[i], ".")
			if len(parts) != 2 {
				panic("implements tag must contain a comma-separated list of interfaces in the form: package.name")
			}
			s.Implements[i] = Interface{
				Name:    parts[1],
				Package: parts[0],
			}
		}
	}
}

// Tag returns the named tag if present, missing otherwise.
func (s *Struct) Tag(name string, missing string) string {
	result := s.Tags.Get(name)
	if result == "" {
		result = missing
	}
	return result
}

// IDName returns the name to give the ID of the type.
func (s *Struct) IDName() string {
	return s.Tag("id", "binaryID"+s.Name())
}

// HasStructTag returns true if any struct in the module has the named tag.
func (m *Module) HasStructTag(name string) bool {
	for _, s := range m.Structs {
		result := s.Tags.Get(name)
		if result != "" {
			return true
		}
	}
	return false
}

type sortEntry struct {
	s       *Struct
	visited bool
}

func walkType(t binary.Type, byname map[string]*sortEntry, structs []*Struct, i int) int {
	switch t := t.(type) {
	case *schema.Primitive:
	case *schema.Struct:
		i = walkStructs(t.String(), byname, structs, i)
	case *schema.Interface:
		i = walkStructs(t.Name, byname, structs, i)
	case *schema.Variant:
		i = walkStructs(t.Name, byname, structs, i)
	case *schema.Pointer:
		i = walkType(t.Type, byname, structs, i)
	case *schema.Array:
		i = walkType(t.ValueType, byname, structs, i)
	case *schema.Slice:
		i = walkType(t.ValueType, byname, structs, i)
	case *schema.Map:
		i = walkType(t.KeyType, byname, structs, i)
		i = walkType(t.ValueType, byname, structs, i)
	}
	return i
}

func walkStructs(name string, byname map[string]*sortEntry, structs []*Struct, i int) int {
	entry, found := byname[name]
	if !found || entry.visited {
		return i
	}
	entry.visited = true
	for _, f := range entry.s.Fields {
		i = walkType(f.Type, byname, structs, i)
	}
	structs[i] = entry.s
	return i + 1
}

func (m *Module) finaliseStructs() {
	names := make(sort.StringSlice, len(m.Structs))
	byname := make(map[string]*sortEntry, len(m.Structs))
	for i, s := range m.Structs {
		name := s.Name()
		names[i] = name
		byname[name] = &sortEntry{s, false}
	}
	names.Sort()
	i := 0
	for _, name := range names {
		i = walkStructs(name, byname, m.Structs, i)
	}
}
