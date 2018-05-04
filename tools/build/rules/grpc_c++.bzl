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

# Based off the original Bazel Skylark file which has the following license.

# Copyright 2016 gRPC authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# This is for the gRPC build system. This isn't intended to be used outsite of
# the BUILD file for gRPC. It contains the mapping for the template system we
# use to generate other platform's build system files.
#
# Please consider that there should be a high bar for additions and changes to
# this file.
# Each rule listed must be re-written for Google's internal build system, and
# each change must be ported from one to the other.
#

# The set of pollers to test against if a test exercises polling

PTHREAD_LINK_FLAG = select({
    "@gapid//tools/build:linux": ["-pthread"],
    "@gapid//tools/build:darwin": ["-pthread"],
    "@gapid//tools/build:windows": [],
    #Android
    "//conditions:default": [],
})

def _maybe_update_cc_library_hdrs(hdrs):
  ret = []
  hdrs_to_update = {
      "third_party/objective_c/Cronet/bidirectional_stream_c.h": "//third_party:objective_c/Cronet/bidirectional_stream_c.h",
  }
  for h in hdrs:
    if h in hdrs_to_update.keys():
      ret.append(hdrs_to_update[h])
    else:
      ret.append(h)
  return ret

# Building GRPC with Bazel on Windows does not support C-ARES.
# Ref: https://github.com/grpc/grpc/issues/13979
def _get_external_deps(external_deps):
  ret = []
  for dep in external_deps:
    if dep == "nanopb":
      ret += ["grpc_nanopb"]
    elif dep == "address_sorting":
      ret += ["//third_party/address_sorting"]
    elif dep == "cares":
      ret += select({"//:grpc_no_ares": [],
                     "@gapid//tools/build:windows": [],
                     "//conditions:default": ["//external:cares"],})
    else:
      ret += ["//external:" + dep]
  return ret

def grpc_cc_library(name, srcs = [], public_hdrs = [], hdrs = [],
                    external_deps = [], deps = [], standalone = False,
                    language = "C++", testonly = False, visibility = None,
                    alwayslink = 0):
  copts = ["-DGRPC_BAZEL_BUILD"]
  if language.upper() == "C":
    copts = copts + ["-std=c99"]
  native.cc_library(
    name = name,
    srcs = srcs,
    defines = select({"//:grpc_no_ares": ["GRPC_ARES=0"],
                      "@gapid//tools/build:windows": ["GRPC_ARES=0"],
                      "//conditions:default": [],}) +
              select({"//:remote_execution":  ["GRPC_PORT_ISOLATED_RUNTIME=1"],
                      "//conditions:default": [],}) +
              select({"//:grpc_allow_exceptions":  ["GRPC_ALLOW_EXCEPTIONS=1"],
                      "//:grpc_disallow_exceptions":
                      ["GRPC_ALLOW_EXCEPTIONS=0"],
                      "//conditions:default": [],}),
    hdrs = _maybe_update_cc_library_hdrs(hdrs + public_hdrs),
    deps = deps + _get_external_deps(external_deps),
    copts = copts,
    visibility = visibility,
    testonly = testonly,
    linkopts = PTHREAD_LINK_FLAG,
    includes = [
        "include"
    ],
    alwayslink = alwayslink,
  )

"""Load dependencies needed to compile and test the grpc library as a 3rd-party consumer."""

