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
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type dumpFBOVerb struct{ DumpFBOFlags }

func init() {
	verb := &dumpFBOVerb{}
	app.AddVerb(&app.Verb{
		Name:      "dump_fbo",
		ShortHelp: "Extract all framebuffer observations from a trace.",
		Action:    verb,
	})
}

func (verb *dumpFBOVerb) writePNGFrame(filename string, frame image.Image) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, frame)
}

func (verb *dumpFBOVerb) pngFrameSink(ctx context.Context, fileprefix string, vidFun videoFrameWriter) error {
	fileprefix = file.Abs(fileprefix).ChangeExt("").System()

	ch := make(chan image.Image, 64)
	var srcErr error
	crash.Go(func() {
		defer close(ch)
		srcErr = vidFun(ch)
	})

	index := 0
	for frame := range ch {
		fn := fmt.Sprintf("%s-%d.png", fileprefix, index)
		if err := verb.writePNGFrame(fn, frame); err != nil {
			return log.Errf(ctx, err, "Error writing %s", fn)
		}
		index++
	}
	return srcErr
}

func (verb *dumpFBOVerb) frameSource(ctx context.Context, client client.Client, capture *path.Capture) (videoFrameWriter, error) {

	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}

	return func(ch chan<- image.Image) error {
		// Get the draw call and end-of-frame events.
		events, err := getEvents(ctx, client, &path.Events{
			Capture:                 capture,
			LastInFrame:             true,
			FramebufferObservations: true,
			Filter:                  filter,
		})
		if err != nil {
			return log.Err(ctx, err, "Couldn't get events")
		}

		var lastFrameEvent *path.Command
		for _, e := range events {
			switch e.Kind {
			case service.EventKind_FramebufferObservation:
				// Find FBO for all presents.
				if lastFrameEvent == nil {
					continue
				}
				lastFrameEvent = nil

				fbo, err := getFBO(ctx, client, e.Command)
				if err != nil {
					return err
				}

				ch <- flipImg(&image.NRGBA{
					Pix:    fbo.Bytes,
					Stride: int(fbo.Width) * 4,
					Rect:   image.Rect(0, 0, int(fbo.Width), int(fbo.Height)),
				})
			case service.EventKind_LastInFrame:
				lastFrameEvent = e.Command
			}
		}

		return nil
	}, nil
}

func (verb *dumpFBOVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	src, err := verb.frameSource(ctx, client, capture)
	if err != nil {
		return err
	}
	return verb.pngFrameSink(ctx, verb.Out, src)
}
