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
load("@gapid//tools/build/third_party:perfetto.bzl", "perfetto")
load("@gapid//tools/build/rules:grpc_c++.bzl", "grpc_deps")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "new_git_repository")

# Defines the repositories for GAPID's dependencies, excluding the
# go dependencies, which require @io_bazel_rules_go to be setup.
#  android - if false, the Android NDK/SDK are not initialized.
#  mingw - if false, our cc toolchain, which uses MinGW on Windows is not initialized.
#  locals - can be used to provide local path overrides for repos:
#     {"foo": "/path/to/foo"} would cause @foo to be a local repo based on /path/to/foo.
def gapid_dependencies(android = True, mingw = True, locals = {}):
    #####################################################
    # Get repositories with workspace rules we need first

    maybe_repository(
        github_repository,
        name = "io_bazel_rules_go",
        locals = locals,
        organization = "bazelbuild",
        project = "rules_go",
        commit = "3c34e66b0507056e83bcbd9c963ab9d7e6cb049f",  # 0.18.3
        sha256 = "eee73ae5c29e421b2bfda62fef86ce266a942f66f19ccb4dfb03b4fa428986a5",
    )

    maybe_repository(
        github_repository,
        name = "bazel_gazelle",
        locals = locals,
        organization = "bazelbuild",
        project = "bazel-gazelle",
        commit = "e443c54b396a236e0d3823f46c6a931e1c9939f2",  # 0.17.0
        sha256 = "ca6dcacc34c159784f01f557dbb0dc5d1772d3b28f1145b51f888ecb3694af1a",
    )

    maybe_repository(
        github_repository,
        name = "com_google_protobuf",
        locals = locals,
        organization = "google",
        project = "protobuf",
        commit = "66dc42d891a4fc8e9190c524fd67961688a37bbe", # 3.6.1 + changes
        sha256 = "23dcfef4adf06e1db7f1d391c035896a92eec546fff5ef9244b6e7b9ee24aa66",
    )

    maybe_repository(
        github_repository,
        name = "com_github_grpc_grpc",
        locals = locals,
        organization = "grpc",
        project = "grpc",
        # v1.11.0
        commit = "bd44e485f69d70ca4095cea92decd98de3892aa6",
        # Override with our own BUILD file, to make Android build work.
        build_file = "@gapid//tools/build/third_party:grpc_c++.BUILD",
        sha256 = "486ed046515f167e0e818442df9eaf91658b1bb29804eb2cf7caf276b4720083",
    )
    grpc_deps(locals)

    ###########################################
    # Now get all our other non-go dependencies

    maybe_repository(
        github_repository,
        name = "com_google_googletest",
        locals = locals,
        organization = "google",
        project = "googletest",
        commit = "62dbaa2947f7d058ea7e16703faea69b1134b024",
        sha256 = "c86258bf52616f5fa52a622ba58ce700eb2dd9f6ec15ff13ad2b2a579afb9c67",
    )

    maybe_repository(
        github_repository,
        name = "astc_encoder",
        locals = locals,
        organization = "ARM-software",
        project = "astc-encoder",
        commit = "b6bf6e7a523ddafdb8cfdc84b068d8fe70ffb45e",
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
        sha256 = "7877eb08c61d8b258c5d4690e924090cb7f303e8be6d74e9a9a611d3177bb5ae",
    )

    maybe_repository(
        breakpad,
        name = "breakpad",
        locals = locals,
        commit = "a61afe7a3e865f1da7ff7185184fe23977c2adca",
        build_file = "@gapid//tools/build/third_party/breakpad:breakpad.BUILD",
    )

    maybe_repository(
        github_repository,
        name = "cityhash",
        locals = locals,
        organization = "google",
        project = "cityhash",
        commit = "8af9b8c2b889d80c22d6bc26ba0df1afb79a30db",
        build_file = "@gapid//tools/build/third_party:cityhash.BUILD",
        sha256 = "3524f5ed43143974a29fddeeece29c8b6348f05db08dd180452da01a2837ddce",
    )

    maybe_repository(
        github_repository,
        name = "glslang",
        locals = locals,
        organization = "KhronosGroup",
        project = "glslang",
        commit = "97e35420a62e112de57a31b265e020662883ef8f",
        build_file = "@gapid//tools/build/third_party:glslang.BUILD",
        sha256 = "4d73467f35b8ac15cc06206cbd8be2802afc630bbfc4e9504b81e711457dde49",
    )

    maybe_repository(
        github_repository,
        name = "llvm",
        locals = locals,
        organization = "llvm-mirror",
        project = "llvm",
        commit = "e562960fe303c0ffab6f3458fcdb1544b56fd81e",
        build_file = "@gapid//tools/build/third_party:llvm.BUILD",
        sha256 = "3ef3d905849d547b6481b16d8e7b473a84efafbe90131e7bc90a0c6aae4cd8e6",
    )

    maybe_repository(
        new_git_repository,
        name = "lss",
        locals = locals,
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "e6527b0cd469e3ff5764785dadcb39bf7d787154",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
    )

    maybe_repository(
        perfetto,
        name = "perfetto",
        locals = locals,
    )

    maybe_repository(
        github_repository,
        name = "spirv_headers",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Headers",
        commit = "9f6846f973a1ef53790e75b9190820ab1557434f",
        build_file = "@gapid//tools/build/third_party:spirv-headers.BUILD",
        sha256 = "1980cefd605c440241f5c948eb4446412166b6df1ad133bf74c47180939477d5",
    )

    maybe_repository(
        github_repository,
        name = "spirv_cross",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "ed55e0ac6d797a338e7c19dad785237f0efc4d86",
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
        sha256 = "a6decf21a137e63f5e9dc01b716c7a905c54eef23fe6a7910058fd253460cec0",
    )

    maybe_repository(
        github_repository,
        name = "spirv_tools",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "8d8a71278bf9e83dd0fb30d5474386d30870b74d",
        build_file = "@gapid//tools/build/third_party:spirv-tools.BUILD",
        sha256 = "8b1dfe726ea9047ef679baf2d40dfbf090e70406512358d236e54a8234e71eae",
    )

    maybe_repository(
        github_repository,
        name = "spirv_reflect",
        locals = locals,
        organization = "chaoticbob",
        project = "SPIRV-Reflect",
        commit = "a861e587bdc924c49272873bbc1744928bc51aac",
        build_file = "@gapid//tools/build/third_party:spirv-reflect.BUILD",
        sha256 = "da636883f8d31fa5d1a8722374b92e76bc1f19ec7c125882c843079623f1c13a",
    )

    if android:
        maybe_repository(
            native.android_sdk_repository,
            name = "androidsdk",
            locals = locals,
            api_level = 26, # This is the target API
        )

        maybe_repository(
            native.android_ndk_repository,
            name = "androidndk",
            locals = locals,
            api_level = 21, # This is the minimum API
        )

        maybe_repository(
            android_native_app_glue,
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
