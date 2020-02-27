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

precision lowp float;
attribute highp vec4 a_position;
attribute lowp vec4 a_color;
attribute mediump vec2 a_texCoord;
uniform highp mat4 u_modelViewProjectionMatrix;

uniform lowp vec4 u_materialAmbient;

varying lowp vec4 v_color;
varying mediump vec2 v_texCoord;

void main() {
	gl_Position = u_modelViewProjectionMatrix * a_position;
	v_color = a_color * vec4(u_materialAmbient.a, u_materialAmbient.a, u_materialAmbient.a, 1.0);
	v_texCoord = a_texCoord;
}
