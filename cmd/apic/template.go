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
	"go/build"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/google/gapid/gapil/resolver"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil/template"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "template",
		ShortHelp: "Passes the ast to a template for code generation",
		Action: &templateVerb{
			Dir: cwd(),
		},
	})
}

type templateVerb struct {
	Dir            string        `help:"The output directory"`
	Tracer         string        `help:"The template function trace expression"`
	Gopath         string        `help:"the go path to use when looking up packages"`
	GlobalList     []string      `help:"A global value setting for the template"`
	Search         file.PathList `help:"The set of paths to search for includes"`
	TemplateSearch file.PathList `help:"Search path apic includes"`
}

func (v *templateVerb) templateLoader(filename string) ([]byte, error) {
	if bytes, err := ioutil.ReadFile(filename); err == nil {
		return bytes, err
	}
	for _, p := range v.TemplateSearch {
		if bytes, err := ioutil.ReadFile(path.Join(p.String(), filename)); err == nil {
			return bytes, err
		}
	}
	return nil, fmt.Errorf("Could not find %s", filename)
}

func (v *templateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	args := flags.Args()
	if len(args) < 2 {
		app.Usage(ctx, "Missing template file")
		return nil
	}

	apis, mappings, err := resolve(ctx, args[0:1], v.Search, resolver.Options{
		ExtractCalls:   true,
		RemoveDeadCode: true,
	})
	if err != nil {
		return err
	}

	api := apis[0]

	if v.Gopath != "" {
		build.Default.GOPATH = filepath.FromSlash(v.Gopath)
	}
	mainTemplate := args[1]
	log.D(ctx, "Reading %v", api.Name())
	options := template.Options{
		Dir:     v.Dir,
		APIFile: api.Name(),
		Loader:  v.templateLoader,
		Globals: v.GlobalList,
		Tracer:  v.Tracer,
	}
	f, err := template.NewFunctions(ctx, api, mappings, options)
	if err != nil {
		return err
	}
	if err := f.Include(mainTemplate); err != nil {
		return fmt.Errorf("%s: %s", mainTemplate, err)
	}
	return nil
}

func cwd() string {
	p, _ := os.Getwd()
	return p
}
