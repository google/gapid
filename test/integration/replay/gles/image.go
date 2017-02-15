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

package gles

import (
	"bytes"
	"encoding/base64"
	"fmt"
	goimg "image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/data/endian"
	gpuimg "github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
)

const referenceImageDir = "reference"

// storeReferenceImage replaces the reference image with img.
func storeReferenceImage(ctx log.Context, outputDir string, name string, img *gpuimg.Image2D) {
	ctx = ctx.S("Name", name)
	data := &bytes.Buffer{}
	i, err := toGoImage(img)
	if err != nil {
		jot.Fatal(ctx, err, "Failed to convert GPU image to Go image")
	}
	if err := png.Encode(data, i); err != nil {
		jot.Fatal(ctx, err, "Failed to encode reference image")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		jot.Fatal(ctx, err, "Failed to create reference image directory")
	}
	path := filepath.Join(outputDir, name+".png")
	if err := ioutil.WriteFile(path, data.Bytes(), 0666); err != nil {
		jot.Fatal(ctx, err, "Failed to store reference image")
	}
}

func readFile(path string) ([]byte, error) {
	b64, found := embedded[path]
	if found {
		return base64.StdEncoding.DecodeString(b64)
	} else {
		return ioutil.ReadFile(path)
	}
}

// loadReferenceImage loads the reference image with the specified name.
func loadReferenceImage(ctx log.Context, name string) *gpuimg.Image2D {
	ctx = ctx.S("Name", name)
	path := filepath.Join(referenceImageDir, name+".png")
	data, err := readFile(path)
	if err != nil {
		jot.Fatal(ctx, err, "Failed to load reference image")
	}
	img, err := png.Decode(bytes.NewBuffer(data))
	if err != nil {
		jot.Fatal(ctx, err, "Failed to decode reference image")
	}
	out, err := toGPUImage(img)
	if err != nil {
		jot.Fatal(ctx, err, "Failed to convert Go image to GPU image")
	}
	return out
}

func toGoImage(in *gpuimg.Image2D) (goimg.Image, error) {
	rect := goimg.Rect(0, 0, int(in.Width), int(in.Height))
	switch in.Format.Key() {
	case gpuimg.RGBA_U8_NORM.Key():
		out := goimg.NewNRGBA(rect)
		out.Pix = in.Data
		return out, nil

	case gpuimg.D_U16_NORM.Key():
		out := goimg.NewGray16(rect)
		out.Pix = make([]byte, len(in.Data))
		// Endian-swap.
		for i, c := 0, len(in.Data); i < c; i += 2 {
			out.Pix[i+0], out.Pix[i+1] = in.Data[i+1], in.Data[i+0]
		}
		return out, nil

	default:
		uncompressed := in.Format.GetUncompressed()
		var converted *gpuimg.Image2D
		var err error
		if depth, _ := uncompressed.Format.Component(stream.Channel_Depth); depth != nil {
			converted, err = in.Convert(gpuimg.D_U16_NORM)
		} else {
			converted, err = in.Convert(gpuimg.RGBA_U8_NORM)
		}
		if err != nil {
			return nil, err
		}
		return toGoImage(converted)
	}
}

func toGPUImage(in goimg.Image) (*gpuimg.Image2D, error) {
	w, h := in.Bounds().Dx(), in.Bounds().Dy()
	out := &gpuimg.Image2D{Width: uint32(w), Height: uint32(h)}
	buf := &bytes.Buffer{}
	e := endian.Writer(buf, device.BigEndian)

	switch in.ColorModel() {
	case color.RGBAModel, color.NRGBAModel:
		out.Format = gpuimg.RGBA_U8_NORM
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, a := in.At(x, y).RGBA()
				e.Uint8(uint8(r >> 8))
				e.Uint8(uint8(g >> 8))
				e.Uint8(uint8(b >> 8))
				e.Uint8(uint8(a >> 8))
			}
		}
	case color.Gray16Model:
		out.Format = gpuimg.D_U16_NORM
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				d, _, _, _ := in.At(x, y).RGBA()
				e.Uint16(uint16(d))
			}
		}
	default:
		return nil, fmt.Errorf("Unsupported color model %v", in.ColorModel())
	}

	out.Data = buf.Bytes()
	return out, nil
}

func quantizeImage(in *gpuimg.Image2D) *gpuimg.Image2D {
	tmp, err := toGoImage(in)
	if err != nil {
		panic(err)
	}
	out, err := toGPUImage(tmp)
	if err != nil {
		panic(err)
	}
	return out
}
