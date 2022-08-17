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

#include <string.h>

#include "astc.h"
#include "third_party/astc-encoder/Source/astcenc.h"

static_assert(sizeof(astc_error) >= sizeof(astcenc_error),
              "astc_error should superset of astcenc_error");

extern "C" astc_error compress_astc(uint8_t* input_image_raw,
                                    uint8_t* output_image_raw, uint32_t width,
                                    uint32_t height, uint32_t block_width,
                                    uint32_t block_height, uint32_t is_srgb) {
  astcenc_profile profile = is_srgb ? ASTCENC_PRF_LDR_SRGB : ASTCENC_PRF_LDR;
  astcenc_config config{};
  astcenc_error result = astcenc_config_init(
      profile, block_width, block_height, 1, ASTCENC_PRE_FASTEST, 0, &config);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astcenc_context* codec_context;
  result = astcenc_context_alloc(&config, 1, &codec_context);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astcenc_image uncompressed_image{width, height, 1, ASTCENC_TYPE_U8,
                                   reinterpret_cast<void**>(&input_image_raw)};
  astcenc_swizzle swz_encode{ASTCENC_SWZ_R, ASTCENC_SWZ_G, ASTCENC_SWZ_B,
                             ASTCENC_SWZ_A};
  result =
      astcenc_compress_image(codec_context, &uncompressed_image, &swz_encode,
                             output_image_raw, 4 * width * height, 0);
  astcenc_context_free(codec_context);
  return result;
}

extern "C" astc_error decompress_astc(uint8_t* input_image_raw,
                                      uint8_t* output_image_raw, uint32_t width,
                                      uint32_t height, uint32_t block_width,
                                      uint32_t block_height) {
  astcenc_config config{};
  astcenc_error result =
      astcenc_config_init(ASTCENC_PRF_LDR, block_width, block_height, 1,
                          ASTCENC_PRE_FASTEST, 0, &config);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astcenc_context* codec_context;
  result = astcenc_context_alloc(&config, 1, &codec_context);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astcenc_image output_image{width, height, 1, ASTCENC_TYPE_U8,
                             reinterpret_cast<void**>(&output_image_raw)};
  astcenc_swizzle swz_decode{ASTCENC_SWZ_R, ASTCENC_SWZ_G, ASTCENC_SWZ_B,
                             ASTCENC_SWZ_A};
  result = astcenc_decompress_image(codec_context, input_image_raw,
                                    4 * width * height, &output_image,
                                    &swz_decode, 0);
  astcenc_context_free(codec_context);
  return ASTCENC_SUCCESS;
}

extern "C" const char* get_astc_error_string(astc_error error_code) {
  switch (error_code) {
    case ASTCENC_ERR_BAD_BLOCK_SIZE:
      return "ERROR: Block size is invalid";
    case ASTCENC_ERR_BAD_CPU_ISA:
      return "ERROR: Required SIMD ISA support missing on this CPU";
    case ASTCENC_ERR_BAD_CPU_FLOAT:
      return "ERROR: astcenc must not be compiled with -ffast-math";
    default:
      return astcenc_get_error_string(static_cast<astcenc_error>(error_code));
  }
}
