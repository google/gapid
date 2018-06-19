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

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
)

func init() {
	RegisterConverter(S3_DXT1_RGB, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeDXT1(r, dst, func(p *pixel) {
				p.setToBlackRGB()
			})
		})
	})
	RegisterConverter(S3_DXT1_RGBA, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeDXT1(r, dst, func(p *pixel) {
				p.setToBlackRGBA()
			})
		})
	})
	RegisterConverter(S3_DXT3_RGBA, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeAlphaDXT3(r, dst)
			decodeColorDXT3_5(r, dst)
		})
	})
	RegisterConverter(S3_DXT5_RGBA, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeAlphaDXT5(r, dst)
			decodeColorDXT3_5(r, dst)
		})
	})
}

type pixel struct {
	r, g, b, a int
}

func (p *pixel) setToBlackRGB() {
	p.r, p.g, p.b = 0, 0, 0
}

func (p *pixel) setToBlackRGBA() {
	p.r, p.g, p.b, p.a = 0, 0, 0, 0
}

func (p *pixel) setColorFrom(c pixel) {
	p.r, p.g, p.b = c.r, c.g, c.b
}

func (p *pixel) setToAverage(c0, c1 pixel) {
	p.r = (c0.r + c1.r) / 2
	p.g = (c0.g + c1.g) / 2
	p.b = (c0.b + c1.b) / 2
}

func (p *pixel) setToMix3(c0, c1 pixel) {
	p.r = (2*c0.r + c1.r) / 3
	p.g = (2*c0.g + c1.g) / 3
	p.b = (2*c0.b + c1.b) / 3
}

func decode4x4Blocks(src []byte, width, height, depth int, decodeBlock func(r binary.Reader, dst []pixel)) ([]byte, error) {
	dst := make([]byte, width*height*depth*4)
	block := make([]pixel, 16)
	r := endian.Reader(bytes.NewReader(src), device.LittleEndian)
	for z := 0; z < depth; z++ {
		dst := dst[z*width*height*4:]
		for y := 0; y < height; y += 4 {
			for x := 0; x < width; x += 4 {
				decodeBlock(r, block)
				copyToDest(block, dst, x, y, width, height)
			}
		}
	}
	return dst, nil
}

func expand565(c int) pixel {
	return pixel{
		((c >> 8) & 0xf8) | ((c >> 13) & 0x7),
		((c >> 3) & 0xfc) | ((c >> 9) & 0x3),
		((c << 3) & 0xf8) | ((c >> 2) & 0x7),
		1,
	}
}

func decodeDXT1(r binary.Reader, dst []pixel, black func(p *pixel)) {
	c0, c1, codes := r.Uint16(), r.Uint16(), r.Uint32()
	p0, p1 := expand565(int(c0)), expand565(int(c1))
	for i := 0; i < 16; i++ {
		dst[i].a = 255
		switch codes & 0x3 {
		case 0:
			dst[i].setColorFrom(p0)
		case 1:
			dst[i].setColorFrom(p1)
		case 2:
			if c0 > c1 {
				dst[i].setToMix3(p0, p1)
			} else {
				dst[i].setToAverage(p0, p1)
			}
		case 3:
			if c0 > c1 {
				dst[i].setToMix3(p1, p0)
			} else {
				black(&dst[i])
			}
		}
		codes >>= 2
	}
}

func decodeColorDXT3_5(r binary.Reader, dst []pixel) {
	c0, c1, codes := r.Uint16(), r.Uint16(), r.Uint32()
	p0, p1 := expand565(int(c0)), expand565(int(c1))
	for i := 0; i < 16; i++ {
		switch codes & 0x3 {
		case 0:
			dst[i].setColorFrom(p0)
		case 1:
			dst[i].setColorFrom(p1)
		case 2:
			dst[i].setToMix3(p0, p1)
		case 3:
			dst[i].setToMix3(p1, p0)
		}
		codes >>= 2
	}
}

func decodeAlphaDXT3(r binary.Reader, dst []pixel) {
	a := r.Uint64()
	for i := 0; i < 16; i++ {
		dst[i].a = int(a&0xf) * 0x11
		a >>= 4
	}
}

func decodeAlphaDXT5(r binary.Reader, dst []pixel) {
	a0, a1, codes := int(r.Uint8()), int(r.Uint8()), uint64(r.Uint16())|(uint64(r.Uint32())<<16)
	for i := 0; i < 16; i++ {
		dst[i].a = mixAlphaDXT5(a0, a1, codes)
		codes >>= 3
	}
}

func mixAlphaDXT5(a0, a1 int, code uint64) int {
	c := int(code & 0x7)
	if a0 > a1 {
		switch c {
		case 0:
			return a0
		case 1:
			return a1
		default:
			return (a0*(8-c) + a1*(c-1)) / 7
		}
	} else {
		switch c {
		case 0:
			return a0
		case 1:
			return a1
		case 6:
			return 0
		case 7:
			return 255
		default:
			return (a0*(6-c) + a1*(c-1)) / 5
		}
	}
}

func copyToDest(block []pixel, dst []byte, x, y, width, height int) {
	o := 4 * (y*width + x)
	for dy := 0; dy < 4 && y+dy < height; dy++ {
		i, p := o, dy*4
		for dx := 0; dx < 4 && x+dx < width; dx++ {
			dst[i+0] = sint.Byte(block[p].r)
			dst[i+1] = sint.Byte(block[p].g)
			dst[i+2] = sint.Byte(block[p].b)
			dst[i+3] = sint.Byte(block[p].a)

			i += 4
			p++
		}
		o += 4 * width
	}
}
