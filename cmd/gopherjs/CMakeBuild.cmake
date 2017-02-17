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

go_install(LINK_PACKAGE "github.com/gopherjs/gopherjs")

function(gopherjs)
    if(DISABLE_GOPHERJS OR NOT TARGET gopherjs)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)

    cmake_parse_arguments(GOPHERJS "MINIFY" "OUTPUT;PACKAGE" "" ${ARGN})

    required(GOPHERJS_OUTPUT "gopherjsjs output file")
    file(RELATIVE_PATH current_package ${GO_SRC} ${CMAKE_CURRENT_SOURCE_DIR})
    default(GOPHERJS_PACKAGE ${current_package})
    set(GOPHERJS_OUTPUT ${CMAKE_CURRENT_SOURCE_DIR}/${GOPHERJS_OUTPUT})
    get_filename_component(GOPHERJS_OUTPUT ${GOPHERJS_OUTPUT} ABSOLUTE)

    set(GOPHERJS_OPTS "")
    if( ${GOPHERJS_MINIFY} )
        set(GOPHERJS_OPTS ${GOPHERJS_OPTS} "--minify")
    endif()

    add_custom_command(
        OUTPUT ${GOPHERJS_OUTPUT}
        COMMAND ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/gopherjs build ${GOPHERJS_OPTS} --output ${GOPHERJS_OUTPUT} ${GOPHERJS_PACKAGE}
        DEPENDS gopherjs ${GOPHERJS_UNPARSED_ARGUMENTS}
        COMMENT "Building ${GOPHERJS_OUTPUT}"
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )
    all_target(gopherjs ${name} ${GOPHERJS_OUTPUT})
    set_source_files_properties(${GOPHERJS_OUTPUT} PROPERTIES GENERATED TRUE)
endfunction(gopherjs)

