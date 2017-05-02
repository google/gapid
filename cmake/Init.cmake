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

# Init is called before the project is initialised, to perform toolchain setup.

# The set of pools we are using and their limits
set_property(GLOBAL PROPERTY JOB_POOLS go=1)

# Module loading paths
set(GPU_CMAKE_PATH "${CMAKE_CURRENT_LIST_DIR}")
list(APPEND CMAKE_MODULE_PATH "${GPU_CMAKE_PATH}")
list(APPEND CMAKE_MODULE_PATH "${GPU_CMAKE_PATH}/Modules")

# Common support libraries
include(CMakeParseArguments)
include(CMakePrintHelpers)
include(ExternalProject)
include(Utils)

# Default the build type to release
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Choose the type of build, options are: Debug Release" FORCE)
endif()

if(NOT DEFINED ROOT_DIR)
    get_filename_component(ROOT_DIR "${CMAKE_SOURCE_DIR}/../../../.." ABSOLUTE)
endif()

if(NOT DEFINED GPU_DIR)
    set(GPU_DIR "${CMAKE_SOURCE_DIR}")
endif()

if(NOT DEFINED GTEST_DIR)
    get_filename_component(GTEST_DIR "${GPU_DIR}/third_party/googletest/googletest" ABSOLUTE)
endif()

if(NOT DEFINED GMOCK_DIR)
    get_filename_component(GMOCK_DIR "${GPU_DIR}/third_party/googletest/googlemock" ABSOLUTE)
endif()

if(NOT DEFINED PROTOBUF_DIR)
    get_filename_component(PROTOBUF_DIR "${GPU_DIR}/third_party/protobuf" ABSOLUTE)
endif()

# Toolchain host detection
if(CMAKE_HOST_APPLE)
    set(ToolchainHost "Darwin")
elseif(CMAKE_HOST_WIN32)
    set(ToolchainHost "Windows")
elseif(CMAKE_HOST_UNIX)
    set(ToolchainHost "Linux")
else()
    message(FATAL_ERROR "Unknown host operating system")
endif()

# Toolchain target selection
if(GAPII_TARGET)
    set(CMAKE_SYSTEM_NAME ${GAPII_TARGET})
else()
    include("Toolchain/AndroidABI")
endif()

if(ANDROID_ABI)
    set(ToolchainTarget "Android")
elseif(CMAKE_SYSTEM_NAME)
    set(ToolchainTarget "${CMAKE_SYSTEM_NAME}")
else()
    set(ToolchainTarget "${ToolchainHost}")
endif()

# Now include the appropriate toolchain file
include("Toolchain/${ToolchainTarget}")

# Language configuration
set(languages)
if(NOT NO_CXX)
    list(APPEND languages C CXX)
endif()
if(NOT NO_GO)
    list(APPEND languages Go)
endif()

# Speed up builds with ccache
find_program(CCACHE_PROGRAM ccache)
if(CCACHE_PROGRAM)
    set_property(GLOBAL PROPERTY RULE_LAUNCH_COMPILE "${CCACHE_PROGRAM}")
    set_property(GLOBAL PROPERTY RULE_LAUNCH_LINK "${CCACHE_PROGRAM}")
endif()
