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

go_install()

function(annotate api)
    if(DISABLED_CODE_GENERATION OR NOT TARGET annotate)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    cmake_parse_arguments(ANNOTATE "" "PATH" "" ${ARGN})
    default(ANNOTATE_PATH ".")
    abs_list(ANNOTATE_PATH ${CMAKE_CURRENT_SOURCE_DIR})
    abs_list(api ${APIC_API_PATH})
    set(generate
        --base64 ${ANNOTATE_PATH}/snippets.base64
        --text ${ANNOTATE_PATH}/snippets.text
        --globals_base64 ${ANNOTATE_PATH}/globals_snippets.base64
        --globals_text ${ANNOTATE_PATH}/globals_snippets.text
    )
    set(output)
    foreach(arg ${generate})
        if(NOT ${arg} MATCHES "^--")
            list(APPEND output "${arg}")
        endif()
    endforeach()
    add_custom_command(
        OUTPUT ${output}
        COMMAND annotate ${generate} ${api}
        DEPENDS annotate ${api} "${CMAKE_CURRENT_SOURCE_DIR}/${name}_binary.go" 
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        COMMENT "Annotate ${name} using ${api}"
    )
    all_target(annotate ${name} ${output})
endfunction(annotate)
