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

glob(sources
    PATH .
    INCLUDE ".cpp$"
    EXCLUDE "_test.cpp$"
)

glob(sources
    PATH .
)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
    add_cmake_target(${abi} VkLayer_VirtualSwapchain ${dst} "libVkLayer_VirtualSwapchain.so"
        DEPENDS ${sources}
        DEPENDEES gapii
        DESTINATION "android/${ANDROID_ABI_PATH_${abi}}"
    )
endforeach()

if(NOT DISABLED_CXX)
    add_library(VkLayer_VirtualSwapchain SHARED ${sources})
    target_compile_options(VkLayer_VirtualSwapchain PRIVATE -fno-exceptions -fno-rtti)

    if(LINUX)
        find_package(PThread REQUIRED)
        target_link_libraries(VkLayer_VirtualSwapchain PThread::Lib)
    endif()

    if(ANDROID)
        target_compile_definitions(VkLayer_VirtualSwapchain PRIVATE "-DVK_USE_PLATFORM_ANDROID_KHR")
        find_package(NDK REQUIRED)
        find_package(STL REQUIRED)
        target_link_libraries(VkLayer_VirtualSwapchain STL::Lib NDK::Lib)
    endif()

    if(NOT ANDROID)
        add_custom_command(TARGET VkLayer_VirtualSwapchain POST_BUILD
            COMMAND ${CMAKE_COMMAND} -E copy_if_different
                ${CMAKE_CURRENT_SOURCE_DIR}/VirtualSwapchainLayer.json
                $<TARGET_FILE_DIR:VkLayer_VirtualSwapchain>)
        install(FILES $<TARGET_FILE_DIR:VkLayer_VirtualSwapchain>/VirtualSwapchainLayer.json
            DESTINATION ${TARGET_INSTALL_PATH})
        install(TARGETS VkLayer_VirtualSwapchain DESTINATION ${TARGET_INSTALL_PATH})
    endif()

    if(UNIX)
        target_compile_definitions(VkLayer_VirtualSwapchain PRIVATE "-DVK_USE_PLATFORM_XCB_KHR")
    endif()
endif()
