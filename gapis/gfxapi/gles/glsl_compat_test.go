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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

const (
	OpenGL_3_0 = "3.0"
)

type glslCompatTest struct {
	name   string
	target string
	lang   ast.Language
	source string
	compat string
}

var glslCompatTests = []glslCompatTest{
	{
		name:   "BumpVersion",
		target: OpenGL_3_0,
		source: `#version 110
uniform mat4 blah[3];
void main() {
    gl_Position = vec4(0., 0., 0., 1.);
}
`,
		compat: `#version 130
uniform mat4 blah[3];
void main() {
    gl_Position = vec4(0., 0., 0., 1.);
}
`,
	}, {
		name:   "StripFunction",
		target: OpenGL_3_0,
		source: `#version 110
highp int f(lowp int b);
`,
		compat: `#version 130
int f(int b);
`,
	}, {
		name:   "StripVarDecl",
		target: OpenGL_3_0,
		source: `#version 110
highp int a;
`,
		compat: `#version 130
int a;
`,
	}, {
		name:   "StripConversion",
		target: OpenGL_3_0,
		source: `#version 110
int a;
int b = lowp int(a);
`,
		compat: `#version 130
int a;
int b = int(a);
`,
	}, {
		name:   "StripElse",
		target: OpenGL_3_0,
		source: `#version 110
void f(bool a) {
    if(a)
        a = true;
    else precision highp int;
}
`,
		compat: `#version 130
void f(bool a) {
    if(a)
        a = true;
    else ;
}
`,
	}, {
		name:   "StripWhile",
		target: OpenGL_3_0,
		source: `#version 110
void f(bool a) {
    while(a) precision highp int;
}
`,
		compat: `#version 130
void f(bool a) {
    while(a) ;
}
`,
	}, {
		name:   "StripDo",
		target: OpenGL_3_0,
		source: `#version 110
void f(bool a) {
    do precision highp int;
    while(a);
}
`,
		compat: `#version 130
void f(bool a) {
    do ;
    while(a);
}
`,
	}, {
		name:   "StripFor",
		target: OpenGL_3_0,
		source: `#version 110
void f(bool a) {
    for(int b = 0; b < 1;  ++b) precision highp int;
}
`,
		compat: `#version 130
void f(bool a) {
    for(int b = 0; b < 1;  ++b) ;
}
`,
	}, {
		name:   "StripTop",
		target: OpenGL_3_0,
		source: `#version 110
precision highp int;`,
		compat: `#version 130
`,
	}, {
		name:   "StripNested",
		target: OpenGL_3_0,
		source: `#version 110
void main() {
    precision highp int;
}
`,
		compat: `#version 130
void main() {
}
`,
	}, {
		name:   "StripIf",
		target: OpenGL_3_0,
		source: `#version 110
void f(bool a) {
    if(a) precision highp int;
}
`,
		compat: `#version 130
void f(bool a) {
    if(a) ;
}
`,
	}, {
		name:   "AttributeToIn",
		target: OpenGL_3_0,
		source: `#version 110
attribute vec4 attr;
`,
		compat: `#version 130
in vec4 attr;
`,
	}, {
		name:   "Declare FragColor",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
void f() {
    gl_FragColor = vec4(1.);
}
`,
		compat: `#version 130
out vec4 FragColor;
void f() {
    FragColor = vec4(vec4(1.));
}
`,
	}, {
		name:   "Declare FragData",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
void f() {
    gl_FragData[1] = vec4(1.);
}
`,
		compat: `#version 130
layout(location = 1) out vec4 FragData1;
void f() {
    FragData1 = vec4(1.);
}
`,
	}, {
		name:   "VaryingToOut",
		target: OpenGL_3_0,
		lang:   ast.LangVertexShader,
		source: `#version 110
varying vec4 attr;
`,
		compat: `#version 130
out vec4 attr;
`,
	}, {
		name:   "VaryingToIn",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
varying vec4 attr;
`,
		compat: `#version 130
in vec4 attr;
`,
	}, {
		name:   "texture2D to texture",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
uniform sampler2D s;
vec4 f() {
    return texture2D(s, vec2(0.));
}
`,
		compat: `#version 130
uniform sampler2D s;
vec4 f() {
    return texture(s, vec2(0.));
}
`,
	}, {
		name:   "texture rename",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
vec4 f(sampler2D texture, vec2 uv) {
    return texture2D(texture, uv);
}
vec4 g(sampler2D texture, vec2 uv) {}
`,
		compat: `#version 130
vec4 f(sampler2D texture_renamed_, vec2 uv) {
    return texture(texture_renamed_, uv);
}
vec4 g(sampler2D texture_renamed_, vec2 uv) {
}
`,
	}, {
		name:   "samplerExternalOES",
		target: OpenGL_3_0,
		lang:   ast.LangFragmentShader,
		source: `#version 110
#extension GL_OES_EGL_image_external : require
varying mediump vec2 v_texCoord;
uniform mediump samplerExternalOES u_texture;
void main() {
  gl_FragColor = texture2D(u_texture, v_texCoord);
}
`,
		compat: `#version 130
out vec4 FragColor;
in vec2 v_texCoord;
uniform sampler2D u_texture;
void main() {
    FragColor = vec4(texture(u_texture, v_texCoord));
}
`,
	}, {
		name:   "Multiview",
		target: OpenGL_3_0,
		source: `#version 110
layout(num_views = 2) in;
void main() {
	  uint id = gl_ViewID_OVR;
    gl_Position = vec4(0., 0., 0., 1.);
}
`,
		compat: `#version 130
uniform uint gapid_gl_ViewID_OVR;

void main() {
    uint id = gapid_gl_ViewID_OVR;
    gl_Position = vec4(0., 0., 0., 1.);
}
`,
	},
}

func TestGLSLCompat(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	for _, test := range glslCompatTests {
		dev := &device.Instance{Configuration: &device.Configuration{
			Drivers: &device.Drivers{
				OpenGL: &device.OpenGLDriver{Version: test.target},
			},
		}}
		got, err := glslCompat(ctx, test.source, test.lang, nil, dev)
		if err != nil {
			t.Errorf("Test '%s' gave unexpected error: %v", test.name, err)
			continue
		}
		if expected := test.compat; got != expected {
			t.Errorf("glslCompat returned unexpected output for '%s'.\nGot:\n%s\nExpected:\n%s",
				test.name, got, expected)
		}
	}
}
