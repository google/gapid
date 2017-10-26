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
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/stream"
)

var (
	ATC_RGB_AMD                     = NewATC_RGB_AMD("ATC_RGB_AMD")
	ATC_RGBA_EXPLICIT_ALPHA_AMD     = NewATC_RGBA_EXPLICIT_ALPHA_AMD("ATC_RGBA_EXPLICIT_ALPHA_AMD")
	ATC_RGBA_INTERPOLATED_ALPHA_AMD = NewATC_RGBA_INTERPOLATED_ALPHA_AMD("ATC_RGBA_INTERPOLATED_ALPHA_AMD")
)

// NewATC_RGB_AMD returns a format representing the ATC_RGB_AMD block texture
// compression format.
func NewATC_RGB_AMD(name string) *Format {
	return &Format{name, &Format_AtcRgbAmd{&FmtATC_RGB_AMD{}}}
}

// NewATC_RGBA_EXPLICIT_ALPHA_AMD returns a format representing the
// ATC_RGBA_EXPLICIT_ALPHA_AMD block texture compression format.
func NewATC_RGBA_EXPLICIT_ALPHA_AMD(name string) *Format {
	return &Format{name, &Format_AtcRgbaExplicitAlphaAmd{&FmtATC_RGBA_EXPLICIT_ALPHA_AMD{}}}
}

// NewATC_RGBA_INTERPOLATED_ALPHA_AMD returns a format representing the
// ATC_RGBA_INTERPOLATED_ALPHA_AMD block compression format.
func NewATC_RGBA_INTERPOLATED_ALPHA_AMD(name string) *Format {
	return &Format{name, &Format_AtcRgbaInterpolatedAlphaAmd{&FmtATC_RGBA_INTERPOLATED_ALPHA_AMD{}}}
}

func (f *FmtATC_RGB_AMD) key() interface{} {
	return *f
}
func (*FmtATC_RGB_AMD) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtATC_RGB_AMD) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtATC_RGB_AMD) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue}
}

func (f *FmtATC_RGBA_EXPLICIT_ALPHA_AMD) key() interface{} {
	return *f
}
func (*FmtATC_RGBA_EXPLICIT_ALPHA_AMD) size(w, h, d int) int {
	return d * sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)
}
func (f *FmtATC_RGBA_EXPLICIT_ALPHA_AMD) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtATC_RGBA_EXPLICIT_ALPHA_AMD) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}

func (f *FmtATC_RGBA_INTERPOLATED_ALPHA_AMD) key() interface{} {
	return *f
}
func (*FmtATC_RGBA_INTERPOLATED_ALPHA_AMD) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtATC_RGBA_INTERPOLATED_ALPHA_AMD) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtATC_RGBA_INTERPOLATED_ALPHA_AMD) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}

