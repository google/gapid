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

package annotate

import (
	"encoding/base64"
	"fmt"
	"io"
	"sort"

	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/semantic/printer"
	"github.com/google/gapid/gapil/snippets"
	"github.com/google/gapid/framework/binary/cyclic"
	"github.com/google/gapid/framework/binary/vle"
)

// Analsyis is a map from atom name to snippet table.
type Analysis map[string]*SnippetTable

// Annotate visits the compiled API file and produces an Analysis object
// which contains all the static analysis snippets added to parameters
// and globals (state variables).
func Annotate(compiled *semantic.API) Analysis {
	analysis := make(Analysis)
	for _, f := range compiled.Functions {
		analysis.function(f, compiled)
	}
	return analysis
}

func (analysis Analysis) function(f *semantic.Function, api *semantic.API) {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("Panic raised: %v\n\nFunction:%v", err, printer.New().WriteFunction(f)))
		}
	}()
	a := New(api)
	// Note the globals are deliberately scoped within the function for the
	// analysis. This is so that we can learn about the globals in the
	// context of a specific atom.
	a.beginScope()
	vars := []semantic.Expression{}
	for _, g := range api.Globals {
		vars = append(vars, g)
	}
	for _, p := range f.FullParameters {
		vars = append(vars, p)
	}
	for _, v := range vars {
		a.declare(v)
	}
	a.scoped(func() { a.visitStatement(f.Block) })
	var table *SnippetTable
	for _, v := range vars {
		loc := a.locate(v)
		if !loc.isEmpty() {
			p := path(v)
			if table == nil {
				table = &SnippetTable{}
			}
			loc.getTable(p, table)
			if len(*table) != 0 {
				analysis[f.Name()] = table
			}
		}
	}
}

// Globals builds a single snippet table which contains all the snippets for
// global paths. When the same global path is inferred for multiple atoms the
// snippets for that path are merged. The snippet table is produced in a
// consistent order.
func (a Analysis) Globals() SnippetTable {
	// Sort atom names so that snippet order is consistent.
	names := sort.StringSlice{}
	for name := range a {
		names = append(names, name)
	}
	names.Sort()

	indexByPathStr := make(map[string]int)
	paths := sort.StringSlice{}
	var table SnippetTable

	i := 0
	for _, name := range names {
		g := a[name].Globals()
		for _, entry := range g {
			pathStr := fmt.Sprint(entry.path)

			if prevIndex, ok := indexByPathStr[pathStr]; ok {
				// Merge with existing entry (order preserving)
				table[prevIndex].snippets.AddSnippets(entry.snippets)
			} else {
				// New entry
				paths = append(paths, pathStr)
				table = append(table, entry)
				indexByPathStr[pathStr] = i
				i++
			}
		}
	}

	// Sort the tables into path order so that the table is in consistent order.
	paths.Sort()
	var sortedTable SnippetTable
	sortedTable = make(SnippetTable, len(table))
	for i, p := range paths {
		sortedTable[i] = table[indexByPathStr[p]]
	}

	return sortedTable
}

// GlobalsGroups builds a snippet global analysis object which is a slice
// of kindred snippets. It contains all the snippets for global paths.
// When the same global path is inferred for multiple atoms the snippets
// for that path are merged. Each path has the snippets grouped by snippet
// type and binary encodable. The groups are in a consistent order.
func (a Analysis) GlobalsGrouped() GlobalAnalysis {
	g := a.Globals()
	return GlobalAnalysis(g.KindredGroups(snippets.GlobalsTypename))
}

// Print outputs the analysis object to out in human readable form.
// Atom names are sorted lexiographically.
func (a Analysis) Print(out io.Writer) {
	names := sort.StringSlice{}
	for name := range a {
		names = append(names, name)
	}
	names.Sort()
	for _, name := range names {
		table := a[name]
		fmt.Fprintf(out, "===================================== %s\n", name)
		fmt.Fprint(out, table)
	}
}

// Base64 outputs the analysis object using binary encoding and base64 encoding.
// Atom names are sorted lexiographically. Base64 encoding is used so that
// the binary file can be embedded in the library using the embed tool.
func (a Analysis) Base64(out io.Writer) error {
	names := sort.StringSlice{}
	for name := range a {
		names = append(names, name)
	}
	names.Sort()

	b64 := base64.NewEncoder(base64.StdEncoding, out)
	e := cyclic.Encoder(vle.Writer(b64))
	for _, name := range names {
		table := a[name].NonGlobals()
		groups := table.KindredGroups(name)
		atomSnips := &snippets.AtomSnippets{AtomName: name, Snippets: groups}
		e.Variant(atomSnips)
	}
	if e.Error() != nil {
		return e.Error()
	}
	return b64.Close()
}

// GlobalAnalysis is kindred grouped table of snippets with paths that apply
// to the global state. It is used to provide snippets for the API state
// object and ultimately the state view in the UI.
type GlobalAnalysis []snippets.KindredSnippets

// Base64 outputs the analysis object using binary encoding and base64 encoding.
// Output is in a consistent order. Base64 encoding is used so that the binary
// file can be embedded in the library using the embed tool.
func (a GlobalAnalysis) Base64(out io.Writer) error {
	b64 := base64.NewEncoder(base64.StdEncoding, out)
	e := cyclic.Encoder(vle.Writer(b64))
	for _, snip := range a {
		e.Variant(snip)
	}
	if e.Error() != nil {
		return e.Error()
	}
	return b64.Close()
}
