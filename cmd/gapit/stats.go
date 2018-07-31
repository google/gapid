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
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/sint"
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
		FirstInFrame:            true,
		LastInFrame:             true,
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

	if verb.Frames.Start < 0 {
		return nil, log.Errf(ctx, nil, "Negative start frame %v is invalid", verb.Frames.Start)
	}
	if verb.Frames.Start >= len(fifIndices) {
		return nil, log.Errf(ctx, nil, "Captured only %v frames, not greater than start frame %v", len(fifIndices), verb.Frames.Start)
	}

	startIndex := fifIndices[verb.Frames.Start]
	endIndex := uint64(math.MaxUint64)
	if verb.Frames.Count >= 0 &&
		verb.Frames.Start+verb.Frames.Count < len(fifIndices) {

		endIndex = fifIndices[verb.Frames.Start+verb.Frames.Count]
	}

	begin := sort.Search(len(events), func(i int) bool {
		return events[i].Command.Indices[0] >= startIndex
	})
	end := sort.Search(len(events), func(i int) bool {
		return events[i].Command.Indices[0] >= endIndex
	})
	return events[begin:end], nil
}

func (verb *infoVerb) drawCallStats(ctx context.Context, client client.Client, c *path.Capture) (int, sint.HistogramStats, error) {
	boxedVal, err := client.Get(ctx, (&path.Stats{
		Capture:  c,
		DrawCall: true,
	}).Path(), nil)
	if err != nil {
		return 0, sint.HistogramStats{}, err
	}
	data := boxedVal.(*service.Stats).DrawCalls

	if verb.Frames.Start < len(data) {
		data = data[verb.Frames.Start:]
	} else {
		data = []uint64{}
	}
	if verb.Frames.Count >= 0 && verb.Frames.Count < len(data) {
		data = data[:verb.Frames.Count]
	}

	hist := make(sint.Histogram, len(data))
	totalDraws := 0
	for i, dat := range data {
		totalDraws += int(dat)
		hist[i] = int(dat)
	}
	return totalDraws, hist.Stats(), nil
}

func (verb *infoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, capture, err := loadCapture(ctx, flags, verb.Gapis)
	if err != nil {
		return err
	}
	defer client.Close()

	events, err := verb.getEventsInRange(ctx, client, capture)

	if err != nil {
		return log.Err(ctx, err, "Couldn't get events")
	}

	counts := map[service.EventKind]int{}
	cmdsPerFrame, frameIdx := sint.Histogram{}, 0
	for i, e := range events {
		counts[e.Kind]++
		switch e.Kind {
		case service.EventKind_AllCommands:
			cmdsPerFrame.Add(frameIdx, 1)
		case service.EventKind_FirstInFrame:
			if i > 0 {
				frameIdx++
			}
		}
	}
	callStats := cmdsPerFrame.Stats()
	totalDraws, drawStats, err := verb.drawCallStats(ctx, client, capture)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 4, 4, 0, ' ', 0)
	fmt.Fprintf(w, "Commands: \t%v\n", counts[service.EventKind_AllCommands])
	fmt.Fprintf(w, "Frames: \t%v\n", counts[service.EventKind_FirstInFrame])
	fmt.Fprintf(w, "Draws: \t%v\n", totalDraws)
	fmt.Fprintf(w, "FBO: \t%v\n", counts[service.EventKind_FramebufferObservation])

	fmt.Fprintf(w, "Avg commands per frame: \t%.2f\n", callStats.Average)
	fmt.Fprintf(w, "Stddev commands per frame: \t%.2f\n", callStats.Stddev)
	fmt.Fprintf(w, "Median commands per frame: \t%v\n", callStats.Median)

	fmt.Fprintf(w, "Avg draw calls per frame: \t%.2f\n", drawStats.Average)
	fmt.Fprintf(w, "Stddev draw calls per frame: \t%.2f\n", drawStats.Stddev)
	fmt.Fprintf(w, "Median draw calls per frame: \t%v\n", drawStats.Median)
	w.Flush()

	return nil
}
