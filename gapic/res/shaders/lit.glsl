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
varying vec3 vViewPosition;

//! VERTEX
in vec3 aVertexPosition;
in vec3 aVertexNormal;

uniform mat4 uModelView;
uniform mat4 uModelViewProj;
uniform mat3 uNormalMatrix;

void main(void) {
  vNormal = uNormalMatrix * aVertexNormal;
  vec4 position = vec4(aVertexPosition, 1.0);
  vViewPosition = -(uModelView * position).xyz;
  gl_Position = uModelViewProj * position;
}

//! FRAGMENT
const int kNumLights = 4;
const float kGamma = 2.2;
const float kFresnelReflectance = 0.028;

uniform vec3 uLightDir[kNumLights];
uniform vec3 uLightColor[kNumLights];
uniform vec3 uLightSpecColor[kNumLights];
uniform float uLightSize[kNumLights];
uniform vec3 uDiffuseColor;
uniform vec3 uSpecularColor;
uniform float uRoughness;

out vec4 fragColor;

float beckmann(float nDotH, float m) {
  float t = nDotH * nDotH;
  float a = m * m;
  return exp((t - 1.0) / (a * t)) / (a * t * t);
}

float brdf(vec3 normal, vec3 toLight, float diffuse, vec3 toEye, float m) {
  float result = 0.0;
  if (diffuse > 0.0) {
    vec3 halfVec = normalize(toLight + toEye);
    float b = beckmann(dot(normal, halfVec), m);
    float fres = kFresnelReflectance + (1.0 - kFresnelReflectance) * pow(1.0 - dot(halfVec, toEye), 5.0);
    float specular = max(b * fres, 0.0);
    result = diffuse * specular;
  }
  return result;
}

void main(void) {
  vec3 normal = normalize(vNormal);
  vec3 toCamera = normalize(vViewPosition);

  float r = 0.2 * pow(smoothstep(0.7, 1.0, 1.0 - max(0.0, dot(normal, toCamera))), 1.8);
  vec3 color = vec3(0);
  for (int i = 0; i < kNumLights; i++) {
    float diffuse = max(dot(normal, uLightDir[i]), 0.0);
    color += diffuse * uLightColor[i] * uDiffuseColor;

    float adjustedRoughness = uRoughness + (1.0 - uRoughness) * uLightSize[i];
    float specular = brdf(normal, uLightDir[i], diffuse, toCamera, adjustedRoughness);
    color += specular * uLightSpecColor[i] * uSpecularColor;
    color += r * diffuse * uLightSpecColor[i];
  }

  fragColor = vec4(pow(color, vec3(1.0 / kGamma)), 1);
}
