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

function(codergen)
    if(DISABLED_CODE_GENERATION OR NOT TARGET codergen)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    set(signatures ${CMAKE_CURRENT_SOURCE_DIR}/signatures.txt)
    file(RELATIVE_PATH package ${GO_SRC} ${CMAKE_CURRENT_SOURCE_DIR})
    set(inputs)
    set(outputs)
    foreach(pkg ${ARGN})
        go_glob("${CMAKE_CURRENT_SOURCE_DIR}/${pkg}")
        foreach(fil ${gofiles})
            if(fil MATCHES "_embed.go")
                # ignore embed generated files
            elseif(fil MATCHES "_binary.go")
                list(APPEND outputs ${fil})
            else()
                list(APPEND inputs ${fil})
            endif()
        endforeach()
        foreach(fil ${testfiles})
            if(fil MATCHES "_binary_test.go")
                list(APPEND outputs ${fil})
            else()
                list(APPEND inputs ${fil})
            endif()
        endforeach()
    endforeach()
    add_custom_command(
        OUTPUT ${signatures} ${outputs}
        COMMAND codergen
        --gopath "${GO_PATH}"
        --signatures ${signatures}
        --go
        -java ${JAVA_BASE}
        ${package}/...
        DEPENDS codergen ${inputs}
        COMMENT "Codergen for ${name}"
    )
    set_source_files_properties(${outputs} PROPERTIES GENERATED TRUE)
    all_target(codergen ${name} ${signatures})
endfunction(codergen)
