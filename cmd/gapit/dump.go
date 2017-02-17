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
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary/schema"
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

func dumpMemory(ctx log.Context, client service.Service, stdout io.Writer, capture *path.Capture, o atom.Observation, i int) error {
	memoryPath := capture.Commands().Index(uint64(i)).MemoryAfter(0, o.Range.Base, o.Range.Size)
	memory, err := client.Get(ctx, memoryPath.Path())
	if err != nil {
		return fmt.Errorf("Failed to acquire the capture's memory: %v", err)
	}
	memoryInfo := memory.(*service.MemoryInfo)
	fmt.Fprintf(stdout, "%v\n", memoryInfo.Data)
	return nil
}

func (verb *dumpVerb) Run(ctx log.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("Could not find capture file '%s': %v", flags.Arg(0), err)
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return fmt.Errorf("Failed to connect to the GAPIS server: %v", err)
	}
	defer client.Close()

	schemaMsg, err := client.GetSchema(ctx)
	if err != nil {
		return fmt.Errorf("Failed to load the schema: %v", err)
	}

	capture, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return fmt.Errorf("Failed to load the capture file '%v': %v", filepath, err)
	}

	boxedAtoms, err := client.Get(ctx, capture.Commands().Path())
	if err != nil {
		return fmt.Errorf("Failed to acquire the capture's atoms: %v", err)
	}
	atoms := boxedAtoms.(*atom.List).Atoms

	stdout := ctx.Raw("").Writer()

	if verb.ShowDeviceInfo {
		for _, a := range atoms {
			if da, ok := a.(*atom.Dynamic); ok {
				if da.Class().Schema().Name() == "Architecture" {
					for _, ex := range da.Extras().All() {
						if obj, ok := ex.(*schema.Object); ok {
							if obj.Type.Name() == "DeviceInfo" {
								fmt.Println(obj.String())
							}
						}
					}
				}
			}
		}
		return nil
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
					if err := dumpMemory(ctx, client, stdout, capture, e, i); err != nil {
						return err
					}
				}
				for _, e := range extras.Observations().Writes {
					if err := dumpMemory(ctx, client, stdout, capture, e, i); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
