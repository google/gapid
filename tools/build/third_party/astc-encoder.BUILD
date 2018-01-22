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
        "Source/astc_block_sizes2.cpp",
        "Source/astc_codec_internals.h",
        "Source/astc_color_unquantize.cpp",
        "Source/astc_decompress_symbolic.cpp",
        "Source/astc_image_load_store.cpp",
        "Source/astc_integer_sequence.cpp",
        "Source/astc_partition_tables.cpp",
        "Source/astc_percentile_tables.cpp",
        "Source/astc_quantization.cpp",
        "Source/astc_symbolic_physical.cpp",
        "Source/astc_weight_quant_xfer_tables.cpp",
        "Source/mathlib.h",
        "Source/softfloat.cpp",
        "Source/softfloat.h",
        "Source/vectypes.h",
    ],
    hdrs = [
        "Source/astc_codec_internals.h",
        "Source/mathlib.h",
        "Source/vectypes.h",
    ],
    copts = ["-Wno-c++11-narrowing"],
    include_prefix = "third_party/astc-encoder",
    visibility = ["//visibility:public"],
)
