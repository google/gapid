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

// Package generate has support processing loaded go code, finding the items
// that require generated code, and converting them to a form the templates can
// easily consume.
package generate

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/types"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/tools/codergen/scan"
)

// Generate is a file generation request.
// It holds all the information needed to generate the file, and is passed in
// to a work queue.
type Generate struct {
	Name   string
	Arg    interface{}
	Output string
	Indent string
}

// Modules holds the list of all modules in a single scan.
type Modules []*Module

// Module represents a go package. In normal go layout, there will be at most
// two modules per directory, one normal module and one test module.
type Module struct {
	Modules    *Modules          // The set of modules this module belongs to.
	Source     *scan.Module      // The source module this was generated from.
	Name       string            // The name of the module.
	Import     string            // The import path of this module.
	IsTest     bool              // whether this module is for a test package.
	Path       string            // The directory name this module was scanned from.
	Directives map[string]string // The set of codergen directives encountered in the files.
	Structs    StructList        // The structs encountered.
	Frozen     StructList        // The frozen structs encountered.
	Imports    Imports           // The set of package imports encountered.
	binary     *types.Interface  // The binary object interface type.
}

// Import represents a go import declaration.
type Import struct {
	Name string // The name if present.
	Path string // The full import path.
}

// ImportGroup represents a group of sorted Import declarations.
type ImportGroup []Import

// Imports holds a set of Import declaration groups.
type Imports []ImportGroup

// ShallowClone returns a shallow clone of m.
func (m *Module) ShallowClone() *Module {
	out := *m
	return &out
}

// DirectiveList returns the comma-separated list of strings from the specified
// directive.
// If no directive with the specified name is found then nil is returned.
func (m *Module) DirectiveList(name string) []string {
	d, ok := m.Directives[name]
	if !ok {
		return nil
	}
	parts := strings.Split(d, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) > 0 {
			out = append(out, part)
		}
	}
	return out
}

// DirectiveSet returns the comma-separated set of strings from the specified
// directive.
// If no directive with the specified name is found then nil is returned.
func (m *Module) DirectiveSet(name string) map[string]bool {
	list := m.DirectiveList(name)
	if list == nil {
		return nil
	}
	set := make(map[string]bool)
	for _, s := range list {
		set[s] = true
	}
	return set
}

// Directive looks up a directive by name, and returns notset if the directive
// is not found.
// It will make an attempt to coerce the return type to match that of notset if
// it is a bool.
func (m *Module) Directive(name string, notset interface{}) interface{} {
	d, ok := m.Directives[name]
	if !ok {
		return notset
	}
	if _, isbool := notset.(bool); isbool {
		//coerce the string to bool
		if b, err := strconv.ParseBool(d); err == nil {
			return b
		}
	}
	return d
}

// ModuleAndName returns the module and type name pair.
func (m *Module) ModuleAndName(v interface{}) (*Module, string) {
	name := ""
	switch v := v.(type) {
	case string:
		name = v
	case *Struct:
		name = v.Name()
	case *schema.Struct:
		name = v.String()
	case *schema.Interface:
		name = v.Name
	case *schema.Variant:
		name = v.Name
	case *schema.Primitive:
		name = v.Name
	case *schema.Array:
		name = v.Alias
	default:
		panic(fmt.Errorf("Invalid type %T to ModuleAndName", v))
	}
	pkg, name := "", name
	if i := strings.LastIndexAny(name, "."); i >= 0 {
		pkg = name[:i]
		name = name[i+1:]
	}
	found := m
	if pkg != "" {
		found = m.FindImport(pkg)
		if found == nil {
			found = &Module{Name: pkg, Import: "missing_" + pkg}
		}
	}
	return found, name
}

func pathToGroup(path string) int {
	dot := strings.IndexRune(path, '.') > 0
	if dot {
		return 1
	}
	return 0
}

func (i ImportGroup) Len() int           { return len(i) }
func (i ImportGroup) Swap(a, b int)      { i[a], i[b] = i[b], i[a] }
func (i ImportGroup) Less(a, b int) bool { return i[a].Path < i[b].Path }

// Add adds a new import to the import list.
func (i *Imports) Add(v Import) {
	g := pathToGroup(v.Path)
	if g >= len(*i) {
		*i = append(*i, make(Imports, (g+1)-len(*i))...)
	}
	group := &((*i)[g])
	n := sort.Search(len(*group), func(n int) bool { return (*group)[n].Path >= v.Path })
	if n < len(*group) && (*group)[n].Path == v.Path {
		return
	}
	*group = append(*group, v)
	sort.Sort(group)
}

