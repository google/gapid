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

set(gles gles/gles.api)
set(vulkan vulkan/vulkan.api)

if(NOT MSVC)
    apic(${gles} PATH ${cc_gapir} TEMPLATE specific_gfx_api.cpp.tmpl OUTPUTS gles_gfx_api.cpp)
    apic(${gles} PATH ${cc_gapir} TEMPLATE specific_gfx_api.h.tmpl OUTPUTS gles_gfx_api.h)

    apic(${vulkan} PATH ${cc_gapir} TEMPLATE specific_gfx_api.cpp.tmpl OUTPUTS vulkan_gfx_api.cpp)
    apic(${vulkan} PATH ${cc_gapir} TEMPLATE specific_gfx_api.h.tmpl OUTPUTS vulkan_gfx_api.h)
    apic(${vulkan} PATH ${cc_gapir} TEMPLATE vulkan_gfx_api_extras.tmpl OUTPUTS vulkan_gfx_api_extras.cpp)
endif()

glob_all_dirs()

glob(sources
    PATH . ${PlatformSourcePath}
    INCLUDE ".cpp$" ".mm$"
    EXCLUDE "_test.cpp$"
)

glob(test_sources
    PATH .
    INCLUDE "_test.cpp$"
)

if(NOT MSVC)
    foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
        set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
        add_cmake_target(${abi} gapir ${dst} "libgapir.so"
            DEPENDEES cc-core
            DEPENDS ${sources} ${android_files}
        )
    endforeach()
endif()

if(MSVC_GAPIR AND WIN32)
    add_cmake_step("gapir-msvc" gapir_static DEPENDEES cc-core breakpad DEPENDS ${sources})
    return()
endif()

if(NOT DISABLED_CXX)
    add_library(gapir_static STATIC  ${sources})
    set_target_properties(gapir_static PROPERTIES OUTPUT_NAME gapir)
    target_link_libraries(gapir_static cc-core)

    if(APPLE)
        find_package(Cocoa REQUIRED)
        target_link_libraries(gapir_static Cocoa::Lib)
    endif()

    if(ANDROID)
        find_package(EGL REQUIRED)
        target_link_libraries(gapir_static EGL::Lib)
    else()
        find_package(GL REQUIRED)
        target_link_libraries(gapir_static GL::Lib)

        add_executable(gapir-tests ${test_sources})
        use_gtest(gapir-tests)
        target_link_libraries(gapir-tests gapir_static)
    endif()
endif()
