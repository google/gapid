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

// DataTypeSize returns the size in bytes of the the specified data type.
func DataTypeSize(t GLenum) int {
	switch t {
	case GLenum_GL_BYTE,
		GLenum_GL_UNSIGNED_BYTE:
		return 1
	case GLenum_GL_SHORT,
		GLenum_GL_UNSIGNED_SHORT,
		GLenum_GL_HALF_FLOAT_ARB,
		GLenum_GL_HALF_FLOAT_OES:
		return 2
	case GLenum_GL_FIXED,
		GLenum_GL_FLOAT,
		GLenum_GL_INT,
		GLenum_GL_UNSIGNED_INT,
		GLenum_GL_UNSIGNED_INT_2_10_10_10_REV:
		return 4
	default:
		panic(fmt.Errorf("Unknown data type %v", t))
	}
}
