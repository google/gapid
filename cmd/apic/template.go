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

var (
	dir        string
	tracer     string
	deps       string
	cmake      string
	gopath     string
	globalList flags.Strings
	searchPath file.PathList
)

func init() {
	verb := &app.Verb{
		Name:      "template",
		ShortHelp: "Passes the ast to a template for code generation",
	}
	verb.Flags.Raw.Var(&globalList, "G", "A global value setting for the template")
	verb.Flags.Raw.Var(&searchPath, "search", "The set of paths to search for includes")
	verb.Flags.Raw.StringVar(&dir, "dir", cwd(), "The output directory")
	verb.Flags.Raw.StringVar(&tracer, "t", "", "The template function trace expression")
	verb.Flags.Raw.StringVar(&deps, "deps", "", "The dependancies file to generate")
	verb.Flags.Raw.StringVar(&cmake, "cmake", "", "The cmake dependancies file to generate")
	verb.Flags.Raw.StringVar(&gopath, "gopath", "", "the go path to use when looking up packages")
	verb.Run = doTemplate
	app.AddVerb(verb)
}

func doTemplate(ctx log.Context, flags flag.FlagSet) error {
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
	if gopath != "" {
		build.Default.GOPATH = filepath.FromSlash(gopath)
	}
	mainTemplate := args[1]
	ctx.Info().S("api", apiName).Log("Reading")
	processor := gapil.NewProcessor()
	if len(searchPath) > 0 {
		processor.Loader = gapil.NewSearchLoader(searchPath)
	}
	compiled, errs := processor.Resolve(apiName)
	for path := range processor.Parsed {
		template.InputDep(path)
	}
	if err := gapil.CheckErrors(apiName, errs, maxErrors); err != nil {
		return err
	}
	options := template.Options{
		Dir:     dir,
		APIFile: apiName,
		Loader:  ioutil.ReadFile,
		Globals: globalList.Strings(),
		Tracer:  tracer,
	}
	f, err := template.NewFunctions(ctx, compiled, processor.Mappings, options)
	if err != nil {
		return err
	}
	if err := f.Include(mainTemplate); err != nil {
		return fmt.Errorf("%s: %s\n", mainTemplate, err)
	}
	writeDeps(ctx)
	writeCMake(ctx)
	return nil
}

func cwd() string {
	p, _ := os.Getwd()
	return p
}

func writeDeps(ctx log.Context) error {
	if len(deps) == 0 {
		return nil
	}
	ctx.Info().S("deps", deps).Log("Write")
	file, err := os.Create(deps)
	if err != nil {
		return err
	}
	defer file.Close()
	template.WriteDeps(ctx, file)
	return file.Close()
}

func writeCMake(ctx log.Context) error {
	if len(cmake) == 0 {
		return nil
	}
	ctx.Info().S("cmake", cmake).Log("Write")
	file, err := os.Create(cmake)
	if err != nil {
		return err
	}
	defer file.Close()
	template.WriteCMake(ctx, file)
	return file.Close()
}
