# Copyright (C) 2022 Google Inc.
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

load("@gapid//tools/build/rules:repository.bzl", "maybe_repository")
load("@rules_fuchsia//fuchsia:deps.bzl", "fuchsia_clang_repository", "fuchsia_sdk_repository", "rules_fuchsia_deps")

def fuchsia_sdk_dependencies(locals = {}):
  rules_fuchsia_deps()

  maybe_repository(
    fuchsia_sdk_repository,
    name = "fuchsia_sdk_dynamic",
    locals = locals,
    cipd_tag = "version:7.20220228.3.1",
    sha256 = {
      "linux": "7f886893f5758272da4f51aced35448fa83805aa0f2b363f2e538db4cf936462",
      "mac": "e18ea6c85b436146ce9f93f6abcea35baea2ba23f9544a87e7b6f5632ade374e",
    },
  )
  native.register_toolchains("@fuchsia_sdk_dynamic//:fuchsia_toolchain_sdk")

  maybe_repository(
    fuchsia_clang_repository,
    name = "fuchsia_clang",
    locals = locals,
    cipd_tag = "git_revision:c9e46219f38da5c3fbfe41012173dc893516826e",
    sdk_root_label = "@fuchsia_sdk_dynamic",
    sha256 = {
      "linux": "573ebb62fc5cd9d2357be630cc079fa97b028cd99e2e87132dcd8be31c425984",
      "mac": "fb9e478d18f35d0a9bb186138a85cf7ef5f9078fbf432fc3ad8f0660b233b3b4",
    },
  )
