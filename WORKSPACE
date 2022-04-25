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

load("@gapid//tools/build:locals.bzl", "user_local_repos")
user_local_repos(__workspace_dir__)
load("@user_locals//:locals.bzl", "LOCALS")

load("@gapid//tools/build:workspace.bzl", "gapid_dependencies")
gapid_dependencies(locals = LOCALS)

load("@gapid//tools/build:workspace_go.bzl", "gapid_go_dependencies")
gapid_go_dependencies()

load("@gapid//tools/build:workspace_gapic.bzl", "gapic_dependencies", "gapic_third_party")
gapic_dependencies(locals = LOCALS)
gapic_third_party()

# Fuchsia rules, disabled by default. See .bazelrc for config to enable.
load("@gapid//tools/build/fuchsia:fuchsia_config.bzl", "fuchsia_config")
fuchsia_config()
load("@local_config_fuchsia//:workspace.bzl", "fuchsia_base_dependencies")
fuchsia_base_dependencies(locals = LOCALS)
load("@local_config_fuchsia//:fuchsia_sdk.bzl", "fuchsia_sdk_dependencies")
fuchsia_sdk_dependencies(locals = LOCALS)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
go_rules_dependencies()
go_register_toolchains("1.17")

# gazelle:repo bazel_gazelle
gazelle_dependencies()

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")
grpc_deps()
load("@com_github_grpc_grpc//bazel:grpc_extra_deps.bzl", "grpc_extra_deps")
grpc_extra_deps()

# Python setup.
load("@rules_python//python:repositories.bzl", "python_register_toolchains")

python_register_toolchains(
    name = "python3_9",
    python_version = "3.9",
)
