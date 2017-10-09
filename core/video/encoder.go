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

package video

import (
	"context"
	"fmt"
	"image"
	"io"
	"os/exec"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
)

// Settings for encoding a video with Encode.
type Settings struct {
	FPS      int // Frames per second. Default: 30
	DataRate int // Target bits-per-second. Default: 5000000
}

var encoder string

func init() {
	encoder, _ = exec.LookPath("avconv")
	if encoder == "" {
		encoder, _ = exec.LookPath("ffmpeg")
	}
}

// Encode will encode the frames written to the returned chan to a video that
// can be read from the Reader.
func Encode(ctx context.Context, settings Settings) (chan<- image.Image, io.Reader, error) {
	if encoder == "" {
		return nil, nil, fmt.Errorf("neither avconv or ffmpeg was found")
	}

	in := make(chan image.Image, 64)
	out, mpg := io.Pipe()

	// Set defaults
	if settings.DataRate == 0 {
		settings.DataRate = 5000000
	}
	if settings.FPS == 0 {
		settings.FPS = 30
	}

	crash.Go(func() {
		// Get the first frame so we know what we're dealing with.
		frame, ok := <-in
		if !ok {
			mpg.Close()
			return // Closed before we got the first frame
		}

		var pixfmt string
		var data func(image.Image) []byte

		switch frame.(type) {
		case *image.NRGBA:
			pixfmt = "rgba"
			data = func(i image.Image) []byte { return (i.(*image.NRGBA)).Pix }
		default:
			mpg.CloseWithError(fmt.Errorf("Unsupported frame type %T", frame))
			return
		}

		debugWriter := log.From(ctx).Writer(log.Debug)
		defer debugWriter.Close()

		stdin, pixels := io.Pipe()
		defer pixels.Close() // Stops the encoder

		crash.Go(func() {
			err := shell.Command(encoder,
				"-v", "verbose",
				"-r", fmt.Sprint(settings.FPS),
				"-pix_fmt", pixfmt,
				"-f", "rawvideo",
				"-s", fmt.Sprintf("%dx%d", frame.Bounds().Dx(), frame.Bounds().Dy()),
				"-i", "pipe:0", // stdin
				"-b:v", fmt.Sprint(settings.DataRate),
				"-f", "mp4", // output should be a mp4
				"-movflags", "frag_keyframe+empty_moov", // fragmented mp4, required for streaming.
				"pipe:1", // stdout
			).Read(stdin).Capture(mpg, debugWriter).Run(ctx)

			if err != nil {
				log.E(ctx, "%v returned error: %v", encoder, err)
			}
			mpg.CloseWithError(err)
		})

		i := 0
		log.D(ctx, "Encoding frame 0")
		pixels.Write(data(frame))
		i++
		for frame := range in {
			log.D(ctx, "Encoding frame %d", i)
			pixels.Write(data(frame))
			i++
		}

		log.I(ctx, "Done")
	})
	return in, out, nil
}
