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

load("//:version.bzl", "version_define_copts")

_ANDROID_COPTS = [
    "-fdata-sections",
    "-ffunction-sections",
    "-fvisibility-inlines-hidden",
    "-DANDROID",
    "-DTARGET_OS_ANDROID",
]

# This should probably all be done by fixing the toolchains...
def cc_copts():
    return version_define_copts() + select({
        "@//tools/build:linux": ["-DTARGET_OS_LINUX"],
        "@//tools/build:darwin": ["-DTARGET_OS_OSX"],
        "@//tools/build:windows": ["-DTARGET_OS_WINDOWS"],
        "@//tools/build:android-armeabi-v7a": _ANDROID_COPTS,
        "@//tools/build:android-arm64-v8a": _ANDROID_COPTS,
        "@//tools/build:android-x86": _ANDROID_COPTS,
    })
