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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

func init() {
	verb := &app.Verb{
		Name:       "rm",
		ShortHelp:  "Removes a benchmark from a .perfz file",
		ShortUsage: "<perfz> <benchmark>",
		Auto:       &rmVerb{},
	}
	app.AddVerb(verb)
}

type rmVerb struct {
	Output string `help:"output .perfz file, same as input if empty"`
}

func (v *rmVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 2 {
		app.Usage(ctx, "Two arguments expected, got %d", flags.NArg())
		return nil
	}

	perfzFile := flags.Arg(0)
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		return err
	}

	benchmarkName := flags.Arg(1)
	_, found := perfz.Benchmarks[benchmarkName]
	if !found {
		return fmt.Errorf("Benchmark not found: %s", benchmarkName)
	}

	delete(perfz.Benchmarks, benchmarkName)

	if v.Output == "" {
		v.Output = perfzFile
	}

	err = perfz.WriteTo(ctx, v.Output)
	if err != nil {
		return log.Err(ctx, err, "perfz.WriteTo")
	}

	return nil
}
