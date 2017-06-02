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
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/template"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "template",
		ShortHelp: "Passes the ast to a template for code generation",
		Auto: &templateVerb{
			Dir: cwd(),
		},
	})
}

type templateVerb struct {
	Dir        string        `help:"The output directory"`
	Tracer     string        `help:"The template function trace expression"`
	Deps       string        `help:"The dependancies file to generate"`
	Cmake      string        `help:"The cmake dependancies file to generate"`
	Gopath     string        `help:"the go path to use when looking up packages"`
	GlobalList flags.Strings `help:"A global value setting for the template"`
	Search     file.PathList `help:"The set of paths to search for includes"`
}

func (v *templateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	args := flags.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Missing api file")
		return nil
	}
	apiName := args[0]
	if len(args) < 2 {
		app.Usage(ctx, "Missing template file")
		return nil
	}
	if v.Gopath != "" {
		build.Default.GOPATH = filepath.FromSlash(v.Gopath)
	}
	mainTemplate := args[1]
	log.D(ctx, "Reading %v", apiName)
	processor := gapil.NewProcessor()
	if len(v.Search) > 0 {
		processor.Loader = gapil.NewSearchLoader(v.Search)
	}
	compiled, errs := processor.Resolve(apiName)
	for path := range processor.Parsed {
		template.InputDep(path)
	}
	if err := gapil.CheckErrors(apiName, errs, maxErrors); err != nil {
		return err
	}
	options := template.Options{
		Dir:     v.Dir,
		APIFile: apiName,
		Loader:  ioutil.ReadFile,
		Globals: v.GlobalList.Strings(),
		Tracer:  v.Tracer,
	}
	f, err := template.NewFunctions(ctx, compiled, processor.Mappings, options)
	if err != nil {
		return err
	}
	if err := f.Include(mainTemplate); err != nil {
		return fmt.Errorf("%s: %s\n", mainTemplate, err)
	}
	v.writeDeps(ctx)
	v.writeCMake(ctx)
	return nil
}

func cwd() string {
	p, _ := os.Getwd()
	return p
}

func (v *templateVerb) writeDeps(ctx context.Context) error {
	if len(v.Deps) == 0 {
		return nil
	}
	log.D(ctx, "Writing deps %v", v.Deps)
	file, err := os.Create(v.Deps)
	if err != nil {
		return err
	}
	defer file.Close()
	template.WriteDeps(ctx, file)
	return file.Close()
}

func (v *templateVerb) writeCMake(ctx context.Context) error {
	if len(v.Cmake) == 0 {
		return nil
	}
	log.D(ctx, "Writing cmake %v", v.Cmake)
	file, err := os.Create(v.Cmake)
	if err != nil {
		return err
	}
	defer file.Close()
	template.WriteCMake(ctx, file)
	return file.Close()
}
