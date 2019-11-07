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

#define MT_GL_ES_3
#define ANDROID

// Mipmapping stuff
#define texture2DWithBias( X, Y ) texture2D( X, Y )
#define ENABLE_SHADER_FEATURES

#ifndef BAKE_DECAL_BACKFACE_TEST
#define BAKE_DECAL_BACKFACE_TEST
#endif
#ifndef USE_STEP_FUNCTION
#define USE_STEP_FUNCTION
#endif
#ifndef USE_LIVERY_TEXTURE_RGB
#define USE_LIVERY_TEXTURE_RGB
#endif


#define MT_FRAGMENT_SHADER

#if defined(BAKE_COLOUR)

uniform lowp sampler2D s_Texture0; // base texture
uniform lowp sampler2D s_Texture1; // mask texture
uniform lowp sampler2D s_Texture2; // decal texture
uniform lowp vec4 u_LiveryColour; // custom colours

varying highp vec2 v_TexCoord0;

void main()
{
	lowp vec4 baseTexture = texture2D( s_Texture0, v_TexCoord0 );
	lowp vec4 maskTexture = texture2D( s_Texture1, v_TexCoord0 );
	lowp vec4 decalTexture = texture2D( s_Texture2, v_TexCoord0 );

	// mix the custom colour with the base texture using the mask
	lowp float mask = maskTexture.r;
	lowp vec3 bg = u_LiveryColour.rgb * mask + baseTexture.rgb * (1.0 - mask);

	// mix the premultipied decal layer with the background using Porter-Duff OVER and the mask (assuming dest alpha is 1)
	gl_FragColor = vec4( decalTexture.rgb * mask + bg.rgb * (1.0 - decalTexture.a * mask), 1.0 );
}

#endif

#if defined(BAKE_COPY)

uniform lowp sampler2D s_Texture0; // decal texture
varying highp vec2 v_TexCoord0;

void main()
{
	lowp vec4 decalTexture = texture2D( s_Texture0, v_TexCoord0 );
	gl_FragColor = decalTexture;
}

#endif

#if defined(BAKE_EDGES)

uniform lowp sampler2D s_Texture0; // decal texture
varying highp vec2 v_TexCoord0;
varying highp vec2 v_TexCoord1;
varying highp vec2 v_TexCoord2;
varying highp vec2 v_TexCoord3;

void sampleTexture(highp vec2 texCoord, inout highp vec4 colourSum, inout highp float weightSum)
{
	lowp vec4 texel = texture2D( s_Texture0, texCoord );
	colourSum += texel * texel.a;
	weightSum += texel.a;
}

void main()
{
	highp vec4 colourSum = vec4(0.0);
	highp float weightSum = 0.0;

	sampleTexture( v_TexCoord0, colourSum, weightSum);
	sampleTexture( v_TexCoord1, colourSum, weightSum);
	sampleTexture( v_TexCoord2, colourSum, weightSum);
	sampleTexture( v_TexCoord3, colourSum, weightSum);

	gl_FragColor = colourSum / weightSum;
}

#endif

#if defined (BAKE_SPEC)

uniform lowp sampler2D s_Texture0; // mask texture
uniform lowp sampler2D s_Texture1; // decal texture

varying highp vec2 v_TexCoord0;

void main()
{
	lowp vec4 maskTexture = texture2D( s_Texture0, v_TexCoord0 );
	lowp vec4 decalTexture = texture2D( s_Texture1, v_TexCoord0 );

	lowp float spec = maskTexture.r * (1.0 - decalTexture.a);
	lowp float decal = maskTexture.r * decalTexture.a;
	gl_FragColor = vec4(spec, decal, maskTexture.b, 1.0);
}

#endif

#if defined(BAKE_STENCIL)

void main()
{
	// just write geometry into stencil buffer
	gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
}

#endif

#if defined(BAKE_DECAL_NO_OCCLUSION) || defined(BAKE_DECAL_DEPTH_TEST) || defined(BAKE_DECAL_BACKFACE_TEST) || defined(USE_LIVERY_TEXTURE_RGB)

uniform lowp sampler2D s_Texture0; // decal texture
uniform highp sampler2D s_Texture1; // depth texture

