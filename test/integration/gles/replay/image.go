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

package replay

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	goimg "image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/gapid/core/data/endian"
	gpuimg "github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
)

// storeReferenceImage replaces the reference image with img.
func storeReferenceImage(ctx context.Context, outputDir string, name string, img *gpuimg.Data) {
	ctx = log.V{"name": name}.Bind(ctx)
	data := &bytes.Buffer{}
	i, err := toGoImage(img)
	if err != nil {
		log.F(ctx, true, "Failed to convert GPU image to Go image: %v", err)
	}
	if err := png.Encode(data, i); err != nil {
		log.F(ctx, true, "Failed to encode reference image: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.F(ctx, true, "Failed to create reference image directory: %v", err)
	}
	path := filepath.Join(outputDir, name+".png")
	if err := ioutil.WriteFile(path, data.Bytes(), 0666); err != nil {
		log.F(ctx, true, "Failed to store reference image: %v", err)
	}
}

// loadReferenceImage loads the reference image with the specified name.
func loadReferenceImage(ctx context.Context, name string) *gpuimg.Data {
	ctx = log.V{"name": name}.Bind(ctx)
	b64, found := embedded[name+".png"]
	if !found {
		log.F(ctx, true, "Embedded reference image '%s' not found", name)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.F(ctx, true, "Failed to load reference image: %v", err)
	}
	img, err := png.Decode(bytes.NewBuffer(data))
	if err != nil {
		log.F(ctx, true, "Failed to decode reference image: %v", err)
	}
	out, err := toGPUImage(img)
	if err != nil {
		log.F(ctx, true, "Failed to convert Go image to GPU image: %v", err)
	}
	return out
}

func toGoImage(in *gpuimg.Data) (goimg.Image, error) {
	rect := goimg.Rect(0, 0, int(in.Width), int(in.Height))
	switch in.Format.Key() {
	case gpuimg.RGBA_U8_NORM.Key():
		out := goimg.NewNRGBA(rect)
		out.Pix = in.Bytes
		return out, nil

	case gpuimg.D_U16_NORM.Key():
		out := goimg.NewGray16(rect)
		out.Pix = make([]byte, len(in.Bytes))
		// Endian-swap.
		for i, c := 0, len(in.Bytes); i < c; i += 2 {
			out.Pix[i+0], out.Pix[i+1] = in.Bytes[i+1], in.Bytes[i+0]
		}
		return out, nil

	default:
		uncompressed := in.Format.GetUncompressed()
		var converted *gpuimg.Data
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

func toGPUImage(in goimg.Image) (*gpuimg.Data, error) {
	w, h := in.Bounds().Dx(), in.Bounds().Dy()
	out := &gpuimg.Data{Width: uint32(w), Height: uint32(h), Depth: 1}
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

	out.Bytes = buf.Bytes()
	return out, nil
}

func quantizeImage(in *gpuimg.Data) *gpuimg.Data {
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
