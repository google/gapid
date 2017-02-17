// Copyright (c) 2016 Google Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

#include "assembly_builder.h"

#include "pass_fixture.h"
#include "pass_utils.h"

namespace {

using namespace spvtools;
using AssemblyBuilderTest = PassTest<::testing::Test>;

TEST_F(AssemblyBuilderTest, MinimalShader) {
  AssemblyBuilder builder;
  std::vector<const char*> expected = {
      // clang-format off
                    "OpCapability Shader",
                    "OpCapability Float64",
               "%1 = OpExtInstImport \"GLSL.std.450\"",
                    "OpMemoryModel Logical GLSL450",
                    "OpEntryPoint Vertex %main \"main\"",
                    "OpName %void \"void\"",
                    "OpName %main_func_type \"main_func_type\"",
                    "OpName %main \"main\"",
                    "OpName %main_func_entry_block \"main_func_entry_block\"",
            "%void = OpTypeVoid",
  "%main_func_type = OpTypeFunction %void",
            "%main = OpFunction %void None %main_func_type",
"%main_func_entry_block = OpLabel",
                    "OpReturn",
                    "OpFunctionEnd",
      // clang-format on
  };

  SinglePassRunAndCheck<opt::NullPass>(builder.GetCode(),
                                       JoinAllInsts(expected),
                                       /* skip_nop = */ false);
}

TEST_F(AssemblyBuilderTest, ShaderWithConstants) {
  AssemblyBuilder builder;
  builder
      .AppendTypesConstantsGlobals({
          // clang-format off
          "%bool = OpTypeBool",
      "%_PF_bool = OpTypePointer Function %bool",
            "%bt = OpConstantTrue %bool",
            "%bf = OpConstantFalse %bool",
           "%int = OpTypeInt 32 1",
       "%_PF_int = OpTypePointer Function %int",
            "%si = OpConstant %int 1",
          "%uint = OpTypeInt 32 0",
      "%_PF_uint = OpTypePointer Function %uint",
            "%ui = OpConstant %uint 2",
         "%float = OpTypeFloat 32",
     "%_PF_float = OpTypePointer Function %float",
             "%f = OpConstant %float 3.14",
        "%double = OpTypeFloat 64",
    "%_PF_double = OpTypePointer Function %double",
             "%d = OpConstant %double 3.14159265358979",
          // clang-format on
      })
      .AppendInMain({
          // clang-format off
          "%btv = OpVariable %_PF_bool Function",
          "%bfv = OpVariable %_PF_bool Function",
           "%iv = OpVariable %_PF_int Function",
           "%uv = OpVariable %_PF_uint Function",
           "%fv = OpVariable %_PF_float Function",
           "%dv = OpVariable %_PF_double Function",
                 "OpStore %btv %bt",
                 "OpStore %bfv %bf",
                 "OpStore %iv %si",
                 "OpStore %uv %ui",
                 "OpStore %fv %f",
                 "OpStore %dv %d",
          // clang-format on
      });

  std::vector<const char*> expected = {
      // clang-format off
                "OpCapability Shader",
                "OpCapability Float64",
           "%1 = OpExtInstImport \"GLSL.std.450\"",
                "OpMemoryModel Logical GLSL450",
                "OpEntryPoint Vertex %main \"main\"",
                "OpName %void \"void\"",
                "OpName %main_func_type \"main_func_type\"",
                "OpName %main \"main\"",
                "OpName %main_func_entry_block \"main_func_entry_block\"",
                "OpName %bool \"bool\"",
                "OpName %_PF_bool \"_PF_bool\"",
                "OpName %bt \"bt\"",
                "OpName %bf \"bf\"",
                "OpName %int \"int\"",
                "OpName %_PF_int \"_PF_int\"",
                "OpName %si \"si\"",
                "OpName %uint \"uint\"",
                "OpName %_PF_uint \"_PF_uint\"",
                "OpName %ui \"ui\"",
                "OpName %float \"float\"",
                "OpName %_PF_float \"_PF_float\"",
                "OpName %f \"f\"",
                "OpName %double \"double\"",
                "OpName %_PF_double \"_PF_double\"",
                "OpName %d \"d\"",
                "OpName %btv \"btv\"",
                "OpName %bfv \"bfv\"",
                "OpName %iv \"iv\"",
                "OpName %uv \"uv\"",
                "OpName %fv \"fv\"",
                "OpName %dv \"dv\"",
        "%void = OpTypeVoid",
"%main_func_type = OpTypeFunction %void",
        "%bool = OpTypeBool",
 "%_PF_bool = OpTypePointer Function %bool",
          "%bt = OpConstantTrue %bool",
          "%bf = OpConstantFalse %bool",
         "%int = OpTypeInt 32 1",
     "%_PF_int = OpTypePointer Function %int",
          "%si = OpConstant %int 1",
        "%uint = OpTypeInt 32 0",
    "%_PF_uint = OpTypePointer Function %uint",
          "%ui = OpConstant %uint 2",
       "%float = OpTypeFloat 32",
   "%_PF_float = OpTypePointer Function %float",
           "%f = OpConstant %float 3.14",
      "%double = OpTypeFloat 64",
  "%_PF_double = OpTypePointer Function %double",
           "%d = OpConstant %double 3.14159265358979",
        "%main = OpFunction %void None %main_func_type",
"%main_func_entry_block = OpLabel",
         "%btv = OpVariable %_PF_bool Function",
         "%bfv = OpVariable %_PF_bool Function",
          "%iv = OpVariable %_PF_int Function",
          "%uv = OpVariable %_PF_uint Function",
          "%fv = OpVariable %_PF_float Function",
          "%dv = OpVariable %_PF_double Function",
                "OpStore %btv %bt",
                "OpStore %bfv %bf",
                "OpStore %iv %si",
                "OpStore %uv %ui",
                "OpStore %fv %f",
                "OpStore %dv %d",
                "OpReturn",
                "OpFunctionEnd",
      // clang-format on
  };
  SinglePassRunAndCheck<opt::NullPass>(builder.GetCode(),
                                       JoinAllInsts(expected),
                                       /* skip_nop = */ false);
}

TEST_F(AssemblyBuilderTest, SpecConstants) {
  AssemblyBuilder builder;
  builder.AppendTypesConstantsGlobals({
      "%bool = OpTypeBool", "%uint = OpTypeInt 32 0", "%int = OpTypeInt 32 1",
      "%float = OpTypeFloat 32", "%double = OpTypeFloat 64",
      "%v2int = OpTypeVector %int 2",

      "%spec_true = OpSpecConstantTrue %bool",
      "%spec_false = OpSpecConstantFalse %bool",
      "%spec_uint = OpSpecConstant %uint 1",
      "%spec_int = OpSpecConstant %int 1",
      "%spec_float = OpSpecConstant %float 1.2",
      "%spec_double = OpSpecConstant %double 1.23456789",

      // Spec constants defined below should not have SpecID.
      "%spec_add_op = OpSpecConstantOp %int IAdd %spec_int %spec_int",
      "%spec_vec = OpSpecConstantComposite %v2int %spec_int %spec_int",
      "%spec_vec_x = OpSpecConstantOp %int CompositeExtract %spec_vec 0",
  });
  std::vector<const char*> expected = {
      // clang-format off
                    "OpCapability Shader",
                    "OpCapability Float64",
               "%1 = OpExtInstImport \"GLSL.std.450\"",
                    "OpMemoryModel Logical GLSL450",
                    "OpEntryPoint Vertex %main \"main\"",
                    "OpName %void \"void\"",
                    "OpName %main_func_type \"main_func_type\"",
                    "OpName %main \"main\"",
                    "OpName %main_func_entry_block \"main_func_entry_block\"",
                    "OpName %bool \"bool\"",
                    "OpName %uint \"uint\"",
                    "OpName %int \"int\"",
                    "OpName %float \"float\"",
                    "OpName %double \"double\"",
                    "OpName %v2int \"v2int\"",
                    "OpName %spec_true \"spec_true\"",
                    "OpName %spec_false \"spec_false\"",
                    "OpName %spec_uint \"spec_uint\"",
                    "OpName %spec_int \"spec_int\"",
                    "OpName %spec_float \"spec_float\"",
                    "OpName %spec_double \"spec_double\"",
                    "OpName %spec_add_op \"spec_add_op\"",
                    "OpName %spec_vec \"spec_vec\"",
                    "OpName %spec_vec_x \"spec_vec_x\"",
                    "OpDecorate %spec_true SpecId 200",
                    "OpDecorate %spec_false SpecId 201",
                    "OpDecorate %spec_uint SpecId 202",
                    "OpDecorate %spec_int SpecId 203",
                    "OpDecorate %spec_float SpecId 204",
                    "OpDecorate %spec_double SpecId 205",
            "%void = OpTypeVoid",
  "%main_func_type = OpTypeFunction %void",
            "%bool = OpTypeBool",
            "%uint = OpTypeInt 32 0",
             "%int = OpTypeInt 32 1",
           "%float = OpTypeFloat 32",
          "%double = OpTypeFloat 64",
           "%v2int = OpTypeVector %int 2",
       "%spec_true = OpSpecConstantTrue %bool",
      "%spec_false = OpSpecConstantFalse %bool",
       "%spec_uint = OpSpecConstant %uint 1",
        "%spec_int = OpSpecConstant %int 1",
      "%spec_float = OpSpecConstant %float 1.2",
     "%spec_double = OpSpecConstant %double 1.23456789",
     "%spec_add_op = OpSpecConstantOp %int IAdd %spec_int %spec_int",
        "%spec_vec = OpSpecConstantComposite %v2int %spec_int %spec_int",
      "%spec_vec_x = OpSpecConstantOp %int CompositeExtract %spec_vec 0",
            "%main = OpFunction %void None %main_func_type",
"%main_func_entry_block = OpLabel",
                    "OpReturn",
                    "OpFunctionEnd",

      // clang-format on
  };

  SinglePassRunAndCheck<opt::NullPass>(builder.GetCode(),
                                       JoinAllInsts(expected),
                                       /* skip_nop = */ false);
}

TEST_F(AssemblyBuilderTest, AppendNames) {
  AssemblyBuilder builder;
  builder.AppendNames({
      "OpName %void \"another_name_for_void\"",
      "I am an invalid OpName instruction and should not be added",
      "OpName %main \"another name for main\"",
  });
  std::vector<const char*> expected = {
      // clang-format off
                    "OpCapability Shader",
                    "OpCapability Float64",
               "%1 = OpExtInstImport \"GLSL.std.450\"",
                    "OpMemoryModel Logical GLSL450",
                    "OpEntryPoint Vertex %main \"main\"",
                    "OpName %void \"void\"",
                    "OpName %main_func_type \"main_func_type\"",
                    "OpName %main \"main\"",
                    "OpName %main_func_entry_block \"main_func_entry_block\"",
                    "OpName %void \"another_name_for_void\"",
                    "OpName %main \"another name for main\"",
            "%void = OpTypeVoid",
  "%main_func_type = OpTypeFunction %void",
            "%main = OpFunction %void None %main_func_type",
"%main_func_entry_block = OpLabel",
                    "OpReturn",
                    "OpFunctionEnd",
      // clang-format on
  };

  SinglePassRunAndCheck<opt::NullPass>(builder.GetCode(),
                                       JoinAllInsts(expected),
                                       /* skip_nop = */ false);
}

}  // anonymous namespace
