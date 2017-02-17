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

package f32

// V4D is a four element vector of float32.
// The elements are in the order X, Y, Z, W.
type Vec4 [4]float32

// SqrMagnitude returns the magnitude of the vector.
func (v Vec4) SqrMagnitude() float32 {
	return v[0]*v[0] + v[1]*v[1] + v[2]*v[2] + v[3]*v[3]
}

// Magnitude returns the magnitude of the vector.
func (v Vec4) Magnitude() float32 {
	return Sqrt(v.SqrMagnitude())
}

// Scale returns the element-wise scaling of v with s.
func (v Vec4) Scale(s float32) Vec4 {
	return Vec4{v[0] * s, v[1] * s, v[2] * s, v[3] * s}
}

// Normalize returns the normalized vector of v.
func (v Vec4) Normalize() Vec4 {
	return v.Scale(1.0 / v.Magnitude())
}

// XYZ returns a V3D formed from first three elements of v.
func (v Vec4) XYZ() Vec3 {
	return Vec3{v[0], v[1], v[2]}
}

// Add4D returns the element-wise addition of vector a and b.
func Add4D(a, b Vec4) Vec4 {
	return Vec4{a[0] + b[0], a[1] + b[1], a[2] + b[2], a[3] + b[3]}
}

// Sub4D returns the element-wise subtraction of vector b from a.
func Sub4D(a, b Vec4) Vec4 {
	return Vec4{a[0] - b[0], a[1] - b[1], a[2] - b[2], a[3] - b[3]}
}
