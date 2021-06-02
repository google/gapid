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

func NewETC2(name string, colorMode FmtETC2_ColorMode, alphaMode FmtETC2_AlphaMode) *Format {
	return &Format{
		Name: name,
		Format: &Format_Etc2{
			&FmtETC2{
				ColorMode: colorMode,
				AlphaMode: alphaMode,
			},
		},
	}
}

func (f *FmtETC2) key() interface{} {
	return f.String()
}

func (f *FmtETC2) size(w, h, d int) int {
	baseSize := d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4))

	// In ETC2 RGBA8 and RG11 compressed with 128 bit per 4x4 block.
	// All the others have 64 bit. Therefore half the size.
	if (f.ColorMode == FmtETC2_RGB || f.ColorMode == FmtETC2_SRGB) && f.AlphaMode == FmtETC2_ALPHA_8BIT {
		return baseSize
	}

	if f.ColorMode == FmtETC2_RG || f.ColorMode == FmtETC2_RG_SIGNED {
		return baseSize
	}

	return baseSize / 2
}

func (f *FmtETC2) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}

func (f *FmtETC2) channels() stream.Channels {
	if f.ColorMode == FmtETC2_R || f.ColorMode == FmtETC2_R_SIGNED {
		return stream.Channels{stream.Channel_Red}
	}

	if f.ColorMode == FmtETC2_RG || f.ColorMode == FmtETC2_RG_SIGNED {
		return stream.Channels{stream.Channel_Red, stream.Channel_Green}
	}

	if f.ColorMode == FmtETC2_RGB || f.ColorMode == FmtETC2_SRGB {
		if f.AlphaMode == FmtETC2_ALPHA_NONE {
			return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue}
		}

		return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
	}

	panic("Unknown ETC color format")
}
