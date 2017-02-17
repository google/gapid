# Copyright (c) 2016 Google Inc.
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

add_spvtools_unittest(TARGET instruction
  SRCS test_instruction.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET ir_loader
  SRCS test_ir_loader.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_manager
  SRCS module_utils.h
       test_pass_manager.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_strip_debug_info
  SRCS test_strip_debug_info.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_freeze_spec_const
  SRCS test_freeze_spec_const.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_eliminate_dead_const
  SRCS test_eliminate_dead_const.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_utils
  SRCS test_utils.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET def_use
  SRCS test_def_use.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET assembly_builder
  SRCS test_assembly_builder.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET types
  SRCS test_types.cpp
  LIBS SPIRV-Tools-opt
)

add_spvtools_unittest(TARGET type_manager
  SRCS test_type_manager.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET iterator
  SRCS test_iterator.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET module
  SRCS module_utils.h
       test_module.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_fold_spec_const_op_composite
  SRCS test_fold_spec_const_op_composite.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET pass_unify_const
  SRCS test_unify_const.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)

add_spvtools_unittest(TARGET line_debug_info
  SRCS test_line_debug_info.cpp pass_utils.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS}
)
