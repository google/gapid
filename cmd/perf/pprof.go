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
	"os"
	"regexp"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/os/shell"
)

func init() {
	verb := &app.Verb{
		Name:       "pprof",
		ShortHelp:  "Runs pprof",
		Run:        pprofVerb,
		ShortUsage: "<perfz> [[benchmark]:[link]]",
	}
	app.AddVerb(verb)
}

func pprofVerb(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() < 1 {
		app.Usage(ctx, "At least one argument expected, got %d", flags.NArg())
		return nil
	}

	perfzFile := flags.Arg(0)
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		return err
	}

	selectedBenchmark, link, err := selectLink(perfz, flags.Arg(1), regexp.MustCompile(`^(cpu|heap)/\d+$`))
	if err != nil {
		return err
	}

	profileFile, isTemp, err := link.Get().DiskFile()
	if isTemp {
		defer os.Remove(profileFile)
	}
	gapisBinary, isTemp, err := selectedBenchmark.Links[gapisLink].Get().DiskFile()
	if isTemp {
		defer os.Remove(gapisBinary)
	}

	return shell.Command("go", "tool", "pprof", gapisBinary, profileFile).Read(os.Stdin).Capture(os.Stdout, os.Stderr).Run(ctx)
}
