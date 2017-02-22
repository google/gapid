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

if (TARGET langsvr)
    set(VSCODE_EXTENSIONS "${home}/.vscode/extensions")

    if (EXISTS ${VSCODE_EXTENSIONS})
        set(extdir "${VSCODE_EXTENSIONS}/gfxapi-0.0.1")

        find_program(npm npm DOC "The node package manager")
        if (NOT npm)
            message("* Install npm from https://nodejs.org/en/download/ to get gfxapi completion for vscode! *")
            return()
        endif()

        # Copy the extension files over to the gfxapi extension directory
        set(files "extension.js" "gfxapi.json" "gfxapi.configuration.json" "package.json")
        set(outputs)
        foreach(fil ${files})
            set(src "${CMAKE_CURRENT_SOURCE_DIR}/vscode/${fil}")
            set(dst "${extdir}/${fil}")
            add_custom_command(
                OUTPUT ${dst}
                COMMAND ${CMAKE_COMMAND} -E copy ${src} ${dst}
                DEPENDS ${src}
            )
            list(APPEND outputs ${dst})
        endforeach()

        # Copy the gfxapi langsvr to the gfxapi extension /bin sub-directory
        set(dst "${extdir}/bin/langsvr${CMAKE_HOST_EXECUTABLE_SUFFIX}")
        add_custom_command(
            OUTPUT ${dst}
            COMMAND ${CMAKE_COMMAND} -E copy "$<TARGET_FILE:langsvr>" ${dst}
            DEPENDS "langsvr"
        )
        list(APPEND outputs ${dst})

        # Run 'npm install' to fetch all the node dependencies.
        set(dst "${extdir}/node_modules")
        add_custom_command(
            OUTPUT ${dst}
            COMMAND ${npm} install
            WORKING_DIRECTORY "${extdir}"
            DEPENDS ${outputs}
        )
        list(APPEND outputs ${dst})

        add_custom_target(vscode ALL DEPENDS ${outputs})
    endif()
endif()