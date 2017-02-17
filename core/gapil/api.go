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

// Package gapil holds the main interface to the api language libraries.
// It provides functions for going from api files to abstract syntax trees and
// processed semantic trees.
package gapil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/google/gapid/core/gapil/ast"
	"github.com/google/gapid/core/gapil/parser"
	"github.com/google/gapid/core/gapil/resolver"
	"github.com/google/gapid/core/gapil/semantic"
	"github.com/google/gapid/core/text/parse"
)

// ParseResult holds the result of parsing a file.
type ParseResult struct {
	API  *ast.API
	Errs parse.ErrorList
}

// ResolveResult holds the result of parsing a file.
type ResolveResult struct {
	API  *semantic.API
	Errs parse.ErrorList
}

// Processor holds the state when resolving multiple api files.
type Processor struct {
	*resolver.Mappings
	Loader              func(path string) ([]byte, error)
	Parsed              map[string]ParseResult // guarded by parsedLock
	Resolved            map[string]ResolveResult
	ResolveOnParseError bool // If true, resolving will be attempted even if parsing failed.

	parsedLock sync.Mutex
}

// NewProcessor returns a new initialized Processor.
func NewProcessor() *Processor {
	return &Processor{
		Mappings: resolver.NewMappings(),
		Loader:   ioutil.ReadFile,
		Parsed:   map[string]ParseResult{},
		Resolved: map[string]ResolveResult{},
	}
}

// Parse parses the api file with a default Processor.
// See Processor.Parse for details.
func Parse(apiname string) (*ast.API, parse.ErrorList) {
	return NewProcessor().Parse(apiname)
}

// Parse returns an ast that represents the supplied filename.
// It if the file has already been parsed, the cached ast will be returned,
// otherwise it invokes parser.Parse on the content of the supplied file name.
// It is safe to parse multiple files simultaniously.
func (p *Processor) Parse(path string) (*ast.API, parse.ErrorList) {
	p.parsedLock.Lock()
	res, ok := p.Parsed[path]
	p.parsedLock.Unlock()
	if ok {
		return res.API, res.Errs
	}

	info, err := p.Loader(path)
	if err != nil {
		return nil, parse.ErrorList{parse.Error{Message: err.Error()}}
	}
	api, errs := parser.Parse(path, string(info), p)
	p.parsedLock.Lock()
	p.Parsed[path] = ParseResult{api, errs}
	p.parsedLock.Unlock()
	return api, errs
}

// Resolve resolves the api file with a default Processor.
// See Processor.Resolve for details.
func Resolve(apiname string) (*semantic.API, parse.ErrorList) {
	return NewProcessor().Resolve(apiname)
}

// Resolve returns a semantic.API that represents the supplied api file name.
// If the file has already been resolved, the cached semantic tree is returned,
// otherwise the file and all dependant files are parsed using Processor.Parse.
// Recursive calls are made to Resolve for all named imports, and then finally
// the ast and all included ast's are handed to resolver.Resolve to do semantic
// processing.
func (p *Processor) Resolve(apiname string) (*semantic.API, parse.ErrorList) {
	absname, err := filepath.Abs(apiname)
	if err != nil {
		return nil, parse.ErrorList{parse.Error{Message: err.Error()}}
	}
	wd, name := filepath.Split(absname)
	return p.resolve(wd, name)
}

func (p *Processor) resolve(wd, name string) (*semantic.API, parse.ErrorList) {
	absname := filepath.Join(wd, name)
	if res, ok := p.Resolved[absname]; ok {
		if res.API == nil && res.Errs == nil { // reentry detected
			return nil, parse.ErrorList{parse.Error{
				Message: fmt.Sprintf("Recursive import %s", absname),
			}}
		}
		return res.API, res.Errs
	}
	p.Resolved[absname] = ResolveResult{} // mark to prevent reentry
	// Parse the API file and gather all the includes
	includes := map[string]*ast.API{}
	allErrs := p.parseIncludesResursive(wd, name, includes)
	// Build a sorted list of includes
	names := make(sort.StringSlice, 0, len(includes))
	for name := range includes {
		names = append(names, name)
	}
	names.Sort()
	list := make([]*ast.API, len(names))
	for i, name := range names {
		list[i] = includes[name]
	}
	// Resolve all the named imports
	imports := &semantic.Symbols{}
	importPaths := map[string]string{}
	for _, api := range list {
		for _, i := range api.Imports {
			if i.Name == nil {
				// unnamed imports have already been included
				continue
			}
			path := filepath.Join(wd, i.Path.Value)
			if importedPath, seen := importPaths[i.Name.Value]; seen {
				if path == importedPath {
					// import with same path and name already included
					continue
				}
				return nil, parse.ErrorList{parse.Error{
					Message: fmt.Sprintf("Import name '%s' used for different paths (%s != %s)",
						i.Name.Value, path, importedPath)},
				}
			}
			wd, name := filepath.Split(path)
			api, errs := p.resolve(wd, name)
			if len(errs) > 0 {
				allErrs = append(errs, allErrs...)
			}
			if api != nil {
				imports.Add(i.Name.Value, api)
				importPaths[i.Name.Value] = path
			}
		}
	}
	if len(allErrs) > 0 && !p.ResolveOnParseError {
		return nil, allErrs
	}
	// Now resolve the api set as a single unit
	api, errs := resolver.Resolve(list, imports, p.Mappings)
	if len(errs) > 0 {
		allErrs = append(errs, allErrs...)
	}
	nameNoExt := name[:len(name)-len(filepath.Ext(name))]
	api.Named = semantic.Named(nameNoExt)
	p.Resolved[absname] = ResolveResult{api, allErrs}
	return api, allErrs
}

// parseUnnamedIncludesResursive resursively parses the unnamed includes from
// apiname in wd. The full list of includes (named and unnamed) is added to
// includes.
func (p *Processor) parseIncludesResursive(wd string, name string, includes map[string]*ast.API) parse.ErrorList {
	path, err := filepath.Abs(filepath.Join(wd, name))
	if err != nil {
		return parse.ErrorList{parse.Error{Message: err.Error()}}
	}
	if _, seen := includes[path]; seen {
		return nil
	}
	api, allErrs := p.Parse(path)
	if api == nil {
		return allErrs
	}
	includes[path] = api
	for _, i := range api.Imports {
		if i.Name != nil {
			// named imports don't get merged
			continue
		}
		path := filepath.Join(wd, i.Path.Value)
		wd, name := filepath.Split(path)
		if errs := p.parseIncludesResursive(wd, name, includes); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

// CheckErrors will, if len(errs) > 0, print each of the error messages for the
// specified api and then return the list as a single error. If errs is zero length,
// CheckErrors does nothing and returns nil.
func CheckErrors(apiName string, errs parse.ErrorList, maxErrors int) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) > maxErrors {
		errs = errs[:maxErrors]
	}
	for _, e := range errs {
		if e.At != nil {
			filename := e.At.Token().Source.Filename
			line, column := e.At.Token().Cursor()
			fmt.Fprintf(os.Stderr, "%s:%v:%v: %s\n", filename, line, column, e.Message)
		} else {
			fmt.Fprintf(os.Stderr, "%s: %s\n", apiName, e.Message)
		}
	}
	if len(errs) > maxErrors {
		fmt.Fprintf(os.Stderr, "And %d more errors\n", len(errs)-maxErrors)
	}
	fmt.Fprintf(os.Stderr, "Stack of first error:\n%s\n", errs[0].Stack)
	return errs
}
