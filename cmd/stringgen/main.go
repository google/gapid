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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/stringtable"
	"github.com/google/gapid/gapis/stringtable/parser"
)

type tableAndTypeMap struct {
	stringTable  *stringtable.StringTable
	paramTypeMap parser.ParameterTypeMap
}

const (
	defaultCultureCode    = "en-us"
	ErrDuplicateParameter = fault.Const("Duplicate stringtable parameter found")
	ErrDuplicateStringKey = fault.Const("Duplicate string key found")
	ErrNoStringtables     = fault.Const("No string table files provided")
	ErrNoEntry            = fault.Const("No entry provided")
	ErrParameterList      = fault.Const("Parameter list different")
)

var (
	defGo  = flag.String("def-go", "", "The path to the Go string definition file")
	defAPI = flag.String("def-api", "", "The path to the API string definition file")
	pkg    = flag.String("pkg", "", "The directory to hold the output string packages.")
)

func main() {
	app.ShortHelp = "stringgen compiles string table files to string packages and a Go definition file."
	app.Run(run)
}

type tableKey struct {
	cultureCode string
}

func key(i *stringtable.Info) tableKey {
	if i == nil {
		return tableKey{}
	}
	return tableKey{i.CultureCode}
}

func (k tableKey) toProto() *stringtable.Info {
	return &stringtable.Info{CultureCode: k.cultureCode}
}

func run(ctx context.Context) error {
	tables := map[tableKey]*tableAndTypeMap{}

	for _, path := range flag.Args() {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		table, paramTypeMap, errs := parser.Parse(path, string(content))
		if len(errs) > 0 {
			return fmt.Errorf("%s", errs)
		}
		if existing, exists := tables[key(table.Info)]; exists {
			// Merge parameter maps, check for duplicates
			for k, v := range paramTypeMap {
				if _, dup := existing.paramTypeMap[k]; !dup {
					existing.paramTypeMap[k] = v
				} else {
					return log.Errf(ctx, ErrDuplicateParameter, "key: %v, parameter: %v", k.EntryKey, k.ParameterKey)
				}
			}
			// Merge tables, check for duplicates.
			for k, v := range table.Entries {
				if _, dup := existing.stringTable.Entries[k]; !dup {
					existing.stringTable.Entries[k] = v
				} else {
					return log.Errf(ctx, ErrDuplicateStringKey, "key: %v", k)
				}
			}
		} else {
			tables[key(table.Info)] = &tableAndTypeMap{stringTable: table, paramTypeMap: paramTypeMap}
		}
	}

	if len(tables) == 0 {
		return log.Err(ctx, ErrNoStringtables, "")
	}

	log.D(ctx, "Found %d string table file(s)", len(tables))

	if err := validate(ctx, tables); err != nil {
		return err
	}

	if *pkg != "" {
		if err := writePackages(tables, *pkg); err != nil {
			return err
		}
	}

	if *defGo != "" {
		for _, t := range tables {
			if err := writeGoDefinitions(t, *defGo); err != nil {
				return err
			}
			break
		}
	}

	if *defAPI != "" {
		for _, t := range tables {
			if err := writeAPIDefinitions(t, *defAPI); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// entry is a map of Info -> parameter list
type entry map[tableKey][]string

func writePackages(tables map[tableKey]*tableAndTypeMap, path string) error {
	for info, table := range tables {
		data, err := proto.Marshal(table.stringTable)
		if err != nil {
			return err
		}
		path := filepath.Join(path, info.cultureCode+".stb")
		os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err := ioutil.WriteFile(path, data, 0755); err != nil {
			return err
		}
	}
	return nil
}

func writeGoDefinitions(table *tableAndTypeMap, path string) error {
	return writeDefinitionsAbstract(table, path, "StringDefGo", stringdefgo_tmpl)
}

// TODO: Merge all the common stuff from this and the one above.
func writeAPIDefinitions(table *tableAndTypeMap, path string) error {
	return writeDefinitionsAbstract(table, path, "StringDefApi", stringdefapi_tmpl)
}

func writeDefinitionsAbstract(table *tableAndTypeMap, path, templateRoutine, templateText string) error {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(abspath), 0755)
	w, err := os.Create(abspath)
	if err != nil {
		return err
	}
	defer w.Close()

	entries := make(EntryList, 0, len(table.stringTable.Entries))
	for key, node := range table.stringTable.Entries {
		entry := Entry{
			Key:        key,
			Parameters: params(key, node, table.paramTypeMap),
		}
		entries = append(entries, entry)
	}

	sort.Sort(entries)

	_, pkg := filepath.Split(filepath.Dir(abspath))
	return Execute(templateRoutine, templateText, pkg, entries, w)
}

// checks for consistency between the various localizations of strings.
func validate(ctx context.Context, tables map[tableKey]*tableAndTypeMap) error {
	all := map[string]entry{}

	for info, table := range tables {
		for key, node := range table.stringTable.Entries {
			e := all[key]
			if e == nil {
				e = make(entry)
			}
			strParams := []string{}
			for _, p := range params(key, node, table.paramTypeMap) {
				strParams = append(strParams, p.Identifier)
			}
			e[info] = strParams
			all[key] = e
		}
	}

	for key, entry := range all {
		ctx := log.V{"Key": key}.Bind(ctx)
		if len(entry) != len(tables) {
			for info := range tables {
				if _, found := entry[info]; !found {
					return log.Errf(ctx, ErrNoEntry, "Table: %v", info)
				}
			}
		}
		var validInfo *stringtable.Info
		var validParams []string
		for info, params := range entry {
			if info.cultureCode == defaultCultureCode {
				validInfo, validParams = info.toProto(), params
				break
			}
		}
		if validInfo == nil || validParams == nil {
			return log.Errf(ctx, ErrNoEntry, "code: %v", defaultCultureCode)
		}
		for info, params := range entry {
			if info.cultureCode != defaultCultureCode &&
				!reflect.DeepEqual(validParams, params) {
				return log.Errf(ctx, ErrParameterList, "first: %v, second: %v", *validInfo, info)
			}
		}
	}

	return nil
}

// params returns the list of parameters used by the stringtable node.
func params(entryKey string, n *stringtable.Node, typeMap parser.ParameterTypeMap) []EntryParameter {
	switch n := n.Node.(type) {
	case *stringtable.Node_Block:
		p := []EntryParameter{}
		for _, n := range n.Block.Children {
			p = append(p, params(entryKey, n, typeMap)...)
		}
		return p
	case *stringtable.Node_Parameter:
		return []EntryParameter{
			{
				Identifier: n.Parameter.Key,
				Type:       typeMap[parser.ParameterID{ParameterKey: n.Parameter.Key, EntryKey: entryKey}],
			},
		}
	}
	return nil
}
