// Copyright (C) 2019 Google Inc.
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

// The smoketests command runs a series of smoke tests.
package main

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

var (
	gapitArg  = flag.String("gapit", "gapit", "Path to gapit executable")
	keepArg   = flag.Bool("keep", false, "Keep the temporary directory even if no errors are found")
	tracesArg = flag.String("traces", "traces", "The directory containing traces to run smoke tests on")
)

func main() {
	app.ShortHelp = "smoketests runs a series of smoke tests"
	app.Name = "smoketests"
	app.Run(run)
}

func run(ctx context.Context) error {

	// Record starting working directory
	startwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Trace directory argument
	traceFilemode, err := os.Lstat(*tracesArg)
	if err != nil {
		return err
	}
	if !traceFilemode.IsDir() {
		return errors.New("Trace path given in argument is not a directory")
	}
	traceDir := *tracesArg
	if !filepath.IsAbs(traceDir) {
		traceDir = filepath.Join(startwd, traceDir)
	}

	// Gapit path argument
	gapitPath := *gapitArg
	// We assume gapitPath is found in PATH environment unless it
	// contains a separator, in which case we make it absolute
	hasSeparator := strings.ContainsAny(gapitPath, string(filepath.Separator))
	if hasSeparator && !filepath.IsAbs(gapitPath) {
		gapitPath = filepath.Join(startwd, gapitPath)
	}

	// Create temporary directory
	tmpdir, err := ioutil.TempDir(startwd, "smoketests.")
	if err != nil {
		return err
	}

	// For each trace, run gapit tests
	traces, err := ioutil.ReadDir(traceDir)
	atLeastOneTraceFound := false
	nbErr := 0

	for _, t := range traces {

		// Filter out non-trace files
		if t.IsDir() {
			continue
		}
		trace := t.Name()
		if !strings.HasSuffix(trace, ".gfxtrace") {
			continue
		}

		atLeastOneTraceFound = true

		// Run smoke tests on trace under temporary directory
		tracewd := filepath.Join(tmpdir, trace)
		tracepath := filepath.Join(traceDir, trace)
		if err := os.Mkdir(tracewd, 0777); err != nil {
			return err
		}
		os.Chdir(tracewd)
		if err := testTrace(ctx, &nbErr, gapitPath, tracepath); err != nil {
			return err
		}
		os.Chdir(startwd)
	}

	if !atLeastOneTraceFound {
		return errors.New("No file ending with '.gfxtrace' found in trace directory")
	}

	// Print number of errors
	errStr := "errors"
	if nbErr == 1 {
		errStr = "error"
	}
	log.I(ctx, "%d %s found", nbErr, errStr)

	// Temporary directory is kept if there are errors or if the user
	// explicitely asked to keep it
	if nbErr > 0 || *keepArg {
		log.I(ctx, "See logs in %s", tmpdir)
	} else {
		os.RemoveAll(tmpdir)
	}
	return nil
}

func testTrace(ctx context.Context, nbErr *int, gapitPath string, tracepath string) error {

	// The trace basename is used for some commands argument
	trace := filepath.Base(tracepath)

	tests := [][]string{
		{"commands", tracepath},
		{"create_graph_visualization", "-format", "dot", "-out", trace + ".dot", tracepath},
		{"dump", tracepath},
		{"dump_fbo", tracepath},
		{"dump_pipeline", tracepath},
		{"dump_replay", tracepath},
		{"dump_resources", tracepath},
		{"export_replay", tracepath},
		{"memory", tracepath},
		{"stats", tracepath},
		{"trim", tracepath},
		{"unpack", tracepath},
	}

	for _, test := range tests {
		if err := gapit(ctx, nbErr, gapitPath, test...); err != nil {
			return err
		}
	}

	return nil
}

func gapit(ctx context.Context, nbErr *int, gapitPath string, args ...string) error {
	// Print command description
	arglen := len(args)
	argsWithoutTrace := args[:arglen-1]
	trace := filepath.Base(args[arglen-1])
	printCmd := gapitPath + " " + strings.Join(argsWithoutTrace, " ") + " " + trace
	log.I(ctx, "run: %s", printCmd)

	// Execute, check error, print status
	cmd := exec.Command(gapitPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// Here the gapit command raised an error
			log.I(ctx, "ERROR %s", printCmd)
			*nbErr += 1
		} else {
			// Here the error comes from somewhere else
			return err
		}
	} else {
		log.I(ctx, "OK %s", printCmd)
	}

	// Write output in log
	verb := args[0]
	if err := ioutil.WriteFile(verb+".log", output, 0666); err != nil {
		return err
	}

	return nil
}
