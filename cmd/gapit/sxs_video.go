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
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

	"github.com/google/gapid/core/event/task"
	img "github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/font"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/f32"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/text/reflow"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type videoFrame struct {
	fbo           *img.Data
	command       *path.Command
	fboIndex      string
	frameIndex    int
	numDrawCalls  int
	renderError   error
	permitNoMatch bool
	rendered      *image.NRGBA
	observed      *image.NRGBA
	difference    *image.NRGBA
	histogramData histogram
	squareError   float64
}

// getFBO fetches the framebuffer observation of the command.
func getFBO(ctx context.Context, client service.Service, a *path.Command) (*img.Data, error) {
	obj, err := client.Get(ctx, a.FramebufferObservation().Path(), nil)
	if err != nil {
		return nil, err
	}
	ii := obj.(*img.Info)
	obj, err = client.Get(ctx, path.NewBlob(ii.Bytes.ID()).Path(), nil)
	if err != nil {
		return nil, err
	}
	data := obj.([]byte)
	return &img.Data{
		Format: ii.Format,
		Width:  ii.Width,
		Height: ii.Height,
		Depth:  ii.Depth,
		Bytes:  data,
	}, nil
}

func (verb *videoVerb) sxsVideoSource(
	ctx context.Context,
	capture *path.Capture,
	client client.Client,
	device *path.Device) (videoFrameWriter, error) {

	filter, err := verb.CommandFilterFlags.commandFilter(ctx, client, capture)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get filter")
	}
	filter.OnlyFramebufferObservations = true

	treePath := capture.CommandTree(filter)

	boxedTree, err := client.Get(ctx, treePath.Path(), nil)
	if err != nil {
		return nil, log.Err(ctx, err, "Failed to load the command tree")
	}

	tree := boxedTree.(*service.CommandTree)

	var allFBOCommands []*path.Command
	traverseCommandTree(ctx, client, tree.Root, func(n *service.CommandTreeNode, prefix string) error {
		if n.Group != "" {
			return nil
		}
		allFBOCommands = append(allFBOCommands, n.Commands.First())
		return nil
	}, "", true)

	// Find maximum frame width / height of all frames, and get all observation
	// command indices.
	videoFrames := []*videoFrame{}
	w, h := 0, 0
	frameIndex, numDrawCalls := 0, 0

	for _, cmd := range allFBOCommands {

		fbo, err := getFBO(ctx, client, cmd)
		if err != nil {
			return nil, err
		}
		if int(fbo.Width) > w {
			w = int(fbo.Width)
		}
		if int(fbo.Height) > h {
			h = int(fbo.Height)
		}

		videoFrames = append(videoFrames, &videoFrame{
			fbo:           fbo,
			fboIndex:      fmt.Sprint(cmd.Indices),
			frameIndex:    frameIndex,
			numDrawCalls:  numDrawCalls,
			command:       cmd,
			permitNoMatch: false,
		})

		frameIndex++
	}

	if verb.Frames.Minimum > len(videoFrames) {
		return nil, log.Errf(ctx, nil, "Captured only %v frames, require %v frames at minimum", len(videoFrames), verb.Frames.Minimum)
	}

	// Get all the observed and rendered frames, and compare them.
	const workers = 32
	execEvents := &task.Events{}
	pool, shutdown := task.Pool(0, workers)
	defer shutdown(ctx)
	executor := task.Batch(pool, execEvents)

	start := time.Now()
	w, h = uniformScale(w, h, verb.Max.Width/2, verb.Max.Height/2)
	for _, v := range videoFrames {
		v := v
		executor(ctx, func(ctx context.Context) error {
			v.observed = &image.NRGBA{
				Pix:    v.fbo.Bytes,
				Stride: int(v.fbo.Width) * 4,
				Rect:   image.Rect(0, 0, int(v.fbo.Width), int(v.fbo.Height)),
			}
			if frame, err := getFrame(ctx, verb.Max.Width, verb.Max.Height, v.command, device, client, verb.NoOpt); err == nil {
				v.rendered = frame
			} else {
				v.renderError = err
			}
			v.observed = flipImg(downsample(v.observed, w, h))
			v.rendered = flipImg(downsample(v.rendered, w, h))
			if v.observed != nil && v.rendered != nil {
				v.difference, v.squareError = getDifference(v.observed, v.rendered, &v.histogramData)
			}
			return nil
		})
	}
	execEvents.Wait(ctx)

	log.D(ctx, "Frames rendered in %v", time.Since(start))
	for _, v := range videoFrames {
		if v.renderError != nil {
			return nil, v.renderError
		}
	}

	// Produce the histogram image
	histogram := getHistogram(videoFrames)
	histogram = resize(histogram, w-2, histogram.Bounds().Dy())

	return func(frames chan<- image.Image) error {

		// Compose and stream out the video frames

		//  p0                 p1
		//    ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━┓
		//    ┃                ┃               ┃
		//    ┃                ┃               ┃
		//    ┃    Observed    ┃    Replay     ┃
		//    ┃                ┃               ┃
		//    ┃                ┃               ┃
		// p2 ┣━━━━━━━━━━━━━━━p3 ━━━━━━━━━━━━━━┫ p4
		//    ┃                ┃               ┃
		//    ┃   Difference   ┃   Details     ┃
		//    ┃             p5 ┣━━━━━━━━━━━━━━━┫
		//    ┃                ┃   Histogram   ┃
		//    ┃                ┣━━━━━━━━━━━━━━━┫ p7
		//    ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━┛
		//                     p6                p8
		var histogramHeight int
		if histogram != nil {
			histogramHeight = histogram.Bounds().Dy()
		}
		p0, p1 := image.Pt(0, 0), image.Pt(w, 0)
		p2, p3, p4 := image.Pt(0, h), image.Pt(w, h), image.Pt(w*2, h)
		p5 := image.Pt(w+2, h+200)
		p6 := image.Pt(w, h*2)
		p7 := image.Pt(w*2, p5.Y+histogramHeight)
		p8 := image.Pt(w*2, h*2)

		white := &image.Uniform{C: color.White}
		black := &image.Uniform{C: color.Black}
		rect := func(min, max image.Point) image.Rectangle {
			return image.Rectangle{Min: min, Max: max}
		}

		b := getBackground(w, h)
		for i, v := range videoFrames {
			// Create side-by-side image.
			sxs := image.NewNRGBA(image.Rect(0, 0, w*2, h*2))

			// Observed
			if o := v.observed; o != nil {
				draw.Draw(sxs, rect(p0, p3), b, image.ZP, draw.Src)
				draw.Draw(sxs, rect(p0, p3), o, image.ZP, draw.Over)
			}

			// Rendered
			if r := v.rendered; r != nil {
				draw.Draw(sxs, rect(p1, p4), b, image.ZP, draw.Src)
				draw.Draw(sxs, rect(p1, p4), r, image.ZP, draw.Over)
			}

			// Difference
			if d := v.difference; d != nil {
				draw.Draw(sxs, rect(p2, p6), d, image.ZP, draw.Src)
			}

			draw.Draw(sxs, rect(p3, p8), black, image.ZP, draw.Src)
			// Histogram
			if h := histogram; h != nil {
				draw.Draw(sxs, rect(p5, p7), histogram, image.ZP, draw.Src)

				// Progress line
				if len(videoFrames) > 1 {
					x := p5.X + (((p7.X - p5.X) * i) / (len(videoFrames) - 1))
					draw.Draw(sxs, image.Rect(x, p5.Y, x+1, p7.Y), white, image.ZP, draw.Src)
				}
			}

			sb := new(bytes.Buffer)
			refw := reflow.New(sb)
			fmt.Fprint(refw, verb.Text)
			fmt.Fprintf(refw, "Dimensions:║%dx%d¶", v.fbo.Width, v.fbo.Height)
			fmt.Fprintf(refw, "Cmd:║%v¶", v.fboIndex)
			fmt.Fprintf(refw, "Frame:║%d¶", v.frameIndex)
			fmt.Fprintf(refw, "Draw calls:║%d¶", v.numDrawCalls)
			fmt.Fprintf(refw, "Difference:║%.4f¶", v.squareError)
			refw.Flush()
			str := sb.String()

			font.DrawString(str, sxs, p3.Add(image.Pt(2, 2)), color.Black)
			font.DrawString(str, sxs, p3, color.White)

			frames <- sxs
		}

		close(frames)

		const threshold = 0.01
		for _, v := range videoFrames {
			if !v.permitNoMatch && v.squareError > threshold {
				return fmt.Errorf("FramebufferObservation did not match replayed framebuffer at %v. Difference: %v%%",
					v.command.Indices, v.squareError*100)
			}
		}
		return nil

	}, nil
}

