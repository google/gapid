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

set(LLVM_CCACHE_BUILD OFF)
if(NOT CMAKE_HOST_WIN32)
    find_program(CCACHE_FOUND ccache)
    if(CCACHE_FOUND)
        set(LLVM_CCACHE_BUILD ON)
    endif()
endif()

function(llvm_abi android_abi out)
    if (${android_abi} STREQUAL "arm64-v8a")
        set(${out} "AArch64" PARENT_SCOPE)
    elseif(${android_abi} STREQUAL "armeabi-v7a")
        set(${out} "ARM" PARENT_SCOPE)
    elseif(${android_abi} STREQUAL "x86")
        set(${out} "X86" PARENT_SCOPE)
    else()
        message(FATAL_ERROR "Unsupported architecture for building LLVM: ${android_abi}")
    endif()
endfunction()

if(ANDROID_ABI)
    llvm_abi(ANDROID_ABI target_arch)
    glob(sources
        PATH "lib" "lib/${target_arch}"
        INCLUDE ".h$" ".cc$"
    )

    if (ANDROID_ABI STREQUAL "arm64-v8a")
        set(LLVM_TARGET_ARCH "${target_arch}")
        set(LLVM_HOST_TRIPLE "aarch64-unknown-linux-android")
    elseif(ANDROID_ABI STREQUAL "armeabi-v7a")
        set(LLVM_TARGET_ARCH "${target_arch}")
        set(LLVM_HOST_TRIPLE "armv8.2a-unknown-linux-android")
    elseif(ANDROID_ABI STREQUAL "x86")
        set(LLVM_TARGET_ARCH "${target_arch}")
        set(LLVM_HOST_TRIPLE "i386-unknown-linux-android")
    else()
        message(FATAL_ERROR "Unsupported architecture for building LLVM: ${LLVM_TARGET_ARCH}")
    endif()


    configure_file(${CMAKE_CURRENT_SOURCE_DIR}/toolchain.cmake.in toolchain.cmake ESCAPE_QUOTES @ONLY)

    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/llvm")
    add_cmake(llvm "${CMAKE_SOURCE_DIR}/third_party/llvm"
        "-DCMAKE_TOOLCHAIN_FILE:PATH=${CMAKE_CURRENT_BINARY_DIR}/toolchain.cmake"
        "-DLLVM_EXTERNAL_PROJECTS:STRING=interceptor"
        "-DLLVM_EXTERNAL_INTERCEPTOR_SOURCE_DIR:PATH=${CMAKE_SOURCE_DIR}/gapii/interceptor-lib/cc"
        "-DLLVM_HOST_TRIPLE:STRING=${LLVM_HOST_TRIPLE}"
        "-DLLVM_TARGET_ARCH:STRING=${LLVM_TARGET_ARCH}"
        "-DLLVM_TARGETS_TO_BUILD:STRING=${LLVM_TARGET_ARCH}"
        "-DPYTHON_EXECUTABLE:PATH=${PYTHON_EXECUTABLE}"
        "-DLLVM_TABLEGEN:PATH=${CMAKE_BINARY_DIR}/../../../bin/llvm/llvm-tblgen${CMAKE_HOST_EXECUTABLE_SUFFIX}"
        "-DLLVM_CCACHE_BUILD:BOOL=${LLVM_CCACHE_BUILD}"
    )
    add_cmake_target(llvm interceptor ${dst} "libinterceptor.so"
        DEPENDS ${sources}
        SOURCE_PATH "lib/libinterceptor.so"
    )
else()
    configure_file(${CMAKE_CURRENT_SOURCE_DIR}/toolchain.cmake.in toolchain.cmake @ONLY)

    glob(sources
        PATH "lib" "lib/AArch64" "lib/ARM" "lib/X86"
        INCLUDE ".h$" ".cc$"
    )

    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/llvm")
    add_cmake(llvm "${CMAKE_SOURCE_DIR}/third_party/llvm"
        "-DCMAKE_TOOLCHAIN_FILE:STRING=${CMAKE_CURRENT_BINARY_DIR}/toolchain.cmake"
        "-DPYTHON_EXECUTABLE:PATH=${PYTHON_EXECUTABLE}"
        "-DLLVM_CCACHE_BUILD:BOOL=${LLVM_CCACHE_BUILD}"
    )
    add_cmake_target(llvm llvm-tblgen ${dst} "llvm-tblgen${CMAKE_HOST_EXECUTABLE_SUFFIX}"
        DEPENDS ${sources}
        SOURCE_PATH "bin/llvm-tblgen${CMAKE_HOST_EXECUTABLE_SUFFIX}"
    )
endif()

foreach(abi ${ANDROID_ACTIVE_ABI_LIST})
    set(dst "${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${ANDROID_BUILD_PATH_${abi}}")

    llvm_abi(${abi} target_arch)
    glob(sources
        PATH "lib" "lib/${target_arch}"
        INCLUDE ".h$" ".cc$"
    )

    add_cmake_target(${abi} llvm-interceptor ${dst} "libinterceptor.so"
        DEPENDS ${sources} llvm-llvm-tblgen
        SOURCE_PATH "llvm/src/llvm-build/lib/libinterceptor.so"
    )
endforeach()
