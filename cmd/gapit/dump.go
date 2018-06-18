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
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
)

type dumpVerb struct{ DumpFlags }

func init() {
	verb := &dumpVerb{}
	app.AddVerb(&app.Verb{
		Name:      "dump",
		ShortHelp: "Dump a textual representation of a .gfxtrace file",
		Action:    verb,
	})
}

func (verb *dumpVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	filepath, err := filepath.Abs(flags.Arg(0))
	ctx = log.V{"filepath": filepath}.Bind(ctx)
	if err != nil {
		return log.Err(ctx, err, "Could not find capture file")
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

	boxedCommands, err := client.Get(ctx, cp.Commands().Path())
	if err != nil {
		return log.Err(ctx, err, "Failed to acquire the capture's commands")
	}
	commands := boxedCommands.(*service.Commands).List

	if verb.ShowDeviceInfo {
		dev, err := json.MarshalIndent(c.Device, "", "  ")
		if err != nil {
			return log.Err(ctx, err, "Failed to marshal capture device to JSON")
		}
		fmt.Printf("Device Information:\n%s\n", string(dev))
	}

	if verb.ShowABIInfo {
		abi, err := json.MarshalIndent(c.ABI, "", "  ")
		if err != nil {
			return log.Err(ctx, err, "Failed to marshal capture abi to JSON")
		}
		fmt.Printf("Trace ABI Information:\n%s\n", string(abi))
	}

	if verb.ShowDeviceInfo || verb.ShowABIInfo {
		return nil // That's all that was requested
	}

	for _, c := range commands {
		if err := getAndPrintCommand(ctx, client, c, verb.Observations); err != nil {
			return err
		}
	}

	return nil
}
