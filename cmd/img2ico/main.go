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

// img2ico is a utility program that converts images to Windows ICO icons.

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"image"
	"image/color"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"

	// Import to register image formats.
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
)

var (
	out = flag.String("out", "-", "Output file, '-' for stdout")
)

func main() {
	app.ShortHelp = "img2ico creates Windows ICO files"
	app.Name = "img2ico"
	app.Run(run)
}

type header struct {
	reserved uint16
	imgType  uint16
	imgCount uint16
}

type entry struct {
	width       uint8
	height      uint8
	paletteSize uint8
	reserved    uint8
	numPlanes   uint16
	bpp         uint16
	imgSize     uint32
	offset      uint32
}

type bmp struct {
	headerSize  uint32
	width       uint32
	height      uint32
	planes      uint16
	bpp         uint16
	compression uint32
	imageSize   uint32
	wResolution uint32
	hResolution uint32
	paletteSize uint32
	important   uint32
}

type pixel struct {
	b, g, r, a uint8
}

func run(ctx context.Context) (err error) {
	if flag.NArg() < 1 {
		app.Usage(ctx, "At least one input image is required")
		return nil
	}

	images := make([]image.Image, 0, flag.NArg())
	pngs := [][]byte{}
	for _, file := range flag.Args() {
		in, err := os.Open(file)
		if err != nil {
			return log.Errf(ctx, err, "Failed to open %s", file)
		}

		img, _, err := image.Decode(in)
		in.Close()
		if err != nil {
			return log.Errf(ctx, err, "Failed to decode image %s", file)
		}

		size := img.Bounds().Size()
		if size.X > 256 || size.Y > 256 {
			return log.Errf(ctx, nil, "Image %s too big. Cannot be larger than 256x256", file)
		}

		images = append(images, img)

		// Use PNG format for large icons.
		if size.X == 256 || size.Y == 256 {
			var pngOut bytes.Buffer
			png.Encode(&pngOut, img)
			pngs = append(pngs, pngOut.Bytes())
		}
	}

	o := os.Stdout
	if *out != "-" {
		o, err = os.Create(*out)
		if err != nil {
			return log.Errf(ctx, err, "Failed to create %s", *out)
		}
		defer o.Close()
	}

	binary.Write(o, binary.LittleEndian, &header{
		imgType:  1, // ICO
		imgCount: uint16(len(images)),
	})

	offset := uint32(6 + 16*len(images))
	pngIdx := 0
	for _, img := range images {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		size := uint32(40 + w*h*4 + h*((w+31) & ^31)/8)

		if w == 256 {
			w = 0
		}
		if h == 256 {
			h = 0
		}

		if w == 0 || h == 0 {
			size = uint32(len(pngs[pngIdx]))
			pngIdx++
		}

		binary.Write(o, binary.LittleEndian, &entry{
			width:     uint8(w),
			height:    uint8(h),
			numPlanes: 1,
			bpp:       32,
			imgSize:   size,
			offset:    offset,
		})
		offset += size
	}

	model := color.NRGBAModel
	pngIdx = 0
	for _, img := range images {
		w, h := img.Bounds().Dx(), img.Bounds().Dy()

		if w == 256 || h == 256 {
			o.Write(pngs[pngIdx])
			pngIdx++
			continue
		}

		binary.Write(o, binary.LittleEndian, &bmp{
			headerSize: 40,
			width:      uint32(w),
			height:     uint32(2 * h),
			planes:     1,
			bpp:        32,
			imageSize:  uint32(w * h * 4),
		})

		matte := make([]byte, h*((w+31) & ^31)/8)
		mattePos := uint(0)
		for y := h - 1; y >= 0; y-- {
			for x := 0; x < w; x++ {
				p := model.Convert(img.At(x, y)).(color.NRGBA)
				binary.Write(o, binary.LittleEndian, &pixel{
					r: p.R,
					g: p.G,
					b: p.B,
					a: p.A,
				})
				if p.A == 0 {
					matte[mattePos/8] |= byte(0x80 >> (mattePos % 8))
				}
				mattePos++
			}
			// Align to 4 bytes.
			mattePos = (mattePos + 31) & ^uint(31)
		}

		o.Write(matte)
	}

	return nil
}
