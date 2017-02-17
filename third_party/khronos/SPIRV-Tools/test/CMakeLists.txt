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

# Add a SPIR-V Tools unit test. Signature:
#   add_spvtools_unittest(
#     TARGET target_name
#     SRCS   src_file.h src_file.cpp
#     LIBS   lib1 lib2
#   )

if (NOT "${SPIRV_SKIP_TESTS}")
  if (TARGET gmock_main)
    message(STATUS "Found Google Mock, building tests.")
  else()
    message(STATUS "Did not find googletest, tests will not be built."
      "To enable tests place googletest in '<spirv-dir>/external/googletest'.")
  endif()
endif()

function(add_spvtools_unittest)
  if (NOT "${SPIRV_SKIP_TESTS}" AND TARGET gmock_main)
    set(one_value_args TARGET)
    set(multi_value_args SRCS LIBS)
    cmake_parse_arguments(
      ARG "" "${one_value_args}" "${multi_value_args}" ${ARGN})
    add_executable(${ARG_TARGET} ${ARG_SRCS})
    spvtools_default_compile_options(${ARG_TARGET})
    if(${COMPILER_IS_LIKE_GNU})
      target_compile_options(${ARG_TARGET} PRIVATE -Wno-undef)
    endif()
    if("${CMAKE_CXX_COMPILER_ID}" STREQUAL "MSVC")
      # Disable C4503 "decorated name length exceeded" warning,
      # triggered by some heavily templated types.
      # We don't care much about that in test code.
      # Important to do since we have warnings-as-errors.
      target_compile_options(${ARG_TARGET} PRIVATE /wd4503)
    endif()
    target_include_directories(${ARG_TARGET} PRIVATE
      ${spirv-tools_SOURCE_DIR}/test
      ${spirv-tools_SOURCE_DIR}
      ${spirv-tools_SOURCE_DIR}/include
      ${SPIRV_HEADER_INCLUDE_DIR}
      ${gtest_SOURCE_DIR}/include
      ${gmock_SOURCE_DIR}/include
    )
    target_link_libraries(${ARG_TARGET} PRIVATE ${ARG_LIBS})
    target_link_libraries(${ARG_TARGET} PRIVATE gmock_main)
    add_test(NAME spirv-tools-${ARG_TARGET} COMMAND ${ARG_TARGET})
  endif()
endfunction()

set(TEST_SOURCES
  ${CMAKE_CURRENT_SOURCE_DIR}/TestFixture.h
  ${CMAKE_CURRENT_SOURCE_DIR}/UnitSPIRV.h

  ${CMAKE_CURRENT_SOURCE_DIR}/AssemblyContext.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/AssemblyFormat.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryDestroy.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryEndianness.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryHeaderGet.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryParse.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryToText.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/BinaryToText.Literal.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/CapabilitySet.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Comment.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/DiagnosticDestroy.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/DiagnosticPrint.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/DiagnosticStream.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ExtInstGLSLstd450.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ExtInst.OpenCL.std.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/FixWord.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/GeneratorMagicNumber.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/HexFloat.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ImmediateInt.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/LibspirvMacros.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/NamedId.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/NameMapper.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OpcodeMake.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OpcodeRequiresCapabilities.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OpcodeSplit.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OpcodeTableGet.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OperandCapabilities.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Operand.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/OperandPattern.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/SoftwareVersion.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TargetEnv.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextAdvance.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextDestroy.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextLiteral.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextStartsNewInst.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Annotation.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Barrier.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Constant.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.ControlFlow.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Debug.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.DeviceSideEnqueue.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Extension.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Function.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Group.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Image.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Literal.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Memory.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.Miscellaneous.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.ModeSetting.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.PipeStorage.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.TypeDeclaration.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextToBinary.SubgroupDispatch.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/TextWordGet.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/UnitSPIRV.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ValidateFixtures.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Validate.Capability.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Validate.CFG.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Validate.Layout.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Validate.Storage.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/Validate.SSA.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ValidateID.cpp
  ${CMAKE_CURRENT_SOURCE_DIR}/ValidationState.cpp
)
add_spvtools_unittest(
  TARGET UnitSPIRV
  SRCS ${TEST_SOURCES}
  LIBS ${SPIRV_TOOLS})

add_spvtools_unittest(
  TARGET cpp_interface
  SRCS cpp_interface.cpp
  LIBS SPIRV-Tools-opt ${SPIRV_TOOLS})

add_subdirectory(opt)
