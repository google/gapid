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

# Build behaviour forcing, set on the cmake line
set(FORCING_GLOB ${FORCE_GLOB})
set(FORCE_GLOB OFF CACHE BOOL "Force regeneration of glob files" FORCE)

set(GLOB_EXTENSIONS
    .go .pb.go
    .cxx .cpp .h .c .cc
    .mm
    .lingo
    .proto
)

set(GLOB_TEMPLATE_FILE "${CMAKE_CURRENT_LIST_FILE}.in")

function(glob VAR)
    cmake_parse_arguments(GLOB "" "" "PATH;INCLUDE;EXCLUDE" ${ARGN})
    default(GLOB_PATH ${CMAKE_CURRENT_SOURCE_DIR})
    set(result)
    _glob_list(result files "${GLOB_PATH}" "${GLOB_INCLUDE}" "${GLOB_EXCLUDE}")
    set(${VAR} ${result} PARENT_SCOPE)
endfunction()

function(glob_dirs VAR)
    cmake_parse_arguments(GLOB "" "" "PATH;INCLUDE;EXCLUDE" ${ARGN})
    default(GLOB_PATH ${CMAKE_CURRENT_SOURCE_DIR})
    set(result)
    _glob_list(result dirs "${GLOB_PATH}" "${GLOB_INCLUDE}" "${GLOB_EXCLUDE}")
    set(${VAR} ${result} PARENT_SCOPE)
endfunction()

function(_glob_list VAR list paths include exclude)
    set(result)
    foreach(path ${paths})
        if(NOT IS_ABSOLUTE "${path}")
            set(path "${CMAKE_CURRENT_SOURCE_DIR}/${path}")
        endif()
        _glob(${path})
        _glob_filter(${list} "${include}" "${exclude}")
        set(result ${result} ${${list}})
    endforeach()
    set(${VAR} ${result} PARENT_SCOPE)
endfunction()

function(_glob_filter VAR include exclude)
    set(result)
    foreach(name ${${VAR}})
        set(entry "${path}/${name}")
        # TODO: Remove this hack if we manage to fix the broken apic outputs knowledge
        if(NOT EXISTS ${entry})
            # Mark all non exsiting files as generated, just in case
            set_source_files_properties(${entry} PROPERTIES GENERATED TRUE)
        endif()
        # TODO: End of hack
        set(matches TRUE)
        if(NOT include STREQUAL "")
            set(matches FALSE)
            foreach(pattern ${include})
                if(entry MATCHES ${pattern})
                    set(matches TRUE)
                endif()
            endforeach()
        endif()
        foreach(pattern ${exclude})
            if(entry MATCHES ${pattern})
                set(matches FALSE)
            endif()
        endforeach()
        if(matches)
            list(APPEND result ${entry})
        endif()
    endforeach()
    set(${VAR} ${result} PARENT_SCOPE)
endfunction()

function(glob_all_dirs)
    _glob(${CMAKE_CURRENT_SOURCE_DIR})
    foreach(child ${dirs})
        set(child_path "${CMAKE_CURRENT_SOURCE_DIR}/${child}")
        _glob(${child_path})
        set(child_files)
        foreach(fil ${files})
            list(APPEND child_files "${child_path}/${fil}")
        endforeach()
        set(${child}_files ${child_files} PARENT_SCOPE)
    endforeach()
endfunction()

# _glob is the internal function that performs the glob if needed, and otherwise loads the results
function(_glob globpath)
    # Do the glob if we need to
    set(globfile "${globpath}/CMakeFiles.cmake")
    file(RELATIVE_PATH rel ${CMAKE_SOURCE_DIR} ${globpath})
    set(files)
    set(dirs)
    if(FORCING_GLOB OR NOT EXISTS ${globfile})
        message(STATUS "Re-globbing ${rel}")
        file(GLOB children RELATIVE ${globpath} ${globpath}/*)
        foreach(child ${children})
            if(IS_DIRECTORY ${globpath}/${child})
                list(APPEND dirs ${child})
            else()
                get_filename_component(type ${child} EXT)
                foreach(test ${GLOB_EXTENSIONS})
                    message("type:${type} -- test:${test}")
                    if(type STREQUAL ${test})
                        list(APPEND files ${child})
                    endif()
                endforeach()
            endif()
        endforeach()
        if(files)
            list(SORT files)
            list(REMOVE_DUPLICATES files)
            string(REPLACE ";" "\n    " files "${files}")
        endif()
        if(dirs)
            list(SORT dirs)
            list(REMOVE_DUPLICATES dirs)
            string(REPLACE ";" "\n    " dirs "${dirs}")
        endif()
        configure_file("${GLOB_TEMPLATE_FILE}" ${globfile} @ONLY)
    endif()
    # Use the glob results
    include(${globfile})
    set(files ${files} PARENT_SCOPE)
    set(dirs ${dirs} PARENT_SCOPE)
endfunction()
