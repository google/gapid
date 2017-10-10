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
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/image/font"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text/reflow"
	"github.com/google/gapid/core/video"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"

	img "github.com/google/gapid/core/image"
)

const allTheWay = -1

type videoVerb struct{ VideoFlags }

func init() {
	verb := &videoVerb{}
	verb.Gapir.Device = "host"
	// The maximum width and height need to match the values in spy.cpp
	// in order to properly compare observed and rendered framebuffers.
	verb.Max.Width = 1920
	verb.Max.Height = 1280
	verb.FPS = 5
	verb.Frames.Count = allTheWay
	app.AddVerb(&app.Verb{
		Name:      "video",
		ShortHelp: "Produce a video or sequence of frames from a .gfxtrace file",
		Action:    verb,
	})
}

type videoFrameWriter func(chan<- image.Image) error
type videoSource func(ctx context.Context, capture *path.Capture, client service.Service, device *path.Device) (videoFrameWriter, error)
type videoSink func(ctx context.Context, filepath string, vidFun videoFrameWriter) error

func (verb *videoVerb) regularVideoSource(
	ctx context.Context,
	capture *path.Capture,
	client service.Service,
	device *path.Device) (videoFrameWriter, error) {

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

	if verb.Frames.Start < len(eofEvents) {
		eofEvents = eofEvents[verb.Frames.Start:]
	}

	if verb.Frames.Count != allTheWay && verb.Frames.Count < len(eofEvents) {
		eofEvents = eofEvents[:verb.Frames.Count]
	}

	frameCount := len(eofEvents)

	log.I(ctx, "Frames: %d", frameCount)

	// Get all the rendered frames
	const workers = 32
	events := &task.Events{}
	pool, shutdown := task.Pool(0, workers)
	defer shutdown(ctx)
	executor := task.Batch(pool, events)
	shouldResize := verb.Type != IndividualFrames
	rendered := make([]*image.NRGBA, frameCount)
	errors := make([]error, frameCount)

	var errorCount uint32
	for i, e := range eofEvents {
		i, e := i, e
		executor(ctx, func(ctx context.Context) error {
			if frame, err := getFrame(ctx, verb.Max.Width, verb.Max.Height, e.Command, device, client); err == nil {
				rendered[i] = flipImg(frame)
			} else {
				errors[i] = err
				atomic.AddUint32(&errorCount, 1)
			}
			return nil
		})
	}
	events.Wait(ctx)

	if errorCount > 0 {
		log.W(ctx, "%d/%d frames errored", errorCount, len(eofEvents))
	}

	// Get the max width and height
	width, height := 0, 0
	for _, frame := range rendered {
		if frame != nil {
			width = sint.Max(width, frame.Bounds().Dx())
			height = sint.Max(height, frame.Bounds().Dy())
		}
	}

	// Video dimensions must be divisible by two.
	if (width & 1) != 0 {
		width++
	}
	if (height & 1) != 0 {
		height++
	}

	log.I(ctx, "Max dimensions: (%d, %d)", width, height)

	return func(frames chan<- image.Image) error {
		for i, frame := range rendered {
			if err := errors[i]; err != nil {
				log.E(ctx, "Error getting frame at %v: %v", eofEvents[i].Command, err)
				continue
			}

			if shouldResize && (frame.Bounds().Dx() != width || frame.Bounds().Dy() != height) {
				src, rect := frame, image.Rect(0, 0, width, height)
				frame = image.NewNRGBA(rect)
				draw.Draw(frame, rect, src, image.ZP, draw.Src)
			}

			sb := new(bytes.Buffer)
			refw := reflow.New(sb)
			fmt.Fprint(refw, verb.Text)
			fmt.Fprintf(refw, "Frame: %d, cmd: %v", i, eofEvents[i].Command.Indices)
			refw.Flush()
			str := sb.String()
			font.DrawString(str, frame, image.Pt(4, 4), color.Black)
			font.DrawString(str, frame, image.Pt(2, 2), color.White)

			frames <- frame
		}
		close(frames)
		return nil
	}, nil
}

