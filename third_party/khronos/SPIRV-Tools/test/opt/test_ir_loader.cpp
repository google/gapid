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

#include <gtest/gtest.h>
#include <algorithm>

#include "opt/libspirv.hpp"

namespace {

using namespace spvtools;

void DoRoundTripCheck(const std::string& text) {
  SpvTools t(SPV_ENV_UNIVERSAL_1_1);
  std::unique_ptr<ir::Module> module = t.BuildModule(text);
  ASSERT_NE(nullptr, module) << "Failed to assemble\n" << text;

  std::vector<uint32_t> binary;
  module->ToBinary(&binary, /* skip_nop = */ false);

  std::string disassembled_text;
  EXPECT_EQ(SPV_SUCCESS, t.Disassemble(binary, &disassembled_text));
  EXPECT_EQ(text, disassembled_text);
}

TEST(IrBuilder, RoundTrip) {
  // #version 310 es
  // int add(int a, int b) { return a + b; }
  // void main() { add(1, 2); }
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
          "%1 = OpExtInstImport \"GLSL.std.450\"\n"
               "OpMemoryModel Logical GLSL450\n"
               "OpEntryPoint Vertex %main \"main\"\n"
               "OpSource ESSL 310\n"
               "OpSourceExtension \"GL_GOOGLE_cpp_style_line_directive\"\n"
               "OpSourceExtension \"GL_GOOGLE_include_directive\"\n"
               "OpName %main \"main\"\n"
               "OpName %add_i1_i1_ \"add(i1;i1;\"\n"
               "OpName %a \"a\"\n"
               "OpName %b \"b\"\n"
               "OpName %param \"param\"\n"
               "OpName %param_0 \"param\"\n"
       "%void = OpTypeVoid\n"
          "%9 = OpTypeFunction %void\n"
        "%int = OpTypeInt 32 1\n"
 "%_ptr_Function_int = OpTypePointer Function %int\n"
         "%12 = OpTypeFunction %int %_ptr_Function_int %_ptr_Function_int\n"
         "%13 = OpConstant %int 1\n"
         "%14 = OpConstant %int 2\n"
       "%main = OpFunction %void None %9\n"
         "%15 = OpLabel\n"
      "%param = OpVariable %_ptr_Function_int Function\n"
    "%param_0 = OpVariable %_ptr_Function_int Function\n"
               "OpStore %param %13\n"
               "OpStore %param_0 %14\n"
         "%16 = OpFunctionCall %int %add_i1_i1_ %param %param_0\n"
               "OpReturn\n"
               "OpFunctionEnd\n"
 "%add_i1_i1_ = OpFunction %int None %12\n"
          "%a = OpFunctionParameter %_ptr_Function_int\n"
          "%b = OpFunctionParameter %_ptr_Function_int\n"
         "%17 = OpLabel\n"
         "%18 = OpLoad %int %a\n"
         "%19 = OpLoad %int %b\n"
         "%20 = OpIAdd %int %18 %19\n"
               "OpReturnValue %20\n"
               "OpFunctionEnd\n");
  // clang-format on
}

TEST(IrBuilder, RoundTripIncompleteBasicBlock) {
  DoRoundTripCheck(
      "%2 = OpFunction %1 None %3\n"
      "%4 = OpLabel\n"
      "OpNop\n");
}

TEST(IrBuilder, RoundTripIncompleteFunction) {
  DoRoundTripCheck("%2 = OpFunction %1 None %3\n");
}

TEST(IrBuilder, KeepLineDebugInfo) {
  // #version 310 es
  // void main() {}
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
          "%1 = OpExtInstImport \"GLSL.std.450\"\n"
               "OpMemoryModel Logical GLSL450\n"
               "OpEntryPoint Vertex %main \"main\"\n"
          "%3 = OpString \"minimal.vert\"\n"
               "OpSource ESSL 310\n"
               "OpName %main \"main\"\n"
               "OpLine %3 10 10\n"
       "%void = OpTypeVoid\n"
               "OpLine %3 100 100\n"
          "%5 = OpTypeFunction %void\n"
       "%main = OpFunction %void None %5\n"
               "OpLine %3 1 1\n"
               "OpNoLine\n"
               "OpLine %3 2 2\n"
               "OpLine %3 3 3\n"
          "%6 = OpLabel\n"
               "OpLine %3 4 4\n"
               "OpNoLine\n"
               "OpReturn\n"
               "OpFunctionEnd\n");
  // clang-format on
}

