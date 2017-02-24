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

include_directories(${CMAKE_CURRENT_SOURCE_DIR})

glob(sources
  PATH .
  INCLUDE ".cpp$"
  EXCLUDE "_test.cpp$"
)

glob(test_sourcs
  PATH .
  INCLUDE "_test.cpp$"
)


foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    add_cmake_step(${abi} cc-memory-tracker DEPENDS ${sources} ${android_files})
endforeach()

if(NOT DISABLED_CXX AND NOT WIN32) # TODO: Windows build not currently supported
  add_library(cc-memory-tracker ${sources})

  if(ANDROID)
    find_package(NDK REQUIRED)
    find_package(STL REQUIRED)
    target_link_libraries(cc-memory-tracker NDK::Lib STL::Lib)
  endif()

  # TODO(qining): Add Windows support

  if(NOT ANDROID AND NOT WIN32)
    add_executable(cc-memory-tracker-tests ${test_sourcs})
    find_package(PThread REQUIRED)
    use_gtest(cc-memory-tracker-tests)
    target_link_libraries(cc-memory-tracker-tests PThread::Lib cc-memory-tracker)
  endif()
endif()
