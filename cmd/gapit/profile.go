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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type profileVerb struct{ GpuProfileFlags }

func init() {
	verb := &profileVerb{GpuProfileFlags{
		DisabledCmds: []flags.U64Slice{},
		DisableAF:    false,
	}}
	app.AddVerb(&app.Verb{
		Name:      "profile",
		ShortHelp: "Profile a replay to get GPU activity and counter data.",
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

	var commands []*path.Command
	if len(verb.DisabledCmds) > 0 {
		for _, cmd := range verb.DisabledCmds {
			commands = append(commands, capturePath.Command(cmd[0], cmd[1:]...))
		}
	}

	req := &service.GpuProfileRequest{
		Capture: capturePath,
		Device:  device,
		Experiments: &service.ProfileExperiments{
			DisabledCommands:            commands,
			DisableAnisotropicFiltering: verb.DisableAF,
		},
	}

	res, err := client.GpuProfile(ctx, req)
	if err != nil {
		return err
	}

	if verb.Json {
		jsonBytes, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return log.Err(ctx, err, "Couldn't marshal trace to JSON")
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
	} else {
		err = proto.MarshalText(os.Stdout, res)
		if err != nil {
			return log.Err(ctx, err, "Couldn't marshal trace to text")
		}
	}
	return nil
}
