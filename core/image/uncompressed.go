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

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
)

var (
	RGBA_F32      = newUncompressed(fmts.RGBA_F32)
	RGB_U8_NORM   = newUncompressed(fmts.RGB_U8_NORM)
	RGBA_U8_NORM  = newUncompressed(fmts.RGBA_U8_NORM)
	SRGB_U8_NORM  = newUncompressed(fmts.SRGB_U8_NORM)
	SRGBA_U8_NORM = newUncompressed(fmts.SRGBA_U8_NORM)
	R_U16_NORM    = newUncompressed(fmts.R_U16_NORM)
	RG_U16_NORM   = newUncompressed(fmts.RG_U16_NORM)
	R_S16_NORM    = newUncompressed(fmts.R_S16_NORM)
	RG_S16_NORM   = newUncompressed(fmts.RG_S16_NORM)
	Gray_U8_NORM  = newUncompressed(fmts.Gray_U8_NORM)
	D_U16_NORM    = newUncompressed(fmts.D_U16_NORM)

	Luminance_R32      = newUncompressed(fmts.L_F32)
	Luminance_U8_NORM  = newUncompressed(fmts.L_U8_NORM)
	LuminanceA_U8_NORM = newUncompressed(fmts.LA_U8_NORM)
)

// newUncompressed returns a new uncompressed format containing with the default
// stream name.
func newUncompressed(f *stream.Format) *Format {
	name := fmt.Sprint(f)
	return NewUncompressed(name, f)
}

// NewUncompressed returns a new uncompressed format wrapping f.
func NewUncompressed(name string, f *stream.Format) *Format {
	return &Format{Name: name, Format: &Format_Uncompressed{&FmtUncompressed{Format: f}}}
}

func (f *FmtUncompressed) key() interface{} {
	return f.Format.String()
}
func (f *FmtUncompressed) size(w, h, d int) int {
	return f.Format.Size(w * h * d)
}
func (f *FmtUncompressed) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (f *FmtUncompressed) channels() stream.Channels {
	out := make(stream.Channels, len(f.Format.Components))
	for i, c := range f.Format.Components {
		out[i] = c.Channel
	}
	return out
}

func (f *FmtUncompressed) resize(data []byte, srcW, srcH, srcD, dstW, dstH, dstD int) ([]byte, error) {
	format := &Format{Name: "Temporary Uncompressed Format", Format: &Format_Uncompressed{f}}
	data, err := Convert(data, srcW, srcH, srcD, format, RGBA_F32)
	if err != nil {
		return nil, err
	}
	data, err = resizeRGBA_F32(data, srcW, srcH, srcD, dstW, dstH, dstD)
	if err != nil {
		return nil, err
	}
	return Convert(data, dstW, dstH, dstD, RGBA_F32, format)
}

func (f *FmtUncompressed) convert(data []byte, w, h, d int, dstFmt *Format) ([]byte, error) {
	if err := f.check(data, w, h, d); err != nil {
		return nil, err
	}
	switch dstFmt := protoutil.OneOf(dstFmt.Format).(type) {
	case *FmtUncompressed:
		return stream.Convert(dstFmt.Format, f.Format, data)
	default:
		return nil, nil
	}
}
