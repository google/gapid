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
load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository")
load("@gapid//tools/build/third_party:breakpad.bzl", "breakpad")
load("@gapid//tools/build/rules:grpc_c++.bzl", "grpc_deps")

# Defines the repositories for GAPID's dependencies, excluding the
# go dependencies, which require @io_bazel_rules_go to be setup.
#  android - if false, the Android NDK/SDK are not initialized.
#  java_client - if false, the Java deps used by the client are not initialized.
#  mingw - if false, our cc toolchain, which uses MinGW on Windows is not initialized.
#  locals - can be used to provide local path overrides for repos:
#     {"foo": "/path/to/foo"} would cause @foo to be a local repo based on /path/to/foo.
def gapid_dependencies(android = True, java_client = True, mingw = True, locals = {}):
    #####################################################
    # Get repositories with workspace rules we need first

    maybe_repository(github_repository,
        name = "io_bazel_rules_go",
        locals = locals,
        organization = "bazelbuild",
        project = "rules_go",
        commit = "01c7b1ba59c7a1819f1003cf58400cb3d4f7d22e",
    )

    maybe_repository(github_repository,
        name = "bazel_gazelle",
        locals = locals,
        organization = "bazelbuild",
        project = "bazel-gazelle",
        commit = "84ec09c308035853493731ef1401ec4812f9674f",
    )

    maybe_repository(github_repository,
        name = "com_google_protobuf",
        locals = locals,
        organization = "google",
        project = "protobuf",
        # Matches with GRPC
        commit = "2761122b810fe8861004ae785cc3ab39f384d342",
        # Override with our own BUILD file, to make the compiler/config selection work.
        build_file = "@gapid//tools/build/third_party:protobuf.BUILD",
    )

    maybe_repository(github_repository,
        name = "com_github_grpc_grpc",
        locals = locals,
        organization = "grpc",
        project = "grpc",
        # v1.11.0
        commit = "bd44e485f69d70ca4095cea92decd98de3892aa6",
        # Override with our own BUILD file, to make Android build work.
        build_file = "@gapid//tools/build/third_party:grpc_c++.BUILD",
    )
    grpc_deps(locals)

    ###########################################
    # Now get all our other non-go dependencies

    maybe_repository(github_repository,
        name = "com_google_googletest",
        locals = locals,
        organization = "google",
        project = "googletest",
        commit = "62dbaa2947f7d058ea7e16703faea69b1134b024",
    )

    maybe_repository(github_repository,
        name = "astc_encoder",
        locals = locals,
        organization = "ARM-software",
        project = "astc-encoder",
        commit = "b6bf6e7a523ddafdb8cfdc84b068d8fe70ffb45e",
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
    )

    maybe_repository(breakpad,
        name = "breakpad",
        locals = locals,
        commit = "a61afe7a3e865f1da7ff7185184fe23977c2adca",
        build_file = "@gapid//tools/build/third_party/breakpad:breakpad.BUILD",
    )

    maybe_repository(github_repository,
        name = "cityhash",
        locals = locals,
        organization = "google",
        project = "cityhash",
        commit = "8af9b8c2b889d80c22d6bc26ba0df1afb79a30db",
        build_file = "@gapid//tools/build/third_party:cityhash.BUILD",
    )

    maybe_repository(github_repository,
        name = "glslang",
        locals = locals,
        organization = "KhronosGroup",
        project = "glslang",
        commit = "97e35420a62e112de57a31b265e020662883ef8f",
        build_file = "@gapid//tools/build/third_party:glslang.BUILD",
    )

    maybe_repository(github_repository,
        name = "llvm",
        locals = locals,
        organization = "llvm-mirror",
        project = "llvm",
        commit = "aeb3e4fcbba63c2750aa27cfe75c4374a8b23550",
        build_file = "@gapid//tools/build/third_party:llvm.BUILD",
    )

    maybe_repository(native.new_git_repository,
        name = "lss",
        locals = locals,
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "e6527b0cd469e3ff5764785dadcb39bf7d787154",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
    )

    maybe_repository(github_repository,
        name = "spirv_headers",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Headers",
        commit = "9f6846f973a1ef53790e75b9190820ab1557434f",
        build_file = "@gapid//tools/build/third_party:spirv-headers.BUILD",
    )

    maybe_repository(github_repository,
        name = "spirv_cross",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "d3b966322d04efbc27c70e2ea8da36eb9c29eb1b",
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
    )

    maybe_repository(github_repository,
        name = "spirv_tools",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "8d8a71278bf9e83dd0fb30d5474386d30870b74d",
        build_file = "@gapid//tools/build/third_party:spirv-tools.BUILD",
    )

    maybe_repository(github_repository,
        name = "spirv_reflect",
        locals = locals,
        organization = "chaoticbob",
        project = "SPIRV-Reflect",
        commit = "5598462f987841f7c1abe9209650ea8d3e727b46",
        build_file = "@gapid//tools/build/third_party:spirv-reflect.BUILD",
    )

    if java_client:
        maybe_repository(github_repository,
            name = "com_github_grpc_java",
            locals = locals,
            organization = "grpc",
            project = "grpc-java",
            commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
            build_file = "@gapid//tools/build/third_party:grpc_java.BUILD",
        )

    if android:
        maybe_repository(native.android_sdk_repository,
            name = "androidsdk",
            locals = locals,
            api_level = 21,
        )

        maybe_repository(native.android_ndk_repository,
            name = "androidndk",
            locals = locals,
            api_level = 21,
        )

        maybe_repository(android_native_app_glue,
            name = "android_native_app_glue",
            locals = locals,
        )

        # Use the LLVM libc++ Android toolchain.
        native.bind(
            name = "android/crosstool",
            actual = "@androidndk//:toolchain-libcpp",
        )

    if mingw:
        cc_configure()
