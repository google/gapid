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
        commit = "efc3212592320c1ab7f986c9a7882770ee06ad3b",  # 0.34.0
        sha256 = "eb10f4436ddc732127afedf78636637d0cc9b256aba9f643a452914289266e6b",
    )

    maybe_repository(
        github_repository,
        name = "rules_python",
        locals = locals,
        organization = "bazelbuild",
        project = "rules_python",
        commit = "ae7a2677b3003b13d45bc9bfc25f1425bed5b407",  # 0.8.1
        sha256 = "f1c3069679395ac1c1104f28a166f06167d30d41bdb1797d154d80b511780d2e",
        patches = [
            # Fix the problem with trying to use /usr/bin/python rather than a versioned python.
            "@gapid//tools/build/third_party:rules_python.patch",
        ],
    )

    maybe_repository(
        github_repository,
        name = "bazel_gazelle",
        locals = locals,
        organization = "bazelbuild",
        project = "bazel-gazelle",
        commit = "e9091445339de2ba7c01c3561f751b64a7fab4a5",  # 0.23.0
        sha256 = "03e266ed67fd21f6fbede975773a569d397312daae71980d34ff7f7e087b7b14",
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

    maybe_repository(
        github_repository,
        name = "com_google_protobuf",
        locals = locals,
        organization = "google",
        project = "protobuf",
        commit = "ab840345966d0fa8e7100d771c92a73bfbadd25c",  # 3.21.5
        sha256 = "0025119f5c97871436b4b271fee48bd6bfdc99956023e0d4fd653dd8eaaeff52",
        repo_mapping = {"@zlib": "@net_zlib"},
    )

    maybe_repository(
        github_repository,
        name = "com_github_grpc_grpc",
        locals = locals,
        organization = "grpc",
        project = "grpc",
        commit = "d2054ec6c6e8abcecf0e24b0b4ee75035d80c3cc",  # 1.48.0
        sha256 = "ea0da456d849eafa5287dc1e9d53c065896dca2cd896a984101ebe0708979dca",
        repo_mapping = {"@zlib": "@net_zlib"},
        patches = [
            # Remove calling the go dependencies, since we do that ourselves.
            "@gapid//tools/build/third_party:com_github_grpc_grpc.patch",
        ],
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
        commit = "2164984243a3fbd2b2429e7562f7fe31914f446b",  # 2.1 (November 2020)
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
        sha256 = "50d0e8c84acf7e16d0ce2c4f3bf658c5a69d0e07d21b7fd629b73cbb143b477a",
    )

    maybe_repository(
        github_repository,
        name = "etc2comp",
        locals = locals,
        organization = "google",
        project = "etc2comp",
        commit = "9cd0f9cae0f32338943699bb418107db61bb66f2", # 2017/04/24
        build_file = "@gapid//tools/build/third_party:etc2comp.BUILD",
        sha256 = "0ddcf7484c0d55bc5a3cb92edb4812dc932ac9f73b4641ad2843fec82ae8cf90",
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

    # Override the gRPC abseil dependency, so we can patch it.
    maybe_repository(
        github_repository,
        name = "com_google_absl",
        locals = locals,
        organization = "abseil",
        project = "abseil-cpp",
        commit = "278e0a071885a22dcd2fd1b5576cc44757299343",  # LTS 20210324, Patch 2
        sha256 = "ff5ea6f91f9bcd0f368346ef707d0a80a372b71de5b6ae69ac11d0ca41688b8f",
        patches = [
            # Workaround for https://github.com/abseil/abseil-cpp/issues/326.
            "@gapid//tools/build/third_party:abseil_macos_fix.patch",
        ],
    )

    maybe_repository(
        github_repository,
        name = "glslang",
        locals = locals,
        organization = "KhronosGroup",
        project = "glslang",
        commit = "ae2a562936cc8504c9ef2757cceaff163147834f",  # 11.5.0
        sha256 = "a89149cd3ed0938fc53f39778b12ecf7326e52ca0ca30179f747db042893fb98",
    )

    maybe_repository(
        github_repository,
        name = "stb",
        locals = locals,
        organization = "nothings",
        project = "stb",
        commit = "3a1174060a7dd4eb652d4e6854bc4cd98c159200",
        sha256 = "9313f6871195b97771ce7da1feae0b3d92c7936456f13099edb54a78096ca93c",
        build_file = "@gapid//tools/build/third_party:stb.BUILD",
    )

    maybe_repository(
        new_git_repository,
        name = "lss",
        locals = locals,
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "e1e7b0ad8ee99a875b272c8e33e308472e897660",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
        shallow_since = "1618241076 +0000",
    )

    maybe_repository(
        git_repository,
        name = "perfetto",
        locals = locals,
        remote = "https://android.googlesource.com/platform/external/perfetto",
        commit = "d1a7b031bbded67e0d67957974e85a83e0b815c0",
        shallow_since = "1619185617 +0100",
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
        commit = "449bc986ba6f4c5e10e32828783f9daef2a77644",
        sha256 = "5624978c0788b06fc66a23ab3066bf99d5d9f421a1698b509417163bb169ac33",
    )

    maybe_repository(
        github_repository,
        name = "spirv_cross",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "9acb9ec31f5a8ef80ea6b994bb77be787b08d3d1",  # 2021-01-15
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
        sha256 = "222af32a8a809612dffb67abf3530863dcdb224dba8babf742f7c754413cfc60",
    )

    maybe_repository(
        github_repository,
        name = "spirv_tools",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "3d4246b4aaa65a4b901abd04e7d5bc1822c582bb",
        sha256 = "3eaefd42e6d34be6cade456ac41cba6e1f955d0f67875f2cd6ef488b9cd9440f",
    )

    maybe_repository(
        github_repository,
        name = "spirv_reflect",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Reflect",
        commit = "272e050728de8d4a4ce9e7101c1244e6ff56e5b0",
        sha256 = "3812779f02cc1860adfebd6f2be11b0253781a552a1fc7ebc4eefec8914c6918",
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
        commit = "8ba8294c86d0e99fcb457bedbd573dd678ccc9b3",  # 1.3.212
        build_file = "@gapid//tools/build/third_party:vulkan-headers.BUILD",
        sha256 = "eaca5dde10b3b6d3cecf07c7f9b84f17f72ab8bce3bd79818a8909978c6fd601",
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

    if mingw:
        cc_configure()
