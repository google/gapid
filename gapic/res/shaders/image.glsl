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
varying vec2 vPos;

//! VERTEX
in vec2 aVertexPosition;

uniform mat4 uTransform;
uniform vec2 uTextureSize;
uniform vec2 uTextureOffset;
uniform bool uFlipped;

void main(void) {
  vPos = uTextureOffset + uTextureSize * (aVertexPosition + 1.0) / 2.0;
  if (uFlipped) {
    vPos.y = 1 - vPos.y;
  }
  gl_Position = uTransform * vec4(aVertexPosition, 0.0, 1.0);
  gl_Position.y *= -1.0; // Flip the y-axis so that our 'NDC' space matches SWT.
}

//! FRAGMENT
uniform sampler2D uTexture;
uniform vec4 uChannels;
uniform vec2 uRange;

out vec4 fragColor;

void tonemap(inout vec4 color) {
  color.rgb = (color.rgb - uRange.x) / uRange.y;
}

void main(void) {
  vec4 color = vec4(0, 0, 0, 1);
  color = texture(uTexture, vPos);
  color.rgb *= uChannels.rgb;
  color.a = (uChannels.a < 0.5) ? 1.0 : color.a;
  tonemap(color);

  fragColor = color;
}
