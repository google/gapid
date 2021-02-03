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
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/stream"
)

// NewETC2_RGB_U8_NORM returns a format representing the COMPRESSED_RGB8_ETC2
// block texture compression format.
func NewETC2_RGB_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbU8Norm{&FmtETC2_RGB_U8_NORM{}}}
}

// NewETC2_SRGB_U8_NORM returns a format representing the COMPRESSED_SRGB8_ETC2
// block texture compression format.
func NewETC2_SRGB_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbU8Norm{&FmtETC2_RGB_U8_NORM{Srgb: true}}}
}

func (f *FmtETC2_RGB_U8_NORM) key() interface{} {
	return "ETC2_RGB_U8_NORM"
}
func (*FmtETC2_RGB_U8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtETC2_RGB_U8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_RGB_U8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue}
}

// NewETC2_RGBA_U8_NORM returns a format representing the
// COMPRESSED_RGBA8_ETC2_EAC block texture compression format.
func NewETC2_RGBA_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbaU8Norm{&FmtETC2_RGBA_U8_NORM{}}}
}

// NewETC2_SRGBA_U8_NORM returns a format representing the
// COMPRESSED_SRGBA8_ETC2_EAC block texture compression format.
func NewETC2_SRGBA_U8_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbaU8Norm{&FmtETC2_RGBA_U8_NORM{Srgb: true}}}
}

func (f *FmtETC2_RGBA_U8_NORM) key() interface{} {
	return "ETC2_RGBA_U8_NORM"
}
func (*FmtETC2_RGBA_U8_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtETC2_RGBA_U8_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_RGBA_U8_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}

// NewETC2_RGBA_U8U8U8U1_NORM returns a format representing the
// COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2 block texture compression format.
func NewETC2_RGBA_U8U8U8U1_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbaU8U8U8U1Norm{&FmtETC2_RGBA_U8U8U8U1_NORM{}}}
}

// NewETC2_SRGBA_U8U8U8U1_NORM returns a format representing the
// COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2 block texture compression format.
func NewETC2_SRGBA_U8U8U8U1_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgbaU8U8U8U1Norm{&FmtETC2_RGBA_U8U8U8U1_NORM{Srgb: true}}}
}

func (f *FmtETC2_RGBA_U8U8U8U1_NORM) key() interface{} {
	return "ETC2_RGBA_U8U8U8U1_NORM"
}
func (*FmtETC2_RGBA_U8U8U8U1_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtETC2_RGBA_U8U8U8U1_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_RGBA_U8U8U8U1_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}

// NewETC2_R_U11_NORM returns a format representing the COMPRESSED_R11_EAC
// block texture compression format.
func NewETC2_R_U11_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RU11Norm{&FmtETC2_R_U11_NORM{}}}
}

func (f *FmtETC2_R_U11_NORM) key() interface{} {
	return "ETC2_R_U11_NORM"
}
func (*FmtETC2_R_U11_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtETC2_R_U11_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_R_U11_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red}
}

// NewETC2_RG_U11_NORM returns a format representing the COMPRESSED_RG11_EAC
// block texture compression format.
func NewETC2_RG_U11_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgU11Norm{&FmtETC2_RG_U11_NORM{}}}
}

func (f *FmtETC2_RG_U11_NORM) key() interface{} {
	return "ETC2_RG_U11_NORM"
}
func (*FmtETC2_RG_U11_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtETC2_RG_U11_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_RG_U11_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green}
}

// NewETC2_R_S11_NORM returns a format representing the
// COMPRESSED_SIGNED_R11_EAC block texture compression format.
func NewETC2_R_S11_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RS11Norm{&FmtETC2_R_S11_NORM{}}}
}

func (f *FmtETC2_R_S11_NORM) key() interface{} {
	return "ETC2_R_S11_NORM"
}
func (*FmtETC2_R_S11_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtETC2_R_S11_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_R_S11_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red}
}

// NewETC2_RG_S11_NORM returns a format representing the COMPRESSED_RG11_EAC
// block texture compression format.
func NewETC2_RG_S11_NORM(name string) *Format {
	return &Format{Name: name, Format: &Format_Etc2RgS11Norm{&FmtETC2_RG_S11_NORM{}}}
}

func (f *FmtETC2_RG_S11_NORM) key() interface{} {
	return "ETC2_RG_S11_NORM"
}
func (*FmtETC2_RG_S11_NORM) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))
}
func (f *FmtETC2_RG_S11_NORM) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtETC2_RG_S11_NORM) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green}
}
