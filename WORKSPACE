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

workspace(name = "gapid")

####################################################################
# Get repositories with rules we need for the rest of the file first

load("@//tools/build/rules:repository.bzl", "github_repository")

github_repository(
    name = "io_bazel_rules_go",
    organization = "bazelbuild",
    project = "rules_go",
    commit = "c949c4d2235a3988ed3c7ac9beb70f707d29d465",
)

github_repository(
    name = "bazel_gazelle",
    organization = "bazelbuild",
    project = "bazel-gazelle",
    commit = "2f186389e2d9a91ee64007914f9b9d0ecae1d8e9",
)

github_repository(
    name = "com_google_protobuf",
    organization = "google",
    project = "protobuf",
    commit = "f08e4dd9845c5ba121b402f8768f3d2617191bbe",
    # Override with our own BUILD file, to make the compiler/config selection work.
    build_file = "//tools/build/third_party:protobuf.BUILD",
)

# rules_go has a second copy of the protobuf repo. Add it here, so we can override the BUILD file.
github_repository(
    name = "com_github_google_protobuf",
    organization = "google",
    project = "protobuf",
    commit = "f08e4dd9845c5ba121b402f8768f3d2617191bbe",
    build_file = "//tools/build/third_party:protobuf.BUILD",
)

github_repository(
    name = "com_github_grpc_grpc",
    organization = "grpc",
    project = "grpc",
    commit = "fa301e3674a1cc786eb4dd4253a0e677f2eb68e3",
)

####################################################################
# Load all our workspace rules

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@//tools/build:cc_toolchain.bzl", "cc_configure")
load("@//tools/build:rules.bzl", "android_native_app_glue", "github_go_repository")

####################################################################
# Run our workspace preparation rules

cc_configure()

android_sdk_repository(
    name="androidsdk",
    api_level=21,
)

android_ndk_repository(
    name="androidndk",
    api_level=21,
)

android_native_app_glue(
    name = "android_native_app_glue",
)

####################################################################
# Now get all our other dependencies

http_archive(
    name = "gtest",
    url = "https://github.com/google/googletest/archive/62dbaa2947f7d058ea7e16703faea69b1134b024.zip",
    sha256 = "c86258bf52616f5fa52a622ba58ce700eb2dd9f6ec15ff13ad2b2a579afb9c67",
    strip_prefix = "googletest-62dbaa2947f7d058ea7e16703faea69b1134b024",
)

github_repository(
    name = "astc-encoder",
    organization = "ARM-software",
    project = "astc-encoder",
    commit = "b6bf6e7a523ddafdb8cfdc84b068d8fe70ffb45e",
    build_file = "//tools/build/third_party:astc-encoder.BUILD",
)

new_git_repository(
    name = "breakpad",
    remote = "https://chromium.googlesource.com/breakpad/breakpad",
    commit = "a61afe7a3e865f1da7ff7185184fe23977c2adca",
    build_file = "//tools/build/third_party:breakpad.BUILD",
)

github_repository(
    name = "cityhash",
    organization = "google",
    project = "cityhash",
    commit = "8af9b8c2b889d80c22d6bc26ba0df1afb79a30db",
    build_file = "//tools/build/third_party:cityhash.BUILD",
)

github_repository(
    name = "glslang",
    organization = "KhronosGroup",
    project = "glslang",
    commit = "778806a69246b8921e867e839c9e87ccddc924f2",
    build_file = "//tools/build/third_party:glslang.BUILD",
)

github_repository(
    name = "llvm",
    organization = "ben-clayton",
    project = "llvm",
    commit = "4c7186401413dad4dc7d6923b69b05554e762cff",
    build_file = "//tools/build/third_party:llvm.BUILD",
)

new_git_repository(
    name = "lss",
    remote = "https://chromium.googlesource.com/linux-syscall-support",
    commit = "e6527b0cd469e3ff5764785dadcb39bf7d787154",
    build_file = "//tools/build/third_party:lss.BUILD",
)

github_repository(
    name = "spirv-headers",
    organization = "KhronosGroup",
    project = "SPIRV-Headers",
    commit = "2bf02308656f97898c5f7e433712f21737c61e4e",
    build_file = "//tools/build/third_party:spirv-headers.BUILD",
)

github_repository(
    name = "spirv-cross",
    organization = "KhronosGroup",
    project = "SPIRV-Cross",
    commit = "98a17431c24b47392cbe343da8dbd1f5ffbb23e8",
    build_file = "//tools/build/third_party:spirv-cross.BUILD",
)

github_repository(
    name = "spirv-tools",
    organization = "KhronosGroup",
    project = "SPIRV-Tools",
    commit = "0b0454c42c6b6f6746434bd5c78c5c70f65d9c51",
    build_file = "//tools/build/third_party:spirv-tools.BUILD",
)

####################################################################
# Go dependencies.

github_go_repository(
    name = "com_github_golang_protobuf",
    organization = "golang",
    project = "protobuf",
    commit = "8ee79997227bf9b34611aee7946ae64735e6fd93",
    importpath = "github.com/golang/protobuf",
)

github_go_repository(
    name = "com_github_google_go_github",
    organization = "google",
    project = "go-github",
    commit = "a89ea1cdf79929726a9416663609269ada774da0",
    importpath = "github.com/google/go-github",
)

github_go_repository(
    name = "com_github_google_go_querystring",
    organization = "google",
    project = "go-querystring",
    commit = "53e6ce116135b80d037921a7fdd5138cf32d7a8a",
    importpath = "github.com/google/go-querystring",
)

github_go_repository(
    name = "com_github_pkg_errors",
    organization = "pkg",
    project = "errors",
    commit = "248dadf4e9068a0b3e79f02ed0a610d935de5302",
    importpath = "github.com/pkg/errors",
)

github_go_repository(
    name = "org_golang_google_grpc",
    organization = "grpc",
    project = "grpc-go",
    commit = "50955793b0183f9de69bd78e2ec251cf20aab121",
    importpath = "google.golang.org/grpc",
)

github_go_repository(
    name = "org_golang_x_crypto",
    organization = "golang",
    project = "crypto",
    commit = "dc137beb6cce2043eb6b5f223ab8bf51c32459f4",
    importpath = "golang.org/x/crypto",
)

github_go_repository(
    name = "org_golang_x_net",
    organization = "golang",
    project = "net",
    commit = "f2499483f923065a842d38eb4c7f1927e6fc6e6d",
    importpath = "golang.org/x/net",
)

github_go_repository(
    name = "org_golang_x_sys",
    organization = "golang",
    project = "sys",
    commit = "d75a52659825e75fff6158388dddc6a5b04f9ba5",
    importpath = "golang.org/x/sys",
)

github_go_repository(
    name = "org_golang_x_tools",
    organization = "golang",
    project = "tools",
    commit = "3da34b1b520a543128e8441cd2ffffc383111d03",
    importpath = "golang.org/x/tools",
)

# Setup the go rules after all our repos have been setup.
go_rules_dependencies()
go_register_toolchains()
gazelle_dependencies()

####################################################################
# Java dependencies.

github_repository(
    name = "com_github_grpc_java",
    organization = "grpc",
    project = "grpc-java",
    commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
    build_file = "//tools/build/third_party:grpc_java.BUILD",
)
