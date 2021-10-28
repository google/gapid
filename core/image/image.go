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
var _ = Convertable((*Info)(nil))

// ConvertTo implements the Convertable interface. It directly calls Convert.
func (i *Info) ConvertTo(ctx context.Context, f *Format) (interface{}, error) {
	return i.Convert(ctx, f)
}

// Convert returns this image Info converted to the format f.
func (i *Info) Convert(ctx context.Context, f *Format) (*Info, error) {
	id, err := database.Store(ctx, &ConvertResolvable{
		Bytes:      i.Bytes,
		Width:      i.Width,
		Height:     i.Height,
		Depth:      i.Depth,
		FormatFrom: i.Format,
		FormatTo:   f,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to convert ImageInfo to format %v: %v", f, err)
	}
	return &Info{
		Format:       f,
		Width:        i.Width,
		Height:       i.Height,
		Depth:        i.Depth,
		Bytes:        NewID(id),
		ComputedSize: uint32(f.Size(int(i.Width), int(i.Height), int(i.Depth))),
	}, nil
}

// Resize returns this image Info resized to the specified dimensions.
func (i *Info) Resize(ctx context.Context, w, h, d uint32) (*Info, error) {
	id, err := database.Store(ctx, &ResizeResolvable{
		Bytes:     i.Bytes,
		Format:    i.Format,
		SrcWidth:  i.Width,
		SrcHeight: i.Height,
		SrcDepth:  i.Depth,
		DstWidth:  w,
		DstHeight: h,
		DstDepth:  d,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to resize ImageInfo to %v x %v: %v", w, h, err)
	}
	return &Info{
		Format:       i.Format,
		Width:        w,
		Height:       h,
		Depth:        d,
		Bytes:        NewID(id),
		ComputedSize: uint32(i.Format.Size(int(w), int(h), int(d))),
	}, nil
}

// Data loads the image from the database and returns a image Data.
func (i *Info) Data(ctx context.Context) (*Data, error) {
	boxedBytes, err := database.Resolve(ctx, i.Bytes.ID())
	if err != nil {
		return nil, err
	}
	return &Data{
		Bytes:  boxedBytes.([]byte),
		Width:  i.Width,
		Height: i.Height,
		Depth:  i.Depth,
		Format: i.Format,
	}, nil
}

// Convert returns the Data converted to the format to.
func (b *Data) Convert(to *Format) (*Data, error) {
	bytes, err := Convert(b.Bytes, int(b.Width), int(b.Height), int(b.Depth), b.Format, to)
	if err != nil {
		return nil, err
	}
	return &Data{Bytes: bytes, Width: b.Width, Height: b.Height, Depth: b.Depth, Format: to}, nil
}

// NewInfo stores the image.Data's content to the database then uses the ID to
// create a new image.Info which describes this image.Data.
func (b *Data) NewInfo(ctx context.Context) (*Info, error) {
	id, err := database.Store(ctx, b.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to create imageInfo from imageData %v: %v", b, err)
	}
	return &Info{
		Format:       b.Format,
		Width:        b.Width,
		Height:       b.Height,
		Depth:        b.Depth,
		Bytes:        NewID(id),
		ComputedSize: uint32(len(b.Bytes)),
	}, nil
}

// Difference returns the normalized square error between the two images.
// A return value of 0 denotes identical images, a return value of 1 denotes
// a complete mismatch (black vs white).
// Only channels that are found in both in a and b are compared. However, if
// there are no common channels then an error is returned.
func Difference(a, b *Data) (float32, error) {
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
		return 1, fmt.Errorf("No common channels between %v and %v",
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

	p := endian.Reader(bytes.NewReader(a.Bytes), device.LittleEndian)
	q := endian.Reader(bytes.NewReader(b.Bytes), device.LittleEndian)
	sqrErr := float32(0)
	c := a.Width * a.Height * uint32(len(channels))
	for i := uint32(0); i < c; i++ {
		err := p.Float32() - q.Float32()
		sqrErr += err * err
	}
	return sqrErr / float32(c), nil
}
