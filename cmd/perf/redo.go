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
	"io"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

// TODO: ensure predictable choice of indices given a seed.

func init() {
	verb := &app.Verb{
		Name:       "redo",
		ShortHelp:  "Runs all benchmarks in the source .perfz and saves the output to a new file",
		ShortUsage: "<source> <destination>",
		Auto: &redoVerb{
			Output: "-",
		},
	}
	app.AddVerb(verb)
}

func clonePerfzInputs(p *Perfz) (*Perfz, error) {
	result := NewPerfz()

	for _, sb := range p.Benchmarks {
		db := result.NewBenchmarkWithName(sb.Input.Name)
		db.Input = sb.Input
		traceLink, err := result.NewLink(sb.Input.Trace.Get())
		if err != nil {
			return nil, err
		}

		db.Input.Trace = traceLink
		gapis, err := newGapisLink(db, sb.Input.Gapis.Get().Bundle)
		if err != nil {
			return nil, err
		}
		db.Input.Gapis = gapis
	}
	return result, nil
}

type redoVerb struct {
	Output string `help:"output results in JSON format"`
}

func (v *redoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 2 {
		app.Usage(ctx, "Two arguments expected, got %d", flags.NArg())
		return nil
	}

	perfzFile := flags.Arg(0)
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		return err
	}

	outputPerfz, err := clonePerfzInputs(perfz)
	if err != nil {
		return err
	}

	for _, bench := range outputPerfz.Benchmarks {
		if err := fullRun(ctx, bench); err != nil {
			return log.Err(ctx, err, "fullRun")
		}
	}

	err = outputPerfz.WriteTo(ctx, flags.Arg(1))
	if err != nil {
		return log.Err(ctx, err, "outputPerfz.WriteTo")
	}

	return writeAllFn(v.Output, func(w io.Writer) error {
		_, err := w.Write([]byte(outputPerfz.String()))
		return err
	})
}
