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

set(api core/core.api)

apic(${api} TEMPLATE api_imports.h.tmpl)
apic(${api} TEMPLATE api_spy.h.tmpl)
apic(${api} TEMPLATE api_spy.cpp.tmpl)
apic(${api} TEMPLATE api_types.cpp.tmpl)
apic(${api} TEMPLATE api_types.h.tmpl)

set(api gles/gles.api)

apic(${api} TEMPLATE ${APIC_API_PATH}/gles/templates/api_exports.cpp.tmpl)
apic(${api} TEMPLATE ${APIC_API_PATH}/gles/templates/api_imports.cpp.tmpl)
apic(${api} TEMPLATE api_imports.h.tmpl)
apic(${api} TEMPLATE api_spy.cpp.tmpl)
apic(${api} TEMPLATE api_spy.h.tmpl)
apic(${api} TEMPLATE api_types.cpp.tmpl)
apic(${api} TEMPLATE api_types.h.tmpl)

apic(${api} PATH windows TEMPLATE opengl32_exports.def.tmpl)
apic(${api} PATH windows TEMPLATE opengl32_resolve.cpp.tmpl)
apic(${api} PATH windows TEMPLATE opengl32_x64.asm.tmpl)

apic(${api} PATH osx TEMPLATE opengl_framework_exports.cpp.tmpl)

set(api vulkan/vulkan.api)

apic(${api} TEMPLATE ${APIC_API_PATH}/vulkan/templates/api_exports.cpp.tmpl)
apic(${api} TEMPLATE ${APIC_API_PATH}/vulkan/templates/api_imports.cpp.tmpl)
apic(${api} TEMPLATE ${APIC_API_PATH}/vulkan/templates/vk_spy_helpers.cpp.tmpl)
apic(${api} TEMPLATE api_imports.h.tmpl)
apic(${api} TEMPLATE api_spy.h.tmpl)
apic(${api} TEMPLATE api_spy.cpp.tmpl)
apic(${api} TEMPLATE api_types.cpp.tmpl)
apic(${api} TEMPLATE api_types.h.tmpl)

glob_all_dirs()

glob(sources
    PATH . ${PlatformSourcePath}
    INCLUDE ".cpp$"
    EXCLUDE "_test.cpp$"
)

list(APPEND sources
    "${PROTO_CC_OUT}/core/data/pack/pack.pb.cc"
    "${PROTO_CC_OUT}/core/os/device/device.pb.cc"
    "${PROTO_CC_OUT}/gapis/atom/atom_pb/atom.pb.cc"
    "${PROTO_CC_OUT}/gapis/memory/memory.pb.cc"
    "${PROTO_CC_OUT}/gapis/memory/memory_pb/memory.pb.cc"
    "${PROTO_CC_OUT}/gapis/capture/capture.pb.cc"
    "${PROTO_CC_OUT}/gapis/gfxapi/core/core_pb/api.pb.cc"
    "${PROTO_CC_OUT}/gapis/gfxapi/vulkan/vulkan_pb/api.pb.cc"
    "${PROTO_CC_OUT}/gapis/gfxapi/gles/gles_pb/api.pb.cc"
    "${PROTO_CC_OUT}/gapis/gfxapi/gles/gles_pb/extras.pb.cc"
)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
    add_cmake_target(${abi} gapii ${dst} "libgapii.so"
        DEPENDEES llvm-interceptor
        DEPENDS ${sources} ${android_files} cc-memory-tracker
    )
endforeach()

if(GAPII_PROJECT)
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${GAPII_PROJECT}")
    add_cmake_target(${GAPII_PROJECT} gapii ${dst} "libgapii.so"
        DEPENDEES cc-core
        DEPENDS ${sources} cc-memory-tracker
    )
endif()

if(NOT DISABLED_CXX)
    add_library(gapii SHARED ${sources})
    target_link_libraries(gapii cc-memory-tracker deviceinfo-static protobuf)

    target_include_directories(gapii PUBLIC "${PROTO_CC_OUT}")
    target_include_directories(gapii PUBLIC "${CMAKE_SOURCE_DIR}/external/protobuf/src")

    if(APPLE)
        find_package(Cocoa REQUIRED)
        target_link_libraries(gapii Cocoa::Lib)
    endif()

    if(ANDROID)
        find_package(EGL REQUIRED)
        target_link_libraries(gapii EGL::Lib)

        set_target_properties(gapii PROPERTIES
          LINK_FLAGS "-Wl,--version-script,${CMAKE_CURRENT_SOURCE_DIR}/gapii.exports")
    elseif(GAPII_TARGET)

    else()
        find_package(GL REQUIRED)
        target_link_libraries(gapii GL::Lib)

        if(WIN32)
            install(TARGETS gapii RUNTIME DESTINATION "${TARGET_INSTALL_PATH}/lib")
        else()
            install(TARGETS gapii LIBRARY DESTINATION "${TARGET_INSTALL_PATH}/lib")
        endif()
    endif()
endif()
