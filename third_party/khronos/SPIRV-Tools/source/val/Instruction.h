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

#ifndef LIBSPIRV_VAL_INSTRUCTION_H_
#define LIBSPIRV_VAL_INSTRUCTION_H_

#include <cstdint>

#include <functional>
#include <utility>
#include <vector>

#include "spirv-tools/libspirv.h"
#include "table.h"

namespace libspirv {

class BasicBlock;
class Function;

/// Wraps the spv_parsed_instruction struct along with use and definition of the
/// instruction's result id
class Instruction {
 public:
  explicit Instruction(const spv_parsed_instruction_t* inst,
                       Function* defining_function = nullptr,
                       BasicBlock* defining_block = nullptr);

  /// Registers the use of the Instruction in instruction \p inst at \p index
  void RegisterUse(const Instruction* inst, uint32_t index);

  uint32_t id() const { return inst_.result_id; }
  uint32_t type_id() const { return inst_.type_id; }
  SpvOp opcode() const { return static_cast<SpvOp>(inst_.opcode); }

  /// Returns the Function where the instruction was defined. nullptr if it was
  /// defined outside of a Function
  const Function* function() const { return function_; }

  /// Returns the BasicBlock where the instruction was defined. nullptr if it
  /// was defined outside of a BasicBlock
  const BasicBlock* block() const { return block_; }

  /// Returns a vector of pairs of all references to this instruction's result
  /// id. The first element is the instruction in which this result id was
  /// referenced and the second is the index of the word in that instruction
  /// where this result id appeared
  const std::vector<std::pair<const Instruction*, uint32_t>>& uses() const {
    return uses_;
  }

  /// The word used to define the Instruction
  uint32_t word(size_t index) const { return words_[index]; }

  /// The words used to define the Instruction
  const std::vector<uint32_t>& words() const { return words_; }

  /// The operands of the Instruction
  const std::vector<spv_parsed_operand_t>& operands() const {
    return operands_;
  }

 private:
  const std::vector<uint32_t> words_;
  const std::vector<spv_parsed_operand_t> operands_;
  spv_parsed_instruction_t inst_;

  /// The function in which this instruction was declared
  Function* function_;

  /// The basic block in which this instruction was declared
  BasicBlock* block_;

  /// This is a vector of pairs of all references to this instruction's result
  /// id. The first element is the instruction in which this result id was
  /// referenced and the second is the index of the word in the referencing
  /// instruction where this instruction appeared
  std::vector<std::pair<const Instruction*, uint32_t>> uses_;
};

#define OPERATOR(OP)                                                \
  bool operator OP(const Instruction& lhs, const Instruction& rhs); \
  bool operator OP(const Instruction& lhs, uint32_t rhs)

OPERATOR(<);
OPERATOR(==);
#undef OPERATOR

}  // namespace libspirv

// custom specialization of std::hash for Instruction
namespace std {
template <>
struct hash<libspirv::Instruction> {
  typedef libspirv::Instruction argument_type;
  typedef std::size_t result_type;
  result_type operator()(const argument_type& inst) const {
    return hash<uint32_t>()(inst.id());
  }
};
}  /// namespace std

#endif  // LIBSPIRV_VAL_INSTRUCTION_H_
