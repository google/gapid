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
	RGBA_F32     = newUncompressed(fmts.RGBA_F32)
	RGB_U8_NORM  = newUncompressed(fmts.RGB_U8_NORM)
	RGBA_U8_NORM = newUncompressed(fmts.RGBA_U8_NORM)
	R_U16_NORM   = newUncompressed(fmts.R_U16_NORM)
	RG_U16_NORM  = newUncompressed(fmts.RG_U16_NORM)
	R_S16_NORM   = newUncompressed(fmts.R_S16_NORM)
	RG_S16_NORM  = newUncompressed(fmts.RG_S16_NORM)
	D_U16_NORM   = newUncompressed(fmts.D_U16_NORM)
)

// newUncompressed returns a new uncompressed format containing with the default
// stream name.
func newUncompressed(f *stream.Format) *Format {
	name := fmt.Sprint(f)
	return NewUncompressed(name, f)
}

// NewUncompressed returns a new uncompressed format wrapping f.
func NewUncompressed(name string, f *stream.Format) *Format {
	return &Format{name, &Format_Uncompressed{&FmtUncompressed{f}}}
}

func (f *FmtUncompressed) key() interface{} {
	return f.Format.String()
}
func (f *FmtUncompressed) size(w, h int) int {
	return f.Format.Size(w * h)
}
func (f *FmtUncompressed) check(d []byte, w, h int) error {
	return checkSize(d, f, w, h)
}
func (f *FmtUncompressed) channels() []stream.Channel {
	out := make([]stream.Channel, len(f.Format.Components))
	for i, c := range f.Format.Components {
		out[i] = c.Channel
	}
	return out
}

func (f *FmtUncompressed) resize(data []byte, srcW, srcH, dstW, dstH int) ([]byte, error) {
	format := &Format{"", &Format_Uncompressed{f}}
	data, err := Convert(data, srcW, srcH, format, RGBA_F32)
	if err != nil {
		return nil, err
	}
	data, err = resizeRGBA_F32(data, srcW, srcH, dstW, dstH)
	if err != nil {
		return nil, err
	}
	return Convert(data, dstW, dstH, RGBA_F32, RGBA_F32)
}

func (f *FmtUncompressed) convert(data []byte, width, height int, dstFmt *Format) ([]byte, error) {
	if err := f.check(data, width, height); err != nil {
		return nil, err
	}
	switch dstFmt := protoutil.OneOf(dstFmt.Format).(type) {
	case *FmtUncompressed:
		return stream.Convert(dstFmt.Format, f.Format, data)
	default:
		return nil, nil
	}
}
