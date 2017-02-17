




precision highp float;
uniform lowp vec4 _WorldSpaceLightPos0;
uniform lowp vec4 _LightColor0;
uniform sampler2D _MainTex;
uniform lowp vec4 _Color;
uniform lowp vec4 _AddColor;
varying highp vec2 xlv_TEXCOORD0;
varying lowp vec3 xlv_TEXCOORD1;
varying lowp vec3 xlv_TEXCOORD2;
void main ()
{
  lowp vec4 c_1;
  lowp vec3 tmpvar_2;
  tmpvar_2 = (texture2D (_MainTex, xlv_TEXCOORD0).xyz * _Color.xyz);
  lowp vec4 c_3;
  lowp float tmpvar_4;
  tmpvar_4 = max (0.0, dot (xlv_TEXCOORD1, _WorldSpaceLightPos0.xyz));
  highp vec3 tmpvar_5;
  tmpvar_5 = (((tmpvar_2 * _LightColor0.xyz) * tmpvar_4) * 2.0);
  c_3.xyz = tmpvar_5;
  c_3.w = 0.0;
  c_1.w = c_3.w;
  c_1.xyz = (c_3.xyz + (tmpvar_2 * xlv_TEXCOORD2));
  c_1.xyz = (c_1.xyz + _AddColor.xyz);
  gl_FragData[0] = c_1;
}



