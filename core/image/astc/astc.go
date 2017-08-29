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

// Package astc implements ASTC texture decompression.
//
// astc is in a separate package from image as it contains cgo code that can
// slow builds.
package astc

// #cgo LDFLAGS: -lastc-encoder -lstdc++
// #include "astc.h"
import "C"

import (
	"unsafe"

	"github.com/google/gapid/core/image"
)

var (
	RGBA_4x4           = NewRGBA_4x4("ASTC_RGBA_4x4")
	RGBA_5x4           = NewRGBA_5x4("ASTC_RGBA_5x4")
	RGBA_5x5           = NewRGBA_5x5("ASTC_RGBA_5x5")
	RGBA_6x5           = NewRGBA_6x5("ASTC_RGBA_6x5")
	RGBA_6x6           = NewRGBA_6x6("ASTC_RGBA_6x6")
	RGBA_8x5           = NewRGBA_8x5("ASTC_RGBA_8x5")
	RGBA_8x6           = NewRGBA_8x6("ASTC_RGBA_8x6")
	RGBA_8x8           = NewRGBA_8x8("ASTC_RGBA_8x8")
	RGBA_10x5          = NewRGBA_10x5("ASTC_RGBA_10x5")
	RGBA_10x6          = NewRGBA_10x6("ASTC_RGBA_10x6")
	RGBA_10x8          = NewRGBA_10x8("ASTC_RGBA_10x8")
	RGBA_10x10         = NewRGBA_10x10("ASTC_RGBA_10x10")
	RGBA_12x10         = NewRGBA_12x10("ASTC_RGBA_12x10")
	RGBA_12x12         = NewRGBA_12x12("ASTC_RGBA_12x12")
	SRGB8_ALPHA8_4x4   = NewSRGB8_ALPHA8_4x4("ASTC_SRGB8_ALPHA8_4x4")
	SRGB8_ALPHA8_5x4   = NewSRGB8_ALPHA8_5x4("ASTC_SRGB8_ALPHA8_5x4")
	SRGB8_ALPHA8_5x5   = NewSRGB8_ALPHA8_5x5("ASTC_SRGB8_ALPHA8_5x5")
	SRGB8_ALPHA8_6x5   = NewSRGB8_ALPHA8_6x5("ASTC_SRGB8_ALPHA8_6x5")
	SRGB8_ALPHA8_6x6   = NewSRGB8_ALPHA8_6x6("ASTC_SRGB8_ALPHA8_6x6")
	SRGB8_ALPHA8_8x5   = NewSRGB8_ALPHA8_8x5("ASTC_SRGB8_ALPHA8_8x5")
	SRGB8_ALPHA8_8x6   = NewSRGB8_ALPHA8_8x6("ASTC_SRGB8_ALPHA8_8x6")
	SRGB8_ALPHA8_8x8   = NewSRGB8_ALPHA8_8x8("ASTC_SRGB8_ALPHA8_8x8")
	SRGB8_ALPHA8_10x5  = NewSRGB8_ALPHA8_10x5("ASTC_SRGB8_ALPHA8_10x5")
	SRGB8_ALPHA8_10x6  = NewSRGB8_ALPHA8_10x6("ASTC_SRGB8_ALPHA8_10x6")
	SRGB8_ALPHA8_10x8  = NewSRGB8_ALPHA8_10x8("ASTC_SRGB8_ALPHA8_10x8")
	SRGB8_ALPHA8_10x10 = NewSRGB8_ALPHA8_10x10("ASTC_SRGB8_ALPHA8_10x10")
	SRGB8_ALPHA8_12x10 = NewSRGB8_ALPHA8_12x10("ASTC_SRGB8_ALPHA8_12x10")
	SRGB8_ALPHA8_12x12 = NewSRGB8_ALPHA8_12x12("ASTC_SRGB8_ALPHA8_12x12")
)

