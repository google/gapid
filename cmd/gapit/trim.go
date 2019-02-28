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
	"flag"
	"io/ioutil"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type trimVerb struct{ TrimFlags }

func init() {
	verb := &trimVerb{}
	verb.Frames.Count = allTheWay

	app.AddVerb(&app.Verb{
		Name:      "trim",
		ShortHelp: "(WIP) Trims a gfx trace to the dependencies of the requested frames",
		Action:    verb,
	})
}

func (verb *trimVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	eofEvents, err := verb.eofEvents(ctx, capture, client)
	if err != nil {
		return err
	}

	dceRequest := verb.getDCERequest(eofEvents, capture)
	if len(dceRequest) > 0 {
		capture, err = client.DCECapture(ctx, capture, dceRequest)
		if err != nil {
			return log.Errf(ctx, err, "DCECapture(%v, %v)", capture, dceRequest)
		}
	}

	data, err := client.ExportCapture(ctx, capture)
	if err != nil {
		return log.Errf(ctx, err, "ExportCapture(%v)", capture)
	}

	output := verb.Out
	if output == "" {
		output = "trimmed.gfxtrace"
	}
	if err := ioutil.WriteFile(output, data, 0666); err != nil {
		return log.Errf(ctx, err, "Writing file: %v", output)
	}
	return nil
}

func (verb *trimVerb) eofEvents(ctx context.Context, capture *path.Capture, client service.Service) ([]*service.Event, error) {
	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}
	requestEvents := path.Events{
		Capture:     capture,
		LastInFrame: true,
		Filter:      filter,
	}

	if verb.Commands {
		requestEvents.LastInFrame = false
		requestEvents.AllCommands = true
	}

	// Get the end-of-frame events.
	eofEvents, err := getEvents(ctx, client, &requestEvents)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get frame events")
	}

	lastFrame := verb.Frames.Start
	if verb.Frames.Count > 0 {
		lastFrame += verb.Frames.Count - 1
	}
	if lastFrame >= len(eofEvents) {
		return nil, log.Errf(ctx, nil, "Requested frame %d, but capture only contains %d frames", lastFrame, len(eofEvents))
	}

	return eofEvents, nil
}

func (verb *trimVerb) getDCERequest(eofEvents []*service.Event, p *path.Capture) []*path.Command {
	frameCount := verb.Frames.Count
	if frameCount < 0 {
		frameCount = len(eofEvents) - verb.Frames.Start
	}
	dceRequest := make([]*path.Command, 0, frameCount+len(verb.ExtraCommands))
	for i := 0; i < frameCount; i++ {
		indices := eofEvents[verb.Frames.Start+i].Command.Indices
		newIndices := make([]uint64, len(indices))
		copy(newIndices, indices)
		cmd := &path.Command{
			Capture: p,
			Indices: newIndices,
		}
		dceRequest = append(dceRequest, cmd)
	}
	for _, id := range verb.ExtraCommands {
		cmd := &path.Command{
			Capture: p,
			Indices: []uint64{id},
		}
		dceRequest = append(dceRequest, cmd)
	}
	return dceRequest
}
