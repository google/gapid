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
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/annotate"
)

const maxErrors = 10

var (
	textFilename          string
	base64Filename        string
	globalsTextFilename   string
	globalsBase64Filename string
)

func main() {
	flag.StringVar(&textFilename, "text", "",
		"Filename for text output of atom snippets")
	flag.StringVar(&base64Filename, "base64", "",
		"Filename for base64 encoded binary objects of atom snippets")
	flag.StringVar(&globalsTextFilename, "globals_text", "",
		"Filename for text output of global state snippets")
	flag.StringVar(&globalsBase64Filename, "globals_base64", "",
		"Filename for base64 encoded binary objects of global state snippets")

	app.ShortHelp = "Annotates entities with metadata from static analysis"
	app.Run(Run)
}

func Run(ctx log.Context) error {
	args := flag.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Missing api file")
		return nil
	}
	if len(textFilename) == 0 && len(base64Filename) == 0 &&
		len(globalsTextFilename) == 0 && len(globalsBase64Filename) == 0 {
		app.Usage(ctx, "Specify a filename for -text/-globals_text and/or -base64/-globals_base64")
		return nil
	}
	apiName := args[0]
	ctx = ctx.S("api", apiName)
	ctx.Print("Parse and resolve")
	compiled, errs := gapil.Resolve(apiName)
	ctx.Print("Check for errors")
	if err := gapil.CheckErrors(apiName, errs, maxErrors); err != nil {
		return err
	}
	ctx.Print("Annotating")
	a := annotate.Annotate(compiled)
	if len(textFilename) != 0 {
		ctx := ctx.S("textFile", textFilename)
		if textFile, err := os.Create(textFilename); err != nil {
			return cause.Explain(ctx, err, "Failed to open {{textFile}} for text output")
		} else {
			buf := bufio.NewWriter(textFile)
			defer buf.Flush()
			a.Print(buf)
		}
	}
	if len(base64Filename) != 0 {
		ctx := ctx.S("base64File", base64Filename)
		if base64File, err := os.Create(base64Filename); err != nil {
			return cause.Explain(ctx, err, "Failed to open {{base64File}} for base64 output")
		} else {
			if err := a.Base64(base64File); err != nil {
				os.Remove(base64Filename)
				return cause.Explain(ctx, err, "Failed to encode {{base64File}}")
			}
		}
	}
	if len(globalsTextFilename) == 0 && len(globalsBase64Filename) == 0 {
		return nil
	}
	ctx.Print("Globals Analysis")
	g := a.Globals()
	if len(globalsTextFilename) != 0 {
		ctx := ctx.S("globalsTextFile", globalsTextFilename)
		if globalsTextFile, err := os.Create(globalsTextFilename); err != nil {
			return cause.Explain(ctx, err, "Failed to open {{globalsTextFile}} for text output")
		} else {
			buf := bufio.NewWriter(globalsTextFile)
			defer buf.Flush()
			fmt.Fprint(buf, &g)
		}
	}
	gg := a.GlobalsGrouped()
	if len(globalsBase64Filename) != 0 {
		ctx := ctx.S("globalsBase64File", globalsBase64Filename)
		if globalsBase64File, err := os.Create(globalsBase64Filename); err != nil {
			return cause.Explain(ctx, err, "Failed to open {{globalsBase64File}} for base64 output")
		} else {
			if err := gg.Base64(globalsBase64File); err != nil {
				os.Remove(globalsBase64Filename)
				return cause.Explain(ctx, err, "Failed to encode {{globalsBase64File}}")
			}
		}
	}
	return nil
}
