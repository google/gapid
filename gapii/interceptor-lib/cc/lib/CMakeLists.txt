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

include_directories(${CMAKE_CURRENT_SOURCE_DIR})

foreach(TARGET ${LLVM_TARGETS_TO_BUILD})
  add_subdirectory(${TARGET})
endforeach()

set(LLVM_LINK_COMPONENTS
  AllTargetsAsmPrinters
  AllTargetsAsmParsers
  AllTargetsDescs
  AllTargetsDisassemblers
  AllTargetsInfos
  MC
  MCParser
  Object
  Support)
 
 llvm_map_components_to_libnames(
   interceptor_libs ${LLVM_LINK_COMPONENTS}
 )
 
 add_library(interceptor SHARED
  code_generator.cc
  constant_pool_data_expr.cc
  disassembler.cc
  error.cc 
  interceptor.cc
  linker.cc
  memory_manager.cc
  target.cc)

target_link_libraries(interceptor PRIVATE ${interceptor_libs})
llvm_update_compile_flags(interceptor)

foreach(TARGET ${LLVM_TARGETS_TO_BUILD})
  target_link_libraries(interceptor PRIVATE
    Interceptor${TARGET}Target)
endforeach()

if (ANDROID)
  target_link_libraries(interceptor PRIVATE log z m)
endif()


# add_llvm_symbol_exports_win is a forked version of LLVM's add_llvm_symbol_exports
# function. It fixes issues with the generation of the exports file.
function(add_llvm_symbol_exports_win target_name export_file)
  set(native_export_file "${target_name}.exports")
  file(READ ${export_file} EXPORTS)
  string(REPLACE "\n" ";\n    " EXPORTS ${EXPORTS})
  configure_file("exports.in" ${native_export_file} @ONLY)
  set_property(TARGET ${target_name} APPEND_STRING PROPERTY
                LINK_FLAGS "  -Wl,--version-script,${CMAKE_CURRENT_BINARY_DIR}/${native_export_file}")
endfunction(add_llvm_symbol_exports_win)

# Only export the public (extern "C") symbols to prevent users from calling
# internal APIs and to make dead code elimination possible.
if(CMAKE_HOST_WIN32)
  add_llvm_symbol_exports_win(interceptor "${CMAKE_CURRENT_SOURCE_DIR}/libinterceptor.exports")
else()
  add_llvm_symbol_exports(interceptor "${CMAKE_CURRENT_SOURCE_DIR}/libinterceptor.exports")
endif()

