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

package gles_test

import (
	"testing"

	"github.com/google/gapid/gapis/api/gles"
)

func TestStubShaderSource(t *testing.T) {
	for _, test := range []struct {
		name   string
		pi     *gles.ProgramInfo
		vs, fs string
	}{
		{
			name: "Simple",
			pi: &gles.ProgramInfo{
				ActiveUniforms: gles.NewUniformIndexːActiveUniformᵐ().Add(0, gles.ActiveUniform{
					Type: gles.GLenum_GL_FLOAT_VEC4,
					Name: "foo",
				}).Add(1, gles.ActiveUniform{
					Type: gles.GLenum_GL_SAMPLER_2D,
					Name: "bar",
				}),
			},
			vs: `#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
uniform vec4 foo;
void main() {
    float no_strip = 0.0;
    no_strip += foo.x;
    gl_Position = vec4(no_strip * 0.000001, 0., 0., 1.);
}`,
			fs: `#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
uniform sampler2D bar;
void main() {
    float no_strip = 0.0;
    no_strip += texture2D(bar, vec2(0.)).x;
    gl_FragColor = vec4(1., no_strip * 0.000001, 1., 1.);
}`,
		}, {
			name: "Array",
			pi: &gles.ProgramInfo{
				ActiveUniforms: gles.NewUniformIndexːActiveUniformᵐ().Add(0, gles.ActiveUniform{
					Type:      gles.GLenum_GL_FLOAT_VEC4,
					Name:      "foo",
					ArraySize: 3,
				}).Add(1, gles.ActiveUniform{
					Type:      gles.GLenum_GL_FLOAT_VEC4,
					Name:      "bar[0]",
					ArraySize: 3,
				}),
			},
			vs: `#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
uniform vec4 bar[3];
uniform vec4 foo[3];
void main() {
    float no_strip = 0.0;
    no_strip += bar[0].x;
    no_strip += bar[1].x;
    no_strip += bar[2].x;
    no_strip += foo[0].x;
    no_strip += foo[1].x;
    no_strip += foo[2].x;
    gl_Position = vec4(no_strip * 0.000001, 0., 0., 1.);
}`,
			fs: `#version 150

/////////////////////////////////////////////
// GAPID stub shader (no source available) //
/////////////////////////////////////////////

precision highp float;
void main() {
    float no_strip = 0.0;
    gl_FragColor = vec4(1., no_strip * 0.000001, 1., 1.);
}`,
		},
	} {
		vs, fs, err := gles.VisibleForTestingStubShaderSource(test.pi)
		if err != nil {
			t.Errorf("Testing '%s': stubShaderSource returned error: %v", test.name, err)
		}
		if vs != test.vs {
			t.Errorf("Testing '%s': Vertex shader was not as expected. Expected:\n%v\nGot:\n%s", test.name, test.vs, vs)
		}
		if fs != test.fs {
			t.Errorf("Testing '%s': Fragment shader was not as expected. Expected:\n%v\nGot:\n%s", test.name, test.fs, fs)
		}
	}
}
