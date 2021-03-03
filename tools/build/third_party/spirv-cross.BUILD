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
    name = "spirv-cross",
    srcs = glob([
        "*.h",
        "*.cpp",
        "*.hpp",
    ]),
    hdrs = [
        "spirv_glsl.hpp",
        "spirv_hlsl.hpp",
        "spirv_parser.hpp",
    ],
    include_prefix = "third_party/SPIRV-Cross",
    local_defines = [
        "SPIRV_CROSS_C_API_GLSL",
        "SPIRV_CROSS_C_API_HLSL",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "@spirv_headers//:spirv_c_headers",
        "@spirv_tools",
    ],
)
