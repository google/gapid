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

// V3D is a three element vector of float32.
// The elements are in the order X, Y, Z.
type Vec3 [3]float32

// SqrMagnitude returns the magnitude of the vector.
func (v Vec3) SqrMagnitude() float32 {
	return v[0]*v[0] + v[1]*v[1] + v[2]*v[2]
}

// Magnitude returns the magnitude of the vector.
func (v Vec3) Magnitude() float32 {
	return Sqrt(v.SqrMagnitude())
}

// Scale returns the element-wise scaling of v with s.
func (v Vec3) Scale(s float32) Vec3 {
	return Vec3{v[0] * s, v[1] * s, v[2] * s}
}

// Normalize returns the normalized vector of v.
func (v Vec3) Normalize() Vec3 {
	return v.Scale(1.0 / v.Magnitude())
}

// W returns a V4D with the first three elements set to v and the fourth set
// to w.
func (v Vec3) W(w float32) Vec4 {
	return Vec4{v[0], v[1], v[2], w}
}

// Add3D returns the element-wise addition of vector a and b.
func Add3D(a, b Vec3) Vec3 {
	return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

// Sub3D returns the element-wise subtraction of vector b from a.
func Sub3D(a, b Vec3) Vec3 {
	return Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

// Cross3D returns the cross product of vector a and b.
func Cross3D(a, b Vec3) Vec3 {
	return Vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
