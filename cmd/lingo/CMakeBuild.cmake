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

function(lingo)
    if(DISABLED_CODE_GENERATION OR NOT TARGET lingo)
        return()
    endif()

    get_filename_component(name ${CMAKE_CURRENT_SOURCE_DIR} NAME)
    set(output)
    glob(lingofiles INCLUDE "\.lingo$")
    foreach(in ${lingofiles})
        string(REPLACE ".lingo" ".go" out "${in}")
        list(APPEND output ${out})
    endforeach()
    add_custom_command(
        OUTPUT ${output}
        COMMAND lingo ARGS ${lingofiles}
        DEPENDS lingo ${lingofiles}
        COMMENT "Lingo for ${name}"
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )
    all_target(lingo ${name} ${output})
endfunction(lingo)

