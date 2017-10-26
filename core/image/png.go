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

package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
)

var PNG = NewPNG("png")

// PNGFrom returns a new Data with the PNG format.
func PNGFrom(data []byte) (*Data, error) {
	cfg, err := png.DecodeConfig(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return &Data{
		Width:  uint32(cfg.Width),
		Height: uint32(cfg.Height),
		Depth:  1,
		Format: PNG,
		Bytes:  data,
	}, nil
}

// NewPNG returns a format representing the the texture compression format with the
// same name.
func NewPNG(name string) *Format { return &Format{name, &Format_Png{&FmtPNG{}}} }

func (f *FmtPNG) key() interface{}                   { return *f }
func (*FmtPNG) size(w, h, d int) int                 { return -1 }
func (*FmtPNG) check(data []byte, w, h, d int) error { return nil }
func (*FmtPNG) resize(data []byte, srcW, srcH, dstW, dstH int) ([]byte, error) {
	return nil, ErrResizeUnsupported
}
func (*FmtPNG) channels() stream.Channels {
	return nil
}
func init() {
	RegisterConverter(RGBA_U8_NORM, PNG,
		func(src []byte, width, height, depth int) ([]byte, error) {
			if depth != 1 {
				return nil, fmt.Errorf("Cannot decode PNG with depth of %d", depth)
			}
			img := image.NewNRGBA(image.Rect(0, 0, width, height))
			i := 0
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					r, g, b, a := src[i+0], src[i+1], src[i+2], src[i+3]
					img.Set(x, y, color.NRGBA{r, g, b, a})
					i += 4
				}
			}

			buffer := bytes.Buffer{}
			png.Encode(&buffer, img)
			return buffer.Bytes(), nil
		})
	RegisterConverter(PNG, RGBA_U8_NORM,
		func(src []byte, width, height, depth int) ([]byte, error) {
			if depth != 1 {
				return nil, fmt.Errorf("Cannot decode PNG with depth of %d", depth)
			}
			img, err := png.Decode(bytes.NewReader(src))
			if err != nil {
				return nil, err
			}
			if w := img.Bounds().Dx(); width != w {
				return nil, fmt.Errorf("PNG width was not as expected. Got: %v, expected: %v", w, width)
			}
			if h := img.Bounds().Dy(); height != h {
				return nil, fmt.Errorf("PNG width was not as expected. Got: %v, expected: %v", h, height)
			}

			var f *Format
			buf := &bytes.Buffer{}
			e := endian.Writer(buf, device.LittleEndian)

			switch img.ColorModel() {
			case color.RGBA64Model:
				return nil, fmt.Errorf("Unsupported color model 'RGBA64'")
			case color.RGBAModel:
				// PNG cannot indicate pre-multiplied alpha, and at the time of writing
				// this color format is only used when there is no alpha channel.
				f = RGBA_U8_NORM
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						c := img.At(x, y).(color.RGBA)
						e.Uint8(c.R)
						e.Uint8(c.G)
						e.Uint8(c.B)
						e.Uint8(c.A)
					}
				}
			case color.NRGBAModel:
				f = RGBA_U8_NORM
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						c := img.At(x, y).(color.NRGBA)
						e.Uint8(c.R)
						e.Uint8(c.G)
						e.Uint8(c.B)
						e.Uint8(c.A)
					}
				}
			case color.NRGBA64Model:
				return nil, fmt.Errorf("Unsupported color model 'NRGBA64'")
			case color.AlphaModel:
				return nil, fmt.Errorf("Unsupported color model 'Alpha'")
			case color.Alpha16Model:
				return nil, fmt.Errorf("Unsupported color model 'Alpha16'")
			case color.GrayModel:
				return nil, fmt.Errorf("Unsupported color model 'Gray'")
			default:
				f = RGBA_F32
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						r, g, b, a := img.At(x, y).RGBA()
						e.Float32(float32(r) / 0xffff)
						e.Float32(float32(g) / 0xffff)
						e.Float32(float32(b) / 0xffff)
						e.Float32(float32(a) / 0xffff)
					}
				}
			}
			return Convert(buf.Bytes(), width, height, 1, f, RGBA_U8_NORM)
		})
}
