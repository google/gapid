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
	"fmt"
	"math"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

func (m *mapping) transform(count int, f func(float64) float64) error {
	data := make([]byte, 8*count)
	tmp := mapping{
		src: m.src,
		dst: buf{
			bytes: data,
			component: &Component{
				DataType: &F64,
				Sampling: m.src.component.GetSampling(),
				Channel:  m.src.component.GetChannel(),
			},
			stride: 64,
		},
	}
	if err := tmp.conv(count, 0, 0); err != nil {
		return err
	}
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	w := endian.Writer(bytes.NewBuffer(data[:0]), device.LittleEndian)
	for i := 0; i < count; i++ {
		w.Float64(f(r.Float64()))
	}
	m.src = tmp.dst
	return nil
}

func (m *mapping) convertCurve(count int) error {
	src := m.src.component.GetSampling().GetCurve()
	dst := m.dst.component.GetSampling().GetCurve()

	switch {
	case src == dst:
		return nil

	case src == Curve_sRGB && dst == Curve_Linear:
		if err := m.transform(count, func(v float64) float64 {
			if v <= 0.04045 {
				return v / 12.92
			}
			return math.Pow((v+0.055)/1.055, 2.4)
		}); err != nil {
			return err
		}

		m.src = m.src.clone()
		m.src.component.Sampling.Curve = Curve_Linear
		return nil

	case src == Curve_Linear && dst == Curve_sRGB:
		if err := m.transform(count, func(v float64) float64 {
			if v <= 0.0031308 {
				return v * 12.92
			}
			return 1.055*math.Pow(v, 1.0/2.4) - 0.055
		}); err != nil {
			return err
		}

		m.src = m.src.clone()
		m.src.component.Sampling.Curve = Curve_sRGB
		return nil
	}

	return fmt.Errorf("Cannot convert curve from %v to %v", src, dst)
}
