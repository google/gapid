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
	"context"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/database"
)

// Interface compliance check
var _ = Convertable((*Info2D)(nil))

// ConvertTo implements the Convertable interface. It directly calls Convert.
func (i *Info2D) ConvertTo(ctx context.Context, f *Format) (interface{}, error) {
	return i.Convert(ctx, f)
}

// Convert returns this image Info converted to the format f.
func (i *Info2D) Convert(ctx context.Context, f *Format) (*Info2D, error) {
	id, err := database.Store(ctx, &ConvertResolvable{
		Data:       i.Data,
		Width:      i.Width,
		Height:     i.Height,
		FormatFrom: i.Format,
		FormatTo:   f,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to convert ImageInfo to format %v: %v", f, err)
	}
	return &Info2D{
		Format: f,
		Width:  i.Width,
		Height: i.Height,
		Data:   NewID(id),
	}, nil
}

// Resize returns this image Info resized to the specified dimensions.
func (i *Info2D) Resize(ctx context.Context, w, h uint32) (*Info2D, error) {
	id, err := database.Store(ctx, &ResizeResolvable{
		Data:      i.Data,
		Format:    i.Format,
		SrcWidth:  i.Width,
		SrcHeight: i.Height,
		DstWidth:  w,
		DstHeight: h,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to resize ImageInfo to %v x %v: %v", w, h, err)
	}
	return &Info2D{
		Format: i.Format,
		Width:  w,
		Height: h,
		Data:   NewID(id),
	}, nil
}

// Convert returns the Image converted to the format to.
func (i *Image2D) Convert(to *Format) (*Image2D, error) {
	data, err := Convert(i.Data, int(i.Width), int(i.Height), i.Format, to)
	if err != nil {
		return nil, err
	}
	return &Image2D{Data: data, Width: i.Width, Height: i.Height, Format: to}, nil
}

// Difference returns the normalized square error between the two images.
// A return value of 0 denotes identical images, a return value of 1 denotes
// a complete mismatch (black vs white).
// Only channels that are found in both in a and b are compared. However, if
// there are no common channels then an error is returned.
func Difference(a, b *Image2D) (float32, error) {
	if a.Width != b.Width || a.Height != b.Height {
		return 1, fmt.Errorf("Image dimensions are not identical. %dx%d vs %dx%d",
			a.Width, a.Height, b.Width, b.Height)
	}

	// Get the union of the channels for a and b.
	aChannels, bChannels := a.Format.Channels(), b.Format.Channels()
	bChannelSet := make(map[stream.Channel]struct{}, len(aChannels))
	for _, c := range bChannels {
		bChannelSet[c] = struct{}{}
	}
	channels := map[stream.Channel]struct{}{}
	for _, c := range aChannels {
		if _, ok := bChannelSet[c]; ok {
			channels[c] = struct{}{}
		}
	}

	if len(channels) == 0 {
		return 1, fmt.Errorf("No common channels between %v and %v.",
			aChannels, bChannels)
	}

	// Create a new uncompressed format which holds all the channels found in
	// a and b of type F32.
	streamFmt := &stream.Format{}
	for c := range channels {
		component := &stream.Component{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  c,
		}
		streamFmt.Components = append(streamFmt.Components, component)
	}

	// Convert a and b to this new uncompressed format.
	uncompressed := newUncompressed(streamFmt)
	a, err := a.Convert(uncompressed)
	if err != nil {
		return 1, err
	}
	b, err = b.Convert(uncompressed)
	if err != nil {
		return 1, err
	}

	p := endian.Reader(bytes.NewReader(a.Data), device.LittleEndian)
	q := endian.Reader(bytes.NewReader(b.Data), device.LittleEndian)
	sqrErr := float32(0)
	c := a.Width * a.Height * uint32(len(channels))
	for i := uint32(0); i < c; i++ {
		err := p.Float32() - q.Float32()
		sqrErr += err * err
	}
	return sqrErr / float32(c), nil
}
