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

#include <stdio.h>

#include "astc.h"

#include "third_party/astc-encoder/Source/astc_codec_internals.h"

// astc-encoder global variables... *sigh*
int alpha_force_use_of_hdr = 0;
int perform_srgb_transform = 0;
int rgb_force_use_of_hdr = 0;
int print_diagnostics = 0;

// Functions that are used in compilation units we depend on, but don't actually
// use.
int astc_codec_unlink(const char *filename) { return 0; }
void astc_codec_internal_error(const char *filename, int linenum) {
    printf("ASTC error: %s:%d\n", filename, linenum);
    exit(1);
}
astc_codec_image *load_ktx_uncompressed_image(const char *filename, int padding, int *result) { return 0; }
astc_codec_image *load_dds_uncompressed_image(const char *filename, int padding, int *result) { return 0; }
astc_codec_image *load_tga_image(const char *tga_filename, int padding, int *result) { return 0; }
astc_codec_image *load_image_with_stb(const char *filename, int padding, int *result) { return 0; }
int store_ktx_uncompressed_image(const astc_codec_image * img, const char *filename, int bitness) { return 0; }
int store_dds_uncompressed_image(const astc_codec_image * img, const char *filename, int bitness) { return 0; }
int store_tga_image(const astc_codec_image * img, const char *tga_filename, int bitness) { return 0; }

uint8_t float2byte(float f) {
    if (f > 1.0f) { return 255; }
    if (f < 0.0f) { return 0; }
    return (uint8_t)(f * 255.0f + 0.5f);
}

extern "C" void init_astc() {
    build_quantization_mode_table();
}

extern "C" void decompress_astc(
        uint8_t* in,
        uint8_t* out,
        uint32_t width,
        uint32_t height,
        uint32_t block_width,
        uint32_t block_height) {

    uint32_t blocks_x = (width + block_width - 1) / block_width;
    uint32_t blocks_y = (height + block_height - 1) / block_height;

    imageblock pb;
    for (uint32_t by = 0; by < blocks_y; by++) {
        for (uint32_t bx = 0; bx < blocks_x; bx++) {
            physical_compressed_block pcb = *(physical_compressed_block*) in;
            symbolic_compressed_block scb;
            physical_to_symbolic(block_width, block_height, 1, pcb, &scb);
            decompress_symbolic_block(DECODE_LDR, block_width, block_height, 1, 0, 0, 0, &scb, &pb);
            in += 16;

            const float* data = pb.orig_data;
            for (uint32_t dy = 0; dy < block_height; dy++) {
                uint32_t y = by*block_height + dy;
                for (uint32_t dx = 0; dx < block_width; dx++) {
                    uint32_t x = bx*block_width + dx;
                    if (x < width && y < height) {
                        uint8_t* pxl = &out[(width*y+x)*4];
                        pxl[0] = float2byte(data[0]);
                        pxl[1] = float2byte(data[1]);
                        pxl[2] = float2byte(data[2]);
                        pxl[3] = float2byte(data[3]);
                    }
                    data += 4;
                }
            }
        }
    }
}