func (verb *videoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
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

	fboEvents, err := getEvents(ctx, client, &path.Events{
		Capture:                 capture,
		FramebufferObservations: true,
	})
	if err != nil {
		return log.Err(ctx, err, "Couldn't get framebuffer observation events")
	}

	var vidSrc videoSource
	var vidFun videoFrameWriter
	var vidOut videoSink

	switch verb.Type {
	case IndividualFrames:
		vidSrc = verb.regularVideoSource
		vidOut = verb.writeFrames
	case RegularVideo:
		vidSrc = verb.regularVideoSource
		vidOut = verb.encodeVideo
	case SxsVideo:
		if len(fboEvents) == 0 {
			return fmt.Errorf("Capture does not contain framebuffer observations")
		}
		vidSrc = verb.sxsVideoSource
		vidOut = verb.encodeVideo
	case AutoVideo:
		if len(fboEvents) > 0 {
			vidSrc = verb.sxsVideoSource
		} else {
			vidSrc = verb.regularVideoSource
		}
		vidOut = verb.encodeVideo
	}

	if vidFun, err = vidSrc(ctx, capture, client, device); err != nil {
		return err
	}

	return vidOut(ctx, filepath, vidFun)
}

func (verb *videoVerb) writeFrames(ctx context.Context, filepath string, vidFun videoFrameWriter) error {
	outFile := verb.Out
	if outFile == "" {
		outFile = file.Abs(filepath).ChangeExt("").System()
	} else {
		pth := file.Abs(outFile)
		if pth.Ext() != "" && !strings.EqualFold(pth.Ext(), ".png") {
			return fmt.Errorf("Only .png output supported")
		}
		outFile = pth.ChangeExt("").System()
	}

	ch := make(chan image.Image, 64)

	crash.Go(func() { vidFun(ch) })

	index := verb.Frames.Start
	var err error
	for frame := range ch {
		fn := fmt.Sprintf("%s-%03d.png", outFile, index)
		if err = verb.writeSingleFrame(frame, fn); err != nil {
			log.E(ctx, "Error writing %s: %s", fn, err.Error())
		}
		index++
	}
	return err
}

func (verb *videoVerb) writeSingleFrame(frame image.Image, fn string) error {
	out, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, frame)
}

func (verb *videoVerb) encodeVideo(ctx context.Context, filepath string, vidFun videoFrameWriter) error {
	// Start an encoder
	frames, video, err := video.Encode(ctx, video.Settings{FPS: verb.FPS})
	if err != nil {
		return err
	}

	vidDone := make(chan error, 1) // buffered so the goroutine always finishes
	crash.Go(func() { vidDone <- vidFun(frames) })

	out := verb.Out
	if out == "" {
		out = file.Abs(filepath).ChangeExt(".mp4").System()
	}
	mpg, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("Error creating video file: %v", err)
	}
	defer mpg.Close()
	if _, err = io.Copy(mpg, video); err != nil {
		return fmt.Errorf("Error writing file: %v", err)
	}

	if vidErr := <-vidDone; vidErr != nil {
		return fmt.Errorf("Error encoding frames: %v", vidErr)
	}

	return nil
}

func getFrame(ctx context.Context, maxWidth, maxHeight int, cmd *path.Command, device *path.Device, client service.Service) (*image.NRGBA, error) {
	ctx = log.V{"cmd": cmd.Indices}.Bind(ctx)
	settings := &service.RenderSettings{MaxWidth: uint32(maxWidth), MaxHeight: uint32(maxHeight)}
	iip, err := client.GetFramebufferAttachment(ctx, device, cmd, api.FramebufferAttachment_Color0, settings, nil)
	if err != nil {
		return nil, log.Errf(ctx, err, "GetFramebufferAttachment failed at %v", cmd)
	}
	iio, err := client.Get(ctx, iip.Path())
	if err != nil {
		return nil, log.Errf(ctx, err, "Get frame image.Info failed at %v", cmd)
	}
	ii := iio.(*img.Info)
	dataO, err := client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path())
	if err != nil {
		return nil, log.Errf(ctx, err, "Get frame image data failed at %v", cmd)
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
