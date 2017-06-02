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
	"io"
	"os"
	"os/exec"
	"text/template"

	"github.com/google/gapid/core/app"
)

const plotTemplate = `#!/usr/bin/env gnuplot
$dataset1 << EOD
#index average median min max
{{range $idx, $s := .First}}
{{$sm := $s.Values}}{{$s.Index}} {{$sm.Average.Seconds}} {{$sm.Median.Seconds}} {{$sm.Min.Seconds}} {{$sm.Max.Seconds}}{{end}}
EOD
{{if .Second}}$dataset2 << EOD
#index average median min max
{{range $idx, $s := .Second}}
{{$sm := $s.Values}}{{$s.Index}} {{$sm.Average.Seconds}} {{$sm.Median.Seconds}} {{$sm.Min.Seconds}} {{$sm.Max.Seconds}}{{end}}
EOD{{end}}

reset
set terminal svg size 1600, 900
{{.Extra}}
set title "{/=20 {{.BenchName}}}\n{/=16 {{.Name1}}{{if .Second}} vs {{.Name2}}{{end}}}"
set xlabel "Index"
set ylabel "Seconds"
set style fill transparent solid 0.2 noborder

plot {{if .ShowMinMax}}'$dataset1' using 1:4:5 with filledcurves title 'min..max:{{.Name1}}', \
     {{end}}{{if .ShowAverage}}'$dataset1' using 1:2 with lp lt 3 pt 7 ps 0.5 lw 1 title 'avg:{{.Name1}}', \
     {{end}}'$dataset1' using 1:3 with lp lt 1 pt 7 ps 0.5 lw 1 title 'median:{{.Name1}}'{{if .Second}}, \
     {{if .ShowMinMax}}'$dataset2' using 1:4:5 with filledcurves title 'min..max:{{.Name2}}', \
     {{end}}{{if .ShowAverage}}'$dataset2' using 1:2 with lp lt 4 pt 7 ps 0.5 lw 1 title 'avg:{{.Name2}}', \
     {{end}}'$dataset2' using 1:3 with lp lt 5 pt 7 ps 0.5 lw 1 title 'median:{{.Name2}}'{{end}}
`

func init() {
	verb := &app.Verb{
		Name:       "plot",
		ShortHelp:  "Plots samples from a benchmark out of one or two perfz files",
		ShortUsage: "<perfz> [perfz]",
		Auto: &plotVerb{
			ShowMinMax:  true,
			ShowAverage: true,
			RunGnuplot:  true,
			Output:      "-",
		},
	}
	app.AddVerb(verb)
}

type plotVerb struct {
	ShowMinMax          bool   `help:"show min and max"`
	ShowAverage         bool   `help:"show average"`
	RunGnuplot          bool   `help:"run gnuplot"`
	Output              string `help:"output file"`
	BenchmarkNameToPlot string `help:"benchmark name"`
}

func getPlotData(ctx context.Context, perfzFile string, benchmarkName string) (IndexedMultisamples, string, error) {
	perfz, err := LoadPerfz(ctx, perfzFile, flagVerifyHashes)
	if err != nil {
		return IndexedMultisamples{}, "", err
	}
	bench, err := selectBenchmark(perfz, benchmarkName)
	if err != nil {
		return IndexedMultisamples{}, "", err
	}
	return bench.Samples.IndexedMultisamples(), bench.Input.Name, nil
}

func (v *plotVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() < 1 {
		app.Usage(ctx, "At least one argument expected.")
		return nil
	}

	args := struct {
		First       IndexedMultisamples
		Second      IndexedMultisamples
		Name1       string
		Name2       string
		BenchName   string
		ShowAverage bool
		ShowMinMax  bool
		Extra       string
	}{
		Name1:       flags.Arg(0),
		Name2:       flags.Arg(1),
		ShowAverage: v.ShowAverage,
		ShowMinMax:  v.ShowMinMax,
		Extra: func() string {
			if v.RunGnuplot && v.Output != "-" {
				return fmt.Sprintf(`set output "%s"`, v.Output)
			}
			return ""
		}(),
	}

	var err error
	args.First, args.BenchName, err = getPlotData(ctx, args.Name1, v.BenchmarkNameToPlot)
	if err != nil {
		return err
	}
	if flags.NArg() >= 2 {
		args.Second, _, err = getPlotData(ctx, args.Name2, args.BenchName)
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("plot").Parse(plotTemplate)
	if err != nil {
		return err
	}

	writeScript := func(w io.Writer) error {
		return tmpl.Execute(w, args)
	}

	if v.RunGnuplot {
		fn, _, err := FuncDataSource(writeScript).DiskFile()
		if err != nil {
			return err
		}
		defer os.Remove(fn)
		cmd := exec.Command("gnuplot", fn)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return writeAllFn(v.Output, writeScript)
}
