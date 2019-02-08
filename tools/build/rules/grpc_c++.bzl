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
load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

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


"""Generates C++ grpc stubs from proto_library rules.

This is an internal rule used by cc_grpc_library, and shouldn't be used
directly.
"""

def _generate_cc_impl(ctx):
  """Implementation of the generate_cc rule."""
  protos = [f for src in ctx.attr.srcs for f in src.proto.direct_sources]
  # protos = [f for src in ctx.attr.srcs for f in src.proto.transitive_sources]
  includes = [f for src in ctx.attr.srcs for f in src.proto.transitive_imports]
  outs = []
  # label_len is length of the path from WORKSPACE root to the location of this build file
  label_len = 0
  # proto_root is the directory relative to which generated include paths should be
  proto_root = ""
  if ctx.label.package:
    # The +1 is for the trailing slash.
    label_len += len(ctx.label.package) + 1
  if ctx.label.workspace_root:
    label_len += len(ctx.label.workspace_root) + 1
    proto_root = "/" + ctx.label.workspace_root

  if ctx.executable.plugin:
    outs += [proto.path[label_len:-len(".proto")] + ".grpc.pb.h" for proto in protos]
    outs += [proto.path[label_len:-len(".proto")] + ".grpc.pb.cc" for proto in protos]
    if ctx.attr.generate_mocks:
      outs += [proto.path[label_len:-len(".proto")] + "_mock.grpc.pb.h" for proto in protos]
  else:
    outs += [proto.path[label_len:-len(".proto")] + ".pb.h" for proto in protos]
    outs += [proto.path[label_len:-len(".proto")] + ".pb.cc" for proto in protos]
  out_files = [ctx.new_file(out) for out in outs]
  dir_out = str(ctx.genfiles_dir.path + proto_root)

  arguments = []
  if ctx.executable.plugin:
    arguments += ["--plugin=protoc-gen-PLUGIN=" + ctx.executable.plugin.path]
    flags = list(ctx.attr.flags)
    if ctx.attr.generate_mocks:
      flags.append("generate_mock_code=true")
    arguments += ["--PLUGIN_out=" + ",".join(flags) + ":" + dir_out]
    additional_input = [ctx.executable.plugin]
  else:
    arguments += ["--cpp_out=" + ",".join(ctx.attr.flags) + ":" + dir_out]
    additional_input = []

  # Import protos relative to their workspace root so that protoc prints the
  # right include paths.
  for include in includes:
    directory = include.path
    if directory.startswith("external"):
      external_sep = directory.find("/")
      repository_sep = directory.find("/", external_sep + 1)
      arguments += ["--proto_path=" + directory[:repository_sep]]
    else:
      arguments += ["--proto_path=."]
  # Include the output directory so that protoc puts the generated code in the
  # right directory.
  arguments += ["--proto_path={0}{1}".format(dir_out, proto_root)]
  arguments += [proto.path for proto in protos]

  # create a list of well known proto files if the argument is non-None
  well_known_proto_files = []
  if ctx.attr.well_known_protos:
    f = ctx.attr.well_known_protos.files.to_list()[0].dirname
    if f != "external/com_google_protobuf/src/google/protobuf":
      print("Error: Only @com_google_protobuf//:well_known_protos is supported")
    else:
      # f points to "external/com_google_protobuf/src/google/protobuf"
      # add -I argument to protoc so it knows where to look for the proto files.
      arguments += ["-I{0}".format(f + "/../..")]
      well_known_proto_files = [f for f in ctx.attr.well_known_protos.files]

  ctx.action(
      inputs = protos + includes + additional_input + well_known_proto_files,
      outputs = out_files,
      executable = ctx.executable._protoc,
      arguments = arguments,
  )

  return struct(files=depset(out_files))

_generate_cc = rule(
    attrs = {
        "srcs": attr.label_list(
            mandatory = True,
            non_empty = True,
            providers = ["proto"],
        ),
        "plugin": attr.label(
            executable = True,
            providers = ["files_to_run"],
            cfg = "host",
        ),
        "flags": attr.string_list(
            mandatory = False,
            allow_empty = True,
        ),
        "well_known_protos" : attr.label(
            mandatory = False,
        ),
        "generate_mocks" : attr.bool(
            default = False,
            mandatory = False,
        ),
        "_protoc": attr.label(
            default = Label("//external:protocol_compiler"),
            executable = True,
            cfg = "host",
        ),
    },
    # We generate .h files, so we need to output to genfiles.
    output_to_genfiles = True,
    implementation = _generate_cc_impl,
)

