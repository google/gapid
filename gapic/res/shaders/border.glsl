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

//! VERTEX
in vec3 aVertexPosition;

uniform mat4 uTransform;
uniform vec2 uBorderWidth;

void main(void) {
  gl_Position = uTransform * vec4(aVertexPosition.xy, 0.0, 1.0);
  gl_Position.xy += aVertexPosition.xy * uBorderWidth * aVertexPosition.z;
  gl_Position.y *= -1.0; // Flip the y-axis so that our 'NDC' space matches SWT.
}

//! FRAGMENT
uniform vec4 uColor;

out vec4 fragColor;

void main(void) {
  fragColor = uColor;
}
