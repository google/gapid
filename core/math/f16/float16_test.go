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

package f16_test

import (
	"math"
	"testing"

	"github.com/google/gapid/core/math/f16"
)

var checks = []struct {
	f16 f16.Number
	f32 float32
}{
	{0x0000, 0.0},
	{0x3c00, 1.0},
	{0x4000, 2.0},
	{0x4200, 3.0},
	{0x4400, 4.0},
	{0x4500, 5.0},
	{0x3555, 0.333251953125},
	{0xbc00, -1.0},
	{0xc000, -2.0},
	{0xc200, -3.0},
	{0xc400, -4.0},
	{0xc500, -5.0},
	{0xb555, -0.333251953125},
	{0x0000, 0.0},
	{0x7a1a, 5e4},
	{0x068d, 1e-4},
	{0x0346, 4.995e-5},
	{0x0053, 4.95e-6},
	{0x0008, 4.77e-7},
}

func TestFloat16To32(t *testing.T) {
	for _, c := range checks {
		expected, got := float64(c.f32), float64(c.f16.Float32())
		esign, gsign := math.Signbit(expected), math.Signbit(got)
		expected, got = math.Abs(expected), math.Abs(got)
		if esign != gsign || got > expected*1.001 || got < expected*0.999 {
			t.Errorf("Expansion of float16(0x%04x) to float32 gave unexpected value.\n"+
				"Expected: %g, got: %g", c.f16, expected, got)
		}
	}
}

func TestFloat32To16(t *testing.T) {
	for _, c := range checks {
		expected, got := c.f16, f16.From(c.f32)
		if expected != got {
			t.Errorf("Encoding of float32 %v gave unexpected value. Expected: %04x, got: %04x",
				c.f32, expected, got)
		}
	}
}

func TestFloat16InfTo32(t *testing.T) {
	if v := f16.Inf(1).Float32(); !math.IsInf(float64(v), 1) {
		t.Errorf("Positive infinity did not expand to positive infinity, but %v.", v)
	}
	if v := f16.Inf(-1).Float32(); !math.IsInf(float64(v), -1) {
		t.Errorf("Negative infinity did not expand to negative infinity, but %v.", v)
	}
	if v := f16.NaN().Float32(); !math.IsNaN(float64(v)) {
		t.Errorf("NaN constant did not expand to NaN, but %v.", v)
	}
}

func TestFloat32InfTo16(t *testing.T) {
	if v := f16.From(float32(math.Inf(1))); !v.IsInf(1) {
		t.Errorf("Positive infinity did not encode to positive infinity, but %04x.", v)
	}
	if v := f16.From(float32(math.Inf(-1))); !v.IsInf(-1) {
		t.Errorf("Negative infinity did not encode to negative infinity, but %04x.", v)
	}
	if v := f16.From(float32(math.NaN())); !v.IsNaN() {
		t.Errorf("NaN did not encode to NaN, but %04x.", v)
	}
	if v := f16.From(float32(1e5)); !v.IsInf(1) {
		t.Errorf("1e5 did not encode to positive infinity, but %04x.", v)
	}
	if v := f16.From(5e-8); v != 0 {
		t.Errorf("5e-8 did not encode to zero, but %04x.", v)
	}
}
