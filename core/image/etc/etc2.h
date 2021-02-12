// Copyright (C) 2021 Google Inc.
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

#ifndef ETC2_H_
#define ETC2_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

enum etc_format {
  // ETC2 Formats
  ETC2_RGB_U8_NORM,
  ETC2_RGBA_U8_NORM,
  ETC2_RGBA_U8U8U8U1_NORM,
  ETC2_SRGB_U8_NORM,
  ETC2_SRGBA_U8_NORM,
  ETC2_SRGBA_U8U8U8U1_NORM,

  // EAC Formats
  ETC2_R_U11_NORM,
  ETC2_RG_U11_NORM,
  ETC2_R_S11_NORM,
  ETC2_RG_S11_NORM,

  // ETC1 Format
  ETC1_RGB_U8_NORM,
};

typedef uint32_t etc_error;

etc_error compress_etc(const uint8_t* input_image, uint8_t* output_image,
                       uint32_t width, uint32_t height, enum etc_format format);

char* get_etc_error_string(etc_error error_code);

#ifdef __cplusplus
}  // extern "C"
#endif

#endif
