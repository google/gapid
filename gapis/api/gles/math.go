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

package gles

// EqualTo returns true if the rectangle has the given dimensions.
func (r Rect) EqualTo(x, y GLint, w, h GLsizei) bool {
	return r.X() == x && r.Y() == y && r.Width() == w && r.Height() == h
}

// IsUnit returns true if the bounding box has a min of [-1, -1, -1, -1]
// and max of [+1, +1, +1, +1].
func (r BoundingBox) IsUnit() bool {
	return r.Min().EqualTo(-1, -1, -1, -1) && r.Max().EqualTo(1, 1, 1, 1)
}

// EqualTo returns true if the color is equal to the given values.
func (c Color) EqualTo(r, g, b, a GLfloat) bool {
	return c.Red() == r && c.Green() == g && c.Blue() == b && c.Alpha() == a
}

// EqualTo returns true if the vector is equal to the given coordinates.
func (v Vec4f) EqualTo(x, y, z, w GLfloat) bool {
	return v.Get(0) == x && v.Get(1) == y && v.Get(2) == z && v.Get(3) == w
}

// EqualTo returns true if the vector is equal to the given coordinates.
func (v Vec4i) EqualTo(x, y, z, w GLint) bool {
	return v.Get(0) == x && v.Get(1) == y && v.Get(2) == z && v.Get(3) == w
}