def cc_grpc_library(name, proto, dep_cc_proto_libs, **kwargs):
  pb_target = "_" + name + "_pb_only"
  grpc_target = "_" + name + "_grpc"

  native.cc_proto_library(
      name = pb_target,
      deps = proto,
  )

  plugin = "//external:grpc_cpp_plugin"
  _generate_cc(
    name = grpc_target,
    srcs = proto,
    plugin = plugin,
    well_known_protos = "@com_google_protobuf//:well_known_protos",
    **kwargs
  )
  grpc_deps = ["//external:grpc++_codegen_proto",
               "//external:protobuf"]
  native.cc_library(
    name = name,
    srcs = [":" + grpc_target],
    hdrs = [":" + grpc_target],
    deps = grpc_deps + dep_cc_proto_libs + [pb_target],
    **kwargs
  )

"""Load dependencies needed to compile and test the grpc library as a 3rd-party consumer."""

def grpc_deps(locals = {}):
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
        http_archive(
            name = "boringssl",
            # on the master-with-bazel branch
            url = "https://boringssl.googlesource.com/boringssl/+archive/6ae5a54bedae2c29e5b67382667871c527e68326.tar.gz",
        )

    maybe_repository(github_repository,
        name = "com_github_madler_zlib",
        locals = locals,
        organization = "madler",
        project = "zlib",
        commit = "cacf7f1d4e3d44d871b605da3b647f07d718623f",
        build_file = "@com_github_grpc_grpc//third_party:zlib.BUILD",
        sha256 = "1cce3828ec2ba80ff8a4cac0ab5aa03756026517154c4b450e617ede751d41bd",
    )

    maybe_repository(github_repository,
        name = "com_github_google_googletest",
        locals = locals,
        organization = "google",
        project = "googletest",
        commit = "ec44c6c1675c25b9827aacd08c02433cccde7780",
        build_file = "@com_github_grpc_grpc//third_party:gtest.BUILD",
        sha256 = "bc258fff04a6511e7106a1575bb514a185935041b2c16affb799e0567393ec30",
    )

    maybe_repository(github_repository,
        name = "com_github_gflags_gflags",
        locals = locals,
        organization = "gflags",
        project = "gflags",
        commit = "30dbc81fb5ffdc98ea9b14b1918bfe4e8779b26e",
        sha256 = "16903f6bb63c00689eee3bf7fb4b8f242934f6c839ce3afc5690f71b712187f9",
    )

    maybe_repository(github_repository,
        name = "com_github_google_benchmark",
        locals = locals,
        organization = "google",
        project = "benchmark",
        commit = "5b7683f49e1e9223cf9927b24f6fd3d6bd82e3f8",
        build_file = "@com_github_grpc_grpc//third_party:benchmark.BUILD",
        sha256 = "866d3c8cadb3323251d4fe0e989ea0df51f30660badb73dad0f2d11e5bf1770f",
    )

    maybe_repository(github_repository,
        name = "com_github_cares_cares",
        locals = locals,
        organization = "c-ares",
        project = "c-ares",
        commit = "3be1924221e1326df520f8498d704a5c4c8d0cce",
        build_file = "@com_github_grpc_grpc//third_party:cares/cares.BUILD",
        sha256 = "932bf7e593d4683fce44fd26920f27d4f0c229113338e4f6d351e35d4d7c7a39",
    )

    maybe_repository(github_repository,
        name = "com_google_absl",
        locals = locals,
        organization = "abseil",
        project = "abseil-cpp",
        commit = "cc4bed2d74f7c8717e31f9579214ab52a9c9c610",
        sha256 = "2e1573146ee96ea7a88628210984675597eeb741ec3662e6a254f23b168b406c",
    )

    maybe_repository(github_repository,
        name = "com_github_bazelbuild_bazeltoolchains",
        locals = locals,
        organization = "bazelbuild",
        project = "bazel-toolchains",
        commit = "b850ccdf53fed1ccab7670f52d6b297d74348d1b",
        sha256 = "3433b9292fe4bfdc39f9e60e251311f2e88aa734181749ae87af73eada730a94",
    )
