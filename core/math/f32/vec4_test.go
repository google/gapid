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

package f32_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/math/f32"
)

func TestV4DSqrMagnitude(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f32.Vec4
		r float32
	}{
		{f32.Vec4{0, 0, 0, 0}, 0},
		{f32.Vec4{1, 0, 0, 0}, 1},
		{f32.Vec4{0, 2, 0, 0}, 4},
		{f32.Vec4{0, 0, -3, 0}, 9},
		{f32.Vec4{0, 0, 0, -4}, 16},
		{f32.Vec4{1, 1, 1, 1}, 4},
	} {
		assert.For("%v.SqrMagnitude", test.v).That(test.v.SqrMagnitude()).Equals(test.r)
	}
}

func TestV4DMagnitude(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f32.Vec4
		r float32
	}{
		{f32.Vec4{0, 0, 0, 0}, 0},
		{f32.Vec4{1, 0, 0, 0}, 1},
		{f32.Vec4{0, 2, 0, 0}, 2},
		{f32.Vec4{0, 0, -3, 0}, 3},
		{f32.Vec4{0, 0, 0, -4}, 4},
		{f32.Vec4{1, 1, 1, 1}, 2},
	} {
		assert.For("%v.Magnitude", test.v).That(test.v.Magnitude()).Equals(test.r)
	}
}

func TestV4DScale(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f32.Vec4
		s float32
		r f32.Vec4
	}{
		{f32.Vec4{1, 0, 0, 0}, -1, f32.Vec4{-1, 0, 0, 0}},
		{f32.Vec4{0, 2, 0, 0}, -2, f32.Vec4{0, -4, 0, 0}},
		{f32.Vec4{0, 0, 3, 0}, -3, f32.Vec4{0, 0, -9, 0}},
		{f32.Vec4{0, 0, 0, 4}, -4, f32.Vec4{0, 0, 0, -16}},
		{f32.Vec4{1, 1, 1, 1}, 0, f32.Vec4{0, 0, 0, 0}},
	} {
		assert.For("%v.Scale", test.v).That(test.v.Scale(test.s)).Equals(test.r)
	}
}

func TestV4DNormalize(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f32.Vec4
		r f32.Vec4
	}{
		{f32.Vec4{1, 0, 0, 0}, f32.Vec4{1, 0, 0, 0}},
		{f32.Vec4{0, -2, 0, 0}, f32.Vec4{0, -1, 0, 0}},
		{f32.Vec4{0, 0, 3, 0}, f32.Vec4{0, 0, 1, 0}},
		{f32.Vec4{0, 0, 0, -4}, f32.Vec4{0, 0, 0, -1}},
		{f32.Vec4{1, 2, -2, 4}, f32.Vec4{1. / 5, 2. / 5, -2. / 5, 4. / 5}},
	} {
		assert.For("%v.Normalize", test.v).That(test.v.Normalize()).Equals(test.r)
	}
}

func TestV4DXYZ(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		v f32.Vec4
		r f32.Vec3
	}{
		{f32.Vec4{0, 0, 0, 0}, f32.Vec3{0, 0, 0}},
		{f32.Vec4{1, 2, 3, 4}, f32.Vec3{1, 2, 3}},
	} {
		assert.For("%v.V3D", test.v).That(test.v.XYZ()).Equals(test.r)
	}
}

func TestAdd4D(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		a f32.Vec4
		b f32.Vec4
		r f32.Vec4
	}{
		{f32.Vec4{0, 0, 0, 0}, f32.Vec4{0, 0, 0, 0}, f32.Vec4{0, 0, 0, 0}},
		{f32.Vec4{1, 2, 3, 4}, f32.Vec4{0, 0, 0, 0}, f32.Vec4{1, 2, 3, 4}},
		{f32.Vec4{0, 0, 0, 0}, f32.Vec4{4, 3, 2, 1}, f32.Vec4{4, 3, 2, 1}},
		{f32.Vec4{1, 2, 3, 4}, f32.Vec4{-1, -2, -3, -4}, f32.Vec4{0, 0, 0, 0}},
	} {
		assert.For("Add4D(%v, %v)", test.a, test.b).
			That(f32.Add4D(test.a, test.b)).Equals(test.r)
	}
}

func TestSub4D(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		a f32.Vec4
		b f32.Vec4
		r f32.Vec4
	}{
		{f32.Vec4{0, 0, 0, 0}, f32.Vec4{0, 0, 0, 0}, f32.Vec4{0, 0, 0, 0}},
		{f32.Vec4{1, 2, 3, 4}, f32.Vec4{0, 0, 0, 0}, f32.Vec4{1, 2, 3, 4}},
		{f32.Vec4{0, 0, 0, 0}, f32.Vec4{4, 3, 2, 1}, f32.Vec4{-4, -3, -2, -1}},
		{f32.Vec4{1, 2, 3, 4}, f32.Vec4{-1, -2, -3, -4}, f32.Vec4{2, 4, 6, 8}},
	} {
		assert.For("Sub4D(%v, %v)", test.a, test.b).
			That(f32.Sub4D(test.a, test.b)).Equals(test.r)
	}
}
