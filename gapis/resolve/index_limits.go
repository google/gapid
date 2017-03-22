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

package resolve

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// MinMax holds a minimum and maximum value.
type MinMax struct {
	Min uint32 // Smallest index found in the index buffer.
	Max uint32 // Largest index found in the index buffer.
}

// IndexLimits returns the lowest and highest index contained in the index
// buffer with identifier id. The buffer holds count elements, each of size
// bytes.
func IndexLimits(ctx context.Context, data id.ID, count int, size int, littleEndian bool) (*MinMax, error) {
	obj, err := database.Build(ctx, &IndexLimitsResolvable{
		IndexSize:    uint64(size),
		Count:        uint64(count),
		LittleEndian: littleEndian,
		Data:         path.NewBlob(data),
	})
	if err != nil {
		return nil, err
	}
	return obj.(*MinMax), nil
}

// Resolve implements the database.Resolver interface.
func (c *IndexLimitsResolvable) Resolve(ctx context.Context) (interface{}, error) {
	min, max := ^uint32(0), uint32(0)
	data, err := database.Resolve(ctx, c.Data.Id.ID())
	if err != nil {
		return nil, err
	}
	byteOrder := device.LittleEndian
	if !c.LittleEndian {
		byteOrder = device.BigEndian
	}
	r := endian.Reader(bytes.NewReader(data.([]byte)), byteOrder)

	var decode func() uint32
	switch c.IndexSize {
	case 1:
		decode = func() uint32 { return uint32(r.Uint8()) }
	case 2:
		decode = func() uint32 { return uint32(r.Uint16()) }
	case 4:
		decode = r.Uint32
	default:
		return nil, fmt.Errorf("Unsupported index size %v", c.IndexSize)
	}

	for i := uint64(0); i < c.Count; i++ {
		v := decode()
		if r.Error() != nil {
			return nil, r.Error()
		}
		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}

	return &MinMax{Min: min, Max: max}, nil
}
