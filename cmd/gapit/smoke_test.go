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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

const tracedirName = "traces"

type smokeTestVerb struct{ smokeTestFlags }

func init() {
	verb := &smokeTestVerb{}

	app.AddVerb(&app.Verb{
		Name:      "smoke_test",
		ShortHelp: "Runs gapit smoke tests",
		Action:    verb,
	})
}

func (verb *smokeTestVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	// Record starting directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Create and change to temporary directory
	tmpdir, err := ioutil.TempDir(".", "gapit-test.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Temporary directory: %s\n", tmpdir)

	// For each trace, run gapit tests
	tracedir := filepath.Join(wd, tracedirName)
	traces, err := ioutil.ReadDir(tracedir)
	nbErr := 0

	for _, t := range traces {
		trace := t.Name()
		if !strings.HasSuffix(trace, ".gfxtrace") {
			continue
		}
		tracewd := filepath.Join(tmpdir, trace)
		tracepath := filepath.Join(tracedir, trace)
		if err := os.Mkdir(tracewd, 0777); err != nil {
			log.Fatal(err)
		}
		os.Chdir(tracewd)
		testTrace(&nbErr, tracepath)
		os.Chdir(wd)
	}

	if nbErr > 0 {
		errStr := "error"
		if nbErr > 1 {
			errStr = "errors"
		}
		fmt.Printf("%d %s found, see logs in %s\n", nbErr, errStr, tmpdir)
	} else {
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
