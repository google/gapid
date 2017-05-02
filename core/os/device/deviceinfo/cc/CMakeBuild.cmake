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

if(GAPII_TARGET)
    return()
endif()

glob_all_dirs()

glob(sources
    PATH . ${PlatformSourcePath}
    INCLUDE ".cpp$" ".mm$"
    EXCLUDE "_test.cpp$"
)

glob(android_sources
    PATH . "android"
    INCLUDE ".cpp$"
    EXCLUDE "_test.cpp$"
)

list(APPEND sources
    "${PROTO_CC_OUT}/core/os/device/device.pb.cc"
)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
    add_cmake_target(${abi} deviceinfo ${dst} "libdeviceinfo.so"
        DEPENDS ${android_sources}
    )
endforeach()

if(NOT DISABLED_CXX)
    add_library(deviceinfo-static STATIC ${sources})

    target_include_directories(deviceinfo-static PUBLIC "${PROTO_CC_OUT}")
    target_include_directories(deviceinfo-static PUBLIC "${CMAKE_SOURCE_DIR}/external/protobuf/src")

    find_package(GL REQUIRED)
    target_link_libraries(deviceinfo-static cc-core protobuf cityhash GL::Lib)

    if(ANDROID)
        find_package(NDK REQUIRED)
        find_package(Log REQUIRED)
        find_package(STL REQUIRED)

        target_link_libraries(deviceinfo-static NDK::Lib Log::Lib STL::Lib)

        add_library(deviceinfo SHARED ${sources})
        target_include_directories(deviceinfo PUBLIC "${PROTO_CC_OUT}")
        target_include_directories(deviceinfo PUBLIC "${CMAKE_SOURCE_DIR}/external/protobuf/src")
        target_link_libraries(deviceinfo cc-core protobuf cityhash)
    endif()
endif()
