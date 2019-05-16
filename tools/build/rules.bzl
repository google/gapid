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

load("//tools/build/rules:android.bzl",
    _android_native_app_glue = "android_native_app_glue",
    _android_native = "android_native",
)
load("//tools/build/rules:apic.bzl",
    _apic_compile = "apic_compile",
    _apic_template = "apic_template",
)
load("//tools/build/rules:cc.bzl",
    _cc_copts = "cc_copts",
    _cc_stripped_binary = "cc_stripped_binary",
    _strip = "strip",
)
load("//tools/build/rules:common.bzl",
    _generate = "generate",
    _copy = "copy",
    _copy_to = "copy_to",
    _copy_tree = "copy_tree",
)
load("//tools/build/rules:dynlib.bzl",
    _android_dynamic_library = "android_dynamic_library",
    _cc_dynamic_library = "cc_dynamic_library",
)
load("//tools/build/rules:embed.bzl",
    _embed = "embed",
)
load("//tools/build/rules:filehash.bzl",
    _filehash = "filehash",
)
load("//tools/build/rules:jni.bzl",
    _jni_library = "jni_library",
)
load("//tools/build/rules:gapil.bzl",
    _api_library = "api_library",
    _api_template = "api_template",
)
load("//tools/build/rules:go.bzl",
    _go_stripped_binary = "go_stripped_binary",
)
load("//tools/build/rules:grpc.bzl",
    _java_grpc_library = "java_grpc_library",
    _cc_grpc_library = "cc_grpc_library",
)
load("//tools/build/rules:lingo.bzl",\
    _lingo = "lingo",
)
load("//tools/build/rules:mm.bzl",
    _mm_library = "mm_library",
)
load("//tools/build/rules:repository.bzl",
    _empty_repository = "empty_repository",
    _github_repository = "github_repository",
)
load("//tools/build/rules:stringgen.bzl",
    _stringgen = "stringgen",
)
load("//tools/build/rules:zip.bzl",
    _extract = "extract",
)

android_native_app_glue = _android_native_app_glue
android_native = _android_native
apic_compile = _apic_compile
apic_template = _apic_template
cc_copts = _cc_copts
cc_stripped_binary = _cc_stripped_binary
strip = _strip
generate = _generate
copy = _copy
copy_to = _copy_to
copy_tree = _copy_tree
android_dynamic_library = _android_dynamic_library
cc_dynamic_library = _cc_dynamic_library
embed = _embed
filehash = _filehash
jni_library = _jni_library
api_library = _api_library
api_template = _api_template
go_stripped_binary = _go_stripped_binary
java_grpc_library = _java_grpc_library
cc_grpc_library = _cc_grpc_library
lingo = _lingo
mm_library = _mm_library
empty_repository = _empty_repository
github_repository = _github_repository
stringgen = _stringgen
extract = _extract
