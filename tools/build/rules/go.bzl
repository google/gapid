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

load("//tools/build/rules:cc.bzl", "strip")
load("//tools/build/rules:common.bzl", "copy_exec")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")

# Macro to replace go_binary rules. Creates the following targets:
#  <name>_unstripped - The unstripped go_binary with debug information.
#  <name> - The stripped go_binary.
def go_stripped_binary(name, **kwargs):
    visibility = kwargs.pop("visibility")

    go_binary(
        name = name + "_unstripped",
        **kwargs
    )
    strip(
        name = name + "_stripped",
        srcs = [name + "_unstripped"],
        visibility = visibility,
    )

    copy_exec(
        name = name,
        srcs = select({
            "@gapid//tools/build:debug": [name + "_unstripped"],
            "//conditions:default": [name + "_stripped"],
        }),
        visibility = visibility,
    )
