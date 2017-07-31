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

function(embed)
    if(DISABLED_CODE_GENERATION OR NOT TARGET embed)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    cmake_parse_arguments(EMBED "WEB" "OUTPUT;PACKAGE" "" ${ARGN})
    if(EMBED_WEB)
        set(EMBED_WEB "--web")
    else()
        set(EMBED_WEB "")
    endif()
    default(EMBED_OUTPUT "${name}_embed.go")
    default(EMBED_PACKAGE ${name})
    required(EMBED_UNPARSED_ARGUMENTS "for embed")
    set(EMBED_OUTPUT ${CMAKE_CURRENT_SOURCE_DIR}/${EMBED_OUTPUT})
    abs_list(EMBED_UNPARSED_ARGUMENTS ${CMAKE_CURRENT_SOURCE_DIR})
    add_custom_command(
        OUTPUT ${EMBED_OUTPUT}
        COMMAND embed --log-level Warning --root ${CMAKE_CURRENT_SOURCE_DIR} --out ${EMBED_OUTPUT} --package ${EMBED_PACKAGE} ${EMBED_WEB} ${EMBED_UNPARSED_ARGUMENTS}
        DEPENDS embed ${EMBED_UNPARSED_ARGUMENTS}
        COMMENT "Embed generating ${EMBED_OUTPUT}"
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )
    all_target(embed ${name} ${EMBED_OUTPUT})
    set_source_files_properties(${EMBED_OUTPUT} PROPERTIES GENERATED TRUE)
endfunction(embed)