varying highp vec2 v_TexCoordDecal;
varying highp vec4 v_Normal;
varying highp float v_Depth;

uniform lowp vec4 u_LiveryColour;

void main()
{
	lowp vec4 decalTexture = texture2D( s_Texture0, v_TexCoordDecal );

#if defined(BAKE_DECAL_DEPTH_TEST)
	highp vec4 depthTexture = texture2D( s_Texture1, v_TexCoordDecal );
	highp float d = v_Depth;
	highp float dt = (depthTexture.r + depthTexture.g + depthTexture.b + depthTexture.a) * 0.25;
#endif

#if defined(USE_STEP_FUNCTION)
	
	
#if defined(USE_LIVERY_TEXTURE_RGB)
	lowp float decalAlpha = decalTexture.a;
#else
	// premultiply colour and assume the blend mode is (1, 1-srcA)
	lowp float decalAlpha = decalTexture.r * u_LiveryColour.a;
#endif

#if defined(USE_FLAT_ALPHA)
	decalAlpha = step( 0.01, decalAlpha );
#endif
#if defined(BAKE_DECAL_DEPTH_TEST)
	decalAlpha *= step( d - 0.005, dt );			// d - 0.005 > dt ||
#endif
#if defined(BAKE_DECAL_DEPTH_TEST) || defined(BAKE_DECAL_BACKFACE_TEST)
	decalAlpha *= step( v_Normal.z, 0.0 );			// v_Normal.z > 0.0 ||
#endif
	decalAlpha *= step( 0.0, v_TexCoordDecal.s );	// v_TexCoordDecal.s < 0.0 ||
	decalAlpha *= step( v_TexCoordDecal.s, 1.0 );	// v_TexCoordDecal.s > 1.0 ||
	decalAlpha *= step( 0.0, v_TexCoordDecal.t );	// v_TexCoordDecal.t < 0.0 ||
	decalAlpha *= step( v_TexCoordDecal.t, 1.0 );	// v_TexCoordDecal.t > 1.0)

#if defined(USE_LIVERY_TEXTURE_RGB)
	gl_FragColor = vec4( decalTexture.rgb, decalAlpha);
#else
	gl_FragColor = vec4( u_LiveryColour.rgb * decalAlpha, decalAlpha );
#endif

#else // USE_STEP_FUNCTION

	if (
#if defined(BAKE_DECAL_DEPTH_TEST)
		d - 0.005 > dt ||
#endif
#if defined(BAKE_DECAL_DEPTH_TEST) || defined(BAKE_DECAL_BACKFACE_TEST)
		v_Normal.z > 0.0 ||
#endif
		v_TexCoordDecal.s < 0.0 ||
		v_TexCoordDecal.s > 1.0 ||
		v_TexCoordDecal.t < 0.0 ||
		v_TexCoordDecal.t > 1.0)
	{
		gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
	}
	else
	{
		// premultiply colour and assume the blend mode is (1, 1-srcA)
#if defined(USE_LIVERY_TEXTURE_RGB)
		lowp float decalAlpha = decalTexture.a;
#else
		lowp float decalAlpha = decalTexture.r * u_LiveryColour.a;
#endif

#if defined(USE_FLAT_ALPHA)
		decalAlpha = ceil( decalAlpha - 0.01 );
#endif

#if defined(USE_LIVERY_TEXTURE_RGB)
		gl_FragColor = vec4( decalTexture.rgb, decalAlpha);
#else
		gl_FragColor = vec4( u_LiveryColour.rgb * decalAlpha, decalAlpha );
#endif
	}

#endif // USE_STEP_FUNCTION
}

#endif

#if defined(BAKE_DEPTH)

void main()
{
	// write depth to texture
	highp float z = gl_FragCoord.z;
	
	highp float r = clamp((z - 0.00) * 4.0, 0.0, 1.0);
	highp float g = clamp((z - 0.25) * 4.0, 0.0, 1.0);
	highp float b = clamp((z - 0.50) * 4.0, 0.0, 1.0);
	highp float a = clamp((z - 0.75) * 4.0, 0.0, 1.0);

	gl_FragColor = vec4(r, g, b, a);
}

#endif
