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

#include "astc.h"
#include "third_party/astc-encoder/Source/astcenccli_internal.h"

static_assert(sizeof(astc_error) >= sizeof(astcenc_error),
              "astc_error should superset of astcenc_error");

struct compression_workload {
  astcenc_context* context;
  astcenc_image* image;
  astcenc_swizzle swizzle;
  uint8_t* data_out;
  size_t data_len;
  astcenc_error error;
};

astc_compressed_image create_astc_compressed_image(uint8_t* data,
                                                   uint32_t width,
                                                   uint32_t height,
                                                   uint32_t block_width,
                                                   uint32_t block_height) {
  uint32_t block_x = (width + block_width - 1) / block_width;
  uint32_t block_y = (height + block_height - 1) / block_height;

  astc_compressed_image image{};
  image.dim_x = width;
  image.dim_y = height;
  image.dim_z = 1;
  image.block_x = block_width;
  image.block_y = block_height;
  image.block_z = 1;
  image.data = data;
  image.data_len = block_x * block_y * 16;
  return image;
}

void write_image(uint8_t* buf, const astcenc_image* img) {
  // This is a custom implementation for "unorm8x4_array_from_astc_img()"
  // original implementation does allocation
  uint8_t*** data8 = static_cast<uint8_t***>(img->data);
  for (uint32_t y = 0; y < img->dim_y; y++) {
    const uint8_t* src = data8[0][y + img->dim_pad] + (4 * img->dim_pad);
    uint8_t* dst = buf + y * img->dim_x * 4;

    for (uint32_t x = 0; x < img->dim_x; x++) {
      dst[4 * x] = src[4 * x];
      dst[4 * x + 1] = src[4 * x + 1];
      dst[4 * x + 2] = src[4 * x + 2];
      dst[4 * x + 3] = src[4 * x + 3];
    }
  }
}

astcenc_image* read_image(const uint8_t* buf, uint32_t width, uint32_t height,
                          uint32_t padding) {
  return astc_img_from_unorm8x4_array(buf, width, height, padding, false);
}

void compression_workload_runner(int thread_count, int thread_id,
                                 void* payload) {
  compression_workload* work = static_cast<compression_workload*>(payload);
  astcenc_error error =
      astcenc_compress_image(work->context, *work->image, work->swizzle,
                             work->data_out, work->data_len, thread_id);

  if (error != ASTCENC_SUCCESS) {
    work->error = error;
  }
}

extern "C" astc_error compress_astc(uint8_t* input_image_raw,
                                    uint8_t* output_image_raw, uint32_t width,
                                    uint32_t height, uint32_t block_width,
                                    uint32_t block_height, uint32_t is_srgb) {
  astcenc_profile profile = is_srgb ? ASTCENC_PRF_LDR_SRGB : ASTCENC_PRF_LDR;
  astcenc_config config{};

  astcenc_error result = astcenc_config_init(profile, block_width, block_height,
                                             1, ASTCENC_PRE_FASTEST, 0, config);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  uint32_t thread_count = get_cpu_count();
  astcenc_context* codec_context;
  result = astcenc_context_alloc(config, thread_count, &codec_context);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astcenc_image* uncompressed_image =
      read_image(input_image_raw, width, height,
                 MAX(config.v_rgba_radius, config.a_scale_radius));

  astcenc_swizzle swz_encode{ASTCENC_SWZ_R, ASTCENC_SWZ_G, ASTCENC_SWZ_B,
                             ASTCENC_SWZ_A};
  uint32_t blocks_x =
      (uncompressed_image->dim_x + config.block_x - 1) / config.block_x;
  uint32_t blocks_y =
      (uncompressed_image->dim_y + config.block_y - 1) / config.block_y;
  size_t buffer_size = blocks_x * blocks_y * 16;

  compression_workload work;
  work.context = codec_context;
  work.image = uncompressed_image;
  work.swizzle = swz_encode;
  work.data_out = output_image_raw;
  work.data_len = buffer_size;
  work.error = ASTCENC_SUCCESS;

  launch_threads(thread_count, compression_workload_runner, &work);
  if (work.error != ASTCENC_SUCCESS) {
    free_image(uncompressed_image);
    astcenc_context_free(codec_context);
    return work.error;
  }

  free_image(uncompressed_image);
  astcenc_context_free(codec_context);

  return ASTCENC_SUCCESS;
}

extern "C" astc_error decompress_astc(uint8_t* input_image_raw,
                                      uint8_t* output_image_raw, uint32_t width,
                                      uint32_t height, uint32_t block_width,
                                      uint32_t block_height) {
  astcenc_config config{};
  astcenc_error result =
      astcenc_config_init(ASTCENC_PRF_LDR, block_width, block_height, 1,
                          ASTCENC_PRE_FASTEST, 0, config);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  uint32_t thread_count = get_cpu_count();
  astcenc_context* codec_context;
  result = astcenc_context_alloc(config, thread_count, &codec_context);
  if (result != ASTCENC_SUCCESS) {
    return result;
  }

  astc_compressed_image input_image = create_astc_compressed_image(
      input_image_raw, width, height, block_width, block_height);

  const uint32_t bitness = 8;
  astcenc_image* output_image = alloc_image(
      bitness, input_image.dim_x, input_image.dim_y, input_image.dim_z, 0);

  astcenc_swizzle swz_decode{ASTCENC_SWZ_R, ASTCENC_SWZ_G, ASTCENC_SWZ_B,
                             ASTCENC_SWZ_A};
  result =
      astcenc_decompress_image(codec_context, input_image.data,
                               input_image.data_len, *output_image, swz_decode);
  if (result != ASTCENC_SUCCESS) {
    free_image(output_image);
    astcenc_context_free(codec_context);
    return result;
  }

  write_image(output_image_raw, output_image);
  free_image(output_image);
  astcenc_context_free(codec_context);
  return ASTCENC_SUCCESS;
}

const char* get_error_string(astc_error error_code) {
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
