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

# required is used to check that a parameter was set.
function (required var message)
    if(NOT ${var})
        message(FATAL_ERROR "${var} expected ${message}")
    endif()
endfunction(required)

# default is used to default a parameter to the supplied value if not set.
function (default var value)
    if(NOT DEFINED ${var})
        set(${var} ${value} PARENT_SCOPE)
    endif()
endfunction(default)

# add_file_copy adds a file copying rule that updates dst if older that src.
function(add_file_copy src dst)
    add_custom_command(
        OUTPUT ${dst}
        COMMAND "${CMAKE_COMMAND}" -E copy ${src} ${dst}
        DEPENDS ${src}
    )
endfunction(add_file_copy)

# all_target is used by steps that want a rule to run all occurences of a step
# type, Most often used by code generators.
# It makes a targets called ${name}-${child} ${name}-all and sets the all
# target to depend on the child target, and the child target to depend on any
# remaining arguments.
function(all_target name child)
    set(target ${name}-${child})
    if(TARGET ${target})
        # additional dependancies, generate a unique name from the first dependancy
        list(GET ARGN 0 first)
        get_filename_component(first ${first} NAME_WE)
        set(extra_target "${target}-${first}")
        add_custom_target(${extra_target} ALL DEPENDS ${ARGN})
        add_dependencies(${target} ${extra_target})
        return()
    endif()
    add_custom_target(${target} ALL DEPENDS ${ARGN})
    set(all_target ${name}-all)
    if(NOT TARGET ${all_target})
        add_custom_target(${all_target} ALL)
    endif()
    add_dependencies(${all_target} ${target})
    if(NOT TARGET generate)
        add_custom_target(generate)
    endif()
    add_dependencies(generate ${target})
endfunction(all_target)


# build_subdirectory is analagous to add_subdirectory, and performs the same
# task if the sub directory has a CMakeLists.txt, but if it has a build.cmake
# then it includes that instead.
# This is needed becuase some of our targets have to be generated in the same
# context but from many directories.
function(build_subdirectory dir)
    set(CMAKE_CURRENT_SOURCE_DIR ${CMAKE_CURRENT_SOURCE_DIR}/${dir})
    if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/CMakeBuild.cmake")
        include("${CMAKE_CURRENT_SOURCE_DIR}/CMakeBuild.cmake")
    else()
        add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR})
    endif()
endfunction(build_subdirectory)

# abs_list uses the specified path as the base to make absolute all entries
# in VAR that are not already absolute paths.
function(abs_list VAR path)
    set(result)
    foreach(entry ${${VAR}})
        if(NOT IS_ABSOLUTE ${entry})
            get_filename_component(entry "${path}/${entry}" ABSOLUTE)
        endif()
        list(APPEND result ${entry})
    endforeach()
    set(${VAR} ${result} PARENT_SCOPE)
endfunction(abs_list)


# paths_to_native transforms the list of paths to the os-native form
# and assigns the results to OUT.
function(paths_to_native OUT list)
    set(result)
    foreach(entry ${${list}})
        file(TO_NATIVE_PATH ${entry} entry)
        list(APPEND result ${entry})
    endforeach()
    set(${OUT} ${result} PARENT_SCOPE)
endfunction(paths_to_native)

# Override and disable link_libraries, it's been a mistake every time it's been used
function(link_libraries)
    message(FATAL_ERROR "Don't use link_libraries, you probably meant target_link_libraries (${ARGV})")
endfunction()

# Create a library target that includes external libraries
function(imported_library tgt)
    if(TARGET ${tgt})
        return()
    endif()
    list(LENGTH ARGN count)
    if(count EQUAL 0)
        add_library(${tgt} INTERFACE IMPORTED)
    elseif(count EQUAL 1)
        add_library(${tgt} UNKNOWN IMPORTED)
        set_target_properties(${tgt} PROPERTIES IMPORTED_LOCATION "${ARGN}")
    else()
        message("Only 0 or 1 libraries allowed right now")
    endif()
endfunction(imported_library)
