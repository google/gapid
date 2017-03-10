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

set(APIC_API_PATH ${CMAKE_SOURCE_DIR}/gapis/gfxapi CACHE PATH INTERNAL)
set(APIC_TEMPLATE_PATH ${CMAKE_SOURCE_DIR}/gapis/gfxapi/templates CACHE PATH INTERNAL)

function(apic api)
    if(DISABLED_CODE_GENERATION OR NOT TARGET apic)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    cmake_parse_arguments(APIC "" "PATH;TEMPLATE" "INPUTS;OUTPUTS;DEFINES" ${ARGN})
    required(APIC_TEMPLATE "for apic")
    default(APIC_PATH ".")
    get_filename_component(api_name ${api} DIRECTORY)
    if (IS_ABSOLUTE ${APIC_TEMPLATE})
        get_filename_component(tmpl ${APIC_TEMPLATE} NAME)
        set(name ${name}_${api_name}_${tmpl})
    else()
        set(name ${name}_${api_name}_${APIC_TEMPLATE})
        abs_list(APIC_TEMPLATE ${APIC_TEMPLATE_PATH})
    endif()

    set(deps ${CMAKE_CURRENT_BINARY_DIR}/api/${name}.deps.cmake)
    set(tag ${CMAKE_CURRENT_BINARY_DIR}/api/${name}.tag)
    abs_list(api ${APIC_API_PATH})
    abs_list(APIC_PATH ${CMAKE_CURRENT_SOURCE_DIR})
    set(APIC_INPUTS ${api} ${APIC_TEMPLATE} ${APIC_INPUTS})
    abs_list(APIC_INPUTS ${CMAKE_CURRENT_SOURCE_DIR})
    if(NOT APIC_OUTPUTS)
        get_filename_component(APIC_OUTPUTS ${APIC_TEMPLATE} NAME)
        string(REGEX REPLACE "\\.tmpl$" "" APIC_OUTPUTS "${APIC_OUTPUTS}")
        string(REGEX REPLACE "^api" "${api_name}" APIC_OUTPUTS "${APIC_OUTPUTS}")
    endif()
    abs_list(APIC_OUTPUTS ${CMAKE_CURRENT_SOURCE_DIR})
    set(apic_args template
        --cmake ${deps}
        --dir ${APIC_PATH}
    )
    if(APIC_DEFINES)
        set(apic_args ${apic_args} -G ${APIC_DEFINES})
    endif()
    set(apic_args ${apic_args} "${api}" ${APIC_TEMPLATE})
    if(NOT EXISTS ${deps})
        # Ugly hacks to make intial dependancies not totally broken...
        set(APIC_INPUTS ${APIC_INPUTS}
            "${GO_SRC}/github.com/google/gapid/gapis/messages/messages.api")
        file(WRITE ${deps} "
            set(api_inputs ${APIC_INPUTS})
            set(api_outputs ${APIC_OUTPUTS})
        ")
    endif()
    set(apic_args ${apic_args} "--gopath" "${GO_PATH}")
    include(${deps})
    add_custom_command(
        OUTPUT ${tag} ${api_outputs}
        COMMAND apic  ${apic_args}
        COMMAND "${CMAKE_COMMAND}" -E touch ${tag}
        DEPENDS apic ${api_inputs}
        COMMENT "Apic on ${api} with ${name}"
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )
    set_source_files_properties(${api_outputs} PROPERTIES GENERATED TRUE)
    all_target(apic ${name} ${tag})
endfunction(apic)
