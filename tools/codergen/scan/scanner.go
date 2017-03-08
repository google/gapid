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

// Package scan has the functionality used by codergen to load and resolve go
// code.
package scan

import (
	"context"
	"go/ast"
	"go/build"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/log"

	"golang.org/x/tools/go/types"
)

// Source holds a filename and content pair as consumed by go/parser.ParseFile.
type Source struct {
	Filename   string            // The filename for this source
	Content    interface{}       // The content of this source, see ParseFiles for details.
	AST        *ast.File         // The parsed syntax tree
	Directives map[string]string // the set of comment overrides
	Parsed     chan struct{}
	Error      error
}

// Module represents a resolvable module. Under normal go layout conditions a
// directory has one module that represents the files that are considered
// when the directory is imported, and a second one that also includes the test
// files. They must be considered separately because otherwise you can get
// import cycles.
type Module struct {
	Directory *Directory     // The directory this module belongs to
	Sources   []Source       // The set of sources to parse
	Files     []*ast.File    // The parsed files included
	Types     *types.Package // The resolved type information
	processed bool
}

// Directory holds information about a scanned directory, including the modules
// found in that directory.
type Directory struct {
	Name       string // The package name (as used in package declarations)
	ImportPath string // The full import path (as used in import statements)
	Dir        string // The actual directory in which the files live
	Scan       bool   // Whether to scan this directory for structs
	Module     Module // The main module data for this directory
	Test       Module // The test module data for this directory
	loaded     bool
}

// Scanner is the main interface to loading and scanning go code for codergen.
type Scanner struct {
	Path        string                // The base path of file scanning
	Directories map[string]*Directory // The set of directories considered
	FileSet     *token.FileSet        // The parser file set
	context     build.Context
	config      types.Config
}

// New creates a new go source scanner.
func New(ctx context.Context, path string, gopath string) *Scanner {
	l := &Scanner{
		Path:        path,
		Directories: map[string]*Directory{},
		FileSet:     token.NewFileSet(),
		context:     build.Default,
		config: types.Config{
			IgnoreFuncBodies: true,
			Error:            func(error) {},
			Packages:         map[string]*types.Package{},
		},
	}
	if gopath != "" {
		l.context.GOPATH = gopath
	}
	l.config.Import = func(pkgs map[string]*types.Package, importPath string) (*types.Package, error) {
		return l.importer(ctx, pkgs, importPath)
	}
	return l
}

// Scan loads and scans the specified package.
// If entry ends in ... then the directory is recursively walked and all the
// children are loaded and scanned as well.
func (s *Scanner) Scan(ctx context.Context, entry string) error {
	base := strings.TrimSuffix(entry, "...")
	pkg, err := s.context.Import(base, s.Path, build.FindOnly)
	if err != nil {
		return err
	}
	log.I(ctx, "Scanning import: %v, dir: %v", pkg.ImportPath, pkg.Dir)
	if len(base) == len(entry) {
		s.ScanPackage(pkg.ImportPath)
		return nil
	}
	return filepath.Walk(pkg.Dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		// Horribly ugly workaround
		if base[0] == '.' || base[0] == '_' || base == "vendor" || base == "third_party" {
			log.I(ctx, "Skipping: %v", base)
			return filepath.SkipDir
		}
		if !s.hasGoFiles(p) {
			return nil
		}
		name := path.Join(pkg.ImportPath, filepath.ToSlash(strings.TrimPrefix(p, pkg.Dir)))
		log.I(ctx, "Reading: %v", name)
		s.ScanPackage(name)
		return nil
	})
}

func (s *Scanner) hasGoFiles(dir string) bool {
	matched, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	return matched != nil
}

// GetDir returns the Directory that matches the supplied import path.
// It will add a new one if needed.
func (s *Scanner) GetDir(importPath string) *Directory {
	dir, ok := s.Directories[importPath]
	if !ok {
		dir = &Directory{
			ImportPath: importPath,
		}
		dir.Module.Directory = dir
		dir.Test.Directory = dir
		s.Directories[importPath] = dir
	}
	return dir
}

// ScanFile adds a fake package with the file as it's only source.
func (s *Scanner) ScanFile(filename, source string) {
	dir := s.GetDir(filename)
	dir.Scan = true
	dir.loaded = true
	dir.Module.addSource(filename, source)
	s.preParse(&dir.Module)
}

// ScanPackage marks the directory specified by the import path as needing to be
// scanned for binary structures.
func (s *Scanner) ScanPackage(importPath string) {
	s.GetDir(importPath).Scan = true
}
