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
)

// ErrResizeUnsupported is returned by Format.Resize() when the format does
// not support resizing.
var ErrResizeUnsupported = fmt.Errorf("Format does not support resizing")

// format is the interface for an image and/or pixel format.
type format interface {
	// check returns an error if the combination of data, image width and image
	// height is invalid for the given format, otherwise check returns nil.
	check(data []byte, width, height int) error

	// size returns the number of bytes required to hold an image of the specified
	// dimensions in this format. If the size varies based on the image data, then
	// size returns -1.
	size(width, height int) int

	// key returns an object that can be used for equality-testing of the format
	// and can be used as a key in a map. Formats of the same type and parameters
	// will always return equal keys.
	// Formats can be deserialized into new objects so testing equality on the
	// Format object directly is not safe.
	key() interface{}

	// Channels returns the list of channels described by this format.
	// If the channels vary based on the image data, then channels returns nil.
	channels() []stream.Channel
}

// Interface compliance check.
var _ = []format{
	&FmtUncompressed{},
	&FmtPNG{},
	&FmtATC_RGB_AMD{},
	&FmtATC_RGBA_EXPLICIT_ALPHA_AMD{},
	&FmtATC_RGBA_INTERPOLATED_ALPHA_AMD{},
	&FmtETC1_RGB_U8_NORM{},
	&FmtETC2_RGB_U8_NORM{},
	&FmtETC2_RGBA_U8_NORM{},
	&FmtETC2_RGBA_U8U8U8U1_NORM{},
	&FmtETC2_R_U11_NORM{},
	&FmtETC2_RG_U11_NORM{},
	&FmtETC2_R_S11_NORM{},
	&FmtETC2_RG_S11_NORM{},
	&FmtS3_DXT1_RGB{},
	&FmtS3_DXT1_RGBA{},
	&FmtS3_DXT3_RGBA{},
	&FmtS3_DXT5_RGBA{},
	&FmtASTC{},
}

// Check returns an error if the combination of data, image width and image
// height is invalid for the given format, otherwise Check returns nil.
func (f *Format) Check(data []byte, width, height int) error {
	return f.format().check(data, width, height)
}

// Size returns the number of bytes required to hold an image of the specified
// dimensions in this format. If the size varies based on the image data, then
// Size returns -1.
func (f *Format) Size(width, height int) int {
	return f.format().size(width, height)
}

// Key returns an object that can be used for equality-testing of the format
// and can be used as a key in a map. Formats of the same type and parameters
// will always return equal keys.
// Formats can be deserialized into new objects so testing equality on the
// Format object directly is not safe.
func (f *Format) Key() interface{} {
	return f.format().key()
}

// Channels returns the list of channels described by this format.
// If the channels vary based on the image data, then Channels returns nil.
func (f *Format) Channels() []stream.Channel {
	return f.format().channels()
}

func (f *Format) format() format {
	return protoutil.OneOf(f.Format).(format)
}

// resizer is the interface implemented by formats that support resizing.
type resizer interface {
	// resize returns an image resized from srcW x srcH to dstW x dstH.
	// If the format does not support image resizing then the error
	// ErrResizeUnsupported is returned.
	resize(data []byte, srcW, srcH, dstW, dstH int) ([]byte, error)
}

// Resize returns an image resized from srcW x srcH to dstW x dstH.
// If the format does not support image resizing then the error
// ErrResizeUnsupported is returned.
func (f *Format) Resize(data []byte, srcW, srcH, dstW, dstH int) ([]byte, error) {
	if r, ok := protoutil.OneOf(f.Format).(resizer); ok {
		return r.resize(data, srcW, srcH, dstW, dstH)
	}
	return nil, ErrResizeUnsupported
}

func checkSize(data []byte, f format, width, height int) error {
	if expected, actual := f.size(width, height), len(data); expected != actual {
		return fmt.Errorf("Image data size (0x%x) did not match expected (0x%x) for dimensions %dx%d",
			actual, expected, width, height)
	}
	return nil
}
