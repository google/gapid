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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type profileVerb struct{ GetTimestampsFlags }

func init() {
	verb := &profileVerb{}
	app.AddVerb(&app.Verb{
		Name:      "profile",
		ShortHelp: "Profile a replay to get the time of executing the commands.",
		Action:    verb,
	})
}

func (verb *profileVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}
	capture, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		log.Errf(ctx, err, "Could not find capture file: %v", flags.Arg(0))
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	capturePath, err := client.LoadCapture(ctx, capture)
	if err != nil {
		return log.Err(ctx, err, "Failed to load the capture file")
	}

	device, err := getDevice(ctx, client, capturePath, verb.Gapir)
	if err != nil {
		return err
	}

	var out io.Writer = os.Stdout
	if verb.Out != "" {
		f, err := os.OpenFile(verb.Out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return log.Err(ctx, err, "Failed to open report output file")
		}
		defer f.Close()
		out = f
	}

	boxedRes, err := client.GetTimestamps(ctx, capturePath, device)
	if err != nil {
		return log.Err(ctx, err, "Failed to get the timestamps")
	}
	res := boxedRes.(*service.GetTimestampsResponse)

	reportWriter := csv.NewWriter(out)
	defer reportWriter.Flush()

	header := []string{"BeginCmd", "EndCmd", "Time(ns)"}
	if err = reportWriter.Write(header); err != nil {
		log.Err(ctx, err, "Failed to write header")
	}

	cmdToString := func(cmd *path.Command) string {
		return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cmd.Indices)), "."), "[]")
	}

	if ts := res.GetTimestamps(); ts != nil {
		for _, t := range ts.Timestamps {
			begin := cmdToString(t.Begin)
			end := cmdToString(t.End)
			record := []string{begin, end, fmt.Sprint(t.TimeInNanoseconds)}
			if err := reportWriter.Write(record); err != nil {
				log.Err(ctx, err, "Failed to write record")
			}
		}
	}

	return nil
}
