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

# glob(test_sources
#     PATH .
#     INCLUDE "_test.cpp$"
# )

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")
    add_cmake_target(${abi} deviceinfo ${dst} "libdeviceinfo.so"
        DEPENDS ${android_sources}
    )
endforeach()

if(NOT DISABLED_CXX)
    if(ANDROID)
        add_library(deviceinfo SHARED ${sources})
    else()
        add_library(deviceinfo STATIC ${sources})
    endif()

    target_include_directories(deviceinfo PUBLIC "${PROTO_CC_OUT}")
    target_include_directories(deviceinfo PUBLIC "${CMAKE_SOURCE_DIR}/external/protobuf/src")

    find_package(GL REQUIRED)
    target_link_libraries(deviceinfo protobuf cityhash GL::Lib)

    if(ANDROID)
        find_package(EGL REQUIRED)
        find_package(NDK REQUIRED)
        find_package(Log REQUIRED)
        find_package(STL REQUIRED)
        target_link_libraries(deviceinfo EGL::Lib NDK::Lib Log::Lib STL::Lib)
    endif()

    # if(LINUX)
    #     find_package(DLOpen REQUIRED)
    #     find_package(PThread REQUIRED)
    #     find_package(RT REQUIRED)
    #     find_package(X11 REQUIRED)
    #     target_link_libraries(deviceinfo DLOpen::Lib PThread::Lib RT::Lib X11::Lib)
    # endif()
    #
    # if(WIN32)
    #     target_link_libraries(deviceinfo)
    # endif()

    # if(NOT ANDROID)
    #     add_executable(deviceinfo-tests ${test_sources})
    #     use_gtest(deviceinfo-tests)
    #     target_link_libraries(deviceinfo-tests deviceinfo)
    # endif()
endif()
