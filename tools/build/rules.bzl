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

load("//tools/build/rules:android.bzl", "android_native_app_glue", "android_native")
load("//tools/build/rules:apic.bzl", "apic_compile", "apic_template")
load("//tools/build/rules:cc.bzl", "cc_copts", "cc_stripped_binary", "strip")
load("//tools/build/rules:common.bzl", "generate", "copy", "copy_to", "copy_tree")
load("//tools/build/rules:dynlib.bzl", "android_dynamic_library", "cc_dynamic_library")
load("//tools/build/rules:embed.bzl", "embed")
load("//tools/build/rules:filehash.bzl", "filehash")
load("//tools/build/rules:gapil.bzl", "api_library", "api_template")
load("//tools/build/rules:go.bzl", "go_stripped_binary")
load("//tools/build/rules:grpc.bzl", "java_grpc_library")
load("//tools/build/rules:lingo.bzl", "lingo")
load("//tools/build/rules:mm.bzl", "mm_library")
load("//tools/build/rules:repository.bzl", "empty_repository", "github_repository")
load("//tools/build/rules:stringgen.bzl", "stringgen")
load("//tools/build/rules:zip.bzl", "extract")
