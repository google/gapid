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
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
)

type infoVerb struct{ InfoFlags }

func init() {
	verb := &infoVerb{}
	app.AddVerb(&app.Verb{
		Name:      "info",
		ShortHelp: "Prints information about a gfx trace capture file",
		Auto:      verb,
	})
}

func (verb *infoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}
	filename := flags.Arg(0)
	fmt.Println("reading file ", filename)
	fstat, err := os.Stat(filename)
	if err != nil {
		return err
	}
	fmt.Println("total file size ", fstat.Size())
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := capture.Import(ctx, "info", f)
	if err != nil {
		return err
	}

	c, err := capture.ResolveFromPath(ctx, path)
	if err != nil {
		return err
	}
	fmt.Println("name is %s", c.Name)
	return err
}
