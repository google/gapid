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

// Package format registers and implements the "format" apic command.
//
// The format command re-formats an API file to a consistent style.
package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/format"
	"github.com/google/gapid/gapil/parser"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "format",
		ShortHelp: "Formats an api file",
		Action:    &formatVerb{},
	})
}

type formatVerb struct{}

func (v *formatVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	args := flags.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Missing api file")
		return nil
	}
	paths := []string{}
	for _, path := range args {
		files, err := filepath.Glob(path)
		if err != nil {
			return err
		}
		paths = append(paths, files...)
	}
	for _, path := range paths {
		ctx := log.V{"file": path}.Bind(ctx)
		f, err := ioutil.ReadFile(path)
		if err != nil {
			log.F(ctx, true, "Failed to read api file. Error: %v", err)
			continue
		}
		m := &ast.Mappings{}
		api, errs := parser.Parse(path, string(f), m)
		if len(errs) > 0 {
			l := log.From(ctx)
			l.E("Errors while parsing api file:")
			for i, e := range errs {
				l.E("%d: %v", i, e)
			}
			continue
		}

		buf := &bytes.Buffer{}
		format.Format(api, m, buf)
		if err = ioutil.WriteFile(path, buf.Bytes(), 0777); err != nil {
			log.E(ctx, "Failed to write formatted api file. Error: %v", err)
		}
	}
	return nil
}
