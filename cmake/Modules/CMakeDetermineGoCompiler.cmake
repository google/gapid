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

if(NOT CMAKE_Go_COMPILER)
    message(STATUS "Looking for a Go compiler")
    set(paths)
    set(flags)
    find_program(CMAKE_Go_COMPILER go PATHS ${paths} DOC "The go compiler" ${flags})
    message(STATUS "Looking for a Go compiler - ${CMAKE_Go_COMPILER}")
    if(NOT CMAKE_Go_COMPILER)
        message(STATUS "Failed to find Go compiler")
        return()
    endif()
    mark_as_advanced(CMAKE_Go_COMPILER)
endif()

# set the go root from where we found the go binary
get_filename_component(GO_ROOT "${CMAKE_Go_COMPILER}" REALPATH)
get_filename_component(GO_ROOT "${GO_ROOT}" DIRECTORY)
get_filename_component(GO_ROOT "${GO_ROOT}" DIRECTORY)

# Find the go source root
set(go_path $ENV{GOPATH})
string(REPLACE "\\" "/" go_path "${go_path}")
string(REPLACE ":" ";" go_path "${go_path}")
if(go_path)
    list(GET go_path 0 go_base)
else()
    set(go_base ${ROOT_DIR})
endif()
if(NOT go_base)
    message(FATAL_ERROR "Could not determine GOPATH")
endif()
set(GO_PATH $ENV{GOPATH} CACHE PATH "The path below which the go code sits")

# Prepare our basic settings
get_filename_component(GO_SRC ${ROOT_DIR}/src ABSOLUTE)
get_filename_component(GO_PKG ${CMAKE_BINARY_DIR}/go/pkg ABSOLUTE)
get_filename_component(GO_DEPS ${CMAKE_BINARY_DIR}/go/deps ABSOLUTE)
get_filename_component(GO_TAGS ${CMAKE_BINARY_DIR}/go/tags ABSOLUTE)
get_filename_component(GO_BIN ${CMAKE_BINARY_DIR}/bin ABSOLUTE)
set(GO_COMPILE ${CMAKE_CURRENT_LIST_DIR}/GoCompile.cmake)
set(GO_BUILD_DEPS ${CMAKE_CURRENT_LIST_DIR}/GoDeps.cmake)
unset(go_path)
unset(go_base)

# Build the go environment variables
string(TOLOWER ${CMAKE_SYSTEM_NAME} GO_OS)
set(GO_ARCH amd64)

set(GO_ENV ${CMAKE_BINARY_DIR}/go/goenv.cmake)
file(WRITE ${GO_ENV} "
file(TO_NATIVE_PATH \"${GO_BIN}\" ENV{GOBIN})
file(TO_NATIVE_PATH \"${GO_ROOT}\" ENV{GOROOT})
file(TO_NATIVE_PATH \"${GO_PATH}\" ENV{GOPATH})
file(TO_NATIVE_PATH \"${GO_OS}\" ENV{GOOS})
file(TO_NATIVE_PATH \"${GO_ARCH}\" ENV{GOARCH})
set(ENV{CGO_LDFLAGS} \"-L${CMAKE_BINARY_DIR}\")
set(ENV{CGO_CFLAGS} \"-I${CMAKE_SOURCE_DIR}\")
set(CMAKE_Go_COMPILER ${CMAKE_Go_COMPILER})
set(GO_PKG ${GO_PKG})
")

if(NOT ToolchainTarget STREQUAL ${ToolchainHost})
    file(APPEND ${GO_ENV} "
set(ENV{CGO_ENABLED} 1)
set(ENV{CC} ${CMAKE_C_COMPILER})
set(ENV{CXX} ${CMAKE_CXX_COMPILER})
set(GO_EXTRA_ARGS \"-ldflags=\\\"-extld=${CMAKE_C_COMPILER}\\\"\")
")
endif()

# Find out which version of go we have
include(${GO_ENV})
execute_process(COMMAND "${CMAKE_Go_COMPILER}" version OUTPUT_VARIABLE go_version_output)
string(REGEX REPLACE "^.*go([0-9]+\\.[0-9]+)[^0-9].*$" "\\1" GO_VERSION ${go_version_output})
if (${GO_VERSION} STREQUAL ${go_version_output})
    message(STATUS "Could not determine Go verison, assuming 1.7+")
    set(GO_VERSION 1.7)
endif()
unset(go_version_output)

if(GO_VERSION VERSION_LESS 1.5)
    message(FATAL_ERROR "Go is too old, at least 1.5 needed")
    return()
elseif(GO_VERSION VERSION_LESS 1.6)
    message(STATUS "Older go version, enabling vendor experiment")
    file(APPEND ${GO_ENV} "
        set(ENV{GO15VENDOREXPERIMENT} 1)
    ")
    # re-include with modified settings
    include(${GO_ENV})
endif()

# configure variables set in this file for fast reload later on
configure_file(${CMAKE_CURRENT_LIST_DIR}/CMakeGoCompiler.cmake.in
    ${CMAKE_PLATFORM_INFO_DIR}/CMakeGoCompiler.cmake
    @ONLY
)
set(CMAKE_Go_COMPILER_ENV_VAR "GO_COMPILER")
