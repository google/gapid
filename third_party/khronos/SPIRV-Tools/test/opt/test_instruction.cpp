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

#include "opt/instruction.h"

#include "gmock/gmock.h"

#include "spirv-tools/libspirv.h"
#include "UnitSPIRV.h"

namespace {

using spvtest::MakeInstruction;
using spvtools::ir::Instruction;
using spvtools::ir::Operand;
using ::testing::Eq;

TEST(InstructionTest, CreateTrivial) {
  Instruction empty;
  EXPECT_EQ(SpvOpNop, empty.opcode());
  EXPECT_EQ(0u, empty.type_id());
  EXPECT_EQ(0u, empty.result_id());
  EXPECT_EQ(0u, empty.NumOperands());
  EXPECT_EQ(0u, empty.NumOperandWords());
  EXPECT_EQ(0u, empty.NumInOperandWords());
  EXPECT_EQ(empty.cend(), empty.cbegin());
  EXPECT_EQ(empty.end(), empty.begin());
}

TEST(InstructionTest, CreateWithOpcodeAndNoOperands) {
  Instruction inst(SpvOpReturn);
  EXPECT_EQ(SpvOpReturn, inst.opcode());
  EXPECT_EQ(0u, inst.type_id());
  EXPECT_EQ(0u, inst.result_id());
  EXPECT_EQ(0u, inst.NumOperands());
  EXPECT_EQ(0u, inst.NumOperandWords());
  EXPECT_EQ(0u, inst.NumInOperandWords());
  EXPECT_EQ(inst.cend(), inst.cbegin());
  EXPECT_EQ(inst.end(), inst.begin());
}

// The words for an OpTypeInt for 32-bit signed integer resulting in Id 44.
uint32_t kSampleInstructionWords[] = {(4 << 16) | uint32_t(SpvOpTypeInt), 44,
                                      32, 1};
// The operands that would be parsed from kSampleInstructionWords
spv_parsed_operand_t kSampleParsedOperands[] = {
    {1, 1, SPV_OPERAND_TYPE_RESULT_ID, SPV_NUMBER_NONE, 0},
    {2, 1, SPV_OPERAND_TYPE_LITERAL_INTEGER, SPV_NUMBER_UNSIGNED_INT, 32},
    {3, 1, SPV_OPERAND_TYPE_LITERAL_INTEGER, SPV_NUMBER_UNSIGNED_INT, 1},
};

// A valid parse of kSampleParsedOperands.
spv_parsed_instruction_t kSampleParsedInstruction = {kSampleInstructionWords,
                                                     uint16_t(4),
                                                     uint16_t(SpvOpTypeInt),
                                                     SPV_EXT_INST_TYPE_NONE,
                                                     0,   // type id
                                                     44,  // result id
                                                     kSampleParsedOperands,
                                                     3};
TEST(InstructionTest, CreateWithOpcodeAndOperands) {
  Instruction inst(kSampleParsedInstruction);
  EXPECT_EQ(SpvOpTypeInt, inst.opcode());
  EXPECT_EQ(0u, inst.type_id());
  EXPECT_EQ(44u, inst.result_id());
  EXPECT_EQ(3u, inst.NumOperands());
  EXPECT_EQ(3u, inst.NumOperandWords());
  EXPECT_EQ(2u, inst.NumInOperandWords());
}

TEST(InstructionTest, GetOperand) {
  Instruction inst(kSampleParsedInstruction);
  EXPECT_THAT(inst.GetOperand(0).words, Eq(std::vector<uint32_t>{44}));
  EXPECT_THAT(inst.GetOperand(1).words, Eq(std::vector<uint32_t>{32}));
  EXPECT_THAT(inst.GetOperand(2).words, Eq(std::vector<uint32_t>{1}));
}

TEST(InstructionTest, GetInOperand) {
  Instruction inst(kSampleParsedInstruction);
  EXPECT_THAT(inst.GetInOperand(0).words, Eq(std::vector<uint32_t>{32}));
  EXPECT_THAT(inst.GetInOperand(1).words, Eq(std::vector<uint32_t>{1}));
}

TEST(InstructionTest, OperandConstIterators) {
  Instruction inst(kSampleParsedInstruction);
  // Spot check iteration across operands.
  auto cbegin = inst.cbegin();
  auto cend = inst.cend();
  EXPECT_NE(cend, inst.cbegin());

  auto citer = inst.cbegin();
  for (int i = 0; i < 3; ++i, ++citer) {
    const auto& operand = *citer;
    EXPECT_THAT(operand.type, Eq(kSampleParsedOperands[i].type));
    EXPECT_THAT(operand.words,
                Eq(std::vector<uint32_t>{kSampleInstructionWords[i + 1]}));
    EXPECT_NE(cend, citer);
  }
  EXPECT_EQ(cend, citer);

  // Check that cbegin and cend have not changed.
  EXPECT_EQ(cbegin, inst.cbegin());
  EXPECT_EQ(cend, inst.cend());

  // Check arithmetic.
  const Operand& operand2 = *(inst.cbegin() + 2);
  EXPECT_EQ(SPV_OPERAND_TYPE_LITERAL_INTEGER, operand2.type);
}

TEST(InstructionTest, OperandIterators) {
  Instruction inst(kSampleParsedInstruction);
  // Spot check iteration across operands, with mutable iterators.
  auto begin = inst.begin();
  auto end = inst.end();
  EXPECT_NE(end, inst.begin());

  auto iter = inst.begin();
  for (int i = 0; i < 3; ++i, ++iter) {
    const auto& operand = *iter;
    EXPECT_THAT(operand.type, Eq(kSampleParsedOperands[i].type));
    EXPECT_THAT(operand.words,
                Eq(std::vector<uint32_t>{kSampleInstructionWords[i + 1]}));
    EXPECT_NE(end, iter);
  }
  EXPECT_EQ(end, iter);

  // Check that begin and end have not changed.
  EXPECT_EQ(begin, inst.begin());
  EXPECT_EQ(end, inst.end());

  // Check arithmetic.
  Operand& operand2 = *(inst.begin() + 2);
  EXPECT_EQ(SPV_OPERAND_TYPE_LITERAL_INTEGER, operand2.type);

  // Check mutation through an iterator.
  operand2.type = SPV_OPERAND_TYPE_TYPE_ID;
  EXPECT_EQ(SPV_OPERAND_TYPE_TYPE_ID, (*(inst.cbegin() + 2)).type);
}

}  // anonymous namespace
