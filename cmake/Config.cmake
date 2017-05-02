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

# Config is called once the project is initialized to configure the build.

# Apply the options and environment to build controls
if(NO_CXX OR NOT CMAKE_CXX_COMPILER)
    set(DISABLED_CXX ON)
endif()
if(NO_GO OR ANDROID OR GAPII_TARGET OR NOT CMAKE_Go_COMPILER)
    set(DISABLED_GO ON)
endif()
if(NO_CODE_GENERATION OR BUILDBOT OR CMAKE_CROSSCOMPILING)
    set(DISABLED_CODE_GENERATION ON)
    set(DISABLE_PROTOC ON)
endif()
if(HOST_ONLY OR CMAKE_CROSSCOMPILING OR ANDROID OR DISABLED_CXX)
    set(ANDROID_ACTIVE_ABI_LIST)
endif()

if(NO_TESTS OR ANDROID OR GAPII_TARGET OR CMAKE_CROSSCOMPILING)
    set(DISABLED_TESTS ON)
endif()
if(NOT DISABLED_TESTS)
    enable_testing()
endif()

if(UNIX AND NOT (APPLE OR ANDROID OR WIN32))
    set(LINUX TRUE)
endif()

# Support functions
include(Glob)
include(UseGo)
include(UseGradle)
include(UseCMake)
include(UseProtoc)
include(UseGtest)

# Global settings
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin")
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin")
set(CMAKE_TEST_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/test")

string(TOLOWER "${TARGET_OS}" TARGET_PATH)
set(PlatformSourcePath ${TARGET_PATH})
if (NOT PlatformSourcePath STREQUAL "windows")
  LIST(APPEND PlatformSourcePath "posix")
endif()

set(JAVA_BASE "${CMAKE_SOURCE_DIR}/gapic/src")
set(JAVA_SERVICE "${JAVA_BASE}/service")

# Install and package settings
set(home $ENV{HOME})
set(gapid_home $ENV{GAPID_HOME})
if(NOT home)
    set(home $ENV{USERPROFILE})
endif()
if(gapid_home)
    set(INSTALL_PREFIX "${gapid_home}" CACHE STRING "Package install directory")
endif()
if(home)
    set(INSTALL_PREFIX "${home}/gapid" CACHE STRING "Package install directory")
endif()
set(PROTO_CC_OUT "${CMAKE_BINARY_DIR}/proto_cc" CACHE PATH "Path to the cc_proto directory")
if(NOT CMAKE_INSTALL_PREFIX STREQUAL ${INSTALL_PREFIX})
    set(CMAKE_INSTALL_PREFIX "${INSTALL_PREFIX}" CACHE STRING "" FORCE)
    message(STATUS "Setting install path to ${CMAKE_INSTALL_PREFIX}")
endif()

set(TARGET_INSTALL_PATH ".")

# Standard compile options
set(CMAKE_POSITION_INDEPENDENT_CODE TRUE)
set(CXX_STANDARD_REQUIRED ON)
add_definitions(-DTARGET_OS_${TARGET_OS})
if (CMAKE_BUILD_TYPE STREQUAL "Debug")
  add_definitions(-DLOG_LEVEL=LOG_LEVEL_VERBOSE)
endif()

if(MSVC)
    find_package(MSVC REQUIRED)
    find_package(WindowsSDK REQUIRED)

    #  warning C4351: new behavior: elements of array ... will be default initialized
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /EHsc /wd4351")
    #set(CMAKE_STATIC_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} /MACHINE:X64")
    #set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} /MACHINE:X64")
    #set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} /MACHINE:X64")
else()
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++0x")
endif()

if (ANDROID)
    set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} ${ANDROID_C_FLAGS}")
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} ${ANDROID_CXX_FLAGS}")
    set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} ${ANDROID_LINKER_FLAGS}")
    set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} -z defs")
endif()

if(CMAKE_COMPILER_IS_GNUCXX AND WIN32)
    set(BUILD_SHARED_LIBRARIES OFF)
    set(CMAKE_EXE_LINKER_FLAGS "-static")
endif()

#compile_options(${name} PUBLIC
#-Werror
#-Wall
#-Wextra
#-Wno-unused-variable
#)