def grpc_deps():
    """Loads dependencies need to compile and test the grpc library."""
    native.bind(
        name = "libssl",
        actual = "@boringssl//:ssl",
    )

    native.bind(
        name = "zlib",
        actual = "@com_github_madler_zlib//:z",
    )

    native.bind(
        name = "protobuf",
        actual = "@com_google_protobuf//:protobuf",
    )

    native.bind(
        name = "protobuf_clib",
        actual = "@com_google_protobuf//:protoc_lib",
    )

    native.bind(
        name = "protobuf_headers",
        actual = "@com_google_protobuf//:protobuf_headers",
    )

    native.bind(
        name = "protocol_compiler",
        actual = "@com_google_protobuf//:protoc",
    )

    native.bind(
        name = "cares",
        actual = "@com_github_cares_cares//:ares",
    )

    native.bind(
        name = "gtest",
        actual = "@com_github_google_googletest//:gtest",
    )

    native.bind(
        name = "gmock",
        actual = "@com_github_google_googletest//:gmock",
    )

    native.bind(
        name = "benchmark",
        actual = "@com_github_google_benchmark//:benchmark",
    )

    native.bind(
        name = "gflags",
        actual = "@com_github_gflags_gflags//:gflags",
    )

    native.bind(
        name = "grpc_cpp_plugin",
        actual = "@com_github_grpc_grpc//:grpc_cpp_plugin"
    )

    native.bind(
        name = "grpc++_codegen_proto",
        actual = "@com_github_grpc_grpc//:grpc++_codegen_proto"
    )

    if "boringssl" not in native.existing_rules():
        native.http_archive(
            name = "boringssl",
            # on the master-with-bazel branch
            url = "https://boringssl.googlesource.com/boringssl/+archive/6ae5a54bedae2c29e5b67382667871c527e68326.tar.gz",
        )

    if "com_github_madler_zlib" not in native.existing_rules():
        native.new_http_archive(
            name = "com_github_madler_zlib",
            build_file = "@com_github_grpc_grpc//third_party:zlib.BUILD",
            strip_prefix = "zlib-cacf7f1d4e3d44d871b605da3b647f07d718623f",
            url = "https://github.com/madler/zlib/archive/cacf7f1d4e3d44d871b605da3b647f07d718623f.tar.gz",
        )

    if "com_google_protobuf" not in native.existing_rules():
        native.http_archive(
            name = "com_google_protobuf",
            strip_prefix = "protobuf-2761122b810fe8861004ae785cc3ab39f384d342",
            url = "https://github.com/google/protobuf/archive/2761122b810fe8861004ae785cc3ab39f384d342.tar.gz",
        )

    if "com_github_google_googletest" not in native.existing_rules():
        native.new_http_archive(
            name = "com_github_google_googletest",
            build_file = "@com_github_grpc_grpc//third_party:gtest.BUILD",
            strip_prefix = "googletest-ec44c6c1675c25b9827aacd08c02433cccde7780",
            url = "https://github.com/google/googletest/archive/ec44c6c1675c25b9827aacd08c02433cccde7780.tar.gz",
        )

    if "com_github_gflags_gflags" not in native.existing_rules():
        native.http_archive(
            name = "com_github_gflags_gflags",
            strip_prefix = "gflags-30dbc81fb5ffdc98ea9b14b1918bfe4e8779b26e",
            url = "https://github.com/gflags/gflags/archive/30dbc81fb5ffdc98ea9b14b1918bfe4e8779b26e.tar.gz",
        )

    if "com_github_google_benchmark" not in native.existing_rules():
        native.new_http_archive(
            name = "com_github_google_benchmark",
            build_file = "@com_github_grpc_grpc//third_party:benchmark.BUILD",
            strip_prefix = "benchmark-5b7683f49e1e9223cf9927b24f6fd3d6bd82e3f8",
            url = "https://github.com/google/benchmark/archive/5b7683f49e1e9223cf9927b24f6fd3d6bd82e3f8.tar.gz",
        )

    if "com_github_cares_cares" not in native.existing_rules():
        native.new_http_archive(
            name = "com_github_cares_cares",
            build_file = "@com_github_grpc_grpc//third_party:cares/cares.BUILD",
            strip_prefix = "c-ares-3be1924221e1326df520f8498d704a5c4c8d0cce",
            url = "https://github.com/c-ares/c-ares/archive/3be1924221e1326df520f8498d704a5c4c8d0cce.tar.gz",
        )

    if "com_google_absl" not in native.existing_rules():
        native.http_archive(
            name = "com_google_absl",
            strip_prefix = "abseil-cpp-cc4bed2d74f7c8717e31f9579214ab52a9c9c610",
            url = "https://github.com/abseil/abseil-cpp/archive/cc4bed2d74f7c8717e31f9579214ab52a9c9c610.tar.gz",
        )

    if "com_github_bazelbuild_bazeltoolchains" not in native.existing_rules():
        native.http_archive(
            name = "com_github_bazelbuild_bazeltoolchains",
            strip_prefix = "bazel-toolchains-b850ccdf53fed1ccab7670f52d6b297d74348d1b",
            urls = [
                "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/b850ccdf53fed1ccab7670f52d6b297d74348d1b.tar.gz",
                "https://github.com/bazelbuild/bazel-toolchains/archive/b850ccdf53fed1ccab7670f52d6b297d74348d1b.tar.gz",
            ],
            sha256 = "d84d6b2fe88ef99963febf91ddce33503eed14c155ace922e2122271b483be64",
        )
