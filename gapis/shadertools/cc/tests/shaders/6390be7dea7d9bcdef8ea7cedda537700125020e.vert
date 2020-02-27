#version 300 es

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

// __multiversion__
#define ATLAS_TEXTURE
#define FANCY
#define LOW_PRECISION
#define SEASONS
#define TEXEL_AA
#define MAT4 lowp mat4
#define POS4 lowp vec4
#define POS3 lowp vec3
precision lowp float;
// This signals the loading code to prepend either #version 100 or #version 300 es as apropriate.

// To use centroid sampling we need to have version 300 es shaders, which requires changing:
// attribute to in
// varying to out when in vertex shaders or in when in fragment shaders
// defining an out vec4 FragColor and replacing uses of gl_FragColor with FragColor
// texture2D to texture
#if __VERSION__ >= 300
#define attribute in
#define varying out

#ifdef MSAA_FRAMEBUFFER_ENABLED
#define _centroid centroid
#else
#define _centroid
#endif

_centroid out vec2 uv0;
_centroid out vec2 uv1;

#else

varying vec2 uv0;
varying vec2 uv1;

#endif


varying vec4 color;
#ifdef FOG
	varying vec4 fogColor;
#endif

#ifdef NEAR_WATER
	varying float cameraDist;
#endif

#ifdef AS_ENTITY_RENDERER
uniform MAT4 WORLDVIEWPROJ;
#else
uniform MAT4 WORLDVIEW;
uniform MAT4 PROJ;
#endif
uniform vec4 FOG_COLOR;
uniform vec2 FOG_CONTROL;
uniform float RENDER_DISTANCE;
uniform vec2 VIEWPORT_DIMENSION;
uniform vec4 CURRENT_COLOR;		//current color r contains the offset to apply to the fog for the "fade in"
uniform POS4 CHUNK_ORIGIN_AND_SCALE;
uniform POS3 VIEW_POS;
uniform float FAR_CHUNKS_DISTANCE;

attribute POS4 POSITION;
attribute vec4 COLOR;
attribute vec2 TEXCOORD_0;
attribute vec2 TEXCOORD_1;

const float rA = 1.0;
const float rB = 1.0;
const vec3 UNIT_Y = vec3(0,1,0);
const float DIST_DESATURATION = 56.0 / 255.0; //WARNING this value is also hardcoded in the water color, don'tchange

void main()
{
    POS4 worldPos;
#ifdef AS_ENTITY_RENDERER
		POS4 pos = WORLDVIEWPROJ * POSITION;
		worldPos = pos;
#else
    worldPos.xyz = (POSITION.xyz * CHUNK_ORIGIN_AND_SCALE.w) + CHUNK_ORIGIN_AND_SCALE.xyz;
    worldPos.w = 1.0;

    // Transform to view space before projection instead of all at once to avoid floating point errors
    // Not required for entities because they are already offset by camera translation before rendering
    // World position here is calculated above and can get huge
    POS4 pos = WORLDVIEW * worldPos;
    pos = PROJ * pos;
#endif
    gl_Position = pos;

    uv0 = TEXCOORD_0;
    uv1 = TEXCOORD_1;
	color = COLOR;

///// find distance from the camera

#if defined(FOG) || defined(NEAR_WATER)
	#ifdef FANCY
		vec3 relPos = -worldPos.xyz;
		float cameraDepth = length(relPos);
		#ifdef NEAR_WATER
			cameraDist = cameraDepth / FAR_CHUNKS_DISTANCE;
		#endif
	#else
		float cameraDepth = pos.z;
		#ifdef NEAR_WATER
			vec3 relPos = -worldPos.xyz;
			float camDist = length(relPos);
			cameraDist = camDist / FAR_CHUNKS_DISTANCE;
		#endif
	#endif
#endif

///// apply fog

#ifdef FOG
	float len = cameraDepth / RENDER_DISTANCE;
	#ifdef ALLOW_FADE
		len += CURRENT_COLOR.r;
	#endif

    fogColor.rgb = FOG_COLOR.rgb;
	fogColor.a = clamp((len - FOG_CONTROL.x) / (FOG_CONTROL.y - FOG_CONTROL.x), 0.0, 1.0);
#endif

///// water magic
#ifdef NEAR_WATER
	#ifdef FANCY  /////enhance water
		float F = dot(normalize(relPos), UNIT_Y);
		F = 1.0 - max(F, 0.1);
		F = 1.0 - mix(F*F*F*F, 1.0, min(1.0, cameraDepth / FAR_CHUNKS_DISTANCE));

		color.rg -= vec2(F * DIST_DESATURATION);

		vec4 depthColor = vec4(color.rgb * 0.5, 1.0);
		vec4 traspColor = vec4(color.rgb * 0.45, 0.8);
		vec4 surfColor = vec4(color.rgb, 1.0);

		vec4 nearColor = mix(traspColor, depthColor, color.a);
		color = mix(surfColor, nearColor, F);
	#else
		// Completely insane, but if I don't have these two lines in here, the water doesn't render on a Nexus 6
		vec4 surfColor = vec4(color.rgb, 1.0);
		color = surfColor;
		color.a = pos.z / FAR_CHUNKS_DISTANCE + 0.5;
	#endif //FANCY
#endif
}
