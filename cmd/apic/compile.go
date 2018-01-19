// Copyright (C) 2018 Google Inc.
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
	"io/ioutil"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil/compiler"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "compile",
		ShortHelp: "Emits code generated from .api files",
		Action:    &compileVerb{},
	})
}

type compileVerb struct {
	Target string `help:"The target device ABI"`
	Output string `help:"The output file path"`
	Emit   struct {
		Exec   bool `help:"Emit executor logic"`
		Encode bool `help:"Emit encoder logic"`
	}
	Optimize bool
	Search   file.PathList `help:"The set of paths to search for includes"`
}

func (v *compileVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	api, mappings, err := resolve(ctx, v.Search, flags)
	if err != nil {
		return err
	}

	settings := compiler.Settings{
		EmitExec:   v.Emit.Exec,
		EmitEncode: v.Emit.Encode,
	}

	prog, err := compiler.Compile(api, mappings, settings)
	if err != nil {
		return err
	}

	if v.Optimize {
		prog.Module.Optimize()
	}

	obj, err := prog.Module.Object(v.Optimize)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(v.Output, obj, 0666); err != nil {
		return err
	}

	return nil
}
