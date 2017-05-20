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
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

type devicesVerb struct{ DevicesFlags }

func init() {
	verb := &devicesVerb{}
	app.AddVerb(&app.Verb{
		Name:      "devices",
		ShortHelp: "Lists the devices available",
		Auto:      verb,
	})
}

func (verb *devicesVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	devices, err := client.GetDevices(ctx)
	if err != nil {
		return log.Err(ctx, err, "Failed to get device list")
	}

	stdout := os.Stdout
	for i, p := range devices {
		fmt.Fprintf(stdout, "-- Device %v: %v --\n", i, p.Id.ID())
		o, err := client.Get(ctx, p.Path())
		if err != nil {
			fmt.Fprintf(stdout, "%v\n", log.Err(ctx, err, "Couldn't resolve device"))
			continue
		}
		d := o.(*device.Instance)
		jsonBytes, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			fmt.Fprintf(stdout, "%v\n", log.Err(ctx, err, "Couldn't marshal device to JSON"))
			continue
		}
		fmt.Fprintln(stdout, string(jsonBytes))
	}

	return nil
}
