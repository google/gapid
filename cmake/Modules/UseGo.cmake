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

# go_install builds a target that invokes the go tool in install mode for the
# specified package.
# It will autmatically add all dependant packages recursivly.
function(go_install)
    if(DISABLED_GO)
        return()
    endif()
    cmake_parse_arguments(GO "" "DESTINATION;LINK_PACKAGE;WIN_UI" "" ${ARGN})
    default(GO_UNPARSED_ARGUMENTS ${CMAKE_CURRENT_SOURCE_DIR})
    foreach(entry ${GO_UNPARSED_ARGUMENTS})
        get_filename_component(CMAKE_CURRENT_SOURCE_DIR "${entry}" ABSOLUTE)
        file(RELATIVE_PATH package ${GO_SRC} ${CMAKE_CURRENT_SOURCE_DIR})
        if(NOT GO_LINK_PACKAGE)
            set(GO_LINK_PACKAGE ${package})
        endif()
        _go_names()
        # Add the executable
        go_glob(${CMAKE_CURRENT_SOURCE_DIR})
        add_executable(${name} ${gofiles})
        _go_deps()
        target_link_libraries(${name} ${Imports})
        set_target_properties(${name} PROPERTIES
            LINK_FLAGS "-DGO_PACKAGE=${GO_LINK_PACKAGE} -DGO_WIN_UI=${GO_WIN_UI}"
            # JOB_POOL_LINK go
        )
        _go_update_deps(${name})
        if(GO_DESTINATION)
            install(TARGETS ${name} DESTINATION ${GO_DESTINATION})
        endif()
        if (WIN32)
            _go_msys_dlls(${name})
        endif(WIN32)
    endforeach()
endfunction(go_install)

# go_package is used to force packages to be tested even if not referred to by
# a normal go_install rule.
function(go_package)
    if(DISABLED_GO)
        return()
    endif()
    file(RELATIVE_PATH package ${GO_SRC} ${CMAKE_CURRENT_SOURCE_DIR})
    _go_package(lib ${package})
endfunction(go_package)

function(_go_deps)
    set(depsfile "${GO_DEPS}/${tag_name}.cmake")
    # Scan it's dependancies
    if(NOT EXISTS ${depsfile})
        message(STATUS "Initial dependancies for ${tag_name}")
        execute_process(
            COMMAND "${CMAKE_COMMAND}" -DGO_ENV=${GO_ENV} -DGO_PACKAGE=${package} -DGO_DEPS_FILE=${depsfile} -P ${GO_BUILD_DEPS}
        )
    endif()
    set(Imports)
    set(TestImports)
    set(XTestImports)
    include(${depsfile} OPTIONAL)
    set(imports)
    set(testImports)
    _go_deps_libs(imports ${Imports})
    _go_deps_libs(testImports ${TestImports})
    _go_deps_libs(testImports ${XTestImports})
    set(depsfile ${depsfile} PARENT_SCOPE)
    set(Imports ${imports} PARENT_SCOPE)
    set(TestImports ${testImports} PARENT_SCOPE)
endfunction(_go_deps)

function(_go_deps_libs VAR)
    foreach(dep ${ARGN})
        _go_package(lib ${dep})
        list(APPEND ${VAR} ${lib})
    endforeach()
    set(${VAR} ${${VAR}} PARENT_SCOPE)
endfunction(_go_deps_libs)

function(_go_update_deps tgt)
    add_custom_command(
        TARGET ${tgt}
        POST_BUILD
        COMMAND "${CMAKE_COMMAND}" -DGO_ENV=${GO_ENV} -DGO_PACKAGE=${package} -DGO_DEPS_FILE=${depsfile} -P ${GO_BUILD_DEPS}
    )
endfunction(_go_update_deps)

