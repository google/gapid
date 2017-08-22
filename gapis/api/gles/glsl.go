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

import (
	"fmt"

	"github.com/google/gapid/gapis/shadertools"
)

// ShaderType returns the ShaderType for the given GLenum.
func (e GLenum) ShaderType() (shadertools.ShaderType, error) {
	switch e {
	case GLenum_GL_VERTEX_SHADER:
		return shadertools.TypeVertex, nil
	case GLenum_GL_TESS_CONTROL_SHADER:
		return shadertools.TypeTessControl, nil
	case GLenum_GL_TESS_EVALUATION_SHADER:
		return shadertools.TypeTessEvaluation, nil
	case GLenum_GL_GEOMETRY_SHADER:
		return shadertools.TypeGeometry, nil
	case GLenum_GL_FRAGMENT_SHADER:
		return shadertools.TypeFragment, nil
	case GLenum_GL_COMPUTE_SHADER:
		return shadertools.TypeCompute, nil
	}
	return 0, fmt.Errorf("%v is not a shader", e)
}