// Count returns the total number of imports across all groups.
func (i *Imports) Count() int {
	count := 0
	for _, g := range *i {
		count += len(g)
	}
	return count
}

// FindName returns the import that matches the supplied name, or an empty
// import if not present. Test the returned .Path to detect this.
func (i Imports) FindName(name string) Import {
	for _, g := range i {
		for _, v := range g {
			if v.Name == name {
				return v
			}
		}
	}
	return Import{}
}

// FindPath finds the import for the specified import path, or an empty
// import if not present. Test the returned .Path to detect this.
func (i Imports) FindPath(path string) Import {
	g := pathToGroup(path)
	n := sort.Search(len(i[g]), func(n int) bool { return i[g][n].Path >= path })
	if n < len(i[g]) && i[g][n].Path == path {
		return i[g][n]
	}
	return Import{}
}

// FindImport searches the modules imports for the specified name, and then
// searches the parent module set for the module that matches the import path
// found.
// It will return nil if either the name is not valid or the module cannot be
// found.
func (m *Module) FindImport(name string) *Module {
	path := m.Imports.FindName(name).Path
	if path == "" {
		for _, o := range *m.Modules {
			if strings.HasSuffix(o.Import, name) {
				return o
			}
		}
		return nil
	}
	for _, o := range *m.Modules {
		if o.Import == path {
			return o
		}
	}
	return nil
}

func fakeStruct(structs map[*types.Struct]*Struct, pkg *types.Package, module string, typename string) {
	t := findStruct(pkg, module, typename)
	if t == nil {
		return
	}
	if _, found := structs[t]; found {
		return
	}
	entity := &Struct{Entity: binary.Entity{Package: module, Identity: typename, Exported: true}}
	entity.Fields = append(entity.Fields, binary.Field{Type: &schema.Interface{}})
	structs[t] = entity
}

// From processes scanned source code to produce the module set it represents.
func From(scanner *scan.Scanner) (Modules, error) {
	result := Modules{}
	var m *Module
	for _, dir := range scanner.Directories {
		if !dir.Scan {
			continue
		}
		if m, err := convert(scanner, &dir.Module, false); err != nil {
			return nil, err
		} else if m != nil {
			m.Modules = &result
			result = append(result, m)
		}
		if m, err := convert(scanner, &dir.Test, true); err != nil {
			return nil, err
		} else if m != nil {
			m.Modules = &result
			result = append(result, m)
		}
	}
	// Now resolve all the cross type depedancies
	structs := map[*types.Struct]*Struct{}
	for _, m = range result {
		for _, s := range m.Structs {
			structs[s.Raw] = s
		}
	}
	for _, m = range result {
		fakeStruct(structs, m.Source.Types, schemaPackage, "Message")
		for _, s := range m.Structs {
			for _, u := range s.unresolved {
				if e, ok := structs[u.t]; ok {
					u.s.Entity = &e.Entity
				} else {
					panic(fmt.Errorf("No match in %s for %s (%T)", s.Name(), u.t.String(), u.t))
				}
			}
		}
	}
	for _, m = range result {
		m.finaliseStructs()
	}
	return result, nil
}

// convert processes a single module from a scan set.
func convert(scanner *scan.Scanner, src *scan.Module, isTest bool) (*Module, error) {
	if src.Types == nil {
		return nil, nil
	}
	directives := map[string]string{}
	for _, file := range src.Sources {
		for k, v := range file.Directives {
			directives[k] = v
		}
	}
	if _, ignored := directives["ignore"]; ignored {
		return nil, nil
	}
	m := &Module{
		Source:     src,
		Name:       src.Directory.Name,
		Path:       src.Directory.Dir,
		Import:     path.Clean(src.Directory.ImportPath),
		Directives: directives,
		IsTest:     isTest,
		binary:     findInterface(src.Types, binaryPackage, "Object"),
	}
	scope := src.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		f := scanner.FileSet.File(obj.Pos())
		filename := f.Name()
		var source *scan.Source
		for i := range src.Sources {
			if src.Sources[i].Filename == filename {
				source = &src.Sources[i]
				break
			}
		}
		if source == nil {
			continue
		}
		if n, ok := obj.(*types.TypeName); ok {
			if t, ok := n.Type().(*types.Named); ok {
				if _, ok := t.Underlying().(*types.Struct); ok {
					m.addStruct(n)
				}
			}
		}
	}
	return m, nil
}
