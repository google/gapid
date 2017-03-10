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

# TODO: Debug builds when CMAKE_BUILD_TYPE=Debug

set(RUN_GRADLE ${CMAKE_CURRENT_LIST_DIR}/RunGradle.cmake)
set(GRADLE_ENV ${CMAKE_BINARY_DIR}/gradleenv.cmake)


set(ANDROID_HOME $ENV{ANDROID_HOME} CACHE PATH "The root path to the Android SDK")
set(JAVA_HOME $ENV{JAVA_HOME} CACHE PATH "The root path to the JDK")
#TODO: should we try to find an installed SDK here?

file(WRITE ${GRADLE_ENV} "
set(ENV{ANDROID_HOME} \"${ANDROID_HOME}\")
set(ENV{JAVA_HOME} \"${JAVA_HOME}\")
")


# gradle adds a target called name that runs gradle in the current source
# directory.
function (gradle name)
    cmake_parse_arguments(GRADLE "" "" "OUTPUT;DEPENDS;DIRECTORY;ARGS" ${ARGN})
    required(GRADLE_OUTPUT "for gradle")
    required(GRADLE_DEPENDS "for gradle")
    set(args ${GRADLE_ARGS})
    if(GRADLE_OFFLINE)
        list(APPEND args "--offline")
    endif()
    if (NOT GRADLE_DIRECTORY)
        set(GRADLE_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})
    endif()
    add_custom_command(
        OUTPUT ${GRADLE_OUTPUT}
        COMMAND "${CMAKE_COMMAND}" -DGRADLE_ENV=${GRADLE_ENV} -DGRADLE_ARGS="${args}" -P ${RUN_GRADLE}
        DEPENDS ${GRADLE_DEPENDS}
        WORKING_DIRECTORY ${GRADLE_DIRECTORY}
        COMMENT "Gradle: ${name}"
        )
    add_custom_target(${name} ALL DEPENDS ${GRADLE_OUTPUT})
endfunction(gradle)
