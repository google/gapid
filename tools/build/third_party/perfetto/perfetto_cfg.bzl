# Copyright (C) 2019 Google Inc.
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

load("@gapid//tools/build/rules:cc.bzl", "cc_stripped_binary")

_ALWAYS_OPTIMIZE_COPTS = [
  "-O2",
  "-DNDEBUG",
]
_STD_MACROS_COPTS = [
  "-D__STDC_FORMAT_MACROS",
]

def _always_optimize_cc_library(**kwargs):
  copts = kwargs.pop("copts", default = []) + _ALWAYS_OPTIMIZE_COPTS + _STD_MACROS_COPTS
  # Override the linkopts from Perfetto (see below to what we add). This is only needed
  # because Perfetto uses the newer OS config rules, which do not currently work with
  # bazel & Android.
  # TODO(pmuetschard): remove once bazel supports the new config rules for Android.
  kwargs.pop("linkopts", default = [])
  native.cc_library(
    copts = copts,
    linkopts = select({
      "@gapid//tools/build:linux": ["-ldl", "-lrt", "-lpthread"],
      "@gapid//tools/build:darwin": [],
      "@gapid//tools/build:windows": [],
      # Android.
      "//conditions:default": ["-ldl"],
    }),
    **kwargs
  )

def _always_optimize_cc_binary(**kwargs):
  copts = kwargs.pop("copts", default = []) + _ALWAYS_OPTIMIZE_COPTS
  # Remove the linkopts from Perfetto. This is only needed because Perfetto uses
  # the newer OS config rules, which do not currently work with  bazel & Android.
  # TODO(pmuetschard): remove once bazel supports the new config rules for Android.
  kwargs.pop("linkopts", default = [])
  visibility = kwargs.pop("visibility", default = ["//visibility:private"])
  cc_stripped_binary(
    copts = copts,
    visibility = visibility,
    **kwargs
  )

PERFETTO_CONFIG = struct(
  root = "//",
  deps = struct(
    build_config = ["@gapid//tools/build/third_party/perfetto:build_config"],
    demangle_wrapper = ["@perfetto//:src_trace_processor_demangle"],
    jsoncpp = [],
    linenoise = [],
    llvm_demangle = [],
    protobuf_descriptor_proto = ["@com_google_protobuf//:descriptor_proto"],
    protobuf_lite = ["@com_google_protobuf//:protobuf_lite"],
    protobuf_full = ["@com_google_protobuf//:protobuf"],
    protoc = ["@com_google_protobuf//:protoc"],
    protoc_lib = ["@com_google_protobuf//:protoc_lib"],
    sqlite = ["@sqlite//:sqlite"],
    sqlite_ext_percentile = ["@sqlite_src//:percentile_ext"],
    version_header = [],
    zlib = ["@net_zlib//:zlib"],
  ),
  public_visibility = [
      "//visibility:public",
  ],
  proto_library_visibility = "//visibility:public",
  go_proto_library_visibility = "//visibility:public",
  deps_copts = struct(
    zlib = [],
    jsoncpp = [],
    linenoise = [],
    sqlite = _ALWAYS_OPTIMIZE_COPTS,
    llvm_demangle = [],
  ),
  rule_overrides = struct(
    cc_library =_always_optimize_cc_library,
    cc_binary = _always_optimize_cc_binary,
  ),
)
