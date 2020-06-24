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
	"strconv"
	"strings"
	"sync"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/flags"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"

	img "github.com/google/gapid/core/image"
)

type screenshotVerb struct{ ScreenshotFlags }

func init() {
	verb := &screenshotVerb{
		ScreenshotFlags{
			At:    []flags.U64Slice{},
			Frame: []int{},
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

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, verb.Gapir, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	device, err := getDevice(ctx, client, capture, verb.Gapir)
	if err != nil {
		return err
	}

	var commands []*path.Command
	if len(verb.At) > 0 {
		for _, at := range verb.At {
			commands = append(commands, capture.Command(at[0], at[1:]...))
		}
	} else {
		commands, err = verb.frameCommands(ctx, capture, client)
		if err != nil {
			return err
		}
	}

	// Submit requests in parallel, so that gapis will batch them.
	var wg sync.WaitGroup
	c := make(chan error)
	multi := len(commands) > 1
	for idx, command := range commands {
		wg.Add(1)
		go func(idx int, command *path.Command) {
			defer wg.Done()

			frame, err := verb.getSingleFrame(ctx, command, device, client)
			if err == nil {
				err = verb.writeSingleFrame(flipImg(frame), formatOut(verb.Out, idx, multi))
			}
			c <- err
		}(idx, command)
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	for err := range c {
		if err != nil {
			return err
		}
	}
	return nil
}

func formatOut(out string, idx int, multi bool) string {
	if strings.Contains(out, "%d") {
		return fmt.Sprintf(out, idx)
	} else if !multi {
		return out
	} else if p := strings.LastIndex(out, "."); p != -1 {
		return fmt.Sprintf("%s_%d%s", out[:p], idx, out[p:])
	}
	return fmt.Sprintf("%s_%d", out, idx)

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
	settings := &path.RenderSettings{
		MaxWidth:                  uint32(0xFFFFFFFF),
		MaxHeight:                 uint32(0xFFFFFFFF),
		DisableReplayOptimization: verb.NoOpt,
		DisplayToSurface:          verb.DisplayToSurface,
	}
	if verb.Overdraw {
		settings.DrawMode = path.DrawMode_OVERDRAW
	}

	attachment, err := verb.getAttachment(ctx)
	if err != nil {
		return nil, log.Errf(ctx, err, "Get color attachment failed")
	}
	fbPath := &path.FramebufferAttachment{
		After:          cmd,
		Index:          attachment,
		RenderSettings: settings,
		Hints:          nil,
	}
	iip, err := client.Get(ctx, fbPath.Path(), &path.ResolveConfig{ReplayDevice: device})
	if err != nil {
		return nil, log.Errf(ctx, err, "GetFramebufferAttachment failed")
	}
	iio, err := client.Get(ctx, iip.(*service.FramebufferAttachment).GetImageInfo().Path(), nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "Get frame image.Info failed")
	}
	ii := iio.(*img.Info)
	dataO, err := client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path(), nil)
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

func (verb *screenshotVerb) frameCommands(ctx context.Context, capture *path.Capture, client service.Service) ([]*path.Command, error) {
	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}

	requestEvents := path.Events{
		Capture:     capture,
		LastInFrame: true,
		DrawCalls:   verb.Draws,
		Filter:      filter,
	}

	// Get the end-of-frame and possibly draw call events.
	events, err := getEvents(ctx, client, &requestEvents)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get frame events")
	}

	// Compute an index of frame to event idx.
	frameIdx := map[int]int{}
	lastFrame := 0
	for i, e := range events {
		if e.Kind == service.EventKind_LastInFrame {
			lastFrame++
			frameIdx[lastFrame] = i
		}
	}

	if len(verb.Frame) == 0 {
		verb.Frame = []int{lastFrame}
	}

	var commands []*path.Command
	for _, frame := range verb.Frame {
		last, ok := frameIdx[frame]
		if !ok {
			return nil, fmt.Errorf("Invalid frame number %d (last frame is %d)", frame, lastFrame)
		}

		first := last
		if verb.Draws {
			if frame == 1 {
				first = 0
			} else {
				first = frameIdx[frame-1]
			}
		}
		for idx := first; idx <= last; idx++ {
			commands = append(commands, events[idx].Command)
		}
	}
	return commands, nil
}

// rescaleBytes scales the values in `data` from [0, `max`] to [0, 255].  If
// `max <= 0`, then the maximum value found in data is used as `max` instead.
func rescaleBytes(ctx context.Context, data []byte, max int) {
	if max <= 0 {
		for _, b := range data {
			if int(b) > max {
				max = int(b)
			}
		}
	}
	log.I(ctx, "Max overdraw: %v", max)
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

func (verb *screenshotVerb) getAttachment(ctx context.Context) (uint32, error) {
	if verb.Attachment == "" {
		return 0, nil
	}

	// TODO: Add-back ability to type "depth" to get the depth attachment
	i, err := strconv.ParseUint(verb.Attachment, 10, 32)
	if err != nil {
		return 0, log.Errf(ctx, nil, "Invalid attachment %v", verb.Attachment)
	}
	return uint32(i), nil
}
