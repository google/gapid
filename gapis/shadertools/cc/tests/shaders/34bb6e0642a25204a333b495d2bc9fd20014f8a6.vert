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