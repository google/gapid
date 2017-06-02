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
	"strings"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
)

func init() {
	verb := &app.Verb{
		Name:       "run",
		ShortHelp:  "Runs a single benchmark and adds the results to the passed perfz file",
		ShortUsage: "<perfz> <trace>",
		Auto: &runVerb{
			Timeout:       -1,
			BenchmarkName: "default",
			SampleLimit:   50,
			SampleOrder:   "random",
			MaxWidth:      1280,
			MaxHeight:     720,
			BenchmarkType: "state",
			Seed:          1,
			Runs:          1,
			TextualOutput: "-",
		},
	}
	app.AddVerb(verb)
}

func fileLink(bench *Benchmark, filename string, nameTemplate string, bundle bool) (*Link, error) {
	dataEntry := &DataEntry{
		DataSource: FileDataSource(filename),
		Bundle:     bundle,
		Source:     filename,
	}
	link, err := bench.Root().NewLink(dataEntry)
	if err != nil {
		return nil, err
	}
	dataEntry.Name = strings.Replace(nameTemplate, "{{id}}", link.Key, -1)
	return link, nil
}

func newGapisLink(bench *Benchmark, bundle bool) (*Link, error) {
	return fileLink(bench, client.GapisPath.System(), "gapis/{{id}}", bundle)
}

type runVerb struct {
	Timeout           time.Duration // `help:"stop after this time interval"`
	BenchmarkName     string        // `help:"benchmark name"`
	BenchmarkComment  string        // `help:"benchmark comment"`
	SampleLimit       int           // `help:"how many samples to grab (if applicable)"`
	SampleOrder       string        // `help:"order in which to process samples (if applicable)"`
	MaxWidth          int           // `help:"maximum frame width to get"`
	MaxHeight         int           // `help:"maximum frame height to get"`
	BenchmarkType     string        // `help:"what to test [state|frames|startup]"`
	Seed              int64         // `help:"seed to pass to the random number generator"`
	Runs              int           // `help:"how many times to repeat the whole process"`
	EnableCPUProfile  bool          // `help:"profile GAPIS CPU usage"`
	EnableHeapProfile bool          // `help:"profile GAPIS heap usage"`
	BundleGapis       bool          // `help:"bundle GAPIS"`
	BundleTraces      bool          // `help:"bundle traces"`
	TextualOutput     string        // `help:"output results in JSON format"`
	PerfzOutput       string        // `help:"output .perfz file, same as input if empty"`
}

func (v *runVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 2 {
		app.Usage(ctx, "Two arguments expected, got %d", flags.NArg())
		return nil
	}

	perfzFile := flags.Arg(0)
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		log.I(ctx, "Could not load .perfz file, starting new one.")
		perfz = NewPerfz()
	}

	traceFile := flags.Arg(1)
	bench := perfz.NewBenchmarkWithName(v.BenchmarkName)
	if err := v.initializeBenchmarkInputFromFlags(flags, traceFile, bench); err != nil {
		return err
	}

	if err := fullRun(ctx, bench); err != nil {
		return log.Err(ctx, err, "fullRun")
	}

	if err := writeAllFn(v.TextualOutput, func(w io.Writer) error {
		_, err := w.Write([]byte(perfz.String()))
		return err
	}); err != nil {
		return log.Err(ctx, err, "writeAll")
	}

	if v.PerfzOutput == "" {
		v.PerfzOutput = perfzFile
	}

	err = perfz.WriteTo(ctx, v.PerfzOutput)
	if err != nil {
		return log.Err(ctx, err, "perfz.WriteTo")
	}

	return nil
}

func (v *runVerb) initializeBenchmarkInputFromFlags(flags flag.FlagSet, traceFile string, bench *Benchmark) error {
	args := &bench.Input
	args.Name = v.BenchmarkName
	args.MaxSamples = v.SampleLimit

	trace, err := fileLink(bench, traceFile, "traces/{{id}}.gfxtrace", v.BundleTraces)
	if err != nil {
		return err
	}
	args.Trace = trace

	g, err := newGapisLink(bench, v.BundleGapis)
	if err != nil {
		return err
	}
	args.Gapis = g

	args.MaxFrameHeight = v.MaxHeight
	args.MaxFrameWidth = v.MaxWidth
	args.Seed = v.Seed
	args.BenchmarkType = v.BenchmarkType
	args.SampleOrder = v.SampleOrder
	args.Runs = v.Runs
	args.Timeout = v.Timeout
	args.EnableHeapProfile = v.EnableHeapProfile
	args.EnableCPUProfile = v.EnableCPUProfile
	args.Comment = v.BenchmarkComment

	return err
}