// diff returns the difference values ranging between [0-0xffff] for each channel.
func diff(x, y color.Color) (r, g, b, a int) {
	r0, g0, b0, a0 := x.RGBA()
	r1, g1, b1, a1 := y.RGBA()
	return sint.Abs(int(r0) - int(r1)), sint.Abs(int(g0) - int(g1)), sint.Abs(int(b0) - int(b1)), sint.Abs(int(a0) - int(a1))
}

var heatGradient = [8][3]int{
	{0x00, 0x00, 0x48},
	{0x00, 0x58, 0xbf},
	{0x00, 0xd8, 0xfe},
	{0x00, 0xed, 0x06},
	{0xb0, 0xeb, 0x00},
	{0xff, 0xb9, 0x19},
	{0xff, 0x00, 0x00},
	{0xff, 0xff, 0xff},
}

// heat returns a head-map RGB value for the value v that ranges between [0-0xffff].
func heat(v int) (r, g, b, a int) {
	c := len(heatGradient) - 1
	i := v * c
	indexA := sint.Min(i>>16, c-1)
	colorA := heatGradient[indexA]
	colorB := heatGradient[indexA+1]
	weightB := i & 0xffff
	weightA := 0x10000 - weightB
	r = (weightA*colorA[0] + weightB*colorB[0]) >> 16
	g = (weightA*colorA[1] + weightB*colorB[1]) >> 16
	b = (weightA*colorA[2] + weightB*colorB[2]) >> 16
	return r, g, b, 0xff
}

