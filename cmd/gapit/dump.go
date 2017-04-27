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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type dumpVerb struct{ DumpFlags }

func init() {
	verb := &dumpVerb{}
	app.AddVerb(&app.Verb{
		Name:      "dump",
		ShortHelp: "Dump a textual representation of a .gfxtrace file",
		Auto:      verb,
	})
}

func dumpMemory(ctx context.Context, client service.Service, stdout io.Writer, cp *path.Capture, o atom.Observation, i int) error {
	memoryPath := cp.Commands().Index(uint64(i)).MemoryAfter(0, o.Range.Base, o.Range.Size)
	memory, err := client.Get(ctx, memoryPath.Path())
	if err != nil {
		return fmt.Errorf("Failed to acquire the capture's memory: %v", err)
	}
	memoryInfo := memory.(*service.MemoryInfo)
	fmt.Fprintf(stdout, "%v\n", memoryInfo.Data)
	return nil
}

func (verb *dumpVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	schemaMsg, err := client.GetSchema(ctx)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the schema")
	}

	cp, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture file")
	}

	boxedCapture, err := client.Get(ctx, cp.Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture")
	}
	c := boxedCapture.(*service.Capture)

	boxedAtoms, err := client.Get(ctx, cp.Commands().Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to acquire the capture's atoms")
	}
	atoms := boxedAtoms.(*atom.List).Atoms

	stdout := os.Stdout

	dev, err := json.MarshalIndent(c.Device, "", "  ")
	if err != nil {
		return log.Err(ctx, err, "Failed to marshal capture device to JSON")
	}
	fmt.Fprintf(stdout, "%s\n", string(dev))

	if verb.ShowDeviceInfo {
		return nil // That's all that was requested
	}

	for i, a := range atoms {
		if dyn, ok := a.(*atom.Dynamic); ok && !verb.Raw {
			fmt.Fprintf(stdout, "%.6d %v\n", i, dyn.StringWithConstants(schemaMsg.Constants))
		} else {
			fmt.Fprintf(stdout, "%.6d %v\n", i, a)
		}
		if verb.Extras || verb.Observations {
			extras := a.Extras()
			for _, e := range extras.All() {
				fmt.Fprintf(stdout, "       %s: [%+v]\n", e.Class().Schema().Identity, e)
			}
			if verb.Observations && extras.Observations() != nil {
				for _, e := range extras.Observations().Reads {
					if err := dumpMemory(ctx, client, stdout, cp, e, i); err != nil {
						return err
					}
				}
				for _, e := range extras.Observations().Writes {
					if err := dumpMemory(ctx, client, stdout, cp, e, i); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
