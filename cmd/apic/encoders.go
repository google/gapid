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
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapil/encoder"
	"github.com/google/gapid/gapil/resolver"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "encoders",
		ShortHelp: "Emits generated code to encode types from .api files",
		Action:    &encodersVerb{},
	})
}

type encodersVerb struct {
	Output    string        `help:"The output file path"`
	Namespace string        `help:"C++ namespace for the generated code"`
	Search    file.PathList `help:"The set of paths to search for includes"`
}

func (v *encodersVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	apis, mappings, err := resolve(ctx, flags.Args(), v.Search, resolver.Options{})
	if err != nil {
		return err
	}

	out, err := os.Create(v.Output)
	if err != nil {
		return err
	}
	defer out.Close()

	settings := encoder.Settings{
		Namespace: v.Namespace,
		Out:       out,
	}

	return encoder.GenerateEncoders(apis, mappings, settings)
}
