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
// width, height and depth into another format. If the conversion succeeds then
// the converted image data is returned, otherwise an error is returned.
type Converter func(data []byte, width, height, depth int) ([]byte, error)

type srcDstFmt struct{ src, dst interface{} }

var registeredConverters = make(map[srcDstFmt]Converter)

// RegisterConverter registers the Converter for converting from src to dst
// formats. If a converter already exists for converting from src to dst, then
// this function panics.
func RegisterConverter(src, dst *Format, c Converter) {
	key := srcDstFmt{src.Key(), dst.Key()}
	if _, found := registeredConverters[key]; found {
		panic(fmt.Errorf("Converter from %s to %s already registered", src.Name, dst.Name))
	}
	registeredConverters[key] = c
}

func Registered(src, dst *Format) bool {
	key := srcDstFmt{src.Key(), dst.Key()}
	_, found := registeredConverters[key]
	return found
}

// converter is the interface implemented by formats that support format
// conversion.
type converter interface {
	// convert converts the image formed from data, width, height, depth to
	// dstFmt.
	// If converter is unable to convert to dstFmt then nil, nil is returned.
	convert(data []byte, width, height, depth int, dstFmt *Format) ([]byte, error)
}

// Convert uses the registered Converters to convert the image formed from
// data, width and height from srcFmt to dstFmt.
// If no direct converter has been registered to convert from srcFmt to dstFmt,
// then Convert may try converting via an intermediate format.
func Convert(data []byte, width, height, depth int, srcFmt, dstFmt *Format) ([]byte, error) {
	out, err := convertDirect(data, width, height, depth, srcFmt, dstFmt)
	if err != nil {
		return nil, err
	}
	if out != nil {
		return out, nil
	}

	// No direct conversion found. Try going via a common intermediate formats.
	for _, via := range []*Format{
		RGBA_U8_NORM, SRGBA_U8_NORM,
	} {
		if data, _ := convertDirect(data, width, height, depth, srcFmt, via); data != nil {
			if data, _ := convertDirect(data, width, height, depth, via, dstFmt); data != nil {
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("No converter registered that can convert from format '%s' to '%s'",
		srcFmt.Name, dstFmt.Name)
}

func convertDirect(data []byte, width, height, depth int, srcFmt, dstFmt *Format) ([]byte, error) {
	srcKey, dstKey := srcFmt.Key(), dstFmt.Key()
	if srcKey == dstKey {
		return data, nil // No conversion required.
	}

	if err := srcFmt.Check(data, width, height, depth); err != nil {
		return nil, fmt.Errorf("Source data of format %s is invalid: %s", srcFmt.Name, err)
	}

	// Look for a registered converter.
	if conv, found := registeredConverters[srcDstFmt{srcKey, dstKey}]; found {
		return conv(data, width, height, depth)
	}

	// Check if the source format supports the converter interface.
	if c, ok := protoutil.OneOf(srcFmt.Format).(converter); ok {
		return c.convert(data, width, height, depth, dstFmt)
	}

	return nil, nil
}

// Resolve returns the byte array holding the converted image for the resolve
// request.
// TODO: Can this be moved to the resolve package?
func (r *ConvertResolvable) Resolve(ctx context.Context) (interface{}, error) {
	boxedBytes, err := database.Resolve(ctx, r.Bytes.ID())
	if err != nil {
		return nil, err
	}
	bytes := boxedBytes.([]byte)
	from, to := r.FormatFrom, r.FormatTo

	sliceLength := from.Size(int(r.Width), int(r.Height), 1)
	sliceStride := int(r.SliceStrideFrom)
	if sliceStride == 0 {
		sliceStride = sliceLength
	}

	rowLength := from.Size(int(r.Width), 1, 1)
	rowStride := int(r.RowStrideFrom)
	if rowStride == 0 {
		rowStride = rowLength
	}

	// Remove any padding from the source image
	if sliceStride != sliceLength || rowStride != rowLength {
		src, dst := bytes, make([]byte, sliceLength*int(r.Depth))
		for z := 0; z < int(r.Depth); z++ {
			dst, src := dst[sliceLength*z:], src[sliceStride*z:]
			for y := 0; y < int(r.Height); y++ {
				copy(dst, src[:rowLength])
				dst, src = dst[rowLength:], src[rowStride:]
			}
		}
	}

	bytes, err = Convert(bytes, int(r.Width), int(r.Height), int(r.Depth), from, to)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
