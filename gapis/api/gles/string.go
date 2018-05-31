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

import "fmt"

func (c Color) String() string {
	return fmt.Sprintf("R:% 6f, G:% 6f, B:% 6f, A:% 6f", c.Red(), c.Green(), c.Blue(), c.Alpha())
}

func (v Vec2f) String() string {
	return fmt.Sprintf("(% 6f, % 6f)", v.Get(0), v.Get(1))
}

func (v Vec3f) String() string {
	return fmt.Sprintf("(% 6f, % 6f, % 6f)", v.Get(0), v.Get(1), v.Get(2))
}

func (v Vec4f) String() string {
	return fmt.Sprintf("(% 6f, % 6f, % 6f, % 6f)", v.Get(0), v.Get(1), v.Get(2), v.Get(3))
}

func (v Vec2i) String() string {
	return fmt.Sprintf("(% 6d, % 6d)", v.Get(0), v.Get(1))
}

func (v Vec3i) String() string {
	return fmt.Sprintf("(% 6d, % 6d, % 6d)", v.Get(0), v.Get(1), v.Get(2))
}

func (v Vec4i) String() string {
	return fmt.Sprintf("(% 6d, % 6d, % 6d, % 6d)", v.Get(0), v.Get(1), v.Get(2), v.Get(3))
}

func (m Mat2f) String() string {
	return fmt.Sprintf(".Get(%), %v]", m.Get(0), m.Get(1))
}

func (m Mat3f) String() string {
	return fmt.Sprintf(".Get(%), %v, %v]", m.Get(0), m.Get(1), m.Get(2))
}

func (m Mat4f) String() string {
	return fmt.Sprintf(".Get(%), %v, %v, %v]", m.Get(0), m.Get(1), m.Get(2), m.Get(3))
}

func (a VertexAttributeArray) String() string {
	if a.Enabled() == GLboolean_GL_TRUE {
		return fmt.Sprintf("%d x %v", int(a.Size()), a.Type())
	}
	return "disabled"
}
