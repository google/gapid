/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef LIBMANAGER_H_
#define LIBMANAGER_H_

#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

typedef struct instruction_t {
  uint32_t id;
  uint32_t opcode;
  uint32_t* words;
  uint32_t words_num;
  char* name;
} instruction_t;

typedef struct debug_instructions_t {
  instruction_t* insts;
  uint32_t insts_num;
} debug_instructions_t;

typedef enum shader_type_t {
  VERTEX,
  TESS_CONTROL,
  TESS_EVALUATION,
  GEOMETRY,
  FRAGMENT,
  COMPUTE
} shader_type;

typedef enum client_type_t {
  OPENGL,
  OPENGLES,
  VULKAN,
} client_type;

typedef struct compile_options_t {
  shader_type shader_type;
  client_type client_type;
  const char* preamble;
} compile_options_t;

typedef struct spirv_binary_t {
  uint32_t* words;
  size_t words_num;
} spirv_binary_t;

typedef struct glsl_compile_result_t {
  bool ok;
  char* message;
  spirv_binary_t binary;
} glsl_compile_result_t;

const char* getDisassembleText(uint32_t*, size_t);

void deleteDisassembleText(const char*);

spirv_binary_t* assembleToBinary(const char*);

void deleteBinary(spirv_binary_t*);

glsl_compile_result_t* compileGlsl(const char* code, const compile_options_t*);

void deleteCompileResult(glsl_compile_result_t*);

#ifdef __cplusplus
}
#endif

#endif  // LIBMANAGER_H_
