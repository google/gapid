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

//! COMMON
varying vec3 vNormal;

//! VERTEX
in vec3 aVertexPosition;
in vec3 aVertexNormal;

uniform mat4 uModelViewProj;
uniform float uInvertNormals;

void main(void) {
  vNormal = uInvertNormals * aVertexNormal;
  gl_Position = uModelViewProj * vec4(aVertexPosition, 1.0);
}

//! FRAGMENT
out vec4 fragColor;

void main(void) {
  vec3 normal = normalize(vNormal);
  fragColor = vec4((normal + 1.0) / 2.0, 1);
}
