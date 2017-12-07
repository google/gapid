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

set(sources "${CMAKE_CURRENT_SOURCE_DIR}/cc/main.cpp")


if(MSVC_GAPIR AND WIN32)
    add_cmake_target("gapir-msvc" gapir "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}" "gapir.exe"
        DEPENDEES gapir_static
        DEPENDS ${sources}
        DESTINATION ${TARGET_INSTALL_PATH}
    )
    ExternalProject_Add_Step("gapir-msvc" dump_syms
        COMMAND "${CMAKE_CURRENT_SOURCE_DIR}/../../third_party/breakpad/src/tools/windows/binaries/dump_syms.exe" "<BINARY_DIR>/bin/gapir.pdb" > "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.sym"
        DEPENDEES gapir
        BYPRODUCTS "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.sym"
    )
    return()
endif()

if(NOT DISABLED_CXX)
    if(ANDROID)
        get_filename_component(glue "${ANDROID_NDK_ROOT}/sources/android/native_app_glue" ABSOLUTE)
        add_library(gapir SHARED ${sources} "${glue}/android_native_app_glue.c")
        target_include_directories(gapir PRIVATE "${glue}")
    elseif(NOT GAPII_TARGET)
        add_executable(gapir ${sources})
        install(TARGETS gapir DESTINATION ${TARGET_INSTALL_PATH})
    endif()
    if(NOT GAPII_TARGET)
        target_link_libraries(gapir gapir_static)
        target_compile_options(gapir PUBLIC "-g")
        if(APPLE)
            add_custom_command(
                TARGET gapir
                POST_BUILD
                COMMAND "dsymutil" "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gapir" -o "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.dSYM"
                COMMAND dump_syms -g "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.dSYM" "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gapir" > "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.sym"
                COMMAND "rm" "-r" "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.dSYM"
                COMMAND "strip" "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gapir"
            )
        elseif(NOT WIN32 AND NOT ANDROID)
            add_custom_command(
                TARGET gapir
                POST_BUILD
                COMMAND dump_syms "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gapir" > "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/../gapir.sym"
                COMMAND "strip" "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gapir"
            )
        endif()
    endif()
endif()
