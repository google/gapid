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

package scan

import (
	"context"
	"go/parser"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/copyright"
	"golang.org/x/tools/go/types"
)

const Tool = "codergen"

func (m *Module) addSource(filename, content string) error {
	if content == "" {
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		content = string(file)
	}

	for _, re := range copyright.Generated {
		match := re.FindStringSubmatch(content)
		if len(match) > 1 {
			for _, test := range match[1:] {
				if test == Tool {
					return nil
				}
			}
		}
	}

	m.Sources = append(m.Sources, Source{Filename: filename, Content: content, Parsed: make(chan struct{})})
	return nil
}

func (s *Scanner) load(dir *Directory) {
	dir.loaded = true
	imp, err := s.context.Import(dir.ImportPath, s.Path, 0)
	if err != nil {
		return
	}
	dir.Name = imp.Name
	dir.ImportPath = imp.ImportPath
	dir.Dir = imp.Dir
	for _, filename := range imp.GoFiles {
		dir.Module.addSource(filepath.Join(dir.Dir, filename), "")
	}
	s.preParse(&dir.Module)
	if dir.Scan {
		for _, filename := range imp.TestGoFiles {
			dir.Test.addSource(filepath.Join(dir.Dir, filename), "")
		}
		s.preParse(&dir.Test)
	}
}

func (s *Scanner) preParse(module *Module) {
	for i := range module.Sources {
		src := &module.Sources[i]
		go func() {
			src.AST, src.Error = parser.ParseFile(s.FileSet, src.Filename, src.Content, parser.ParseComments)
			close(src.Parsed)
		}()
	}
}

var directive = regexp.MustCompile(`binary: *([^= ]+) *(= *(.+))? *`)

func (s *Scanner) parse(ctx context.Context, module *Module) error {
	for i := range module.Sources {
		src := &module.Sources[i]
		<-src.Parsed
		if src.Error != nil {
			return src.Error
		}
		module.Files = append(module.Files, src.AST)
		src.Directives = make(map[string]string)
		for _, group := range src.AST.Comments {
			for _, comment := range group.List {
				if matches := directive.FindStringSubmatch(comment.Text); len(matches) >= 1 {
					k := matches[1]
					v := matches[3]
					if matches[2] == "" {
						v = "true"
					}
					src.Directives[k] = v
				}
			}
		}
	}
	return nil
}

func (s *Scanner) importer(ctx context.Context, pkgs map[string]*types.Package, importPath string) (*types.Package, error) {
	if importPath == "unsafe" {
		pkgs[importPath] = types.Unsafe
		return types.Unsafe, nil
	}
	dir := s.GetDir(importPath)
	if err := s.process(ctx, dir); err != nil {
		return nil, err
	}
	if dir.Module.Types == nil {
		return nil, log.Errf(ctx, fault.Const("Cyclic import"), "package: %v", dir.ImportPath)
	}
	pkgs[importPath] = dir.Module.Types
	return dir.Module.Types, nil
}
