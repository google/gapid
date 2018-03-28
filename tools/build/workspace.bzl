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

# Defines macros to be called from a WORKSPACE file to setup the GAPID
# dependencies and toolchains.

load("@gapid//tools/build:cc_toolchain.bzl", "cc_configure")
load("@gapid//tools/build/rules:android.bzl", "android_native_app_glue")
load("@gapid//tools/build/rules:repository.bzl", "github_http_args", "github_repository")
load("@gapid//tools/build/third_party:breakpad.bzl", "breakpad")

# Defines the repositories for GAPID's dependencies, excluding the
# go dependencies, which require @io_bazel_rules_go to be setup.
def gapid_dependencies(android = True, java_client = True, mingw = True):
    #####################################################
    # Get repositories with workspace rules we need first

    _maybe(github_repository,
        name = "io_bazel_rules_go",
        organization = "bazelbuild",
        project = "rules_go",
        commit = "2d3336269eab48bac7adcaff03e7232e14463619",
    )

    _maybe(github_repository,
        name = "bazel_gazelle",
        organization = "bazelbuild",
        project = "bazel-gazelle",
        commit = "f4ae892927eeabd060c59693c38e82303f41558d",
    )

    _maybe(github_repository,
        name = "com_google_protobuf",
        organization = "google",
        project = "protobuf",
        commit = "f08e4dd9845c5ba121b402f8768f3d2617191bbe",
        # Override with our own BUILD file, to make the compiler/config selection work.
        build_file = "@gapid//tools/build/third_party:protobuf.BUILD",
    )

    _maybe(github_repository,
        name = "com_github_grpc_grpc",
        organization = "grpc",
        project = "grpc",
        commit = "fa301e3674a1cc786eb4dd4253a0e677f2eb68e3",
    )

    ###########################################
    # Now get all our other non-go dependencies

    _maybe(github_repository,
        name = "com_google_googletest",
        organization = "google",
        project = "googletest",
        commit = "62dbaa2947f7d058ea7e16703faea69b1134b024",
    )

    _maybe(github_repository,
        name = "astc-encoder",
        organization = "ARM-software",
        project = "astc-encoder",
        commit = "b6bf6e7a523ddafdb8cfdc84b068d8fe70ffb45e",
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
    )

    _maybe(breakpad,
        name = "breakpad",
        commit = "a61afe7a3e865f1da7ff7185184fe23977c2adca",
    )

    _maybe(github_repository,
        name = "cityhash",
        organization = "google",
        project = "cityhash",
        commit = "8af9b8c2b889d80c22d6bc26ba0df1afb79a30db",
        build_file = "@gapid//tools/build/third_party:cityhash.BUILD",
    )

    _maybe(github_repository,
        name = "glslang",
        organization = "KhronosGroup",
        project = "glslang",
        commit = "56e8056582c92e0226d87418171d06f4e74ff29b",
        build_file = "@gapid//tools/build/third_party:glslang.BUILD",
    )

    _maybe(github_repository,
        name = "llvm",
        organization = "ben-clayton",
        project = "llvm",
        commit = "4c7186401413dad4dc7d6923b69b05554e762cff",
        build_file = "@gapid//tools/build/third_party:llvm.BUILD",
    )

    _maybe(native.new_git_repository,
        name = "lss",
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "e6527b0cd469e3ff5764785dadcb39bf7d787154",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
    )

    _maybe(github_repository,
        name = "spirv-headers",
        organization = "KhronosGroup",
        project = "SPIRV-Headers",
        commit = "9f6846f973a1ef53790e75b9190820ab1557434f",
        build_file = "@gapid//tools/build/third_party:spirv-headers.BUILD",
    )

    _maybe(github_repository,
        name = "spirv-cross",
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "29315f3b3fd6dcafab0075e1a3d898c3ff995fed",
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
    )

    _maybe(github_repository,
        name = "spirv-tools",
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "8d8a71278bf9e83dd0fb30d5474386d30870b74d",
        build_file = "@gapid//tools/build/third_party:spirv-tools.BUILD",
    )

    if java_client:
        _maybe(github_repository,
            name = "com_github_grpc_java",
            organization = "grpc",
            project = "grpc-java",
            commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
            build_file = "@gapid//tools/build/third_party:grpc_java.BUILD",
        )

    if android:
        _maybe(native.android_sdk_repository,
            name = "androidsdk",
            api_level = 21,
        )

        _maybe(native.android_ndk_repository,
            name = "androidndk",
            api_level = 21,
        )

        _maybe(android_native_app_glue,
            name = "android_native_app_glue",
        )

    if mingw:
        cc_configure()

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
