# GAPIC Shader Language Enhancements

To simplify the editing of shaders used in the Graphics API Client UI (GAPIC),
the source to both the vertex and fragment shaders of a program are stored in
the same file. Special "comment markers" are used to indicate which shader the
following lines of code belong to.

These program files are stored in the
[shader resource](https://github.com/google/agi/tree/master/gapic/res/shaders)
directory in files ending with the ```.glsl``` extension. The name of the file
is used as the identifier of the shader program.

## Comment Markers

The shader loader looks for the following special comment markers. The markers
have to be on their own line with nothing but optional whitespace preceeding or
following them, and are case sensitive.

 * ```//! COMMON``` - This marker indicates that the following lines will be
included in both shaders. This is the default if the program file does not start
with a marker.
 * ```//! VERTEX``` - The following lines are part of the vertex shader only.
 * ```//! FRAGMENT``` - The following lines are part of the fragment shader
only.

## Vertex Output / Fragment Input

To simplify the declarations of vertex shader output and fragment shader input,
and avoid spelling bugs, the ```varying``` keyword is reintroduced. Thus, these
variables can be declared only once in a common section. The shader loader will
automatically insert a ```#define varying out``` directive in the vertex shader
and ```#define varying in``` in the fragment shader.

## Version Directive

To simplify cross-platform OpenGL development, the shader loader will add the
correct ```#version``` directive to the shaders and it **must not** be part of
the source file. Shaders should be written against GLSL version 1.50, i.e.
OpenGL 3.2 Core.

## Example Shader

The following is an example shader program following this syntax.

```glsl
//! COMMON
varying vec3 vNormal;
varying vec2 vUV;

//! VERTEX
in vec2 aVertexPosition;
in vec3 aVertexNormal;
in vec3 aVertexUV;

uniform mat4 uProjection;
uniform mat4 uModelView;

void main() {
  gl_Position = uProjection * uModelView *
      vec4(aVertexPosition, 1);
  vNormal = mat3(uModelView) * aVertexNormal;
  vUV = aVertexUV;
}

//! FRAGMENT
uniform sampler2D uTexture;
out vec4 fragColor;

void main(void) {
  // ...
  vec4 color = texture(uTexture, vUV);
  // ...
  fragColor = color;
}
```
