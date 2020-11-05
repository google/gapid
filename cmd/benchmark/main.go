// Copyright (C) 2020 Google Inc.
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

// The embed command is used to embed text files into Go executables as strings.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/os/shell"
)

var (
	gapit string
	root  string
)

func main() {
	flag.StringVar(&gapit, "gapit", "gapit", "the path to the gapit command")
	flag.StringVar(&root, "root", "", "the root directory to resolves paths against")
	app.ShortHelp = "benchmark: A tool to run and summarize gapit benchmarks."
	app.Run(run)
}

func run(ctx context.Context) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}

	tmpOut, err := ioutil.TempDir("", "benchmark")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpOut)

	res := []*results{}
	for i := range cfg.Traces {
		r, err := runTrace(ctx, cfg, i, file.Abs(tmpOut))
		if err != nil {
			return err
		}
		res = append(res, r)
	}

	fmt.Println("------------------------")
	printResults(res)
	fmt.Println("------------------------")
	printSummary(res)

	return nil
}

type results struct {
	name   string
	titles []string
	values [][]float64
}

func (r *results) append(path file.Path) error {
	in, err := os.Open(path.System())
	if err != nil {
		return err
	}
	defer in.Close()

	records, err := csv.NewReader(in).ReadAll()
	if err != nil {
		return err
	}

	if len(records) != 2 {
		return fmt.Errorf("Expected two summary rows, got %d", len(records))
	}

	if r.titles != nil && len(r.titles) != len(records[0]) {
		return fmt.Errorf("Unmatched number of titles: got %d, expected %d", len(records[0]), len(r.titles))
	}
	r.titles = records[0]

	if len(records[0]) != len(records[1]) {
		return fmt.Errorf("Unmatched number of values: got %d, expected %d", len(records[1]), len(records[0]))
	}

	values := make([]float64, len(records[1]))
	for i := range values {
		v, err := strconv.ParseFloat(records[1][i], 64)
		if err != nil {
			return fmt.Errorf("Failed to parse summary value \"%s\": %v", records[1][i], err)
		}
		values[i] = v
	}
	r.values = append(r.values, values)

	return nil
}

func runTrace(ctx context.Context, cfg *config, idx int, tmpOut file.Path) (*results, error) {
	trace := cfg.Traces[idx]
	log.I(ctx, "Measuring %v (%v)...", trace.Name, trace.Input)

	res := &results{
		name: trace.Name,
	}
	for i := 0; i < cfg.Iterations; i++ {
		summaryOut := tmpOut.Join(fmt.Sprintf("summry_%d_%d.csv", idx, i))
		args := []string{
			"benchmark2",
			"--numdraws", strconv.Itoa(cfg.Draws),
			"--summaryout", summaryOut.System(),
		}
		if trace.Secondary {
			args = append(args, "--secondary")
		}
		for _, path := range trace.Paths {
			args = append(args, "--paths", path)
		}
		args = append(args, file.Abs(root).Join(trace.Input).System())

		err := shell.Command(gapit, args...).
			Verbose().
			In(root).
			Run(ctx)
		if err != nil {
			return nil, err
		}

		if err := res.append(summaryOut); err != nil {
			return nil, err
		}
	}

	return res, nil
}

type config struct {
	Iterations int
	Draws      int
	Traces     []struct {
		Name      string
		Input     string
		Paths     []string
		Secondary bool
	}
}

func readConfig() (*config, error) {
	args := flag.Args()
	if len(args) != 1 {
		return nil, errors.New("Expected a config file as a paramter")
	}

	in, err := os.Open(args[0])
	if err != nil {
		return nil, err
	}
	defer in.Close()

	dec := json.NewDecoder(in)
	c := config{
		Iterations: 1,
	}
	if err := dec.Decode(&c); err != nil {
		return nil, err
	}

	if len(c.Traces) == 0 {
		return nil, errors.New("No traces in config file, expected at least one")
	}

	return &c, nil
}

func printResults(rs []*results) {
	out := csv.NewWriter(os.Stdout)
	out.Write(append([]string{"Application"}, rs[0].titles...))

	for _, r := range rs {
		for _, vs := range r.values {
			printResultRow(out, r.name, vs)
		}
	}

	out.Flush()
}

func printSummary(rs []*results) {
	out := csv.NewWriter(os.Stdout)
	out.Write(append([]string{"Application"}, rs[0].titles...))

	for _, r := range rs {
		avgs := make([]float64, len(r.values[0]))
		for _, vs := range r.values {
			for i := range avgs {
				avgs[i] += vs[i]
			}
		}
		for i := range avgs {
			avgs[i] /= float64(len(r.values))
		}
		printResultRow(out, r.name, avgs)
	}

	out.Flush()
}

func printResultRow(out *csv.Writer, name string, row []float64) {
	r := make([]string, 1+len(row))
	r[0] = name
	for i, v := range row {
		r[i+1] = fmt.Sprintf("%0.3f", v)
	}
	out.Write(r)
}
