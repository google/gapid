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

#include "opt/libspirv.hpp"

namespace {

using namespace spvtools;

TEST(CppInterface, SuccessfulRoundTrip) {
  const std::string input_text = "%2 = OpSizeOf %1 %3\n";
  SpvTools t(SPV_ENV_UNIVERSAL_1_1);

  std::vector<uint32_t> binary;
  EXPECT_EQ(SPV_SUCCESS, t.Assemble(input_text, &binary));
  EXPECT_TRUE(binary.size() > 5u);
  EXPECT_EQ(SpvMagicNumber, binary[0]);
  EXPECT_EQ(SpvVersion, binary[1]);

  std::string output_text;
  EXPECT_EQ(SPV_SUCCESS, t.Disassemble(binary, &output_text));
  EXPECT_EQ(input_text, output_text);
}

TEST(CppInterface, AssembleWithWrongTargetEnv) {
  const std::string input_text = "%r = OpSizeOf %type %pointer";
  SpvTools t(SPV_ENV_UNIVERSAL_1_0);

  std::vector<uint32_t> binary;
  EXPECT_EQ(SPV_ERROR_INVALID_TEXT, t.Assemble(input_text, &binary));
}

TEST(CppInterface, DisassembleWithWrongTargetEnv) {
  const std::string input_text = "%r = OpSizeOf %type %pointer";
  SpvTools t11(SPV_ENV_UNIVERSAL_1_1);
  SpvTools t10(SPV_ENV_UNIVERSAL_1_0);

  std::vector<uint32_t> binary;
  EXPECT_EQ(SPV_SUCCESS, t11.Assemble(input_text, &binary));

  std::string output_text;
  EXPECT_EQ(SPV_ERROR_INVALID_BINARY, t10.Disassemble(binary, &output_text));
}

}  // anonymous namespace
