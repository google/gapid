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
	"github.com/google/gapid/gapil/bapi"
	"github.com/google/gapid/gapil/resolver"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "binary",
		ShortHelp: "Parses and resolves an api file and stores it to a binary file",
		Action:    &binaryVerb{},
	})
}

type binaryVerb struct {
	Search file.PathList `help:"The set of paths to search for includes"`
	Output file.Path     `help:"Output file path"`
}

func (v *binaryVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	apis, mappings, err := resolve(ctx, flags.Args(), v.Search, resolver.Options{})
	if err != nil {
		return err
	}
	data, err := bapi.Encode(apis, mappings)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(v.Output.String(), data, 0666)
}
