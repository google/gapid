# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

cc_library(
    name = "astc-encoder",
    srcs = [
        "Source/astcenc.h",
        "Source/astcenc_averages_and_directions.cpp",
        "Source/astcenc_block_sizes.cpp",
        "Source/astcenc_color_quantize.cpp",
        "Source/astcenc_color_unquantize.cpp",
        "Source/astcenc_compress_symbolic.cpp",
        "Source/astcenc_compute_variance.cpp",
        "Source/astcenc_decompress_symbolic.cpp",
        "Source/astcenc_diagnostic_trace.cpp",
        "Source/astcenc_diagnostic_trace.h",
        "Source/astcenc_entry.cpp",
        "Source/astcenc_find_best_partitioning.cpp",
        "Source/astcenc_ideal_endpoints_and_weights.cpp",
        "Source/astcenc_image.cpp",
        "Source/astcenc_integer_sequence.cpp",
        "Source/astcenc_internal.h",
        "Source/astcenc_mathlib.cpp",
        "Source/astcenc_mathlib.h",
        "Source/astcenc_mathlib_softfloat.cpp",
        "Source/astcenc_partition_tables.cpp",
        "Source/astcenc_percentile_tables.cpp",
        "Source/astcenc_pick_best_endpoint_format.cpp",
        "Source/astcenc_platform_isa_detection.cpp",
        "Source/astcenc_quantization.cpp",
        "Source/astcenc_symbolic_physical.cpp",
        "Source/astcenc_vecmathlib.h",
        "Source/astcenc_vecmathlib_avx2_8.h",
        "Source/astcenc_vecmathlib_common_4.h",
        "Source/astcenc_vecmathlib_neon_4.h",
        "Source/astcenc_vecmathlib_none_4.h",
        "Source/astcenc_vecmathlib_sse_4.h",
        "Source/astcenc_weight_align.cpp",
        "Source/astcenc_weight_quant_xfer_tables.cpp",
    ],
    hdrs = [
        "Source/astcenc.h",
        "Source/astcenc_mathlib.h",
        "Source/astcenc_vecmathlib.h",
        "Source/astcenc_vecmathlib_common_4.h",
        "Source/astcenc_vecmathlib_sse_4.h",
        "Source/astcenccli_internal.h",
    ],
    copts = [
        "-Wno-c++11-narrowing",
        "-DASTCENC_POPCNT=0",
        "-DASTCENC_AVX=0",
        "-DASTCENC_ISA_INVARIANCE=0",
        "-DASTCENC_VECALIGN=16",
    ] + select({
        # Melih TODO: We may need to update this for performance
        # in the future. Currently SIMD options for compilers
        # are a bit of issue.
        "@gapid//tools/build:darwin_arm64": ["-ffast-math"],
        "//conditions:default": [
            "-mfpmath=sse",
            "-msse2",
            "-DASTCENC_SSE=20",
        ],
    }),
    include_prefix = "third_party/astc-encoder",
    visibility = ["//visibility:public"],
)
