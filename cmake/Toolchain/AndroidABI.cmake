# Copyright (C) 2017 Google Inc.
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

if(NOT ANDROID_ABI)
    set(ANDROID_ACTIVE_ABI_LIST "armeabi-v7a" "arm64-v8a" "x86")
endif()

set(ANDROID_ABI_NAME_armeabi-v7a "armeabi")
set(ANDROID_APK_NAME_armeabi-v7a "armeabi")
set(ANDROID_BUILD_PATH_armeabi-v7a "android-armv7a")

set(ANDROID_ABI_NAME_arm64-v8a "arm64-v8a")
set(ANDROID_APK_NAME_arm64-v8a "aarch64")
set(ANDROID_BUILD_PATH_arm64-v8a "android-armv8a")

set(ANDROID_ABI_NAME_x86 "x86")
set(ANDROID_APK_NAME_x86 "x86")
set(ANDROID_BUILD_PATH_x86 "android-x86")
