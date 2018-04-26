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
    name = "spirv-include",
    hdrs = glob([
        "include/**/*.h",
        "include/**/*.hpp",
    ]),
    strip_include_prefix = "include/",
    visibility = ["//visibility:private"],
)

cc_library(
    name = "spirv-source",
    hdrs = glob([
        "source/**/*.h",
        "source/**/*.inc",
    ]),
    strip_include_prefix = "source/",
    visibility = ["//visibility:private"],
    deps = [
        "@gapid//tools/build/third_party:spirv-tools-generated",
        "@spirv_headers//:spirv-headers",
    ],
)

cc_library(
    name = "spirv-tools",
    srcs = glob([
        "*.h",
        "*.cpp",
        "*.hpp",
        "source/**/*.h",
        "source/**/*.cpp",
        "source/**/*.hpp",
    ]),
    hdrs = glob([
        "include/spirv-tools/*",
        "source/**/*.h",
    ]),
    include_prefix = "third_party/SPIRV-Tools/",
    visibility = ["//visibility:public"],
    deps = [
        ":spirv-include",
        ":spirv-source",
        "@spirv_headers//:spirv-headers",
    ],
)
