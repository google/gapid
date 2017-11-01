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

func NewASTC(name string, blockWidth, blockHeight uint32, srgb bool) *Format {
	return &Format{name, &Format_Astc{&FmtASTC{blockWidth, blockHeight, srgb}}}
}

func (f *FmtASTC) key() interface{} {
	return f.String()
}
func (f *FmtASTC) size(w, h, d int) int {
	bw, bh := int(f.BlockWidth), int(f.BlockHeight)
	return (16 * sint.AlignUp(w, bw) * sint.AlignUp(h, bh)) / (bw * bh)
}
func (f *FmtASTC) check(data []byte, w, h, d int) error {
	if actual, expected := len(data), f.size(w, h, d); expected != actual {
		return fmt.Errorf("Image data size (0x%x) did not match expected (0x%x) for dimensions %dx%dx%d",
			actual, expected, w, h, d)
	}
	return nil
}
func (*FmtASTC) channels() stream.Channels {
	return stream.Channels{stream.Channel_Red, stream.Channel_Green, stream.Channel_Blue, stream.Channel_Alpha}
}
