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

load("@gapid//tools/build:rules.bzl", "cc_copts")

cc_library(
    name = "glslang",
    srcs = glob([
        "glslang/GenericCodeGen/*.h",
        "glslang/GenericCodeGen/*.cpp",
        "glslang/Include/*.h",
        "glslang/MachineIndependent/**/*.h",
        "glslang/MachineIndependent/**/*.cpp",
        "glslang/OSDependent/*.h",
        "glslang/OSDependent/*.cpp",
        "glslang/Public/*.h",
        "glslang/Public/*.cpp",
        "OGLCompilersDLL/*.h",
        "OGLCompilersDLL/*.cpp",
        "SPIRV/*.h",
        "SPIRV/*.cpp",
        "SPIRV/*.hpp",
    ]) + select({
        "@gapid//tools/build:windows": ["glslang/OSDependent/Windows/ossource.cpp"],
        "//conditions:default": ["glslang/OSDependent/Unix/ossource.cpp"],
    }),
    hdrs = glob([
        "glslang/Include/*.h",
        "glslang/Public/*.h",
        "glslang/MachineIndependent/*.h",
        "SPIRV/*.h",
    ]),
    copts = cc_copts() + [
        "-DNV_EXTENSIONS",
        "-Wno-unused-variable",
    ] + select({
        "@gapid//tools/build:linux": [
            "-Wno-error=class-memaccess",  # TODO(#3100): Remove this when glslang fixes the bug
            "-Wno-maybe-uninitialized",
        ],
        "//conditions:default": [],
    }),
    include_prefix = "third_party/glslang",
    linkopts = select({
        "@gapid//tools/build:windows": [],
        "//conditions:default": ["-lpthread"],
    }),
    visibility = ["//visibility:public"],
)
