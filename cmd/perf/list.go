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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

func init() {
	verb := &app.Verb{
		Name:       "list",
		ShortHelp:  "Lists benchmarks and associated data in .perfz",
		Run:        listVerb,
		ShortUsage: "<perfz>",
	}
	app.AddVerb(verb)
}

func listVerb(ctx log.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "One argument expected, got %d", flags.NArg())
		return nil
	}

	perfzFile := flags.Arg(0)
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		return err
	}

	for _, b := range perfz.Benchmarks {
		fmt.Println(b.Input.Name)
		for k, l := range b.Links {
			bundled := ""
			if l.Get().Bundle {
				bundled = "*"
			}
			fmt.Printf("  %s%s\n", k, bundled)
		}
	}

	return nil
}