func NewRGBA_4x4(name string) *image.Format           { return image.NewASTC(name, 4, 4, false) }
func NewRGBA_5x4(name string) *image.Format           { return image.NewASTC(name, 5, 4, false) }
func NewRGBA_5x5(name string) *image.Format           { return image.NewASTC(name, 5, 5, false) }
func NewRGBA_6x5(name string) *image.Format           { return image.NewASTC(name, 6, 5, false) }
func NewRGBA_6x6(name string) *image.Format           { return image.NewASTC(name, 6, 6, false) }
func NewRGBA_8x5(name string) *image.Format           { return image.NewASTC(name, 8, 5, false) }
func NewRGBA_8x6(name string) *image.Format           { return image.NewASTC(name, 8, 6, false) }
func NewRGBA_8x8(name string) *image.Format           { return image.NewASTC(name, 8, 8, false) }
func NewRGBA_10x5(name string) *image.Format          { return image.NewASTC(name, 10, 5, false) }
func NewRGBA_10x6(name string) *image.Format          { return image.NewASTC(name, 10, 6, false) }
func NewRGBA_10x8(name string) *image.Format          { return image.NewASTC(name, 10, 8, false) }
func NewRGBA_10x10(name string) *image.Format         { return image.NewASTC(name, 10, 10, false) }
func NewRGBA_12x10(name string) *image.Format         { return image.NewASTC(name, 12, 10, false) }
func NewRGBA_12x12(name string) *image.Format         { return image.NewASTC(name, 12, 12, false) }
func NewSRGB8_ALPHA8_4x4(name string) *image.Format   { return image.NewASTC(name, 4, 4, true) }
func NewSRGB8_ALPHA8_5x4(name string) *image.Format   { return image.NewASTC(name, 5, 4, true) }
func NewSRGB8_ALPHA8_5x5(name string) *image.Format   { return image.NewASTC(name, 5, 5, true) }
func NewSRGB8_ALPHA8_6x5(name string) *image.Format   { return image.NewASTC(name, 6, 5, true) }
func NewSRGB8_ALPHA8_6x6(name string) *image.Format   { return image.NewASTC(name, 6, 6, true) }
func NewSRGB8_ALPHA8_8x5(name string) *image.Format   { return image.NewASTC(name, 8, 5, true) }
func NewSRGB8_ALPHA8_8x6(name string) *image.Format   { return image.NewASTC(name, 8, 6, true) }
func NewSRGB8_ALPHA8_8x8(name string) *image.Format   { return image.NewASTC(name, 8, 8, true) }
func NewSRGB8_ALPHA8_10x5(name string) *image.Format  { return image.NewASTC(name, 10, 5, true) }
func NewSRGB8_ALPHA8_10x6(name string) *image.Format  { return image.NewASTC(name, 10, 6, true) }
func NewSRGB8_ALPHA8_10x8(name string) *image.Format  { return image.NewASTC(name, 10, 8, true) }
func NewSRGB8_ALPHA8_10x10(name string) *image.Format { return image.NewASTC(name, 10, 10, true) }
func NewSRGB8_ALPHA8_12x10(name string) *image.Format { return image.NewASTC(name, 12, 10, true) }
func NewSRGB8_ALPHA8_12x12(name string) *image.Format { return image.NewASTC(name, 12, 12, true) }

func init() {
	C.init_astc()

	for _, f := range []struct {
		src *image.Format
		dst *image.Format
	}{
		{RGBA_4x4, image.RGBA_U8_NORM},
		{RGBA_5x4, image.RGBA_U8_NORM},
		{RGBA_5x5, image.RGBA_U8_NORM},
		{RGBA_6x5, image.RGBA_U8_NORM},
		{RGBA_6x6, image.RGBA_U8_NORM},
		{RGBA_8x5, image.RGBA_U8_NORM},
		{RGBA_8x6, image.RGBA_U8_NORM},
		{RGBA_8x8, image.RGBA_U8_NORM},
		{RGBA_10x5, image.RGBA_U8_NORM},
		{RGBA_10x6, image.RGBA_U8_NORM},
		{RGBA_10x8, image.RGBA_U8_NORM},
		{RGBA_10x10, image.RGBA_U8_NORM},
		{RGBA_12x10, image.RGBA_U8_NORM},
		{RGBA_12x12, image.RGBA_U8_NORM},
		{SRGB8_ALPHA8_4x4, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_5x4, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_5x5, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_6x5, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_6x6, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_8x5, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_8x6, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_8x8, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_10x5, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_10x6, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_10x8, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_10x10, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_12x10, image.SRGBA_U8_NORM},
		{SRGB8_ALPHA8_12x12, image.SRGBA_U8_NORM},
	} {
		f := f
		image.RegisterConverter(f.src, f.dst, func(src []byte, w, h, d int) ([]byte, error) {
			dst := make([]byte, w*h*d*4)
			sliceSize := f.src.Size(w, h, 1)
			for z := 0; z < d; z++ {
				dst, src := dst[z*w*h*4:], src[z*sliceSize:]
				in := (unsafe.Pointer)(&src[0])
				out := (unsafe.Pointer)(&dst[0])
				blockW := f.src.GetAstc().BlockWidth
				blockH := f.src.GetAstc().BlockHeight
				C.decompress_astc(
					(*C.uint8_t)(in),
					(*C.uint8_t)(out),
					(C.uint32_t)(w),
					(C.uint32_t)(h),
					(C.uint32_t)(blockW),
					(C.uint32_t)(blockH))
			}
			return dst, nil
		})
	}
}