func init() {
	RegisterConverter(ATC_RGB_AMD, RGBA_U8_NORM,
		func(src []byte, width, height, depth int) ([]byte, error) {
			dst, j := make([]byte, width*height*depth*4), 0

			blockWidth := sint.Max(width/4, 1)
			blockHeight := sint.Max(height/4, 1)

			bs := binary.BitStream{Data: src}

			c := [4][3]uint64{
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
			}

			for z := 0; z < depth; z++ {
				dst := dst[z*width*height*4:]
				for by := 0; by < blockHeight; by++ {
					for bx := 0; bx < blockWidth; bx++ {
						c[0][2] = bs.Read(5) << 3
						c[0][1] = bs.Read(5) << 3
						c[0][0] = bs.Read(5) << 3
						alt := bs.ReadBit() != 0
						c[3][2] = bs.Read(5) << 3
						c[3][1] = bs.Read(6) << 2
						c[3][0] = bs.Read(5) << 3
						for i := 0; i < 3; i++ {
							if alt {
								c[2][i] = c[0][i]
								c[1][i] = c[0][i] - c[3][i]/4
								c[0][i] = 0
							} else {
								c[1][i] = (c[0][i]*2 + c[3][i]*1) / 3
								c[2][i] = (c[0][i]*1 + c[3][i]*2) / 3
							}
						}
						for y := by * 4; y < (by+1)*4; y++ {
							for x := bx * 4; x < (bx+1)*4; x++ {
								idx := bs.Read(2)
								if x < width && y < height {
									j = 4 * (y*width + x)
									dst[j+0] = uint8(c[idx][0])
									dst[j+1] = uint8(c[idx][1])
									dst[j+2] = uint8(c[idx][2])
									dst[j+3] = 255
								}
							}
						}
					}
				}
			}

			return dst, nil
		})

	RegisterConverter(ATC_RGBA_EXPLICIT_ALPHA_AMD, RGBA_U8_NORM,
		func(src []byte, width, height, depth int) ([]byte, error) {
			dst, j := make([]byte, width*height*depth*4), 0

			blockWidth := sint.Max(width/4, 1)
			blockHeight := sint.Max(height/4, 1)

			bs := binary.BitStream{Data: src}

			a := [16]uint8{
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			}

			c := [4][3]uint64{
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
			}

			for z := 0; z < depth; z++ {
				dst := dst[z*width*height*4:]
				for by := 0; by < blockHeight; by++ {
					for bx := 0; bx < blockWidth; bx++ {
						for i := 0; i < 16; i++ {
							a[i] = uint8(bs.Read(4) << 4)
						}
						c[0][2] = bs.Read(5) << 3
						c[0][1] = bs.Read(5) << 3
						c[0][0] = bs.Read(5) << 3
						bs.ReadBit()
						c[3][2] = bs.Read(5) << 3
						c[3][1] = bs.Read(6) << 2
						c[3][0] = bs.Read(5) << 3
						for i := 0; i < 3; i++ {
							c[1][i] = (c[0][i]*2 + c[3][i]*1) / 3
							c[2][i] = (c[0][i]*1 + c[3][i]*2) / 3
						}
						p := 0
						for y := by * 4; y < (by+1)*4; y++ {
							for x := bx * 4; x < (bx+1)*4; x++ {
								idx := bs.Read(2)
								if x < width && y < height {
									j = 4 * (y*width + x)
									dst[j+0] = uint8(c[idx][0])
									dst[j+1] = uint8(c[idx][1])
									dst[j+2] = uint8(c[idx][2])
									dst[j+3] = a[p]
									p++
								}
							}
						}
					}
				}
			}

			return dst, nil
		})

	RegisterConverter(ATC_RGBA_INTERPOLATED_ALPHA_AMD, RGBA_U8_NORM,
		func(src []byte, width, height, depth int) ([]byte, error) {
			dst, j := make([]byte, width*height*depth*4), 0

			blockWidth := sint.Max(width/4, 1)
			blockHeight := sint.Max(height/4, 1)

			bs := binary.BitStream{Data: src}

			t := [8]uint8{}

			a := [16]uint8{
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			}

			c := [4][3]uint64{
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
				{0, 0, 0},
			}

			for z := 0; z < depth; z++ {
				dst := dst[z*width*height*4:]
				for by := 0; by < blockHeight; by++ {
					for bx := 0; bx < blockWidth; bx++ {
						t0 := uint(bs.Read(8))
						t1 := uint(bs.Read(8))
						t[0] = uint8(t0)
						t[1] = uint8(t1)

						if t0 > t1 {
							t[2] = uint8((6*t0 + 1*t1) / 7)
							t[3] = uint8((5*t0 + 2*t1) / 7)
							t[4] = uint8((4*t0 + 3*t1) / 7)
							t[5] = uint8((3*t0 + 4*t1) / 7)
							t[6] = uint8((2*t0 + 5*t1) / 7)
							t[7] = uint8((1*t0 + 6*t1) / 7)
						} else {
							t[2] = uint8((4*t0 + 1*t1) / 5)
							t[3] = uint8((3*t0 + 2*t1) / 5)
							t[4] = uint8((2*t0 + 3*t1) / 5)
							t[5] = uint8((1*t0 + 3*t1) / 5)
							t[6] = 0
							t[7] = 255
						}

						for i := range a {
							a[i] = t[bs.Read(3)]
						}

						c[0][2] = bs.Read(5) << 3
						c[0][1] = bs.Read(5) << 3
						c[0][0] = bs.Read(5) << 3
						bs.ReadBit()
						c[3][2] = bs.Read(5) << 3
						c[3][1] = bs.Read(6) << 2
						c[3][0] = bs.Read(5) << 3
						for i := 0; i < 3; i++ {
							c[1][i] = (c[0][i]*2 + c[3][i]*1) / 3
							c[2][i] = (c[0][i]*1 + c[3][i]*2) / 3
						}
						p := 0
						for y := by * 4; y < (by+1)*4; y++ {
							for x := bx * 4; x < (bx+1)*4; x++ {
								idx := bs.Read(2)
								if x < width && y < height {
									j = 4 * (y*width + x)
									dst[j+0] = uint8(c[idx][0])
									dst[j+1] = uint8(c[idx][1])
									dst[j+2] = uint8(c[idx][2])
									dst[j+3] = a[p]
									p++
								}
							}
						}
					}
				}
			}

			return dst, nil
		})
}
