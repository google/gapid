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
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"

	img "github.com/google/gapid/core/image"
)

type screenshotVerb struct{ ScreenshotFlags }

func init() {
	verb := &screenshotVerb{
		ScreenshotFlags{
			At:    flags.U64Slice{},
			Frame: -1,
			Out:   "screenshot.png",
			NoOpt: false,
		},
	}

	app.AddVerb(&app.Verb{
		Name:      "screenshot",
		ShortHelp: "Produce a screenshot at a particular command from a .gfxtrace file",
		Action:    verb,
	})
}

func (verb *screenshotVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return log.Errf(ctx, err, "Finding file: %v", flags.Arg(0))
	}

	client, err := getGapis(ctx, verb.Gapis, verb.Gapir)
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer client.Close()

	capture, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Errf(ctx, err, "LoadCapture(%v)", filepath)
	}

	device, err := getDevice(ctx, client, capture, verb.Gapir)
	if err != nil {
		return err
	}

	var command *path.Command
	if len(verb.At) > 0 {
		command = capture.Command(verb.At[0], verb.At[1:]...)
	} else {
		var err error
		command, err = verb.frameCommand(ctx, capture, client)
		if err != nil {
			return err
		}
	}

	if frame, err := verb.getSingleFrame(ctx, command, device, client); err == nil {
		return verb.writeSingleFrame(flipImg(frame), verb.Out)
	} else {
		return err
	}

}

func (verb *screenshotVerb) writeSingleFrame(frame image.Image, fn string) error {
	out, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, frame)
}

func (verb *screenshotVerb) getSingleFrame(ctx context.Context, cmd *path.Command, device *path.Device, client service.Service) (*image.NRGBA, error) {
	ctx = log.V{"cmd": cmd.Indices}.Bind(ctx)
	settings := &service.RenderSettings{MaxWidth: uint32(0xFFFFFFFF), MaxHeight: uint32(0xFFFFFFFF)}
	if verb.Overdraw {
		settings.DrawMode = service.DrawMode_OVERDRAW
	}
	iip, err := client.GetFramebufferAttachment(ctx,
		&service.ReplaySettings{
			Device: device,
			DisableReplayOptimization: verb.NoOpt,
		},
		cmd, api.FramebufferAttachment_Color0, settings, nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "GetFramebufferAttachment failed")
	}
	iio, err := client.Get(ctx, iip.Path())
	if err != nil {
		return nil, log.Errf(ctx, err, "Get frame image.Info failed")
	}
	ii := iio.(*img.Info)
	dataO, err := client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path())
	if err != nil {
		return nil, log.Errf(ctx, err, "Get frame image data failed")
	}
	w, h, data := int(ii.Width), int(ii.Height), dataO.([]byte)

	ctx = log.V{
		"width":  w,
		"height": h,
		"format": ii.Format,
	}.Bind(ctx)
	if ii.Width == 0 || ii.Height == 0 {
		return nil, log.Err(ctx, nil, "Framebuffer has zero dimensions")
	}
	format := ii.Format
	if verb.Overdraw {
		format = img.Gray_U8_NORM
		rescaleBytes(ctx, data, verb.Max.Overdraw)
	}
	data, err = img.Convert(data, w, h, 1, format, img.RGBA_U8_NORM)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to convert frame to RGBA")
	}
	stride := w * 4
	return &image.NRGBA{
		Rect:   image.Rect(0, 0, w, h),
		Stride: stride,
		Pix:    data,
	}, nil
}

func (verb *screenshotVerb) frameCommand(ctx context.Context, capture *path.Capture, client service.Service) (*path.Command, error) {
	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}

	requestEvents := path.Events{
		Capture:     capture,
		LastInFrame: true,
		Filter:      filter,
	}

	// Get the end-of-frame events.
	eofEvents, err := getEvents(ctx, client, &requestEvents)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get frame events")
	}

	if verb.Frame == -1 {
		verb.Frame = int64(len(eofEvents)) - 1
	}
	fmt.Printf("Frame Command: %v\n", eofEvents[verb.Frame].Command.GetIndices())
	return eofEvents[verb.Frame].Command, nil
}

func rescaleBytes(ctx context.Context, data []byte, max int) {
	if max <= 0 {
		for _, b := range data {
			if int(b) > max {
				max = int(b)
			}
		}
	}
	log.D(ctx, "Max overdraw: %v", max)
	if max < 1 {
		max = 1
	}

	for i, b := range data {
		if int(b) >= max {
			data[i] = 255
		} else {
			data[i] = byte(int(data[i]) * 255 / max)
		}
	}
}
