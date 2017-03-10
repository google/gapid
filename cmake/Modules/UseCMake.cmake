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

include(ExternalProject)

set(CMAKE_RUN_SCRIPT "${CMAKE_CURRENT_LIST_DIR}/RunCMakeWithLock.cmake")

function (add_cmake proj src)
    ExternalProject_Add(${proj}
        PREFIX ${proj}
        SOURCE_DIR ${src}
        CMAKE_ARGS ""
        CMAKE_CACHE_ARGS
        "${ARGN}"
        CMAKE_CACHE_DEFAULT_ARGS
        "-DPYTHON_EXECUTABLE:PATH=${PYTHON_EXECUTABLE}"
        "-DPROTO_CC_OUT:PATH=${PROTO_CC_OUT}"
        "-DCMAKE_BUILD_TYPE:STRING=${CMAKE_BUILD_TYPE}"
        "-DCMAKE_MAKE_PROGRAM:STRING=${CMAKE_MAKE_PROGRAM}"
        "-DANDROID_NDK_ROOT:STRING=${ANDROID_NDK_ROOT}"
        "-DANDROID_HOME:STRING=${ANDROID_HOME}"
        "-DJAVA_HOME:STRING=${JAVA_HOME}"
        "-DCMAKE_Go_COMPILER:PATH=${CMAKE_Go_COMPILER}"

        DOWNLOAD_COMMAND ""
        UPDATE_COMMAND ""
        PATCH_COMMAND ""
        BUILD_COMMAND ""
        INSTALL_COMMAND ""
        TEST_COMMAND ""
    )
endfunction(add_cmake)

function (add_cmake_step proj tgt)
    ExternalProject_Add_Step(${proj} ${tgt}
        COMMAND "${CMAKE_COMMAND}" "-DRUN_CMAKE_DIR=<BINARY_DIR>" "-DRUN_CMAKE_TARGET=${tgt}" "-P" "${CMAKE_RUN_SCRIPT}"
        DEPENDEES configure
        ${ARGN}
    )
endfunction (add_cmake_step)

function (add_cmake_target proj tgt dst name)
    cmake_parse_arguments(ADD_CMAKE "" "DESTINATION;SOURCE_PATH" "" ${ARGN})
    if (NOT ADD_CMAKE_SOURCE_PATH)
        set(ADD_CMAKE_SOURCE_PATH bin/${name})
    endif()

    ExternalProject_Get_Property(${proj} PREFIX BINARY_DIR)
    set(built ${BINARY_DIR}/${ADD_CMAKE_SOURCE_PATH})
    set(fulldst ${dst}/${name})
    add_cmake_step(${proj} ${tgt}
        BYPRODUCTS ${built}
        ${ADD_CMAKE_UNPARSED_ARGUMENTS}
    )
    add_custom_command(
        COMMAND "${CMAKE_COMMAND}"
        -E copy ${built} ${fulldst}
        DEPENDS ${built} ${proj}
        OUTPUT ${fulldst}
    )
    add_custom_target("${proj}-${tgt}" ALL DEPENDS ${fulldst})
    if(ADD_CMAKE_DESTINATION)
        install(FILES ${fulldst} DESTINATION ${ADD_CMAKE_DESTINATION})
    endif()
endfunction (add_cmake_target)
