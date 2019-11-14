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
load("@gapid//tools/build/rules:android.bzl", "android_native_app_glue", "ndk_vk_validation_layer")
load("@gapid//tools/build/rules:repository.bzl", "github_repository", "maybe_repository")
load("@gapid//tools/build/third_party:breakpad.bzl", "breakpad")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository", "new_git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

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
        commit = "6fc21c78143ff1d4ea98100e8fd7a928d45abd00",  # 0.18.6
        sha256 = "6356b0c591659b2da6f8149dfe7207a23d2cc41d3ed3932f0be3aa5dad7a4d2f",
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
        commit = "815ff7e1fb2d417d5aebcbf5fc46e626b18dc834", # Head of 3.8.x branch
        sha256 = "083646275522dc57e145f769c2daf39d469757bafcc5b7d09b119dfaf1b873b8",
        repo_mapping = {"@zlib": "@net_zlib"},
    )

    maybe_repository(
        github_repository,
        name = "com_github_grpc_grpc",
        locals = locals,
        organization = "grpc",
        project = "grpc",
        # v1.20.1
        commit = "7741e806a213cba63c96234f16d712a8aa101a49",
        sha256 = "9ed7d944d8d07deac365c9edcda10ce8159c1436119e1b0792a1e830cb20606c",
    )
    _grpc_deps(locals)

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
        name = "lodepng",
        locals = locals,
        remote = "https://github.com/lvandeve/lodepng",
        commit = "3606db5a71359a4165308e7cd5afe67bd42f7edf",
        build_file = "@gapid//tools/build/third_party:lodepng.BUILD",
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
        git_repository,
        name = "perfetto",
        locals = locals,
        remote = "https://android.googlesource.com/platform/external/perfetto",
        commit = "aa9175d6761d7ff7a172d637fda4a17df64453b5",
        shallow_since = "1570644465 +0000",
    )

    maybe_repository(
        http_archive,
        name = "sqlite",
        locals = locals,
        url = "https://storage.googleapis.com/perfetto/sqlite-amalgamation-3250300.zip",
        sha256 = "2ad5379f3b665b60599492cc8a13ac480ea6d819f91b1ef32ed0e1ad152fafef",
        strip_prefix = "sqlite-amalgamation-3250300",
        build_file = "@perfetto//bazel:sqlite.BUILD",
    )

    maybe_repository(
        http_archive,
        name = "sqlite_src",
        locals = locals,
        url = "https://storage.googleapis.com/perfetto/sqlite-src-3250300.zip",
        sha256 = "c7922bc840a799481050ee9a76e679462da131adba1814687f05aa5c93766421",
        strip_prefix = "sqlite-src-3250300",
        build_file = "@perfetto//bazel:sqlite.BUILD",
    )

    maybe_repository(
        native.new_local_repository,
        name = "perfetto_cfg",
        locals = locals,
        path = "tools/build/third_party/perfetto",
        build_file = "@gapid//tools/build/third_party/perfetto:BUILD.bazel",
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

    maybe_repository(
        http_archive,
        name = "vscode-languageclient",
        locals = locals,
        url = "https://registry.npmjs.org/vscode-languageclient/-/vscode-languageclient-2.6.3.tgz",
        build_file = "@gapid//tools/build/third_party:vscode-languageclient.BUILD",
        sha256 = "42ad6dc73bbf24a067d1e21038d35deab975cb207ac2d63b81c37a977d431d8f",
    )

    maybe_repository(
        http_archive,
        name = "vscode-jsonrpc",
        locals = locals,
        url = "https://registry.npmjs.org/vscode-jsonrpc/-/vscode-jsonrpc-2.4.0.tgz",
        build_file = "@gapid//tools/build/third_party:vscode-jsonrpc.BUILD",
        sha256= "bed9b2facb7d179f14c8a710db8e613be56bd88b2a75443143778813048b5c89",
    )

    maybe_repository(
        http_archive,
        name = "vscode-languageserver-types",
        locals = locals,
        url = "https://registry.npmjs.org/vscode-languageserver-types/-/vscode-languageserver-types-1.0.4.tgz",
        build_file = "@gapid//tools/build/third_party:vscode-languageserver-types.BUILD",
        sha256 = "0cd219ac388c41a70c3ff4f72d25bd54fa351bc0850196c25c6c3361e799ac79",
    )

    maybe_repository(
        github_repository,
        name = "vulkan-headers",
        locals = locals,
        organization = "KhronosGroup",
        project = "Vulkan-Headers",
        commit = "5bc459e2921304c32568b73edaac8d6df5f98b84",  # 1.1.127
        build_file = "@gapid//tools/build/third_party:vulkan-headers.BUILD",
        sha256 = "e5d3169f6d2e2f0bcd95f89c5e2f7bb3090af1682f96e3117bd0a74fca78936d",
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
            api_level = 23, # This is the minimum API
        )

        maybe_repository(
            android_native_app_glue,
            name = "android_native_app_glue",
            locals = locals,
        )

        maybe_repository(
            ndk_vk_validation_layer,
            name = "ndk_vk_validation_layer",
            locals = locals,
        )

        # Use the LLVM libc++ Android toolchain.
        native.bind(
            name = "android/crosstool",
            actual = "@androidndk//:toolchain-libcpp",
        )

        maybe_repository(
            http_archive,
            name = "libinterceptor",
            locals = locals,
            url = "https://github.com/google/gapid/releases/download/libinterceptor-v1.0/libinterceptor.zip",
            build_file = "@gapid//tools/build/third_party:libinterceptor.BUILD",
            sha256 = "307e0e3ec7451a244811b4edf21453d55d1e90a5f868a73dc42d4975ef74aec9",
        )

    if mingw:
        cc_configure()

# Function to setup all the GRPC deps and bindings.
def _grpc_deps(locals):
    maybe_repository(http_archive,
        name = "boringssl",
        locals = locals,
        # on the master-with-bazel branch
        url = "https://boringssl.googlesource.com/boringssl/+archive/afc30d43eef92979b05776ec0963c9cede5fb80f.tar.gz",
    )

    maybe_repository(github_repository,
        name = "net_zlib", # name used by rules_go
        locals = locals,
        organization = "madler",
        project = "zlib",
        commit = "cacf7f1d4e3d44d871b605da3b647f07d718623f",
        build_file = "@gapid//tools/build/third_party:zlib.BUILD",
        sha256 = "1cce3828ec2ba80ff8a4cac0ab5aa03756026517154c4b450e617ede751d41bd",
    )

    maybe_repository(github_repository,
        name = "com_github_nanopb_nanopb",
        locals = locals,
        organization = "nanopb",
        project = "nanopb",
        commit = "f8ac463766281625ad710900479130c7fcb4d63b",
        build_file = "@com_github_grpc_grpc//third_party:nanopb.BUILD",
        sha256 = "e7e635b26fa11246e8fd1c46df141d2f094a659b905ac61e957234018308f883",
    )

    native.bind(
        name = "libssl",
        actual = "@boringssl//:ssl",
    )

    native.bind(
        name = "zlib",
        actual = "@net_zlib//:z",
    )

    native.bind(
        name = "nanopb",
        actual = "@com_github_nanopb_nanopb//:nanopb",
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
