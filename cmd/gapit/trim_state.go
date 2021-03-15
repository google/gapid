// Copyright (C) 2021 Google Inc.
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
	"io/ioutil"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

type trimStateVerb struct{ TrimStateFlags }

func init() {
	verb := &trimStateVerb{}
	app.AddVerb(&app.Verb{
		Name:      "trim_state",
		ShortHelp: "Trims a gfx trace's initial state to the resources actually used by the commands",
		Action:    verb,
	})
}

func (verb *trimStateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	c, err := client.TrimCaptureInitialState(ctx, capture)
	if err != nil {
		return log.Errf(ctx, err, "TrimCaptureInitialState(%v)", capture)
	}

	data, err := client.ExportCapture(ctx, c)
	if err != nil {
		return log.Errf(ctx, err, "ExportCapture(%v)", c)
	}

	output := verb.Out
	if output == "" {
		output = "initialStateTrimmed.gfxtrace"
	}
	if err := ioutil.WriteFile(output, data, 0666); err != nil {
		return log.Errf(ctx, err, "Writing file: %v", output)
	}
	return nil
}
