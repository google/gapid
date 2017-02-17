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

# The settings common to all android builds
set(ANDROID ON CACHE INTERNAL "Target system was of type android")
set(TARGET_OS ANDROID)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(ANDROID_C_FLAGS "${ANDROID_C_FLAGS} -O2 -fPIC")
else()
    set(ANDROID_C_FLAGS "${ANDROID_C_FLAGS} -O0 -fPIC")
endif()

# TODO: set(ANDROID_LINKER_FLAGS "${ANDROID_LINKER_FLAGS} -B ${ANDROID_COMPILER_BASE}")

set(ANDROID_SYSROOT "${ANDROID_NDK_ROOT}/platforms/android-${ANDROID_NDK_VERSION}/arch-${ANDROID_SYSTEM_ARCH}")
set(ANDROID_STL_ROOT "${ANDROID_NDK_ROOT}/sources/cxx-stl/gnu-libstdc++/${ANDROID_COMPILER_VERSION}")
set(ANDROID_STL_LIBS "${ANDROID_STL_ROOT}/libs/${ANDROID_ABI_NAME}")

# Enable link time garbage collection to reduce the binary size
set(ANDROID_C_FLAGS "${ANDROID_C_FLAGS} -fdata-sections -ffunction-sections")
set(ANDROID_LINKER_FLAGS "${ANDROID_LINKER_FLAGS} -Wl,--gc-sections")

# Set all derived variables
set(ANDROID_C_FLAGS "--sysroot=${ANDROID_SYSROOT} ${ANDROID_C_FLAGS} -funwind-tables -fsigned-char -no-canonical-prefixes")
set(ANDROID_CXX_FLAGS "${ANDROID_C_FLAGS} ${ANDROID_CXX_FLAGS}")
set(ANDROID_LINKER_FLAGS "${ANDROID_LINKER_FLAGS} -shared")

# Update cmake settings from android  ones
set(CMAKE_SYSROOT ${ANDROID_SYSROOT})
set(CMAKE_STAGING_PREFIX ${CMAKE_BINARY_DIR}/stage)
set(CMAKE_SYSTEM_NAME Linux)
set(CMAKE_SYSTEM_PROCESSOR "${ANDROID_PROCESSOR}")
#TODO: list(APPEND CMAKE_PREFIX_PATH "${ANDROID_COMPILER_PATH}")
include_directories("${ANDROID_SYSROOT}/usr/include")
list(APPEND CMAKE_LIBRARY_PATH "${ANDROID_STL_LIBS}")

# only search for libraries and includes in the ndk toolchain
#TODO: set(CMAKE_FIND_ROOT_PATH "${ANDROID_COMPILER_PATH}/bin" "${ANDROID_COMPILER_PATH}/${ANDROID_COMPILER_NAME}" "${ANDROID_SYSROOT}")
set(CMAKE_FIND_ROOT_PATH "${ANDROID_SYSROOT}")
set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)

include (CMakeForceCompiler)
CMAKE_FORCE_C_COMPILER("${ANDROID_GCC_PATH}" GNU)
CMAKE_FORCE_CXX_COMPILER("${ANDROID_GXX_PATH}" GNU)
set(CMAKE_RANLIB "${ANDROID_RANLIB_PATH}" CACHE FILEPATH "" FORCE)
set(CMAKE_AR "${ANDROID_AR_PATH}" CACHE FILEPATH "" FORCE)
set(CMAKE_LINKER "${ANDROID_LD_PATH}" CACHE FILEPATH "" FORCE)
set(CMAKE_C_COMPILER_WORKS TRUE)
set(CMAKE_CXX_COMPILER_WORKS TRUE)
