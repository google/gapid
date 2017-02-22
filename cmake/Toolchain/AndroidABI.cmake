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

# Common android settings
set(ANDROID_COMPILER_VERSION "4.9")
set(ANDROID_NDK_VERSION 21)

# Detect the NDK root directory
if(NOT ANDROID_NDK_ROOT)
    # Using the installed NDK for the android compilers
    set(ANDROID_NDK_ROOT "$ENV{ANDROID_NDK_ROOT}")
    if(NOT ANDROID_NDK_ROOT)
        # TODO: search well known locations for an NDK?
        message(FATAL_ERROR "You must have ANDROID_NDK_ROOT set")
    endif()
    string(REPLACE "\\" "/" ANDROID_NDK_ROOT "${ANDROID_NDK_ROOT}")
endif()

# Define and document the set of variables each android abi must declare
set(ANDROID_ABI_VARS
        ANDROID_ABI_NAME # The normal name of the abi
        ANDROID_ABI_PATH # The path used in the installed package for this abi
        ANDROID_BUILD_PATH # The path used by the build system for this abi
        ANDROID_PROCESSOR # The processor name this abi targets
        ANDROID_C_FLAGS # The extra c flags to use when compiling for this abi
        ANDROID_SYSTEM_ARCH # The ndk system architecture to use to compile for this abi
        ANDROID_TOOLCHAIN_PREFIX # The prefix to the specific host to android abi compiler tools
        # Constructed variables
        ANDROID_GCC_PATH # The path to the gcc tool for this abi
        ANDROID_GXX_PATH # The path to the g++ tool for this abi
        ANDROID_RANLIB_PATH # The path to the ranlib tool for this abi
        ANDROID_AR_PATH # The path to the ar tool for this abi
        ANDROID_LD_PATH # The path to the ld tool for this abi
)

set(ANDROID_ABI_NAME_armeabi "armeabi")
set(ANDROID_ABI_PATH_armeabi "armeabi-v7a")
set(ANDROID_BUILD_PATH_armeabi "android-armv7a")
set(ANDROID_PROCESSOR_armeabi "armv5te")
set(ANDROID_C_FLAGS_armeabi "-mthumb -march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=softfp")
set(ANDROID_SYSTEM_ARCH_armeabi "arm")

set(ANDROID_ABI_NAME_aarch64 "arm64-v8a")
set(ANDROID_ABI_PATH_aarch64 "arm64-v8a")
set(ANDROID_BUILD_PATH_aarch64 "android-armv8a")
set(ANDROID_PROCESSOR_aarch64 "aarch64")
set(ANDROID_C_FLAGS_aarch64 "")
set(ANDROID_SYSTEM_ARCH_aarch64 "arm64")

set(ANDROID_ABI_NAME_x86 "x86")
set(ANDROID_ABI_PATH_x86 "x86")
set(ANDROID_BUILD_PATH_x86 "android-x86")
set(ANDROID_PROCESSOR_x86 "x86")
set(ANDROID_C_FLAGS_x86 "-m32")
set(ANDROID_SYSTEM_ARCH_x86 "x86")

# Using the installed NDK for the android compilers
set(toolchains "${ANDROID_NDK_ROOT}/toolchains")
if(ToolchainHost STREQUAL "Linux")
    set(compiler_host "linux-x86_64")
endif()
if(ToolchainHost STREQUAL "Darwin")
    set(compiler_host "darwin-x86_64")
endif()
if(ToolchainHost STREQUAL "Windows")
    set(compiler_host "windows-x86_64")
endif()

set(ANDROID_TOOLCHAIN_PREFIX_armeabi "${toolchains}/arm-linux-androideabi-4.9/prebuilt/${compiler_host}/bin/arm-linux-androideabi-")
set(ANDROID_TOOLCHAIN_PREFIX_aarch64 "${toolchains}/aarch64-linux-android-4.9/prebuilt/${compiler_host}/bin/aarch64-linux-android-")
set(ANDROID_TOOLCHAIN_PREFIX_x86 "${toolchains}/x86-4.9/prebuilt/${compiler_host}/bin/i686-linux-android-")

# Look up the set of android ABI's by detecting the ANDROID_ABI_NAME_{$abi} variables
set(ANDROID_ABI_LIST)
get_cmake_property(vars VARIABLES)
foreach (var ${vars})
    if(var MATCHES "^ANDROID_ABI_NAME_(.*)")
        list(APPEND ANDROID_ABI_LIST ${CMAKE_MATCH_1})
    endif()
endforeach()

foreach(abi ${ANDROID_ABI_LIST})
    # Build any constructed variables
    set(ANDROID_GCC_PATH_${abi} "${ANDROID_TOOLCHAIN_PREFIX_${abi}}gcc${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    set(ANDROID_GXX_PATH_${abi} "${ANDROID_TOOLCHAIN_PREFIX_${abi}}g++${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    set(ANDROID_RANLIB_PATH_${abi} "${ANDROID_TOOLCHAIN_PREFIX_${abi}}ranlib${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    set(ANDROID_AR_PATH_${abi} "${ANDROID_TOOLCHAIN_PREFIX_${abi}}ar${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    set(ANDROID_LD_PATH_${abi} "${ANDROID_TOOLCHAIN_PREFIX_${abi}}ld${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    # Verify that all required variables are set
    foreach(var ${ANDROID_ABI_VARS})
        if(NOT DEFINED ${var}_${abi})
            message(FATAL_ERROR "ABI ${abi} does not define ${var}_${abi}")
        endif()
    endforeach()
endforeach()

if(NOT ANDROID_VALID_ABI_LIST)
    # Build the active abi list by detecting whether the configured compiler path is valid
    set(ANDROID_VALID_ABI_LIST)
    foreach(abi ${ANDROID_ABI_LIST})
        if(EXISTS ${ANDROID_GCC_PATH_${abi}})
            message(STATUS "Found ${abi} using ${ANDROID_GCC_PATH_${abi}}")
            list(APPEND ANDROID_VALID_ABI_LIST ${abi})
        else()
            message(STATUS "Disable ${abi}, missing ${ANDROID_GCC_PATH_${abi}}")
        endif()
    endforeach()
    set(ANDROID_VALID_ABI_LIST  "${ANDROID_VALID_ABI_LIST}" CACHE STRING "The set of android abi's to compile" FORCE)
endif()

# If we are actually compiling for android
if(ANDROID_ABI)
    # Check it's a valid host->abi toolchain
    list (FIND ANDROID_VALID_ABI_LIST ${ANDROID_ABI} is_valid)
    if (${is_valid} EQUAL -1)
        message(FATAL_ERROR "Android abi ${ANDROID_ABI} is not valid on this host")
    endif()
    # Set the default values of all the variables from the abi ones
    foreach(var ${ANDROID_ABI_VARS})
        set(${var} ${${var}_${ANDROID_ABI}})
    endforeach()
else()
    set(ANDROID_ACTIVE_ABI_LIST ${ANDROID_VALID_ABI_LIST})
endif()
