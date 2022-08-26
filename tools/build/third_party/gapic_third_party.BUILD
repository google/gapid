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

# This is a templated BUILD file. See /tools/build/workspace_gapic.bzl.

package(default_visibility = ["//visibility:public"])

java_library(
    name = "grpc",
    exports = [
        "{{io_grpc_api}}",
        "{{io_grpc_context}}",
        "{{io_grpc_core}}",
        "{{io_grpc_okhttp}}",
        "{{io_grpc_protobuf_lite}}",
        "{{io_grpc_protobuf}}",
        "{{io_grpc_stub}}",
        "{{io_opencensus_api}}",
        "{{io_opencensus_contrib_grpc_metrics}}",
        "{{io_perfmark_api}}",
        "{{javax_annotation_api}}",
    ],
)

java_library(
    name = "okhttp",
    exports = [
        "{{com_squareup_okhttp}}",
        "{{com_squareup_okio}}",
    ],
)

java_library(
    name = "lwjgl",
    exports = [
        "{{org_lwjgl_core}}",
        "{{org_lwjgl_opengl}}",
    ],
    runtime_deps = select({
        "@gapid//tools/build:linux": [
            "{{org_lwjgl_core_natives_linux}}",
            "{{org_lwjgl_opengl_natives_linux}}",
        ],
        "@gapid//tools/build:windows": [
            "{{org_lwjgl_core_natives_windows}}",
            "{{org_lwjgl_opengl_natives_windows}}",
        ],
        "@gapid//tools/build:darwin": [
            "{{org_lwjgl_core_natives_macos}}",
            "{{org_lwjgl_opengl_natives_macos}}",
        ],
        "@gapid//tools/build:darwin_arm64": [
            "{{org_lwjgl_core_natives_macos}}",
            "{{org_lwjgl_opengl_natives_macos}}",
        ],
    }),
)

java_library(
    name = "guava",
    exports = [
        "{{com_google_guava-failureaccess}}",
        "{{com_google_guava}}",
    ],
)

alias(
    name = "jface",
    actual = "{{jface}}",
)

alias(
    name = "swt",
    actual = "{{swt}}",
)
