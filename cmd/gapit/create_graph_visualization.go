// Copyright (C) 2018 Google Inc.
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
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"os"
)

type createGraphVisualizationVerb struct{ CreateGraphVisualizationFlags }

func init() {
	verb := &createGraphVisualizationVerb{}
	app.AddVerb(&app.Verb{
		Name:      "create_graph_visualization",
		ShortHelp: "Create graph visualization file from capture",
		Action:    verb,
	})
}

func (verb *createGraphVisualizationVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}
	if verb.Format == "" {
		app.Usage(ctx, "specify an output format with --format <format> (supported formats: pbtxt and dot)")
		return nil
	}
	var format service.GraphFormat
	if verb.Format == "pbtxt" {
		format = service.GraphFormat_PBTXT
	} else if verb.Format == "dot" {
		format = service.GraphFormat_DOT
	} else {
		app.Usage(ctx, "invalid format (supported formats: pbtxt and dot)")
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, GapirFlags{}, flags.Arg(0), CaptureFileFlags{})
	if err != nil {
		return err
	}
	defer client.Close()

	log.I(ctx, "Creating graph visualization file from capture id: %s", capture.ID)

	graphVisualization, err := client.GetGraphVisualization(ctx, capture, format)
	if err != nil {
		return log.Errf(ctx, err, "GetGraphVisualization(%v)", capture)
	}

	filePath := verb.Out
	if filePath == "" {
		filePath = "graph_visualization." + verb.Format
	}

	file, err := os.Create(filePath)
	if err != nil {
		return log.Errf(ctx, err, "Creating file (%v)", filePath)
	}
	defer file.Close()

	bytesWritten, err := file.Write(graphVisualization)
	if err != nil {
		return log.Errf(ctx, err, "Error after writing %d bytes to file", bytesWritten)
	}
	return nil
}
