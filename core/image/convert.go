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
	"context"
	"fmt"

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/gapis/database"
)

// Converter is used to convert the the image formed from the parameters data,
// width and height into another format. If the conversion succeeds then the
// converted image data is returned, otherwise an error is returned.
type Converter func(data []byte, width int, height int) ([]byte, error)

type srcDstFmt struct{ src, dst interface{} }

var registeredConverters = make(map[srcDstFmt]Converter)

// RegisterConverter registers the Converter for converting from src to dst
// formats. If a converter already exists for converting from src to dst, then
// this function panics.
func RegisterConverter(src, dst *Format, c Converter) {
	key := srcDstFmt{src.Key(), dst.Key()}
	if _, found := registeredConverters[key]; found {
		panic(fmt.Errorf("Converter from %s to %s already registered", src, dst))
	}
	registeredConverters[key] = c
}

// converter is the interface implemented by formats that support format
// conversion.
type converter interface {
	// convert converts the image formed from data, width and height to dstFmt.
	// If converter is unable to convert to dstFmt then nil, nil is returned.
	convert(data []byte, width, height int, dstFmt *Format) ([]byte, error)
}

// Convert uses the registered Converters to convert the image formed from
// data, width and height from srcFmt to dstFmt.
// If no direct converter has been registered to convert from srcFmt to dstFmt,
// then Convert may try converting via an intermediate format.
func Convert(data []byte, width, height int, srcFmt, dstFmt *Format) ([]byte, error) {
	srcKey, dstKey := srcFmt.Key(), dstFmt.Key()
	if srcKey == dstKey {
		return data, nil // No conversion required.
	}

	if err := srcFmt.Check(data, width, height); err != nil {
		return nil, fmt.Errorf("Source data of format %s is invalid: %s", srcFmt, err)
	}

	// Look for a registered converter.
	if conv, found := registeredConverters[srcDstFmt{srcKey, dstKey}]; found {
		return conv(data, width, height)
	}

	// Check if the source format supports the converter interface.
	if c, ok := protoutil.OneOf(srcFmt.Format).(converter); ok {
		data, err := c.convert(data, width, height, dstFmt)
		if data != nil || err != nil {
			return data, err
		}
	}

	// No direct conversion found. Try going via RGBA_U8_NORM.
	rgbaU8Key := RGBA_U8_NORM.Key()
	if convA, found := registeredConverters[srcDstFmt{srcKey, rgbaU8Key}]; found {
		if convB, found := registeredConverters[srcDstFmt{rgbaU8Key, dstKey}]; found {
			if data, err := convA(data, width, height); err != nil {
				return convB(data, width, height)
			}
		}
	}

	return nil, fmt.Errorf("No converter registered that can convert from format '%s' to '%s'\n",
		srcFmt, dstFmt)
}

// Resolve returns the byte array holding the converted image for the resolve
// request.
// TODO: Can this be moved to the resolve package?
func (r *ConvertResolvable) Resolve(ctx context.Context) (interface{}, error) {
	data, err := database.Resolve(ctx, r.Data.ID())
	if err != nil {
		return nil, err
	}
	from, to := r.FormatFrom, r.FormatTo
	rowLength := from.Size(int(r.Width), 1)
	if r.StrideFrom != 0 && r.StrideFrom != uint32(rowLength) {
		// Remove any padding from the source image
		packed := make([]byte, from.Size(int(r.Width), int(r.Height)))
		src, dst := data.([]byte), packed
		for y := 0; y < int(r.Height); y++ {
			copy(dst, src[:rowLength])
			dst, src = dst[rowLength:], src[r.StrideFrom:]
		}
		data = packed
	}

	data, err = Convert(data.([]byte), int(r.Width), int(r.Height), from, to)
	if err != nil {
		return nil, err
	}

	return data, nil
}
