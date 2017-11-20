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

if(NOT DISABLED_CXX)
    if(ANDROID)
        get_filename_component(glue "${ANDROID_NDK_ROOT}/sources/android/native_app_glue" ABSOLUTE)
        add_library(gapir SHARED ${sources} "${glue}/android_native_app_glue.c")
        target_include_directories(gapir PRIVATE "${glue}")
    elseif(NOT GAPII_TARGET)
        add_executable(gapir ${sources})
        install(TARGETS gapir DESTINATION ${TARGET_INSTALL_PATH})
    endif()
    if(CMAKE_BUILD_TYPE MATCHES Release)
        target_compile_options(gapir PRIVATE "-g")
    endif()
    if(NOT GAPII_TARGET)
        target_link_libraries(gapir gapir_static)
    endif()
endif()
