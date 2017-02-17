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
	"fmt"

	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/stream"
)

func NewASTC_RGBA_4x4(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 4, BlockHeight: 4, Srgb: false}}}
}
func NewASTC_RGBA_5x4(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 5, BlockHeight: 4, Srgb: false}}}
}
func NewASTC_RGBA_5x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 5, BlockHeight: 5, Srgb: false}}}
}
func NewASTC_RGBA_6x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 6, BlockHeight: 5, Srgb: false}}}
}
func NewASTC_RGBA_6x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 6, BlockHeight: 6, Srgb: false}}}
}
func NewASTC_RGBA_8x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 5, Srgb: false}}}
}
func NewASTC_RGBA_8x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 6, Srgb: false}}}
}
func NewASTC_RGBA_8x8(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 8, Srgb: false}}}
}
func NewASTC_RGBA_10x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 5, Srgb: false}}}
}
func NewASTC_RGBA_10x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 6, Srgb: false}}}
}
func NewASTC_RGBA_10x8(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 8, Srgb: false}}}
}
func NewASTC_RGBA_10x10(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 10, Srgb: false}}}
}
func NewASTC_RGBA_12x10(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 12, BlockHeight: 10, Srgb: false}}}
}
func NewASTC_RGBA_12x12(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 12, BlockHeight: 12, Srgb: false}}}
}
func NewASTC_SRGB8_ALPHA8_4x4(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 4, BlockHeight: 4, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_5x4(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 5, BlockHeight: 4, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_5x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 5, BlockHeight: 5, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_6x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 6, BlockHeight: 5, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_6x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 6, BlockHeight: 6, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_8x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 5, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_8x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 6, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_8x8(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 8, BlockHeight: 8, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_10x5(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 5, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_10x6(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 6, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_10x8(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 8, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_10x10(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 10, BlockHeight: 10, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_12x10(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 12, BlockHeight: 10, Srgb: true}}}
}
func NewASTC_SRGB8_ALPHA8_12x12(name string) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{BlockWidth: 12, BlockHeight: 12, Srgb: true}}}
}

func (f *FmtASTC) key() interface{} {
	return f.String()
}
func (f *FmtASTC) size(w, h int) int {
	bw, bh := int(f.BlockWidth), int(f.BlockHeight)
	return (16 * sint.AlignUp(w, bw) * sint.AlignUp(h, bh)) / (bw * bh)
}
func (f *FmtASTC) check(d []byte, w, h int) error {
	if actual, expected := len(d), f.size(w, h); expected != actual {
		return fmt.Errorf("Image data size (0x%x) did not match expected (0x%x) for dimensions %dx%d",
			actual, expected, w, h)
	}
	return nil
}
func (*FmtASTC) channels() []stream.Channel {
	return []stream.Channel{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}

func init() {
	fmts := []*Format{
		NewASTC_RGBA_4x4(""),
		NewASTC_RGBA_5x4(""),
		NewASTC_RGBA_5x5(""),
		NewASTC_RGBA_6x5(""),
		NewASTC_RGBA_6x6(""),
		NewASTC_RGBA_8x5(""),
		NewASTC_RGBA_8x6(""),
		NewASTC_RGBA_8x8(""),
		NewASTC_RGBA_10x5(""),
		NewASTC_RGBA_10x6(""),
		NewASTC_RGBA_10x8(""),
		NewASTC_RGBA_10x10(""),
		NewASTC_RGBA_12x10(""),
		NewASTC_RGBA_12x12(""),
		NewASTC_SRGB8_ALPHA8_4x4(""),
		NewASTC_SRGB8_ALPHA8_5x4(""),
		NewASTC_SRGB8_ALPHA8_5x5(""),
		NewASTC_SRGB8_ALPHA8_6x5(""),
		NewASTC_SRGB8_ALPHA8_6x6(""),
		NewASTC_SRGB8_ALPHA8_8x5(""),
		NewASTC_SRGB8_ALPHA8_8x6(""),
		NewASTC_SRGB8_ALPHA8_8x8(""),
		NewASTC_SRGB8_ALPHA8_10x5(""),
		NewASTC_SRGB8_ALPHA8_10x6(""),
		NewASTC_SRGB8_ALPHA8_10x8(""),
		NewASTC_SRGB8_ALPHA8_10x10(""),
		NewASTC_SRGB8_ALPHA8_12x10(""),
		NewASTC_SRGB8_ALPHA8_12x12(""),
	}
	for _, f := range fmts {
		RegisterConverter(f, RGBA_U8_NORM, func(src []byte, width, height int) ([]byte, error) {
			// TODO: Implement decompressor.
			return make([]byte, width*height*4), nil
		})
	}
}
