#version 330

uniform sampler2D _MainTex;
in mediump vec2 xlv_TEXCOORD2;

mat3 theMatrix;
mat3 matrix2;

int i;


struct struct_type {
  float a;
  int b;
  bool arr[5];
} struct_val;



void main ()
{

  i = 0;
  float arr4[7];

  while (i < 7) {
    arr4[i] = 3;
  }
  arr4[6] = 4;

  float arr3[] = arr4;

  float arr2[6] = float[6](1, 2,3,  4, 5, 6);
  float arr[] = float[](1,2,3,4,5);
  i = arr.length();

  struct_val = struct_type(1.2, 3, bool[5](true, true, true, false, false));
  struct_val.a = 2.2;
  struct_val.arr[struct_val.b] = (struct_val.a > 2);


  theMatrix[1] = vec3(3.0, 3.0, 3.0); //Sets the second column to all 3.0s
  theMatrix[2][0] = 16.0;

  matrix2 = theMatrix;

  gl_FragDepth = 1.2;
}
