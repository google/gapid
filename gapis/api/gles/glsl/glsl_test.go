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

package glsl

import (
	"testing"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
)

func TestParseVersion(t *testing.T) {
	test := map[string]Version{
		"100": {1, 0, 0},
		"123": {1, 2, 3},
	}
	for str, expected := range test {
		got := ParseVersion(str)
		if got != expected {
			t.Errorf("Parsed version '%v' was not as expected. Expected %v, Got %v",
				str, expected, got)
		}
	}
}

func TestParseShader(t *testing.T) {
	// Shader source taken from MoreTeapots NDK sample.
	source := `//
// Copyright (C) 2015 The Android Open Source Project
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
//
#version 300 es
precision mediump float;

//
//Shader with phoneshading + geometry instancing support
//Parameters with %PARAM_NAME% will be replaced to actual parameter at compile time
//

const int NUM_OBJECTS = 512;
layout(location=0) in highp vec3    myVertex;
layout(location=1) in highp vec3    myNormal;

layout(std140) uniform ParamBlock {
    mat4      uPMatrix[NUM_OBJECTS];
    mat4      uMVMatrix[NUM_OBJECTS];
    vec3      vMaterialDiffuse[NUM_OBJECTS];
};

uniform highp vec3      vLight0;
uniform lowp vec3       vMaterialAmbient;
uniform lowp vec4       vMaterialSpecular;

out lowp    vec4    colorDiffuse;

out mediump vec3 position;
out mediump vec3 normal;

void main(void)
{
    highp vec4 p = vec4(myVertex,1);
    gl_Position = uPMatrix[gl_InstanceID] * p;

    highp vec3 worldNormal = vec3(mat3(uMVMatrix[gl_InstanceID][0].xyz,
            uMVMatrix[gl_InstanceID][1].xyz,
            uMVMatrix[gl_InstanceID][2].xyz) * myNormal);
    highp vec3 ecPosition = p.xyz;

    colorDiffuse = dot( worldNormal, normalize(-vLight0+ecPosition) ) * vec4(vMaterialDiffuse[gl_InstanceID], 1.f)  + vec4( vMaterialAmbient, 1 );

    normal = worldNormal;
    position = ecPosition;
}`
	_, _, _, errs := Parse(source, ast.LangVertexShader)
	for i, err := range errs {
		t.Errorf("Error %d: %v", i, err)
	}
}
