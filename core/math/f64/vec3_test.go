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

package f64_test

import (
	"math"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/math/f64"
)

func TestV3DSqrMagnitude(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f64.Vec3
		r float64
	}{
		{f64.Vec3{0, 0, 0}, 0},
		{f64.Vec3{1, 0, 0}, 1},
		{f64.Vec3{0, 2, 0}, 4},
		{f64.Vec3{0, 0, -3}, 9},
		{f64.Vec3{1, 1, 1}, 3},
	} {
		assert.For("%v.SqrMagnitude", test.v).That(test.v.SqrMagnitude()).Equals(test.r)
	}
}

func TestV3DMagnitude(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f64.Vec3
		r float64
	}{
		{f64.Vec3{0, 0, 0}, 0},
		{f64.Vec3{1, 0, 0}, 1},
		{f64.Vec3{0, 2, 0}, 2},
		{f64.Vec3{0, 0, -3}, 3},
		{f64.Vec3{1, 1, 1}, math.Sqrt(3)},
	} {
		assert.For("%v.Magnitude", test.v).That(test.v.Magnitude()).Equals(test.r)
	}
}

func TestV3DScale(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f64.Vec3
		s float64
		r f64.Vec3
	}{
		{f64.Vec3{1, 0, 0}, -1, f64.Vec3{-1, 0, 0}},
		{f64.Vec3{0, 2, 0}, -2, f64.Vec3{0, -4, 0}},
		{f64.Vec3{0, 0, 3}, -3, f64.Vec3{0, 0, -9}},
		{f64.Vec3{1, 1, 1}, 0, f64.Vec3{0, 0, 0}},
	} {
		assert.For("%v.Scale", test.v).That(test.v.Scale(test.s)).Equals(test.r)
	}
}

func TestV3DNormalize(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f64.Vec3
		r f64.Vec3
	}{
		{f64.Vec3{1, 0, 0}, f64.Vec3{1, 0, 0}},
		{f64.Vec3{0, -2, 0}, f64.Vec3{0, -1, 0}},
		{f64.Vec3{0, 0, 3}, f64.Vec3{0, 0, 1}},
		{f64.Vec3{1, 2, -2}, f64.Vec3{1. / 3, 2. / 3, -2. / 3}},
	} {
		assert.For("%v.Normalize", test.v).That(test.v.Normalize()).Equals(test.r)
	}
}

func TestV3DW(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f64.Vec3
		s float64
		r f64.Vec4
	}{
		{f64.Vec3{0, 0, 0}, -4, f64.Vec4{0, 0, 0, -4}},
		{f64.Vec3{1, 2, 3}, 4, f64.Vec4{1, 2, 3, 4}},
	} {
		assert.For("%v.W", test.v).That(test.v.W(test.s)).Equals(test.r)
	}
}

func TestAdd3D(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		a f64.Vec3
		b f64.Vec3
		r f64.Vec3
	}{
		{f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}},
		{f64.Vec3{1, 2, 3}, f64.Vec3{0, 0, 0}, f64.Vec3{1, 2, 3}},
		{f64.Vec3{0, 0, 0}, f64.Vec3{3, 2, 1}, f64.Vec3{3, 2, 1}},
		{f64.Vec3{1, 2, 3}, f64.Vec3{-1, -2, -3}, f64.Vec3{0, 0, 0}},
	} {
		assert.For("Add3D(%v, %v)", test.a, test.b).
			That(f64.Add3D(test.a, test.b)).Equals(test.r)
	}
}

func TestSub3D(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		a f64.Vec3
		b f64.Vec3
		r f64.Vec3
	}{
		{f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}},
		{f64.Vec3{1, 2, 3}, f64.Vec3{0, 0, 0}, f64.Vec3{1, 2, 3}},
		{f64.Vec3{0, 0, 0}, f64.Vec3{3, 2, 1}, f64.Vec3{-3, -2, -1}},
		{f64.Vec3{1, 2, 3}, f64.Vec3{-1, -2, -3}, f64.Vec3{2, 4, 6}},
	} {
		assert.For("Sub3D(%v, %v)", test.a, test.b).
			That(f64.Sub3D(test.a, test.b)).Equals(test.r)
	}
}

func TestCross3D(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		a f64.Vec3
		b f64.Vec3
		r f64.Vec3
	}{
		{f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}, f64.Vec3{0, 0, 0}},
		{f64.Vec3{1, 0, 0}, f64.Vec3{0, 4, 0}, f64.Vec3{0, 0, 4}},
		{f64.Vec3{0, 2, 0}, f64.Vec3{0, 0, 5}, f64.Vec3{10, 0, 0}},
		{f64.Vec3{0, 0, 3}, f64.Vec3{6, 0, 0}, f64.Vec3{0, 18, 0}},
	} {
		assert.For("Cross3D(%v, %v)", test.a, test.b).
			That(f64.Cross3D(test.a, test.b)).Equals(test.r)
	}
}