const bins = 32

type histogram [bins][4]int

func getHistogram(videoFrames []*videoFrame) *image.NRGBA {
	const exaggeration = 0.1
	w, h := len(videoFrames), bins*4

	pixels := make([]byte, w*h*4)
	out := &image.NRGBA{Pix: pixels, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	for i := range pixels {
		if i%4 == 3 {
			pixels[i] = 255
		}
	}

	// Layout into RGBA32 bitmap.
	f32s := make([]float32, bins*w)
	for c := 0; c < 4; c++ {
		// r, g, b, a
		var peak float32
		for bin := 0; bin < bins; bin++ {
			for frame := 0; frame < w; frame++ {
				v := float32(videoFrames[frame].histogramData[bin][c])
				peak = f32.MaxOf(peak, v)
				f32s[frame+bin*w] = v
			}
		}

		// Normalize into RGBA8 bitmap
		for _, f := range f32s {
			v := byte(255 * math.Pow(float64(f/peak), exaggeration))
			if c == 3 {
				// alpha
				pixels[0] = v
				pixels[1] = v
			} else {
				pixels[c] = v
			}
			pixels = pixels[4:]
		}
	}

	return out
}

func getDifference(a, b *image.NRGBA, hist *histogram) (*image.NRGBA, float64) {
	w := sint.Max(a.Bounds().Dx(), b.Bounds().Dx())
	h := sint.Max(a.Bounds().Dy(), b.Bounds().Dy())

	data := make([]byte, 0, w*h*4)
	sqrErr := float64(0)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dr, dg, db, da := diff(a.At(x, y), b.At(x, y))
			hr, hg, hb, ha := heat(sint.MaxOf(dr, dg, db, da))
			data = append(data, byte(hr), byte(hg), byte(hb), byte(ha))
			hist[(dr*(bins-1))/0xffff][0]++
			hist[(dg*(bins-1))/0xffff][1]++
			hist[(db*(bins-1))/0xffff][2]++
			hist[(da*(bins-1))/0xffff][3]++
			sqrErr += (float64(dr) / 0xffff) * (float64(dr) / 0xffff)
			sqrErr += (float64(dg) / 0xffff) * (float64(dg) / 0xffff)
			sqrErr += (float64(db) / 0xffff) * (float64(db) / 0xffff)
			sqrErr += (float64(da) / 0xffff) * (float64(da) / 0xffff)
		}
	}
	sqrErr = sqrErr / float64(w*h*4)
	return &image.NRGBA{Pix: data, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}, sqrErr
}

