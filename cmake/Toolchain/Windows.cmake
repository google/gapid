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

set(TARGET_OS WINDOWS)

if(ToolchainTarget STREQUAL ${ToolchainHost})
    set(MSYS2_PATH "" CACHE PATH "Path to the msys2 installation directory")
    if(MSYS2_PATH)
        set(MINGW_PATH "${MSYS2_PATH}/mingw64")
        set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -Wa,-mbig-obj")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wa,-mbig-obj")
    elseif(IS_DIRECTORY ${MSVC_PATH})
        include_directories("${MSVC_PATH}/include")
        set(CMAKE_C_FLAGS "/wd4267 /wd4996 /D__x86_64 /Zi")
        set(CMAKE_CXX_FLAGS "/wd4267 /wd4996 /D__x86_64 /Zi")
    else()
        message(FATAL_ERROR "MSYS2_PATH not set!")
    endif()

else()
    # Cross compiling
    message(FATAL_ERROR "Cross compiling to windows was only supported with PREBUILTS")
    if(NOT ToolchainHost STREQUAL "Linux")
        message(FATAL_ERROR "Cross compiling to windows only supported on Linux")
    endif()

    set(CMAKE_SYSTEM_NAME "Windows")
    set(CMAKE_LIBRARY_ARCHITECTURE "x86_64-w64-mingw32")
    set(MINGW_PATH "${PREBUILTS}/gcc/linux-x86/host/x86_64-w64-mingw32-4.8")
    list(APPEND CMAKE_LIBRARY_PATH "${MINGW_PATH}/x86_64-w64-mingw32/lib")
    list(APPEND CMAKE_PREFIX_PATH "${PREBUILTS}/go/linux-x86")
endif()

if(MINGW_PATH)
    set(CMAKE_C_COMPILER "${MINGW_PATH}/bin/x86_64-w64-mingw32-gcc${CMAKE_HOST_EXECUTABLE_SUFFIX}")
    set(CMAKE_CXX_COMPILER "${MINGW_PATH}/bin/x86_64-w64-mingw32-g++${CMAKE_HOST_EXECUTABLE_SUFFIX}")
endif()
