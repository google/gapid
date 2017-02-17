// Copyright (c) 2016 Google
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

#include "TestFixture.h"
#include "gmock/gmock.h"

namespace {

using ::spvtest::MakeInstruction;
using ::testing::Eq;

using OpTypePipeStorageTest = spvtest::TextToBinaryTest;

TEST_F(OpTypePipeStorageTest, OpcodeUnrecognizedInV10) {
  EXPECT_THAT(CompileFailure("%res = OpTypePipeStorage", SPV_ENV_UNIVERSAL_1_0),
              Eq("Invalid Opcode name 'OpTypePipeStorage'"));
}

TEST_F(OpTypePipeStorageTest, ArgumentCount) {
  EXPECT_THAT(
      CompileFailure("OpTypePipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq("Expected <result-id> at the beginning of an instruction, found "
         "'OpTypePipeStorage'."));
  EXPECT_THAT(
      CompiledInstructions("%res = OpTypePipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq(MakeInstruction(SpvOpTypePipeStorage, {1})));
  EXPECT_THAT(CompileFailure("%res = OpTypePipeStorage %1 %2 %3 %4 %5",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("'=' expected after result id."));
}

using OpConstantPipeStorageTest = spvtest::TextToBinaryTest;

TEST_F(OpConstantPipeStorageTest, OpcodeUnrecognizedInV10) {
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 3 4 5",
                             SPV_ENV_UNIVERSAL_1_0),
              Eq("Invalid Opcode name 'OpConstantPipeStorage'"));
}

TEST_F(OpConstantPipeStorageTest, ArgumentCount) {
  EXPECT_THAT(
      CompileFailure("OpConstantPipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq("Expected <result-id> at the beginning of an instruction, found "
         "'OpConstantPipeStorage'."));
  EXPECT_THAT(
      CompileFailure("%1 = OpConstantPipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq("Expected operand, found end of stream."));
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 3 4",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Expected operand, found end of stream."));
  EXPECT_THAT(CompiledInstructions("%1 = OpConstantPipeStorage %2 3 4 5",
                                   SPV_ENV_UNIVERSAL_1_1),
              Eq(MakeInstruction(SpvOpConstantPipeStorage, {1, 2, 3, 4, 5})));
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 3 4 5 %6 %7",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("'=' expected after result id."));
}

TEST_F(OpConstantPipeStorageTest, ArgumentTypes) {
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 %3 4 5",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Invalid unsigned integer literal: %3"));
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 3 %4 5",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Invalid unsigned integer literal: %4"));
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage 2 3 4 5",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Expected id to start with %."));
  EXPECT_THAT(CompileFailure("%1 = OpConstantPipeStorage %2 3 4 \"ab\"",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Invalid unsigned integer literal: \"ab\""));
}

using OpCreatePipeFromPipeStorageTest = spvtest::TextToBinaryTest;

TEST_F(OpCreatePipeFromPipeStorageTest, OpcodeUnrecognizedInV10) {
  EXPECT_THAT(CompileFailure("%1 = OpCreatePipeFromPipeStorage %2 %3",
                             SPV_ENV_UNIVERSAL_1_0),
              Eq("Invalid Opcode name 'OpCreatePipeFromPipeStorage'"));
}

TEST_F(OpCreatePipeFromPipeStorageTest, ArgumentCount) {
  EXPECT_THAT(
      CompileFailure("OpCreatePipeFromPipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq("Expected <result-id> at the beginning of an instruction, found "
         "'OpCreatePipeFromPipeStorage'."));
  EXPECT_THAT(
      CompileFailure("%1 = OpCreatePipeFromPipeStorage", SPV_ENV_UNIVERSAL_1_1),
      Eq("Expected operand, found end of stream."));
  EXPECT_THAT(CompileFailure("%1 = OpCreatePipeFromPipeStorage %2 OpNop",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Expected operand, found next instruction instead."));
  EXPECT_THAT(CompiledInstructions("%1 = OpCreatePipeFromPipeStorage %2 %3",
                                   SPV_ENV_UNIVERSAL_1_1),
              Eq(MakeInstruction(SpvOpCreatePipeFromPipeStorage, {1, 2, 3})));
  EXPECT_THAT(CompileFailure("%1 = OpCreatePipeFromPipeStorage %2 %3 %4 %5",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("'=' expected after result id."));
}

TEST_F(OpCreatePipeFromPipeStorageTest, ArgumentTypes) {
  EXPECT_THAT(CompileFailure("%1 = OpCreatePipeFromPipeStorage \"\" %3",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Expected id to start with %."));
  EXPECT_THAT(CompileFailure("%1 = OpCreatePipeFromPipeStorage %2 3",
                             SPV_ENV_UNIVERSAL_1_1),
              Eq("Expected id to start with %."));
}

}  // anonymous namespace
