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

uniform ivec2 uSize;
uniform ivec2 uOffset;
uniform vec2 uPixelSize;
uniform vec2 uTextureSize;
uniform vec2 uTextureOffset;
uniform bool uFlipped;

void main(void) {
  vPos = uTextureOffset + uTextureSize * (aVertexPosition + 1.0) / 2.0;
  if (uFlipped) {
    vPos.y = 1 - vPos.y;
  }
  gl_Position = vec4((aVertexPosition * uSize + 2.0 * uOffset) * uPixelSize, 0.0, 1.0);
}

//! FRAGMENT
uniform sampler2D uTexture;
uniform vec4 uChannels;
uniform vec3 uColor;
uniform int uMode;

uniform vec2 uRange;

out vec4 fragColor;

const float kCheckerSize = 15.0;
const vec3 kCheckerDark = vec3(0.753);
const vec3 kCheckerLight = vec3(1);

void tonemap(inout vec4 color) {
  color.rgb = (color.rgb - uRange.x) / uRange.y;
}

void main(void) {
  vec4 color = vec4(0, 0, 0, 1);
  if (uMode == 0) {
    if (vPos.x < 0.0 || vPos.x > 1.0 || vPos.y < 0.0 || vPos.y > 1.0) {
      discard;
    }
    color = texture(uTexture, vPos);
    color.rgb *= uChannels.rgb;
    color.a = (uChannels.a < 0.5) ? 1.0 : color.a;
    tonemap(color);
  } else if (uMode == 1) {
    vec2 x = mod(gl_FragCoord.xy, 2 * kCheckerSize);
    color.rgb = ((x.x < kCheckerSize) != (x.y < kCheckerSize)) ? kCheckerDark : kCheckerLight;
  } else if (uMode == 2) {
    color.rgb = uColor;
  }

  fragColor = color;
}
