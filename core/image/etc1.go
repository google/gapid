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

var (
	ETC1_RGB8 = NewETC1_RGB8("ETC1_RGB8")
)

// NewETC1_RGB8 returns a format representing the ETC1_RGB8 block texture
// compression format.
func NewETC1_RGB8(name string) *Format {
	return &Format{name, &Format_Etc1Rgb8{&FmtETC1_RGB8{}}}
}

func (f *FmtETC1_RGB8) key() interface{} {
	return *f
}
func (*FmtETC1_RGB8) size(w, h int) int {
	return (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (*FmtETC1_RGB8) check(d []byte, w, h int) error {
	return checkSize(d, sint.Max(sint.AlignUp(w, 4), 4), sint.Max(sint.AlignUp(h, 4), 4), 4)
}
func (*FmtETC1_RGB8) channels() []stream.Channel {
	return []stream.Channel{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue}
}

func init() {
	RegisterConverter(ETC1_RGB8, RGBA_U8_NORM, func(src []byte, width, height int) ([]byte, error) {
		return Convert(src, width, height, ETC2_RGB8, RGBA_U8_NORM)
	})
}
