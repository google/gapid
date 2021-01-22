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

#ifndef ASTC_H_
#define ASTC_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef int astc_error;

astc_error compress_astc(uint8_t* in, uint8_t* out, uint32_t width,
                         uint32_t height, uint32_t block_width,
                         uint32_t block_height, uint32_t is_srgb);

astc_error decompress_astc(uint8_t* in, uint8_t* out, uint32_t width,
                           uint32_t height, uint32_t block_width,
                           uint32_t block_height);

const char* get_error_string(astc_error error_code);

#ifdef __cplusplus
}  // extern "C"
#endif

#endif
