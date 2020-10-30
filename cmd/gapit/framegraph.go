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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// Default output file name.
const defaultOutFilename = "framegraph.dot"

type framegraphVerb struct{ FramegraphFlags }

func init() {
	verb := &framegraphVerb{}
	app.AddVerb(&app.Verb{
		Name:      "framegraph",
		ShortHelp: "Create frame graph (in DOT format) from capture",
		Action:    verb,
	})
}

// Run is the main logic for the 'gapit framegraph' command.
func (verb *framegraphVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	captureFilename := flags.Arg(0)

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, GapirFlags{}, captureFilename, CaptureFileFlags{})
	if err != nil {
		return err
	}
	defer client.Close()

	boxedFramegraph, err := client.Get(ctx, capture.Framegraph().Path(), nil)
	if err != nil {
		return err
	}
	framegraph := boxedFramegraph.(*api.Framegraph)

	dot := framegraph2dot(framegraph, captureFilename)

	// Write the DOT representation of the framegraph into a file
	filePath := verb.Out
	if filePath == "" {
		filePath = defaultOutFilename
	}
	file, err := os.Create(filePath)
	if err != nil {
		return log.Errf(ctx, err, "Creating file (%v)", filePath)
	}
	defer file.Close()
	bytesWritten, err := fmt.Fprint(file, dot)
	if err != nil {
		return log.Errf(ctx, err, "Error after writing %d bytes to file", bytesWritten)
	}

	return nil
}

// framegraph2dot formats a framegraph in the Graphviz DOT format.
// https://graphviz.org/doc/info/lang.html
func framegraph2dot(framegraph *api.Framegraph, captureFilename string) string {
	s := "digraph agiFramegraph {\n"
	// Graph title: use capture filename, on top
	s += "label = \"" + captureFilename + "\";\n"
	s += "labelloc = \"t\";\n"
	// Use monospace font everywhere
	s += "node [fontname = \"Monospace\"];\n"
	s += "\n"
	// Node IDs cannot start with a digit, so use "n<node.Id>", e.g. n0 n1 n2
	for _, node := range framegraph.Nodes {
		s += fmt.Sprintf("n%v [label=\"%s\"];\n", node.Id, node.Text)
	}
	s += "\n"
	for _, edge := range framegraph.Edges {
		s += fmt.Sprintf("n%v -> n%v;\n", edge.Origin, edge.Destination)
	}
	s += "}\n"
	return s
}
