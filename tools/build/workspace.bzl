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
        commit = "a2b7f4288fc7ad4ed387aa20cb2d09bf497a1b10",  # 0.11.0
        sha256 = "226f62de4dfd78fc6a2ee82c4747d4cce4d39dbf67108887ca5c7aa07a30dbd2",
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
        commit = "21767c654d31d2dccdde4330529775c6c5fd5389",  # 1.2.12
        build_file = "@gapid//tools/build/third_party:zlib.BUILD",
        sha256 = "b860a877983100f28c7bcf2f3bb7abbca8833e7ce3af79edfda21358441435d3",
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
        commit = "58d77fa8070e8cec2dc1ed015d66b454c8d78850",  # 1.12.1
        sha256 = "ab78fa3f912d44d38b785ec011a25f26512aaedc5291f51f3807c592b506d33a",
    )

    maybe_repository(
        github_repository,
        name = "astc_encoder",
        locals = locals,
        organization = "ARM-software",
        project = "astc-encoder",
        commit = "f6236cf158a877b3279a2090dbea5e9a4c105d64",  # 4.0.0
        build_file = "@gapid//tools/build/third_party:astc-encoder.BUILD",
        sha256 = "28305281b0fe89b0e57c61f684ed7f6145a5079a3f4f03a4fd3fe0c27df0bb45",
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
        commit = "335e61656fa6034fabc3431a91e5800ba6fc3dc9",
        build_file = "@gapid//tools/build/third_party/breakpad:breakpad.BUILD",
    )

    maybe_repository(
        github_repository,
        name = "cityhash",
        locals = locals,
        organization = "google",
        project = "cityhash",
        commit = "f5dc54147fcce12cefd16548c8e760d68ac04226",
        build_file = "@gapid//tools/build/third_party:cityhash.BUILD",
        sha256 = "20ab6da9929826c7c81ea3b7348190538a23f823a8b749c2da9715ecb7a6b545",
    )

    # Override the gRPC abseil dependency, so we can patch it.
    maybe_repository(
        github_repository,
        name = "com_google_absl",
        locals = locals,
        organization = "abseil",
        project = "abseil-cpp",
        commit = "273292d1cfc0a94a65082ee350509af1d113344d",  # LTS 20220623, Patch 0
        sha256 = "6764f226bd6e2d8ab9fe2f3cab5f45fb1a4a15c04b58b87ba7fa87456054f98b",
        patches = [
            # Workaround for https://github.com/abseil/abseil-cpp/issues/326.
            "@gapid//tools/build/third_party:abseil_macos_fix.patch",
            # Pick up bcrypt library on Windows.
            "@gapid//tools/build/third_party:abseil_windows_fix.patch",
        ],
    )

    maybe_repository(
        github_repository,
        name = "glslang",
        locals = locals,
        organization = "KhronosGroup",
        project = "glslang",
        commit = "73c9630da979017b2f7e19c6549e2bdb93d9b238",  # 11.11.0
        sha256 = "9304cb73d86fc8e3f1cbcdbd157cd2750baad10cb9e3a798986bca3c3a1be1f0",
    )

    maybe_repository(
        github_repository,
        name = "stb",
        locals = locals,
        organization = "nothings",
        project = "stb",
        commit = "af1a5bc352164740c1cc1354942b1c6b72eacb8a",
        sha256 = "e3d0edbecd356506d3d69b87419de2f9d180a98099134c6343177885f6c2cbef",
        build_file = "@gapid//tools/build/third_party:stb.BUILD",
    )

    maybe_repository(
        new_git_repository,
        name = "lss",
        locals = locals,
        remote = "https://chromium.googlesource.com/linux-syscall-support",
        commit = "c0c9689369b4c5e46b440993807ce4b0a7c9af8a",
        build_file = "@gapid//tools/build/third_party:lss.BUILD",
        shallow_since = "1660655052 +0000",
    )

    maybe_repository(
        github_repository,
        name = "perfetto",
        locals = locals,
        organization = "google",
        project = "perfetto",
        commit = "99ead408d98eaa25b7819c7e059734bea42fa148",  # 28.0
        sha256 = "24440b99bd8400be7bca44b767488641d15f099eebb4e09715a5c0b9bf6c8d68",
        patches = [
            # Fix a Windows MinGW build issue.
            "@gapid//tools/build/third_party:perfetto.patch",
        ]
    )

    maybe_repository(
        http_archive,
        name = "sqlite",
        locals = locals,
        url = "https://storage.googleapis.com/perfetto/sqlite-amalgamation-3350400.zip",
        sha256 = "f3bf0df69f5de0675196f4644e05d07dbc698d674dc563a12eff17d5b215cdf5",
        strip_prefix = "sqlite-amalgamation-3350400",
        build_file = "@perfetto//bazel:sqlite.BUILD",
    )

    maybe_repository(
        http_archive,
        name = "sqlite_src",
        locals = locals,
        url = "https://storage.googleapis.com/perfetto/sqlite-src-3320300.zip",
        sha256 = "9312f0865d3692384d466048f746d18f88e7ffd1758b77d4f07904e03ed5f5b9",
        strip_prefix = "sqlite-src-3320300",
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
        commit = "b2a156e1c0434bc8c99aaebba1c7be98be7ac580",  # 1.3.216.0
        sha256 = "fbb4e256c2e9385169067d3b6f2ed3800f042afac9fb44a348b619aa277bb1fd",
    )

    maybe_repository(
        github_repository,
        name = "spirv_cross",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Cross",
        commit = "0e2880ab990e79ce6cc8c79c219feda42d98b1e8",  # 2021-08-30
        build_file = "@gapid//tools/build/third_party:spirv-cross.BUILD",
        sha256 = "7ae1069c29f507730ffa5143ac23a5be87444d18262b3b327dfb00ca53ae07cd",
    )

    maybe_repository(
        github_repository,
        name = "spirv_tools",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Tools",
        commit = "b930e734ea198b7aabbbf04ee1562cf6f57962f0",  # 1.3.216.0
        sha256 = "2d956e7d49a8795335d13c3099c44aae4fe501eb3ec0dbf7e1bfa28df8029b43",
    )

    maybe_repository(
        github_repository,
        name = "spirv_reflect",
        locals = locals,
        organization = "KhronosGroup",
        project = "SPIRV-Reflect",
        commit = "0f142bbfe9bd7aeeae6b3c703bcaf837dba8df9d",  # 1.3.216.0
        sha256 = "8eae9dcd2f6954b452a9a53b02ce7507dd3dcd02bacb678c4316f336dab79d86",
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
        commit = "3ef4c97fd6ea001d75a8e9da408ee473c180e456",  # 1.3.216
        build_file = "@gapid//tools/build/third_party:vulkan-headers.BUILD",
        sha256 = "64a7fc6994501b36811af47b21385251a56a136a3ed3cf92673465c9d62985a1",
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
