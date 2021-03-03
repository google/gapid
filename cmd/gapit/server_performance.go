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
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	gapidPath "github.com/google/gapid/gapis/service/path"
)

type serverPerformanceVerb struct{ ServerPerformanceFlags }

func init() {
	verb := &serverPerformanceVerb{}
	app.AddVerb(&app.Verb{
		Name:      "server_performance",
		ShortHelp: "Test the performance of GAPIS replay generation.",
		Action:    verb,
	})
}

func (verb *serverPerformanceVerb) Run(ctx context.Context, flags flag.FlagSet) error {

	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capturePath, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	var device *gapidPath.Device
	if !verb.OriginalDevice {
		device, err = getDevice(ctx, client, capturePath, verb.Gapir)
		if err != nil {
			return err
		}
	}

	opts := &service.ExportReplayOptions{
		FramebufferAttachments: make([]*gapidPath.FramebufferAttachment, 0),
		GetTimestampsRequest:   nil,
		DisplayToSurface:       false,
	}

	start := time.Now()
	if err := client.ExportReplay(ctx, capturePath, device, "replay_export", opts); err != nil {
		return log.Err(ctx, err, "Failed to export replay")
	}
	elapsed := time.Since(start)

	startRepeat := time.Now()
	if err := client.ExportReplay(ctx, capturePath, device, "replay_export", opts); err != nil {
		return log.Err(ctx, err, "Failed to export replay")
	}
	elapsedRepeat := time.Since(startRepeat)

	log.W(ctx, "\n\n\n\n")
	log.W(ctx, "--------------------------------")
	log.W(ctx, "RESULTS:")
	log.W(ctx, "First time replay generation:\t%s\t", elapsed)
	log.W(ctx, "Repeat replay generation:\t%s\t", elapsedRepeat)
	log.W(ctx, "--------------------------------\n\n\n\n")

	return nil
}
