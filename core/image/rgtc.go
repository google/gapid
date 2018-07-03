// Copyright (C) 2018 Google Inc.
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
	RGTC1_BC4_R_U8_NORM  = NewRGTC1_BC4_R_U8_NORM("RGTC1_BC4_R_U8_NORM")
	RGTC1_BC4_R_S8_NORM  = NewRGTC1_BC4_R_S8_NORM("RGTC1_BC4_R_S8_NORM")
	RGTC2_BC5_RG_U8_NORM = NewRGTC2_BC5_RG_U8_NORM("RGTC2_BC5_RG_U8_NORM")
	RGTC2_BC5_RG_S8_NORM = NewRGTC2_BC5_RG_S8_NORM("RGTC2_BC5_RG_S8_NORM")
)

func init() {
	RegisterConverter(RGTC1_BC4_R_U8_NORM, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeRGTC(r, dst, false, false)
		})
	})
	RegisterConverter(RGTC1_BC4_R_S8_NORM, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeRGTC(r, dst, true, false)
		})
	})
	RegisterConverter(RGTC2_BC5_RG_U8_NORM, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeRGTC(r, dst, false, true)
		})
	})
	RegisterConverter(RGTC2_BC5_RG_S8_NORM, RGBA_U8_NORM, func(src []byte, w, h, d int) ([]byte, error) {
		return decode4x4Blocks(src, w, h, d, func(r binary.Reader, dst []pixel) {
			decodeRGTC(r, dst, true, true)
		})
	})
}

// NewRGTC1_BC4_R_U8_NORM returns a format representing the RGTC1_BC4_R_U8_NORM
// block texture compression.
func NewRGTC1_BC4_R_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Rgtc1Bc4RU8Norm{&FmtRGTC1_BC4_R_U8_NORM{}}}
}

func (f *FmtRGTC1_BC4_R_U8_NORM) key() interface{} {
	return "RGTC1_BC4_U"
}
func (*FmtRGTC1_BC4_R_U8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtRGTC1_BC4_R_U8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtRGTC1_BC4_R_U8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red}
}

// NewRGTC1_BC4_R_S8_NORM returns a format representing the RGTC1_BC4_R_S8_NORM
// block texture compression.
func NewRGTC1_BC4_R_S8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Rgtc1Bc4RS8Norm{&FmtRGTC1_BC4_R_S8_NORM{}}}
}

func (f *FmtRGTC1_BC4_R_S8_NORM) key() interface{} {
	return "RGTC1_BC4_S"
}
func (*FmtRGTC1_BC4_R_S8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtRGTC1_BC4_R_S8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtRGTC1_BC4_R_S8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red}
}

// NewRGTC2_BC5_RG_U8_NORM returns a format representing the RGTC2_BC5_RG_U8_NORM
// block texture compression.
func NewRGTC2_BC5_RG_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Rgtc2Bc5RgU8Norm{&FmtRGTC2_BC5_RG_U8_NORM{}}}
}

func (f *FmtRGTC2_BC5_RG_U8_NORM) key() interface{} {
	return "RGTC2_BC5_U"
}
func (*FmtRGTC2_BC5_RG_U8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtRGTC2_BC5_RG_U8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtRGTC2_BC5_RG_U8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green}
}

// NewRGTC2_BC5_RG_S8_NORM returns a format representing the RGTC2_BC5_RG_S8_NORM
// block texture compression.
func NewRGTC2_BC5_RG_S8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Rgtc2Bc5RgS8Norm{&FmtRGTC2_BC5_RG_S8_NORM{}}}
}

func (f *FmtRGTC2_BC5_RG_S8_NORM) key() interface{} {
	return "RGTC2_BC5_S"
}
func (*FmtRGTC2_BC5_RG_S8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtRGTC2_BC5_RG_S8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtRGTC2_BC5_RG_S8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green}
}

func decodeRGTC(r binary.Reader, dst []pixel, signed bool, hasGreen bool) {
	r0, r1, rc := r.Uint8(), r.Uint8(), uint64(r.Uint16())|(uint64(r.Uint32())<<16)
	mixer := &rgtcMixer{
		signed: signed,
		r0:     r0,
		r1:     r1,
		rc:     rc,
	}
	if hasGreen {
		g0, g1, gc := r.Uint8(), r.Uint8(), uint64(r.Uint16())|(uint64(r.Uint32())<<16)
		mixer.hasGreen, mixer.g0, mixer.g1, mixer.gc = hasGreen, g0, g1, gc
	}
	for i := 0; i < 16; i++ {
		dst[i].r = mixer.red(i)
		dst[i].g = mixer.green(i)
		dst[i].b = 0
		dst[i].a = 255
	}
}

type rgtcMixer struct {
	signed, hasGreen bool
	r0, r1           uint8
	g0, g1           uint8
	rc, gc           uint64
}

func (m *rgtcMixer) red(pos int) int {
	return m.mix(m.r0, m.r1, m.rc, pos)
}

func (m *rgtcMixer) green(pos int) int {
	if !m.hasGreen {
		return 0
	}
	return m.mix(m.g0, m.g1, m.gc, pos)
}

func (m *rgtcMixer) mix(v0, v1 uint8, codes uint64, pos int) int {
	codes = codes >> uint(pos*3)
	if !m.signed {
		return mixAlphaDXT5(int(v0), int(v1), codes)
	}
	return mixRGTCSigned(int8(v0), int8(v1), codes)
}

func mixRGTCSigned(v0, v1 int8, code uint64) int {
	toFloat := func(i int8) float32 {
		if int(i) > -128 {
			return float32(int(i)) / 127.0
		}
		return -1.0
	}
	f0 := toFloat(v0)
	f1 := toFloat(v1)
	var f float32
	c := code & 0x7
	if f0 > f1 {
		switch c {
		case 0:
			f = f0
		case 1:
			f = f1
		default:
			f = (f0*float32(8-c) + f1*float32(c-1)) / 7.0
		}
	} else {
		switch c {
		case 0:
			f = f0
		case 1:
			f = f1
		case 6:
			f = -1.0
		case 7:
			f = 1.0
		default:
			f = (f0*float32(6-c) + f1*float32(c-1)) / 5.0
		}
	}
	return int((f + 1.0) * 255.0 / 2.0)
}
