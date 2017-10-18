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

glob_all_dirs()


glob(sources
    PATH . gl ${PlatformSourcePath}
    INCLUDE ".cpp$"
    EXCLUDE "_test.cpp$"
)

if(GAPII_TARGET)
glob(sources
    PATH . gl posix armlinux
    INCLUDE ".cpp$"
    EXCLUDE "_test.cpp$"
)
endif()

glob(test_sources
    PATH .
    INCLUDE "_test.cpp$"
)

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    add_cmake_step(${abi} cc-core DEPENDS ${sources} ${android_files})
endforeach()

if(GAPII_PROJECT)
    add_cmake_step(${GAPII_PROJECT} cc-core DEPENDS ${sources})
endif()


if(NOT DISABLED_CXX)
    add_library(cc-core ${sources})
    target_link_libraries(cc-core cityhash)
    target_link_libraries(cc-core breakpad)

    if(ANDROID)
        target_link_libraries(cc-core log android)
    endif()

    if(LINUX)
        find_package(DLOpen REQUIRED)
        find_package(PThread REQUIRED)
        find_package(RT REQUIRED)
        if(GAPII_TARGET)
            target_link_libraries(cc-core DLOpen::Lib PThread::Lib RT::Lib)
        else()
            find_package(X11 REQUIRED)
            target_link_libraries(cc-core DLOpen::Lib PThread::Lib RT::Lib X11::Lib)
        endif()
    endif()

    if(WIN32)
        find_package(Winsock REQUIRED)
        target_link_libraries(cc-core Winsock::Lib)
    endif()

    if(NOT ANDROID AND NOT GAPII_TARGET)
        add_executable(cc-core-tests ${test_sources})
        use_gtest(cc-core-tests)
        target_link_libraries(cc-core-tests cc-core)
    endif()
endif()
