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
	"sort"
	"sync"

	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
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

// Loader is the interface to something that finds and loads api imports.
type Loader interface {
	// Find recieves the path to a desired import relative to the current file.
	// The path it is supplied may not be valid, the loader should transform it
	// into a valid path if possible or return a fully invalid path if the api
	// file cannot be found.
	Find(path file.Path) file.Path
	// Load takes a path returned by Find and returns the content the path
	// represents, or an error if the path was not valid.
	Load(file.Path) ([]byte, error)
}

// Processor holds the state when resolving multiple api files.
type Processor struct {
	*semantic.Mappings
	Loader              Loader
	Parsed              map[string]ParseResult // guarded by parsedLock
	Resolved            map[string]ResolveResult
	ResolveOnParseError bool // If true, resolving will be attempted even if parsing failed.
	Options             resolver.Options

	parsedLock sync.Mutex
}

// NewProcessor returns a new initialized Processor.
func NewProcessor() *Processor {
	return &Processor{
		Mappings: &semantic.Mappings{},
		Loader:   absLoader{},
		Parsed:   map[string]ParseResult{},
		Resolved: map[string]ResolveResult{},
	}
}

type (
	absLoader        struct{}
	dataLoader       struct{ data []byte }
	searchListLoader struct {
		search file.PathList
	}
)

func (l absLoader) Find(path file.Path) file.Path        { return path }
func (l absLoader) Load(path file.Path) ([]byte, error)  { return ioutil.ReadFile(path.System()) }
func (l dataLoader) Find(path file.Path) file.Path       { return path }
func (l dataLoader) Load(path file.Path) ([]byte, error) { return l.data, nil }

func NewDataLoader(data []byte) Loader {
	return dataLoader{data: data}
}

func NewSearchLoader(search file.PathList) Loader {
	return searchListLoader{search: search}
}

func (l searchListLoader) Find(path file.Path) file.Path {
	rooted := l.search.RootOf(path)
	if rooted.Root.IsEmpty() {
		return path
	}
	found := l.search.Find(rooted.Fragment)
	if found.Root.IsEmpty() {
		return path
	}
	return found.Path()
}

func (l searchListLoader) Load(path file.Path) ([]byte, error) {
	return ioutil.ReadFile(path.System())
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
	return p.parse(file.Abs(path))
}

func (p *Processor) parse(path file.Path) (*ast.API, parse.ErrorList) {
	path = p.Loader.Find(path)
	absname := path.System()
	p.parsedLock.Lock()
	res, ok := p.Parsed[absname]
	p.parsedLock.Unlock()
	if ok {
		return res.API, res.Errs
	}
	info, err := p.Loader.Load(path)
	if err != nil {
		return nil, parse.ErrorList{parse.Error{Message: err.Error()}}
	}
	api, errs := parser.Parse(absname, string(info), &p.Mappings.AST)
	p.parsedLock.Lock()
	p.Parsed[absname] = ParseResult{api, errs}
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
	return p.resolve(file.Abs(apiname))
}

func (p *Processor) resolve(base file.Path) (*semantic.API, parse.ErrorList) {
	base = p.Loader.Find(base)
	absname := base.System()
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
	allErrs := p.parseIncludesResursive(base, includes)
	if len(allErrs) > 0 && !p.ResolveOnParseError {
		return nil, allErrs
	}
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
	// Now resolve the api set as a single unit
	api, errs := resolver.Resolve(list, p.Mappings, p.Options)
	if len(errs) > 0 {
		allErrs = append(errs, allErrs...)
	}
	_, nameNoExt, _ := base.Smash()
	api.Named = semantic.Named(nameNoExt)
	p.Resolved[absname] = ResolveResult{api, allErrs}
	return api, allErrs
}

// parseUnnamedIncludesResursive resursively parses the unnamed includes from
// apiname in wd. The full list of includes (named and unnamed) is added to
// includes.
func (p *Processor) parseIncludesResursive(base file.Path, includes map[string]*ast.API) parse.ErrorList {
	absname := base.System()
	if _, seen := includes[absname]; seen {
		return nil
	}
	api, allErrs := p.parse(base)
	if api == nil {
		return allErrs
	}
	includes[absname] = api
	for _, i := range api.Imports {
		child := p.Loader.Find(base.Parent().Join(i.Path.Value))
		if errs := p.parseIncludesResursive(child, includes); len(errs) > 0 {
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
			filename := e.At.Tok().Source.Filename
			line, column := e.At.Tok().Cursor()
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
