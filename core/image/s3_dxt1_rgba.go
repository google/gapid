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
	S3_DXT1_RGBA = NewS3_DXT1_RGBA("S3_DXT1_RGBA")
)

// NewS3_DXT1_RGBA returns a format representing the S3_DXT1_RGBA block texture
// compression.
func NewS3_DXT1_RGBA(name string) *Format {
	return &Format{Name: name, Format: &Format_S3Dxt1Rgba{&FmtS3_DXT1_RGBA{}}}
}

func (f *FmtS3_DXT1_RGBA) key() interface{} {
	return "S3_DXT1_RGBA"
}
func (*FmtS3_DXT1_RGBA) size(w, h, d int) int {
	return d * (sint.Max(sint.AlignUp(w, 4), 4) * sint.Max(sint.AlignUp(h, 4), 4)) / 2
}
func (f *FmtS3_DXT1_RGBA) check(data []byte, w, h, d int) error {
	return checkSize(data, f, w, h, d)
}
func (*FmtS3_DXT1_RGBA) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}
