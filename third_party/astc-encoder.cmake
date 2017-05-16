# Copyright (C) 2017 Google Inc.
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

set(sources
    "third_party/astc-encoder/Source/astc_block_sizes2.cpp"
    "third_party/astc-encoder/Source/astc_color_unquantize.cpp"
    "third_party/astc-encoder/Source/astc_decompress_symbolic.cpp"
    "third_party/astc-encoder/Source/astc_image_load_store.cpp"
    "third_party/astc-encoder/Source/astc_integer_sequence.cpp"
    "third_party/astc-encoder/Source/astc_partition_tables.cpp"
    "third_party/astc-encoder/Source/astc_percentile_tables.cpp"
    "third_party/astc-encoder/Source/astc_quantization.cpp"
    "third_party/astc-encoder/Source/astc_symbolic_physical.cpp"
    "third_party/astc-encoder/Source/astc_weight_quant_xfer_tables.cpp"
    "third_party/astc-encoder/Source/softfloat.cpp"
)

if(NOT DISABLED_CXX)
    add_library(astc-encoder ${sources})

    if(WIN32)
        target_compile_options(astc-encoder PRIVATE "-Wno-narrowing"
            "-D__NO_INLINE__"                     # Avoids a lvalue compile error
            "-D_GLIBCXX_INCLUDE_NEXT_C_HEADERS"   # Avoids redefinition compile error
        )
    else()
        target_compile_options(astc-encoder PRIVATE "-Wno-c++11-narrowing")
    endif()
    target_include_directories(astc-encoder PUBLIC "${astc-encoder_gen}")
endif()