func getBackground(w, h int) *image.NRGBA {
	data := make([]byte, 0, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := byte(0xff), byte(0x00), byte(0xff), byte(0xff)
			if (x&16 != 0) != (y&16 != 0) {
				r, g, b = byte(0x00), byte(0xff), byte(0x00)
			}
			data = append(data, r, g, b, a)
		}
	}
	return &image.NRGBA{Pix: data, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
}

func resize(i *image.NRGBA, w, h int) *image.NRGBA {
	if i == nil {
		return nil
	}
	data, err := img.RGBA_U8_NORM.Resize(i.Pix, i.Bounds().Dx(), i.Bounds().Dy(), 1, w, h, 1)
	if err != nil {
		return nil
	}
	return &image.NRGBA{Pix: data, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
}

func uniformScale(srcW, srcH, maxW, maxH int) (dstW, dstH int) {
	// Calculate the minimal scaling factor as integer fraction.
	mul, div := 1, 1
	if mul*srcW > maxW*div {
		// if mul/div > maxW/srcW
		mul, div = maxW, srcW
	}
	if mul*srcH > maxH*div {
		// if mul/div > maxH/srcH
		mul, div = maxH, srcH
	}
	// Calculate the final dimensions.
	// Round up to keep the following numerically stable:
	//  w, h = uniformScale(srcW, srcH, maxW, maxH)
	//  w, h = uniformScale(srcW, srcH, w, h)
	return (srcW*mul + div - 1) / div, (srcH*mul + div - 1) / div
}

func flipImg(i *image.NRGBA) *image.NRGBA {
	if i == nil {
		return nil
	}
	data, stride := i.Pix, i.Stride
	out := make([]byte, len(data))
	if len(out) > 0 {
		for i, c := 0, len(data)/stride; i < c; i++ {
			copy(out[(c-i-1)*stride:(c-i)*stride], data[i*stride:])
		}
	}
	return &image.NRGBA{Pix: out, Stride: stride, Rect: i.Rect}
}

func downsample(src *image.NRGBA, maxW, maxH int) *image.NRGBA {
	if src == nil || (src.Rect.Dx() <= maxW && src.Rect.Dy() <= maxH) {
		return src
	}
	srcData, srcStride, srcW, srcH := src.Pix, src.Stride, src.Rect.Dx(), src.Rect.Dy()
	dstW, dstH := uniformScale(srcW, srcH, maxW, maxH)
	dstData := make([]byte, dstW*dstH*4)
	for srcY, y, dstY := 0, 0, 0; dstY < dstH; srcY, dstY = y, dstY+1 {
		for srcX, x, dstX := 0, 0, 0; dstX < dstW; srcX, dstX = x, dstX+1 {
			r, g, b, a, n := 0, 0, 0, 0, 0
			// We need to loop over srcX/srcY ranges several times, so we keep them in x/y,
			// and we update srcX/srcY to the last x/y only once we are done with the pixel.
			for y = srcY; y*dstH < (dstY+1)*srcH; y++ {
				// while y*yScale < dstY+1
				for x = srcX; x*dstW < (dstX+1)*srcW; x++ {
					// while x*xScale < dstX+1
					srcOffset := y*srcStride + x*4
					r += int(srcData[srcOffset+0])
					g += int(srcData[srcOffset+1])
					b += int(srcData[srcOffset+2])
					a += int(srcData[srcOffset+3])
					n += 1
				}
			}
			dstOffset := (dstX + dstY*dstW) * 4
			dstData[dstOffset+0] = byte(r / n)
			dstData[dstOffset+1] = byte(g / n)
			dstData[dstOffset+2] = byte(b / n)
			dstData[dstOffset+3] = byte(a / n)
		}
	}
	return &image.NRGBA{Pix: dstData, Stride: dstW * 4, Rect: image.Rect(0, 0, dstW, dstH)}
}
