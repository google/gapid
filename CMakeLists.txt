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

cmake_minimum_required(VERSION 3.2)
cmake_policy(SET CMP0009 NEW) # GLOB_RECURSE does not recurse symlinks
cmake_policy(SET CMP0053 NEW) # Simplify variable reference
cmake_policy(SET CMP0003 NEW) # Link by full path
cmake_policy(SET CMP0058 NEW) # Custom command DEPENDS on files that are not specified as OUTPUT or BYPRODUCTS

set(GAPID_CMAKE_VERSION "1")
set(GAPID_CMAKE_VERSION_STORE ${GAPID_CMAKE_VERSION} CACHE STRING "Version of the gapid cmake build system")
if(NOT ${GAPID_CMAKE_VERSION} STREQUAL ${GAPID_CMAKE_VERSION_STORE})
    message(FATAL_ERROR "
    Output directory version is ${GAPID_CMAKE_VERSION_STORE}, required version is ${GAPID_CMAKE_VERSION}
    The cmake output directory needs to be deleted and regenerated...
")
endif()

set(CMAKE_HOST_EXECUTABLE_SUFFIX "")
if(CMAKE_HOST_WIN32)
    set(CMAKE_HOST_EXECUTABLE_SUFFIX ".exe")
endif()

file(LOCK ${CMAKE_BINARY_DIR} DIRECTORY GUARD PROCESS RESULT_VARIABLE locked)

# Basic selection controls
option(BUILDBOT "Running in buildbot mode")
option(DISABLE_CXX "Disable all c++ building steps")
option(DISABLE_GO "Disable all go building steps")
option(DISABLE_PROTOC "Disable all proto building steps")
option(DISABLE_CODE_GENERATION "Disable all code generation building steps")
option(ONLY_HOST "Disable non host build steps")

include(cmake/Init.cmake)
project(GAPID ${languages})
include(Config)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    # Build a cmake output directory for the specific abi
    add_cmake(${abi} ${CMAKE_SOURCE_DIR} "-DANDROID_ABI:STRING=${abi}")
endforeach()

if(ARMLINUX_GAPII)
    set(GAPII_TOOLCHAIN "ArmLinux")
    string(TOLOWER "${GAPII_TOOLCHAIN}" GAPII_PROJECT)
    add_cmake(${GAPII_PROJECT} ${CMAKE_SOURCE_DIR} "-DGAPII_TARGET:STRING=${GAPII_TOOLCHAIN}")
endif()

# Third party libraries
include(third_party/cityhash.cmake)
include(third_party/khronos.cmake)
include(third_party/protobuf.cmake)
include(third_party/grpc-java.cmake)

# Must come first (declares rules like apic).
build_subdirectory(cmd)

build_subdirectory(core)
build_subdirectory(gapii)
if(NOT GAPII_TARGET)
build_subdirectory(gapil)
build_subdirectory(gapir)
build_subdirectory(gapis)
build_subdirectory(gapidapk)
build_subdirectory(test)
endif()

scan_for_protos()

# Add root as include path for all c includes.
include_directories(${CMAKE_SOURCE_DIR})

include(Package)
