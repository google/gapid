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

var (
	flagTimeout           time.Duration
	flagBenchmarkName     string
	flagBenchmarkComment  string
	flagSampleLimit       int
	flagSampleOrder       string
	flagMaxWidth          int
	flagMaxHeight         int
	flagBenchmarkType     string
	flagSeed              int64
	flagRuns              int
	flagEnableCpuProfile  bool
	flagEnableHeapProfile bool
	flagBundleGapis       bool
	flagBundleTraces      bool
	flagTextualOutput     string
	flagPerfzOutput       string
)

func init() {
	verb := &app.Verb{
		Name:       "run",
		ShortHelp:  "Runs a single benchmark and adds the results to the passed perfz file",
		Run:        runVerb,
		ShortUsage: "<perfz> <trace>",
	}

	verb.Flags.Raw.DurationVar(&flagTimeout, "timeout", -1, "stop after this time interval")
	verb.Flags.Raw.StringVar(&flagBenchmarkName, "name", "default", "benchmark name")
	verb.Flags.Raw.StringVar(&flagBenchmarkComment, "comment", "", "benchmark comment")
	verb.Flags.Raw.IntVar(&flagSampleLimit, "samples", 50, "how many samples to grab (if applicable)")
	verb.Flags.Raw.StringVar(&flagSampleOrder, "order", "random", "order in which to process samples (if applicable)")
	verb.Flags.Raw.IntVar(&flagMaxWidth, "max_width", 1280, "maximum frame width to get")
	verb.Flags.Raw.IntVar(&flagMaxHeight, "max_height", 720, "maximum frame height to get")
	verb.Flags.Raw.StringVar(&flagBenchmarkType, "what", "state", "what to test [state|frames|startup]")
	verb.Flags.Raw.Int64Var(&flagSeed, "seed", 1, "seed to pass to the random number generator")
	verb.Flags.Raw.IntVar(&flagRuns, "runs", 1, "how many times to repeat the whole process")
	verb.Flags.Raw.BoolVar(&flagEnableCpuProfile, "cpu", false, "profile GAPIS CPU usage")
	verb.Flags.Raw.BoolVar(&flagEnableHeapProfile, "heap", false, "profile GAPIS heap usage")
	verb.Flags.Raw.BoolVar(&flagBundleGapis, "bg", false, "bundle GAPIS")
	verb.Flags.Raw.BoolVar(&flagBundleTraces, "bt", false, "bundle traces")
	verb.Flags.Raw.StringVar(&flagTextualOutput, "json", "-", "output results in JSON format")
	verb.Flags.Raw.StringVar(&flagPerfzOutput, "o", "", "output .perfz file, same as input if empty")
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

func initializeBenchmarkInputFromFlags(flags flag.FlagSet, traceFile string, bench *Benchmark) error {
	args := &bench.Input
	args.Name = flagBenchmarkName
	args.MaxSamples = flagSampleLimit

	trace, err := fileLink(bench, traceFile, "traces/{{id}}.gfxtrace", flagBundleTraces)
	if err != nil {
		return err
	}
	args.Trace = trace

	g, err := newGapisLink(bench, flagBundleGapis)
	if err != nil {
		return err
	}
	args.Gapis = g

	args.MaxFrameHeight = flagMaxHeight
	args.MaxFrameWidth = flagMaxWidth
	args.Seed = flagSeed
	args.BenchmarkType = flagBenchmarkType
	args.SampleOrder = flagSampleOrder
	args.Runs = flagRuns
	args.Timeout = flagTimeout
	args.EnableHeapProfile = flagEnableHeapProfile
	args.EnableCpuProfile = flagEnableCpuProfile
	args.Comment = flagBenchmarkComment

	return err
}

func runVerb(ctx context.Context, flags flag.FlagSet) error {
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
	bench := perfz.NewBenchmarkWithName(flagBenchmarkName)
	if err := initializeBenchmarkInputFromFlags(flags, traceFile, bench); err != nil {
		return err
	}

	if err := fullRun(ctx, bench); err != nil {
		return log.Err(ctx, err, "fullRun")
	}

	if err := writeAllFn(flagTextualOutput, func(w io.Writer) error {
		_, err := w.Write([]byte(perfz.String()))
		return err
	}); err != nil {
		return log.Err(ctx, err, "writeAll")
	}

	if flagPerfzOutput == "" {
		flagPerfzOutput = perfzFile
	}

	err = perfz.WriteTo(ctx, flagPerfzOutput)
	if err != nil {
		return log.Err(ctx, err, "perfz.WriteTo")
	}

	return nil
}
