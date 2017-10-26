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

package stream

import (
	"bytes"
	"math"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

func convertSharedExponent(dst, src *Format, data []byte) ([]byte, error) {
	// This only covers the bias of GL_EXT_texture_shared_exponent. If we find
	// more shared-exponent formats, then we'll need to parameterize this.
	const exponentBias = 24

	// Create an intermediate format that expands the shared exponent to U32 bits
	// and all other components to F32.
	format := &Format{
		Components: []*Component{
			&Component{
				Channel:  Channel_SharedExponent,
				DataType: &U32,
				Sampling: Linear,
			},
		},
	}
	for _, c := range dst.Components {
		if c.Channel != Channel_SharedExponent && src.Channels().Contains(c.Channel) {
			format.Components = append(format.Components, &Component{
				Channel:  c.Channel,
				DataType: &F32,
				Sampling: Linear,
			})
		}
	}

	// Convert the data to this intermediate format.
	data, err := Convert(format, src, data)
	if err != nil {
		return nil, err
	}

	// All components are 4 bytes long.
	count := len(data) / (4 * len(format.Components))

	// In-place scale all non-exponent components by the exponent.
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	w := endian.Writer(bytes.NewBuffer(data[:0]), device.LittleEndian)
	for i := 0; i < count; i++ {
		exp := r.Uint32()
		scale := float32(math.Pow(2, float64(exp)-exponentBias))
		w.Uint32(0) // padding for exponent
		for c := 0; c < len(format.Components)-1; c++ {
			v := r.Float32()
			w.Float32(v * scale)
		}
	}

	// Replace the exponent component with padding, and convert to the target
	// format.
	format.Components[0].Channel = Channel_Undefined
	return Convert(dst, format, data)
}
