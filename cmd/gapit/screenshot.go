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
			At: flags.U64Slice{},
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

	command := capture.Command(verb.At[0], verb.At[1:]...)

	if frame, err := getSingleFrame(ctx, command, device, client); err == nil {
		return verb.writeSingleFrame(flipImg(frame), "screenshot.png")
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

func getSingleFrame(ctx context.Context, cmd *path.Command, device *path.Device, client service.Service) (*image.NRGBA, error) {
	ctx = log.V{"cmd": cmd.Indices}.Bind(ctx)
	settings := &service.RenderSettings{MaxWidth: uint32(0xFFFFFFFF), MaxHeight: uint32(0xFFFFFFFF)}
	iip, err := client.GetFramebufferAttachment(ctx,
		&service.ReplaySettings{
			Device: device,
			DisableReplayOptimization: false,
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
	data, err = img.Convert(data, w, h, 1, ii.Format, img.RGBA_U8_NORM)
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
