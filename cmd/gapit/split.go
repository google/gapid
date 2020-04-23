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

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

type splitVerb struct{ SplitFlags }

func init() {
	verb := &splitVerb{}

	app.AddVerb(&app.Verb{
		Name:      "split",
		ShortHelp: "Splits traces by carving out subranges into new traces",
		Action:    verb,
	})
}

func (verb *splitVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	newCapture, err := client.SplitCapture(ctx, capture.CommandRange(verb.From, verb.To))
	if err != nil {
		return err
	}
	log.I(ctx, "Created new capture; id: %s", newCapture.ID)

	output := verb.Out
	if output == "" {
		output = "split.gfxtrace"
	}
	return client.SaveCapture(ctx, newCapture, output)
}
