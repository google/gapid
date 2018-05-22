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
	"flag"
	"fmt"
	"math"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type infoVerb struct{ StatsFlags }

func init() {
	verb := &infoVerb{}
	verb.Frames.Count = -1
	app.AddVerb(&app.Verb{
		Name:      "stats",
		ShortHelp: "Prints information about a capture file",
		Action:    verb,
	})
}

func loadCapture(ctx context.Context, flags flag.FlagSet, gapisFlags GapisFlags) (client.Client, *path.Capture, error) {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil, nil, nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return nil, nil, log.Errf(ctx, err, "Finding file: %v", flags.Arg(0))
	}

	client, err := getGapis(ctx, gapisFlags, GapirFlags{})
	if err != nil {
		return nil, nil, log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}

	capture, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return nil, nil, log.Errf(ctx, err, "LoadCapture(%v)", filepath)
	}

	return client, capture, nil
}

func (verb *infoVerb) getEventsInRange(ctx context.Context, client service.Service, capture *path.Capture) ([]*service.Event, error) {
	events, err := getEvents(ctx, client, &path.Events{
		Capture:                 capture,
		AllCommands:             true,
		DrawCalls:               true,
		FirstInFrame:            true,
		FramebufferObservations: true,
	})
	if err != nil {
		return nil, err
	}

	if verb.Frames.Start == 0 && verb.Frames.Count == -1 {
		return events, err
	}

	fifIndices := []uint64{}
	for _, e := range events {
		if e.Kind == service.EventKind_FirstInFrame {
			fifIndices = append(fifIndices, e.Command.Indices[0])
		}
	}

	if verb.Frames.Start > len(fifIndices) {
		return nil, log.Errf(ctx, nil, "Captured only %v frames, less than start frame %v", len(fifIndices), verb.Frames.Start)
	}

	startIndex := fifIndices[verb.Frames.Start]
	endIndex := uint64(math.MaxUint64)
	if verb.Frames.Count >= 0 &&
		verb.Frames.Start+verb.Frames.Count < len(fifIndices) {

		endIndex = fifIndices[verb.Frames.Start+verb.Frames.Count]
	}

	begin, end := len(events), len(events)
	for i, e := range events {
		if i < begin && e.Command.Indices[0] >= startIndex {
			begin = i
		}
		if i < end && e.Command.Indices[0] >= endIndex {
			end = i
		}
	}
	return events[begin:end], nil
}

func (verb *infoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, capture, err := loadCapture(ctx, flags, verb.Gapis)
	if err != nil {
		return err
	}
	defer client.Close()

	events, err := verb.GetEventsInRange(ctx, client, capture)

	if err != nil {
		return log.Err(ctx, err, "Couldn't get events")
	}

	counts := map[service.EventKind]int{}
	for _, e := range events {
		counts[e.Kind] = counts[e.Kind] + 1
	}

	fmt.Println("Commands: ", counts[service.EventKind_AllCommands])
	fmt.Println("Frames:   ", counts[service.EventKind_FirstInFrame])
	fmt.Println("Draws:    ", counts[service.EventKind_DrawCall])
	fmt.Println("FBO:      ", counts[service.EventKind_FramebufferObservation])
	return err
}
