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

embed(PACKAGE main stringdefgo.tmpl stringdefapi.tmpl)

go_install()

function(stringgen)
    if(NOT TARGET stringgen OR CMAKE_CROSSCOMPILING)
        return()
    endif()
    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    cmake_parse_arguments(STRINGGEN "" "INPUT;OUTPUT_GO;OUTPUT_API;PACKAGE" "" ${ARGN})
    required(STRINGGEN_INPUT "for stringgen")
    required(STRINGGEN_OUTPUT_GO "for stringgen")
    required(STRINGGEN_OUTPUT_API "for stringgen")
    required(STRINGGEN_PACKAGE "for stringgen")
    set(STRINGGEN_OUTPUT_GO ${CMAKE_CURRENT_SOURCE_DIR}/${STRINGGEN_OUTPUT_GO})
    set(STRINGGEN_OUTPUT_API ${CMAKE_CURRENT_SOURCE_DIR}/${STRINGGEN_OUTPUT_API})
    set(STRINGGEN_INPUT ${CMAKE_CURRENT_SOURCE_DIR}/${STRINGGEN_INPUT})
    add_custom_command(
        OUTPUT ${STRINGGEN_OUTPUT_GO} ${STRINGGEN_OUTPUT_API}
        COMMAND stringgen
        --log-level Warning
        --def-go ${STRINGGEN_OUTPUT_GO}
        --def-api ${STRINGGEN_OUTPUT_API}
        --pkg ${STRINGGEN_PACKAGE}
        ${STRINGGEN_INPUT}
        DEPENDS stringgen ${STRINGGEN_INPUT}
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        COMMENT "stringgen using ${STRINGGEN_INPUT}"
    )
    all_target(stringgen ${name} ${STRINGGEN_OUTPUT_GO})
    all_target(stringgen ${name} ${STRINGGEN_OUTPUT_API})
endfunction(stringgen)
