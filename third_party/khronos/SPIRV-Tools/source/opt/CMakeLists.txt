add_library(SPIRV-Tools-opt
  basic_block.h
  constants.h
  def_use_manager.h
  eliminate_dead_constant_pass.h
  function.h
  fold_spec_constant_op_and_composite_pass.h
  freeze_spec_constant_value_pass.h
  instruction.h
  ir_loader.h
  libspirv.hpp
  module.h
  null_pass.h
  reflect.h
  pass.h
  passes.h
  pass_manager.h
  strip_debug_info_pass.h
  types.h
  type_manager.h
  unify_const_pass.h

  def_use_manager.cpp
  eliminate_dead_constant_pass.cpp
  function.cpp
  fold_spec_constant_op_and_composite_pass.cpp
  freeze_spec_constant_value_pass.cpp
  instruction.cpp
  ir_loader.cpp
  libspirv.cpp
  module.cpp
  strip_debug_info_pass.cpp
  types.cpp
  type_manager.cpp
  unify_const_pass.cpp
)

spvtools_default_compile_options(SPIRV-Tools-opt)
target_include_directories(SPIRV-Tools-opt
  PRIVATE ${spirv-tools_SOURCE_DIR}/include
  PUBLIC ${SPIRV_HEADER_INCLUDE_DIR}
)
