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

# This CMake invocation is compiling for android

if(NOT ANDROID_NDK_ROOT)
    message(FATAL ERROR "ANDROID_NDK_ROOT not set")
endif()

# Sub-cmake builds must not attempt to generate code.
set(DISABLED_CODE_GENERATION 1)

# The settings common to all android builds
set(ANDROID ON CACHE INTERNAL "Target system was of type android")
set(TARGET_OS ANDROID)

set(CMAKE_HOST_SYSTEM_NAME ${ToolchainHost})
set(ANDROID_TOOLCHAIN clang)
set(ANDROID_PLATFORM "android-21")

# Import the NDK's android.toolchain.cmake file.
include(${ANDROID_NDK_ROOT}/build/cmake/android.toolchain.cmake)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    # Strip all to reduce release executable sizes
    set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} -Wl,--strip-all")
endif()

# Update cmake settings from android ones
set(CMAKE_STAGING_PREFIX ${CMAKE_BINARY_DIR}/stage)
set(CMAKE_SYSTEM_NAME Linux)
set(CMAKE_SYSTEM_PROCESSOR "${ANDROID_PROCESSOR}")
