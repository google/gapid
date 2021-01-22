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
	"bytes"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
)

func NewASTC(name string, blockWidth, blockHeight uint32, srgb bool) *Format {
	return &Format{Name: name, Format: &Format_Astc{&FmtASTC{BlockWidth: blockWidth, BlockHeight: blockHeight, Srgb: srgb}}}
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

// ASTCFrom reads a raw astc image(with header), extracts the header and creates an
// ASTC image format object.
func ASTCFrom(data []byte) (*Data, error) {
	r := endian.Reader(bytes.NewBuffer(data), device.LittleEndian)

	if got := r.Uint32(); got != 0x5ca1ab13 {
		return nil, fmt.Errorf("Invalid header. Got: %x", got)
	}

	blockWidth := uint32(r.Uint8())
	blockHeight := uint32(r.Uint8())
	blockDepth := uint32(r.Uint8())

	if blockDepth != 1 {
		return nil, fmt.Errorf("Got a block depth of %v. Only 2D textures are currently supported", blockDepth)
	}

	texelWidth := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())
	texelHeight := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())
	texelDepth := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())

	blocksX := (texelWidth + blockWidth - 1) / blockWidth
	blocksY := (texelHeight + blockHeight - 1) / blockHeight
	blocksZ := (texelDepth + blockDepth - 1) / blockDepth

	texelData := make([]byte, blocksX*blocksY*blocksZ*16)
	r.Data(texelData)

	if err := r.Error(); err != nil {
		return nil, err
	}

	return &Data{
		Format: NewASTC("astc", blockWidth, blockHeight, false),
		Width:  texelWidth,
		Height: texelHeight,
		Depth:  texelDepth,
		Bytes:  texelData,
	}, nil
}
