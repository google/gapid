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

package image_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/os/device"
)

func TestRGBAF32Resize(t *testing.T) {
	_4x1 := []float32{
		/*
			RGBA
		*/
		1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1,
	}
	_4x4 := []float32{
		/*
			RRGG
			BBAA
		*/
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
	}
	_8x8 := []float32{
		/*
			RRRRGGGG
			RRRRGGGG
			BBBBAAAA
			BBBBAAAA
		*/
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
	}

	for _, test := range []struct {
		name     string
		src      []float32
		sW, sH   int
		dW, dH   int
		expected []float32
	}{
		{name: "8x8 -> 4x4", src: _8x8, sW: 8, sH: 8, dW: 4, dH: 4, expected: _4x4},
		{name: "8x8 -> 4x2", src: _8x8, sW: 8, sH: 8, dW: 4, dH: 2, expected: []float32{
			1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		}},
		{name: "8x8 -> 2x4", src: _8x8, sW: 8, sH: 8, dW: 2, dH: 4, expected: []float32{
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
		}},
		{name: "8x8 -> 1x1", src: _8x8, sW: 8, sH: 8, dW: 1, dH: 1, expected: []float32{
			0.25, 0.25, 0.25, 0.25,
		}},
		{name: "8x8 -> 2x2", src: _8x8, sW: 8, sH: 8, dW: 2, dH: 2, expected: []float32{
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
		}},
		{name: "8x8 -> 3x3", src: _8x8, sW: 8, sH: 8, dW: 3, dH: 3, expected: []float32{
			1.00, 0.00, 0.00, 0.00 /**/, 0.50, 0.50, 0.00, 0.00 /**/, 0.00, 1.00, 0.00, 0.00,
			0.50, 0.00, 0.50, 0.00 /**/, 0.25, 0.25, 0.25, 0.25 /**/, 0.00, 0.50, 0.00, 0.50,
			0.00, 0.00, 1.00, 0.00 /**/, 0.00, 0.00, 0.50, 0.50 /**/, 0.00, 0.00, 0.00, 1.00,
		}},

		{name: "4x1 -> 3x1", src: _4x1, sW: 4, sH: 1, dW: 3, dH: 1, expected: []float32{
			1, 0, 0, 0 /**/, 0, 0.5, 0.5, 0 /**/, 0, 0, 0, 1,
		}},
		{name: "4x4 -> 4x2", src: _4x4, sW: 4, sH: 4, dW: 4, dH: 2, expected: []float32{
			1, 0, 0, 0 /**/, 1, 0, 0, 0 /**/, 0, 1, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 1, 0 /**/, 0, 0, 0, 1 /**/, 0, 0, 0, 1,
		}},
		{name: "4x4 -> 2x4", src: _4x4, sW: 4, sH: 4, dW: 2, dH: 4, expected: []float32{
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
		}},
		{name: "4x4 -> 1x1", src: _4x4, sW: 4, sH: 4, dW: 1, dH: 1, expected: []float32{
			0.25, 0.25, 0.25, 0.25,
		}},
		{name: "4x4 -> 2x2", src: _4x4, sW: 4, sH: 4, dW: 2, dH: 2, expected: []float32{
			1, 0, 0, 0 /**/, 0, 1, 0, 0,
			0, 0, 1, 0 /**/, 0, 0, 0, 1,
		}},
		{name: "4x4 -> 3x3", src: _4x4, sW: 4, sH: 4, dW: 3, dH: 3, expected: []float32{
			1.00, 0.00, 0.00, 0.00 /**/, 0.50, 0.50, 0.00, 0.00 /**/, 0.00, 1.00, 0.00, 0.00,
			0.50, 0.00, 0.50, 0.00 /**/, 0.25, 0.25, 0.25, 0.25 /**/, 0.00, 0.50, 0.00, 0.50,
			0.00, 0.00, 1.00, 0.00 /**/, 0.00, 0.00, 0.50, 0.50 /**/, 0.00, 0.00, 0.00, 1.00,
		}},
	} {
		in := make([]byte, test.sW*test.sH*4*4)
		w := endian.Writer(bytes.NewBuffer(in[:0]), device.LittleEndian)
		for i, c := 0, test.sW*test.sH*4; i < c; i++ {
			w.Float32(test.src[i])
		}

		res, err := image.RGBA_F32.Resize(in, test.sW, test.sH, test.dW, test.dH)
		if err != nil {
			t.Errorf("RGBAF32 resize of %s failed with: %v", test.name, err)
		}

		r, i := endian.Reader(bytes.NewReader(res), device.LittleEndian), 0
		for y := 0; y < test.dH; y++ {
			for x := 0; x < test.dW; x++ {
				eR, gR := test.expected[i+0], r.Float32()
				eG, gG := test.expected[i+1], r.Float32()
				eB, gB := test.expected[i+2], r.Float32()
				eA, gA := test.expected[i+3], r.Float32()
				if eR != gR || eG != gG || eB != gB || eA != gA {
					t.Errorf("Resized of %s at (%d, %d) gave unexpected value. Expected: (%v,%v,%v,%v), Got: (%v,%v,%v,%v)",
						test.name, x, y, eR, eG, eB, eA, gR, gG, gB, gA)
				}
				i += 4
			}
		}
	}
}
