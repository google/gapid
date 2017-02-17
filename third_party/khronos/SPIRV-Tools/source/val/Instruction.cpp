// Copyright (c) 2015-2016 The Khronos Group Inc.
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

#include "val/Instruction.h"

#include <utility>

using std::make_pair;

namespace libspirv {
#define OPERATOR(OP)                                                 \
  bool operator OP(const Instruction& lhs, const Instruction& rhs) { \
    return lhs.id() OP rhs.id();                                     \
  }                                                                  \
  bool operator OP(const Instruction& lhs, uint32_t rhs) {           \
    return lhs.id() OP rhs;                                          \
  }

OPERATOR(<)
OPERATOR(==)
#undef OPERATOR

Instruction::Instruction(const spv_parsed_instruction_t* inst,
                         Function* defining_function,
                         BasicBlock* defining_block)
    : words_(inst->words, inst->words + inst->num_words),
      operands_(inst->operands, inst->operands + inst->num_operands),
      inst_({words_.data(), inst->num_words, inst->opcode, inst->ext_inst_type,
             inst->type_id, inst->result_id, operands_.data(),
             inst->num_operands}),
      function_(defining_function),
      block_(defining_block),
      uses_() {}

void Instruction::RegisterUse(const Instruction* inst, uint32_t index) {
  uses_.push_back(make_pair(inst, index));
}
}  // namespace libspirv
