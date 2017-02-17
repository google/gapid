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

// This file defines the language constructs for representing a SPIR-V
// module in memory.

#ifndef LIBSPIRV_OPT_BASIC_BLOCK_H_
#define LIBSPIRV_OPT_BASIC_BLOCK_H_

#include <functional>
#include <memory>
#include <utility>
#include <vector>

#include "instruction.h"
#include "iterator.h"

namespace spvtools {
namespace ir {

class Function;

// A SPIR-V basic block.
class BasicBlock {
 public:
  using iterator = UptrVectorIterator<Instruction>;
  using const_iterator = UptrVectorIterator<Instruction, true>;

  // Creates a basic block with the given starting |label|.
  inline explicit BasicBlock(std::unique_ptr<Instruction> label);

  // Sets the enclosing function for this basic block.
  void SetParent(Function* function) { function_ = function; }
  // Appends an instruction to this basic block.
  inline void AddInstruction(std::unique_ptr<Instruction> i);
  // Prepends vector of Instructions to this basic block
  void PrependInstructions(std::vector<std::unique_ptr<Instruction>>& insts) {
    insts_.insert(insts_.begin(), std::make_move_iterator(insts.begin()), std::make_move_iterator(insts.end()));
  }

  void SetInstructions(std::vector<std::unique_ptr<Instruction>>&& insts) {
    insts_ = std::move(insts);
  }

  uint32_t GetLabelId() { return label_->result_id(); }

  iterator begin() { return iterator(&insts_, insts_.begin()); }
  iterator end() { return iterator(&insts_, insts_.end()); }
  const_iterator cbegin() { return const_iterator(&insts_, insts_.cbegin()); }
  const_iterator cend() { return const_iterator(&insts_, insts_.cend()); }

  // Runs the given function |f| on each instruction in this basic block, and
  // optionally on the debug line instructions that might precede them.
  inline void ForEachInst(const std::function<void(Instruction*)>& f,
                          bool run_on_debug_line_insts = false);
  inline void ForEachInst(const std::function<void(const Instruction*)>& f,
                          bool run_on_debug_line_insts = false) const;

 private:
  // The enclosing function.
  Function* function_;
  // The label starting this basic block.
  std::unique_ptr<Instruction> label_;
  // Instructions inside this basic block, but not the OpLabel.
  std::vector<std::unique_ptr<Instruction>> insts_;
};

inline BasicBlock::BasicBlock(std::unique_ptr<Instruction> label)
    : function_(nullptr), label_(std::move(label)) {}

inline void BasicBlock::AddInstruction(std::unique_ptr<Instruction> i) {
  insts_.emplace_back(std::move(i));
}

inline void BasicBlock::ForEachInst(const std::function<void(Instruction*)>& f,
                                    bool run_on_debug_line_insts) {
  if (label_) label_->ForEachInst(f, run_on_debug_line_insts);
  for (auto& inst : insts_) inst->ForEachInst(f, run_on_debug_line_insts);
}

inline void BasicBlock::ForEachInst(
    const std::function<void(const Instruction*)>& f,
    bool run_on_debug_line_insts) const {
  if (label_)
    static_cast<const Instruction*>(label_.get())
        ->ForEachInst(f, run_on_debug_line_insts);
  for (const auto& inst : insts_)
    static_cast<const Instruction*>(inst.get())
        ->ForEachInst(f, run_on_debug_line_insts);
}

}  // namespace ir
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_BASIC_BLOCK_H_
