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

#include "pass_fixture.h"
#include "pass_utils.h"

#include <algorithm>
#include <tuple>
#include <vector>

namespace {

using namespace spvtools;

struct FreezeSpecConstantValueTypeTestCase {
  const char* type_decl;
  const char* spec_const;
  const char* expected_frozen_const;
};

using FreezeSpecConstantValueTypeTest =
    PassTest<::testing::TestWithParam<FreezeSpecConstantValueTypeTestCase>>;

TEST_P(FreezeSpecConstantValueTypeTest, PrimaryType) {
  auto& test_case = GetParam();
  std::vector<const char*> text = {"OpCapability Shader",
                                   "OpMemoryModel Logical GLSL450",
                                   test_case.type_decl, test_case.spec_const};
  std::vector<const char*> expected = {
      "OpCapability Shader", "OpMemoryModel Logical GLSL450",
      test_case.type_decl, test_case.expected_frozen_const};
  SinglePassRunAndCheck<opt::FreezeSpecConstantValuePass>(
      JoinAllInsts(text), JoinAllInsts(expected), /* skip_nop = */ false);
}

// Test each primary type.
INSTANTIATE_TEST_CASE_P(
    PrimaryTypeSpecConst, FreezeSpecConstantValueTypeTest,
    ::testing::ValuesIn(std::vector<FreezeSpecConstantValueTypeTestCase>({
        // Type declaration, original spec constant definition, expected frozen
        // spec constants.
        {"%int = OpTypeInt 32 1", "%2 = OpSpecConstant %int 1",
         "%2 = OpConstant %int 1"},
        {"%uint = OpTypeInt 32 0", "%2 = OpSpecConstant %uint 1",
         "%2 = OpConstant %uint 1"},
        {"%float = OpTypeFloat 32", "%2 = OpSpecConstant %float 3.14",
         "%2 = OpConstant %float 3.14"},
        {"%double = OpTypeFloat 64", "%2 = OpSpecConstant %double 3.1415926",
         "%2 = OpConstant %double 3.1415926"},
        {"%bool = OpTypeBool", "%2 = OpSpecConstantTrue %bool",
         "%2 = OpConstantTrue %bool"},
        {"%bool = OpTypeBool", "%2 = OpSpecConstantFalse %bool",
         "%2 = OpConstantFalse %bool"},
    })));

using FreezeSpecConstantValueRemoveDecorationTest = PassTest<::testing::Test>;

TEST_F(FreezeSpecConstantValueRemoveDecorationTest,
       RemoveDecorationInstWithSpecId) {
  std::vector<const char*> text = {
      // clang-format off
               "OpCapability Shader",
               "OpCapability Float64",
          "%1 = OpExtInstImport \"GLSL.std.450\"",
               "OpMemoryModel Logical GLSL450",
               "OpEntryPoint Vertex %main \"main\"",
               "OpSource GLSL 450",
               "OpSourceExtension \"GL_GOOGLE_cpp_style_line_directive\"",
               "OpSourceExtension \"GL_GOOGLE_include_directive\"",
               "OpName %main \"main\"",
               "OpDecorate %3 SpecId 200",
               "OpDecorate %4 SpecId 201",
               "OpDecorate %5 SpecId 202",
               "OpDecorate %6 SpecId 203",
       "%void = OpTypeVoid",
          "%8 = OpTypeFunction %void",
        "%int = OpTypeInt 32 1",
          "%3 = OpSpecConstant %int 3",
      "%float = OpTypeFloat 32",
          "%4 = OpSpecConstant %float 3.14",
     "%double = OpTypeFloat 64",
          "%5 = OpSpecConstant %double 3.14159265358979",
       "%bool = OpTypeBool",
          "%6 = OpSpecConstantTrue %bool",
          "%13 = OpSpecConstantFalse %bool",
       "%main = OpFunction %void None %8",
         "%14 = OpLabel",
               "OpReturn",
               "OpFunctionEnd",
      // clang-format on
  };
  std::string expected_disassembly = SelectiveJoin(text, [](const char* line) {
    return std::string(line).find("SpecId") != std::string::npos;
  });
  std::vector<std::pair<const char*, const char*>> opcode_replacement_pairs = {
      {" OpSpecConstant ", " OpConstant "},
      {" OpSpecConstantTrue ", " OpConstantTrue "},
      {" OpSpecConstantFalse ", " OpConstantFalse "},
  };
  for (auto& p : opcode_replacement_pairs) {
    EXPECT_TRUE(FindAndReplace(&expected_disassembly, p.first, p.second))
        << "text:\n"
        << expected_disassembly << "\n"
        << "find_str:\n"
        << p.first << "\n"
        << "replace_str:\n"
        << p.second << "\n";
  }
  SinglePassRunAndCheck<opt::FreezeSpecConstantValuePass>(
      JoinAllInsts(text), expected_disassembly,
      /* skip_nop = */ true);
}
}  // anonymous namespace
