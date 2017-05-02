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

if(ANDROID OR GAPII_TARGET)
    return()
endif()

set(gtest_sources
    ${GTEST_DIR}/src/gtest-all.cc
    ${GTEST_DIR}/src/gtest_main.cc
)

set(gmock_sources
    ${GMOCK_DIR}/src/gmock-all.cc
)

if(NOT TARGET gmock)
    add_library(gmock STATIC ${gmock_sources})
    target_include_directories(gmock
        PRIVATE ${GMOCK_DIR} ${GTEST_DIR}
        PUBLIC ${GMOCK_DIR}/include ${GTEST_DIR}/include)
    target_link_libraries(gmock)
endif()

if(NOT TARGET gtest_main)
    add_library(gtest_main STATIC ${gtest_sources})
    target_include_directories(gtest_main
        PRIVATE ${GTEST_DIR}
        PUBLIC ${GTEST_DIR}/include)
    target_link_libraries(gtest_main PUBLIC gmock)
endif()

function(use_gtest TARGET)
    target_link_libraries(${TARGET} gtest_main)
    set_target_properties(${TARGET} PROPERTIES
        RUNTIME_OUTPUT_DIRECTORY ${CMAKE_TEST_OUTPUT_DIRECTORY}
    )
    add_test(NAME "${TARGET}" COMMAND ${TARGET})
endfunction(use_gtest)
