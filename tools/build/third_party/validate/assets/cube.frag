/*
 * Copyright (c) 2015-2016 The Khronos Group Inc.
 * Copyright (c) 2015-2016 Valve Corporation
 * Copyright (c) 2015-2016 LunarG, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/*
 * Fragment shader for cube demo
 */
#version 400
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_shading_language_420pack : enable
layout(std140, binding = 0) uniform buf {
        mat4 MVP;
        vec4 position[12*3];
        vec4 attr[12*3];
        highp int gpu_workload;
} ubuf;

layout (binding = 1) uniform sampler2D tex;

layout (location = 0) in vec4 texcoord;
layout (location = 1) in vec3 frag_pos;
layout (location = 0) out vec4 uFragColor;

const vec3 lightDir= vec3(0.424, 0.566, 0.707);

void main() {
   float tint = 0;

   for(int i = 0; i < ubuf.gpu_workload; ++i) {
     tint += 0.0001;
   }

   vec3 dX = dFdx(frag_pos);
   vec3 dY = dFdy(frag_pos);
   vec3 normal = normalize(cross(dX,dY));
   float light = max(0.0, dot(lightDir, normal));
   uFragColor = light * texture(tex, texcoord.xy);
   uFragColor.x += tint;
}
