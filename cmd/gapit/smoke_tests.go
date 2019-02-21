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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
)

type smokeTestsVerb struct{ SmokeTestsFlags }

func init() {
	verb := &smokeTestsVerb{}
	app.AddVerb(&app.Verb{
		Name:      "smoke_test",
		ShortHelp: "Run smoke tests on a set of traces using the gapit command found in PATH",
		Action:    verb,
	})
}

func (verb *smokeTestsVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one argument expected: path to directory containing traces to run smoke tests on")
		return nil
	}

	// Record starting working directory
	startwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create temporary directory
	tmpdir, err := ioutil.TempDir(startwd, "gapit-test.")
	if err != nil {
		return err
	}

	// For each trace, run gapit tests
	tracedir := flags.Arg(0)
	if !filepath.IsAbs(tracedir) {
		tracedir = filepath.Join(startwd, tracedir)
	}
	traces, err := ioutil.ReadDir(tracedir)
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
		tracepath := filepath.Join(tracedir, trace)
		if err := os.Mkdir(tracewd, 0777); err != nil {
			log.Fatal(err)
		}
		os.Chdir(tracewd)
		testTrace(&nbErr, tracepath)
		os.Chdir(startwd)
	}

	if !atLeastOneTraceFound {
		return errors.New("No file ending with '.gfxtrace' found in trace directory")
	}

	if nbErr > 0 {
		errStr := "error"
		if nbErr > 1 {
			errStr = "errors"
		}
		fmt.Printf("%d %s found, see logs in %s\n", nbErr, errStr, tmpdir)
	} else {
		// Temporary directory is removed only when there are no errors
		os.RemoveAll(tmpdir)
	}
	return nil
}

func testTrace(nbErr *int, tracepath string) {
	trace := filepath.Base(tracepath)
	gapit(nbErr, "commands", tracepath)
	gapit(nbErr, "create_graph_visualization", "-format", "dot", "-out", trace+".dot", tracepath)
	gapit(nbErr, "dump", tracepath)
	gapit(nbErr, "dump_fbo", tracepath)
	gapit(nbErr, "dump_pipeline", tracepath)
	gapit(nbErr, "dump_replay", tracepath)
	gapit(nbErr, "dump_resources", tracepath)
	gapit(nbErr, "export_replay", tracepath)
	gapit(nbErr, "memory", tracepath)
	gapit(nbErr, "stats", tracepath)
	gapit(nbErr, "trim", "-frames-start", "2", "-frames-count", "2", tracepath)
	gapit(nbErr, "unpack", tracepath)
}

func gapit(nbErr *int, args ...string) {
	// Print command description
	arglen := len(args)
	argsWithoutTrace := args[:arglen-1]
	trace := filepath.Base(args[arglen-1])
	printCmd := "gapit " + strings.Join(argsWithoutTrace, " ") + " " + trace
	fmt.Printf("%-70.70s ", printCmd)

	// Execute, check error, print status
	cmd := exec.Command("gapit", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// Here the gapit command raised an error
			fmt.Printf("ERR\n")
			*nbErr += 1
		} else {
			// Here the error comes from somewhere else
			log.Fatal(err)
		}
	} else {
		fmt.Printf("OK\n")
	}

	// Write output
	verb := args[0]
	err = ioutil.WriteFile(verb+".log", output, 0666)
	if err != nil {
		log.Fatal(err)
	}
}