TEST(IrBuilder, LocalGlobalVariables) {
  // #version 310 es
  //
  // float gv1 = 10.;
  // float gv2 = 100.;
  //
  // float f() {
  //   float lv1 = gv1 + gv2;
  //   float lv2 = gv1 * gv2;
  //   return lv1 / lv2;
  // }
  //
  // void main() {
  //   float lv1 = gv1 - gv2;
  // }
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
          "%1 = OpExtInstImport \"GLSL.std.450\"\n"
               "OpMemoryModel Logical GLSL450\n"
               "OpEntryPoint Vertex %main \"main\"\n"
               "OpSource ESSL 310\n"
               "OpName %main \"main\"\n"
               "OpName %f_ \"f(\"\n"
               "OpName %gv1 \"gv1\"\n"
               "OpName %gv2 \"gv2\"\n"
               "OpName %lv1 \"lv1\"\n"
               "OpName %lv2 \"lv2\"\n"
               "OpName %lv1_0 \"lv1\"\n"
       "%void = OpTypeVoid\n"
         "%10 = OpTypeFunction %void\n"
      "%float = OpTypeFloat 32\n"
         "%12 = OpTypeFunction %float\n"
 "%_ptr_Private_float = OpTypePointer Private %float\n"
        "%gv1 = OpVariable %_ptr_Private_float Private\n"
         "%14 = OpConstant %float 10\n"
        "%gv2 = OpVariable %_ptr_Private_float Private\n"
         "%15 = OpConstant %float 100\n"
 "%_ptr_Function_float = OpTypePointer Function %float\n"
       "%main = OpFunction %void None %10\n"
         "%17 = OpLabel\n"
      "%lv1_0 = OpVariable %_ptr_Function_float Function\n"
               "OpStore %gv1 %14\n"
               "OpStore %gv2 %15\n"
         "%18 = OpLoad %float %gv1\n"
         "%19 = OpLoad %float %gv2\n"
         "%20 = OpFSub %float %18 %19\n"
               "OpStore %lv1_0 %20\n"
               "OpReturn\n"
               "OpFunctionEnd\n"
         "%f_ = OpFunction %float None %12\n"
         "%21 = OpLabel\n"
        "%lv1 = OpVariable %_ptr_Function_float Function\n"
        "%lv2 = OpVariable %_ptr_Function_float Function\n"
         "%22 = OpLoad %float %gv1\n"
         "%23 = OpLoad %float %gv2\n"
         "%24 = OpFAdd %float %22 %23\n"
               "OpStore %lv1 %24\n"
         "%25 = OpLoad %float %gv1\n"
         "%26 = OpLoad %float %gv2\n"
         "%27 = OpFMul %float %25 %26\n"
               "OpStore %lv2 %27\n"
         "%28 = OpLoad %float %lv1\n"
         "%29 = OpLoad %float %lv2\n"
         "%30 = OpFDiv %float %28 %29\n"
               "OpReturnValue %30\n"
               "OpFunctionEnd\n");
  // clang-format on
}

TEST(IrBuilder, OpUndefOutsideFunction) {
  // #version 310 es
  // void main() {}
  const std::string text =
      // clang-format off
               "OpMemoryModel Logical GLSL450\n"
        "%int = OpTypeInt 32 1\n"
       "%uint = OpTypeInt 32 0\n"
      "%float = OpTypeFloat 32\n"
          "%4 = OpUndef %int\n"
          "%5 = OpConstant %int 10\n"
          "%6 = OpUndef %uint\n"
       "%bool = OpTypeBool\n"
          "%8 = OpUndef %float\n"
     "%double = OpTypeFloat 64\n";
  // clang-format on

  SpvTools t(SPV_ENV_UNIVERSAL_1_1);
  std::unique_ptr<ir::Module> module = t.BuildModule(text);
  ASSERT_NE(nullptr, module);

  const auto opundef_count = std::count_if(
      module->types_values_begin(), module->types_values_end(),
      [](const ir::Instruction& inst) { return inst.opcode() == SpvOpUndef; });
  EXPECT_EQ(3, opundef_count);

  std::vector<uint32_t> binary;
  module->ToBinary(&binary, /* skip_nop = */ false);

  std::string disassembled_text;
  EXPECT_EQ(SPV_SUCCESS, t.Disassemble(binary, &disassembled_text));
  EXPECT_EQ(text, disassembled_text);
}

TEST(IrBuilder, OpUndefInBasicBlock) {
  DoRoundTripCheck(
      // clang-format off
               "OpMemoryModel Logical GLSL450\n"
               "OpName %main \"main\"\n"
       "%void = OpTypeVoid\n"
       "%uint = OpTypeInt 32 0\n"
     "%double = OpTypeFloat 64\n"
          "%5 = OpTypeFunction %void\n"
       "%main = OpFunction %void None %5\n"
          "%6 = OpLabel\n"
          "%7 = OpUndef %uint\n"
          "%8 = OpUndef %double\n"
               "OpReturn\n"
               "OpFunctionEnd\n");
  // clang-format on
}

TEST(IrBuilder, KeepLineDebugInfoBeforeType) {
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
               "OpMemoryModel Logical GLSL450\n"
          "%1 = OpString \"minimal.vert\"\n"
               "OpLine %1 1 1\n"
               "OpNoLine\n"
       "%void = OpTypeVoid\n"
               "OpLine %1 2 2\n"
          "%3 = OpTypeFunction %void\n");
  // clang-format on
}

TEST(IrBuilder, KeepLineDebugInfoBeforeLabel) {
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
               "OpMemoryModel Logical GLSL450\n"
          "%1 = OpString \"minimal.vert\"\n"
       "%void = OpTypeVoid\n"
          "%3 = OpTypeFunction %void\n"
       "%4 = OpFunction %void None %3\n"
          "%5 = OpLabel\n"
   "OpBranch %6\n"
               "OpLine %1 1 1\n"
               "OpLine %1 2 2\n"
          "%6 = OpLabel\n"
               "OpBranch %7\n"
               "OpLine %1 100 100\n"
          "%7 = OpLabel\n"
               "OpReturn\n"
               "OpFunctionEnd\n");
  // clang-format on
}

TEST(IrBuilder, KeepLineDebugInfoBeforeFunctionEnd) {
  DoRoundTripCheck(
      // clang-format off
               "OpCapability Shader\n"
               "OpMemoryModel Logical GLSL450\n"
          "%1 = OpString \"minimal.vert\"\n"
       "%void = OpTypeVoid\n"
          "%3 = OpTypeFunction %void\n"
       "%4 = OpFunction %void None %3\n"
               "OpLine %1 1 1\n"
               "OpLine %1 2 2\n"
               "OpFunctionEnd\n");
  // clang-format on
}

}  // anonymous namespace