set(GO_TEST_SCRIPT "${CMAKE_CURRENT_LIST_DIR}/GoTest.cmake")
# _go_package builds the interface library for a package, and also tests if needed.
function(_go_package VAR package)
    _go_names()
    set(${VAR} "${tag_name}" PARENT_SCOPE)
    if(TARGET ${tag_name})
        return()
    endif()
    if(NOT IS_DIRECTORY ${path} OR ${path} MATCHES "third_party" OR NOT ${path} MATCHES "github.com/google/gapid")
      add_library(${tag_name} INTERFACE)
      # It is difficult to handle external packages correctly during the build,
      # so compile them explicitly before everything else (single threaded).
      if(NOT ${package} STREQUAL "C")
          message(STATUS "Build external package ${tag_name}")
          execute_process(
              COMMAND "${CMAKE_COMMAND}" -DGO_ENV=${GO_ENV} -DGO_PACKAGE=${package} -P ${GO_COMPILE}
          )
      endif()
      return()
    endif()
    go_glob(${path})
    add_library(${tag_name} ${gofiles})
    add_custom_command(
      TARGET ${tag_name}
      PRE_BUILD
      COMMAND "${CMAKE_COMMAND}"
              -DGO_ENV=${GO_ENV}
              -DGO_PACKAGE=${package}
              -P ${GO_COMPILE}
    )
    _go_deps()
    if(Imports)
      list(SORT Imports)
      list(REMOVE_DUPLICATES Imports)
    endif()
    target_link_libraries(${tag_name} ${Imports})
    set_target_properties(${tag_name} PROPERTIES
        SUFFIX ".tag"
        ARCHIVE_OUTPUT_DIRECTORY ${GO_TAGS}
    )
    _go_update_deps(${tag_name})
    if(DISABLED_TESTS OR NOT testfiles)
        return()
    endif()
    set(test_name "test-${tag_name}")
    if(TARGET test_name)
        return()
    endif()
    set(test_deps ${Imports} ${TestImports})
    # Do not self-reference - foo_test does not depend on foo and can be done in parallel
    list(REMOVE_ITEM test_deps ${tag_name})
    list(REMOVE_DUPLICATES test_deps)
    add_executable(${test_name} ${gofiles} ${testfiles})
    target_link_libraries(${test_name} ${test_deps})
    set_target_properties(${test_name} PROPERTIES
        LINK_FLAGS "-DGO_PACKAGE=${package}"
        # JOB_POOL_LINK go
        RUNTIME_OUTPUT_DIRECTORY ${CMAKE_TEST_OUTPUT_DIRECTORY}
    )
    if(NOT NO_RUN_TESTS AND NOT "${test_name}" MATCHES "integration")
        # Make the test run on build
        add_custom_command(
            TARGET ${test_name}
            POST_BUILD
            COMMAND "${CMAKE_COMMAND}"
                "-DTESTER=$<TARGET_FILE:${test_name}>"
                "-DGO_PATH=\"${GO_PATH}\""
                -P "${GO_TEST_SCRIPT}"
            # Let the test run in the original source directory so that
            # we can properly locate resources referenced by the test.
            # Needed here since this is a POST_BUILD custom command.
            WORKING_DIRECTORY ${path}
        )
    endif()
    # Add the test to the full testing suite
    add_test(
        NAME test-${tag_name}
        COMMAND "${test_name}"
        # Also need to specify here so tests invoked by ctest directly
        # are run under the correct directory.
        WORKING_DIRECTORY ${path})
endfunction(_go_package)

# _go_names is an internal function that works out what the name, path and
# tag_name should be for a given package name.
function(_go_names)
    set(path "${GO_SRC}/${package}")
    if(IS_DIRECTORY ${path})
        file(RELATIVE_PATH short ${CMAKE_SOURCE_DIR} ${path})
    else()
        set(path "${package}")
        set(short "external-${package}")
    endif()
    get_filename_component(name "${path}" NAME)
    string(REPLACE "/" "-" tag_name "${short}")
    set(name ${name} PARENT_SCOPE)
    set(path ${path} PARENT_SCOPE)
    set(tag_name "${tag_name}" PARENT_SCOPE)
endfunction(_go_names)

# _go_msys_dlls is an internal function that copies the MSYS DLLs to the output.
# The binaries on Windows currently have a dependency on some DLLs from MSYS.
# TODO - remove this dependency
function(_go_msys_dlls tgt)
    list(APPEND go_msys_dlls
        ${MINGW_PATH}/bin/libwinpthread-1.dll
        ${MINGW_PATH}/bin/libstdc++-6.dll
        ${MINGW_PATH}/bin/libgcc_s_seh-1.dll
    )

    add_custom_command(TARGET ${tgt} POST_BUILD
        COMMAND "${CMAKE_COMMAND}" -E copy_if_different
            ${go_msys_dlls} $<TARGET_FILE_DIR:${tgt}>)
    if(GO_DESTINATION)
        install(FILES ${go_msys_dlls} DESTINATION ${GO_DESTINATION})
    endif()
endfunction(_go_msys_dlls)

# go_glob does the glob file generation step for a given directory.
function(go_glob path)
    # Glob for the source files of the package
    glob(gofiles PATH "${path}" INCLUDE "\.go$" EXCLUDE "_test\.go$")
    set_source_files_properties(${gofiles} PROPERTIES EXTERNAL_OBJECT TRUE)
    glob(testfiles PATH "${path}" INCLUDE "_test\.go$")
    set_source_files_properties(${testfiles} PROPERTIES EXTERNAL_OBJECT TRUE)
    set(gofiles "${gofiles}" PARENT_SCOPE)
    set(testfiles "${testfiles}" PARENT_SCOPE)
endfunction(go_glob)

# cgo_dependency adds a cgo library dependeny to the go package.
function(cgo_dependency)
    if(DISABLED_GO)
        return()
    endif()
    file(RELATIVE_PATH package ${GO_SRC} ${CMAKE_CURRENT_SOURCE_DIR})
    _go_package(lib ${package})
    target_link_libraries(${lib} ${ARGN})
    if(TARGET "test-${lib}")
        target_link_libraries("test-${lib}" ${ARGN})
    endif()
endfunction(cgo_dependency)
