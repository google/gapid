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

glob(android_sources
    PATH .
)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
    add_cmake_target(${abi} VkLayerGraphicsSpy ${dst} "libVkLayerGraphicsSpy.so"
        DEPENDS ${android_sources}
        DEPENDEES gapii
    )
endforeach()

if(NOT DISABLED_CXX)
    add_library(VkLayerGraphicsSpy SHARED ${sources})
    target_compile_options(VkLayerGraphicsSpy PRIVATE -fno-exceptions -fno-rtti)

    if(LINUX)
        find_package(DLOpen REQUIRED)
        target_link_libraries(VkLayerGraphicsSpy DLOpen::Lib)
    endif()

    if(ANDROID)
        find_package(NDK REQUIRED)
        target_link_libraries(VkLayerGraphicsSpy NDK::Lib)
    endif()

    if(NOT ANDROID)
        add_custom_command(TARGET VkLayerGraphicsSpy POST_BUILD
            COMMAND "${CMAKE_COMMAND}" -E copy_if_different
                ${CMAKE_CURRENT_SOURCE_DIR}/GraphicsSpyLayer.json
                $<TARGET_FILE_DIR:VkLayerGraphicsSpy>)
        install(FILES $<TARGET_FILE_DIR:VkLayerGraphicsSpy>/GraphicsSpyLayer.json
            DESTINATION "${TARGET_INSTALL_PATH}/lib")
    endif()

endif()
