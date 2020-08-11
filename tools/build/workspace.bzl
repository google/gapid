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
load("@gapid//tools/build/rules:android.bzl", "android_native_app_glue", "ndk_vk_validation_layer", "ndk_version_check")
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
        commit = "a94abcb35d08aac590dac3a1ac5961cfb16caf05",  # 0.22.1
        sha256 = "9b9cadda86ba1769cd944f17ccc2f5445de771b2d7322930929a106246cb0125",
    )

    maybe_repository(
        github_repository,
        name = "bazel_gazelle",
        locals = locals,
        organization = "bazelbuild",
        project = "bazel-gazelle",
        commit = "9b9f0226e01d10a70cbf268f9967cb715c8ff067",  # 0.20.0
        sha256 = "def44e3a1af0f1ea63910e99f8ad5bd7760416abe165c1511b6b93f65ad10a97",
    )

    maybe_repository(
        github_repository,
        name = "com_google_protobuf",
        locals = locals,
        organization = "google",
        project = "protobuf",
        commit = "d0bfd5221182da1a7cc280f3337b5e41a89539cf",  # 3.11.4
        sha256 = "c5fd8f99f0d30c6f9f050bf008e021ccc70d7645ac1f64679c6038e07583b2f3",
        repo_mapping = {"@zlib": "@net_zlib"},
    )

    maybe_repository(
        github_repository,
        name = "com_github_grpc_grpc",
        locals = locals,
        organization = "grpc",
        project = "grpc",
        commit = "c599e6a922a80e40e24a2d3c994a6dd51046796b",  # 1.22.1
        sha256 = "d17ead923510b3c8a03eec623fffe4cba64d43e10b3695f027a1c8f10c03756a",
	# This patch works around a naming conflict in grpc which leads to
	# compilation issues in recent gcc/glibc. This issue is fixed on recent
	# grpc versions (since
	# https://github.com/grpc/grpc/commit/de6255941a5e1c2fb2d50e57f84e38c09f45023d),
	# but updating our grpc version leads to errors in compiling the abseil
	# dependency of grpc
	# (https://github.com/abseil/abseil-cpp/issues/326). We tried to pull
	# abseil ourselves and patch it, but abseil also fails to compile with
	# gcc on windows, so we choose to patch grpc directly. Once grpc has a
	# version that builds fine on all our targets, we can update grpc and
	# drop this patch.
	patch_file = "@gapid//tools/build/third_party/com_github_grpc_grpc:com_github_grpc_grpc_fix.patch",
    )
    _grpc_deps(locals)

    maybe_repository(
        github_repository,
        name = "rules_python",
        locals = locals,
        organization = "bazelbuild",
        project = "rules_python",
        commit = "9150caa9d857e3768a4cf5ef6c3e88668b7ec84f",  # 0.0.1
        sha256 = "8eece92b8e286ac60b2847f0f00d0a949b3b0192669ffcc9e8d3c8365f889d1e",
    )

    ###########################################
    # Now get all our other non-go dependencies

    maybe_repository(
        github_repository,
        name = "com_google_googletest",
        locals = locals,
        organization = "google",
        project = "googletest",
        commit = "703bd9caab50b139428cea1aaff9974ebee5742e",  # 1.10.0
        sha256 = "2db427be8b258ad401177c411c2a7c2f6bc78548a04f1a23576cc62616d9cd38",
    )

    maybe_repository(
        github_repository,
        name = "astc_encoder",
        locals = locals,
        organization = "ARM-software",
        project = "astc-encoder",
        commit = "de61b3511fa2c3e2e0ae87f5a7874a928e3a3111",  # 1.x (March 2020)
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
        sha256 = "ac3318f1120e66d195fe4c5d3c413e20f9eeddc4c378349eeb906ca1d405d4f2",
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
        commit = "8db9eccc0baf30c9d22c496ab28db0fe1e4e97c5",  # 8.13.3559
        sha256 = "5c11a228d41ec011918b9c8beb60b6556745d30c8c856ec622beab5c5469152d",
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
        name = "stb",
        locals = locals,
        remote = "https://github.com/nothings/stb",
        commit = "f54acd4e13430c5122cab4ca657705c84aa61b08",
        build_file = "@gapid//tools/build/third_party:stb.BUILD",
        shallow_since = "1580905940 -0800",
    )

    maybe_repository(
        new_git_repository,
        name = "lss",
        locals = locals,
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "fd00dbbd0c06a309c657d89e9430143b179ff6db",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
       shallow_since = "1583885669 +0000",
    )

    maybe_repository(
        git_repository,
        name = "perfetto",
        locals = locals,
        remote = "https://android.googlesource.com/platform/external/perfetto",
        commit = "1dd60f85b97ef5068e9a39a4165087ccbdf6b09f",
        shallow_since = "1596837015 -0700",
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
        commit = "f8bf11a0253a32375c32cad92c841237b96696c0",
        sha256 = "2ca7c37db06ab526c8c5c31767a0bbdbd30de74909dc1a4900302d7a8f537de7",
    )

    maybe_repository(
        github_repository,
        name = "spirv_cross",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "871c85d7f0edc6b613e3959bc51d13bfbc2fe2df",
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
        sha256 = "6aba055d6a9a7c33ec2761c4883b21c9d67c7fef2550797cea677a77fd65055a",
    )

    maybe_repository(
        github_repository,
        name = "spirv_tools",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "60104cd97446877dad8ed1010a635218937a2f18",
        sha256 = "6050c012fec919087ebc3b083b24f874648fc1593b55ac8e3742df760aec19fc",
    )

    maybe_repository(
        github_repository,
        name = "spirv_reflect",
        locals = locals,
        organization = "chaoticbob",
        project = "SPIRV-Reflect",
        commit = "3c77a11472a1da7830d055306b4299c5e2398e7c",
        build_file = "@gapid//tools/build/third_party:spirv-reflect.BUILD",
        sha256 = "1af7c64657db1af191d7b8f12928d63159e1c1eafdbedac0b4add905d0430e8c",
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
        commit = "7264358702061d3ed819d62d3d6fd66ab1da33c3",  # 1.2.132
        build_file = "@gapid//tools/build/third_party:vulkan-headers.BUILD",
        sha256 = "d44112f625cb2152fd7c8906a15e4e98abc5946d1ef85c2e17b3cb5c247586d3",
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

        maybe_repository(
            ndk_version_check,
            name = "ndk_version_check",
            locals = locals,
        )

        # Use the LLVM libc++ Android toolchain.
        native.bind(
            name = "android/crosstool",
            actual = "@androidndk//:toolchain-libcpp",
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
        name = "madler_zlib",
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
