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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/layout"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

var (
	gapitArg  = flag.String("gapit", "", "Path to gapit executable")
	keepArg   = flag.Bool("keep", false, "Keep the temporary directory even if no errors are found")
	tracesArg = flag.String("traces", "traces", "The directory containing traces to run smoke tests on")
)

func main() {
	app.ShortHelp = "smoketests runs a series of smoke tests"
	app.Name = "smoketests"
	app.Run(run)
}

func sigInt(ctx context.Context, c chan os.Signal) {
	s := <-c
	log.F(ctx, true, "Received signal: ", s)
}

func run(ctx context.Context) error {

	// Register SIGINT handler
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go sigInt(ctx, signalChan)

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

	var gapitPath file.Path
	if *gapitArg != "" {
		gapitPath = file.Abs(*gapitArg)
	} else {
		gapitPath, err = layout.Gapit(ctx)
		if err != nil {
			return err
		}
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
		if err := testTrace(ctx, &nbErr, gapitPath.System(), tracepath); err != nil {
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

	if nbErr > 0 {
		// Return a non-nil error to force a non-zero exit value
		return fmt.Errorf("Smoketests: %d %s found", nbErr, errStr)
	}
	return nil
}

func testTrace(ctx context.Context, nbErr *int, gapitPath string, tracepath string) error {

	// The trace basename is used for some commands argument
	trace := filepath.Base(tracepath)

	tests := [][]string{

		{"commands", tracepath},
		{"commands", "-groupbyapi", tracepath},
		{"commands", "-groupbydrawcall", tracepath},
		{"commands", "-groupbyframe", tracepath},
		{"commands", "-groupbythread", tracepath},
		{"commands", "-groupbyusermarkers", tracepath},
		{"commands", "-groupbysubmission", tracepath},
		{"commands", "-maxchildren", "1", tracepath},
		{"commands", "-name", "vkQueueSubmit", tracepath},
		{"commands", "-observations-ranges", tracepath},
		{"commands", "-observations-data", tracepath},
		{"commands", "-raw", tracepath},
		{"create_graph_visualization", "-format", "dot", "-out", trace + ".dot", tracepath},
		{"dump", tracepath},
		{"dump_fbo", tracepath},
		{"dump_pipeline", tracepath},
		{"dump_resources", tracepath},
		{"memory", tracepath},
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
	printCmd := "gapit " + strings.Join(argsWithoutTrace, " ") + " " + trace

	// Execute, check error, print status
	cmd := exec.Command(gapitPath, append(layout.GoArgs(ctx), args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// Here the gapit command raised an error
			fmt.Printf("FAIL %s\n", printCmd)
			fmt.Println("===============================================")
			fmt.Println(string(output))
			fmt.Println("===============================================")
			*nbErr++
		} else {
			// Here the error comes from somewhere else
			return err
		}
	} else {
		fmt.Printf("PASS %s\n", printCmd)
	}

	// Write output in log
	logFilename := strings.Join(argsWithoutTrace, "_") + ".log"
	if err := ioutil.WriteFile(logFilename, output, 0666); err != nil {
		return err
	}

	return nil
}
