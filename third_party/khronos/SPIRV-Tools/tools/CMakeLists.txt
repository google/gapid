# Copyright (c) 2015-2016 The Khronos Group Inc.
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and/or associated documentation files (the
# "Materials"), to deal in the Materials without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Materials, and to
# permit persons to whom the Materials are furnished to do so, subject to
# the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Materials.
#
# MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
# KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
# SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
#    https://www.khronos.org/registry/
#
# THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
# CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
# TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
# MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

add_subdirectory(lesspipe)

# Add a SPIR-V Tools command line tool. Signature:
#   add_spvtools_tool(
#     TARGET target_name
#     SRCS   src_file1.cpp src_file2.cpp
#     LIBS   lib_target1 lib_target2
#   )
function(add_spvtools_tool)
  set(one_value_args TARGET)
  set(multi_value_args SRCS LIBS)
  cmake_parse_arguments(
    ARG "" "${one_value_args}" "${multi_value_args}" ${ARGN})

  add_executable(${ARG_TARGET} ${ARG_SRCS})
  spvtools_default_compile_options(${ARG_TARGET})
  target_link_libraries(${ARG_TARGET} PRIVATE ${ARG_LIBS})
  target_include_directories(${ARG_TARGET} PRIVATE ${spirv-tools_SOURCE_DIR})
endfunction()

if (NOT ${SPIRV_SKIP_EXECUTABLES})
  add_spvtools_tool(TARGET spirv-as SRCS as/as.cpp LIBS ${SPIRV_TOOLS})
  add_spvtools_tool(TARGET spirv-dis SRCS dis/dis.cpp LIBS ${SPIRV_TOOLS})
  add_spvtools_tool(TARGET spirv-val SRCS val/val.cpp LIBS ${SPIRV_TOOLS})
  add_spvtools_tool(TARGET spirv-opt SRCS opt/opt.cpp LIBS SPIRV-Tools-opt ${SPIRV_TOOLS})
  add_spvtools_tool(TARGET spirv-cfg
                    SRCS cfg/cfg.cpp
                         cfg/bin_to_dot.h
                         cfg/bin_to_dot.cpp
                    LIBS ${SPIRV_TOOLS})
  target_include_directories(spirv-cfg PRIVATE ${spirv-tools_SOURCE_DIR}
                                               ${SPIRV_HEADER_INCLUDE_DIR})

  set(SPIRV_INSTALL_TARGETS spirv-as spirv-dis spirv-val spirv-opt spirv-cfg)
  install(TARGETS ${SPIRV_INSTALL_TARGETS}
    RUNTIME DESTINATION bin
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib)
endif()
