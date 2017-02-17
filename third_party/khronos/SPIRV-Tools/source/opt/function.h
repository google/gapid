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

#ifndef LIBSPIRV_OPT_CONSTRUCTS_H_
#define LIBSPIRV_OPT_CONSTRUCTS_H_

#include <functional>
#include <memory>
#include <utility>
#include <vector>

#include "basic_block.h"
#include "instruction.h"
#include "iterator.h"

namespace spvtools {
namespace ir {

class Module;

// A SPIR-V function.
class Function {
 public:
  using iterator = UptrVectorIterator<BasicBlock>;
  using const_iterator = UptrVectorIterator<BasicBlock, true>;

  // Creates a function instance declared by the given OpFunction instruction
  // |def_inst|.
  inline explicit Function(std::unique_ptr<Instruction> def_inst);

  uint32_t GetNameId() { return def_inst_->result_id(); }

  // Sets the enclosing module for this function.
  void SetParent(Module* module) { module_ = module; }
  // Appends a parameter to this function.
  inline void AddParameter(std::unique_ptr<Instruction> p);
  // Appends a basic block to this function.
  inline void AddBasicBlock(std::unique_ptr<BasicBlock> b);

  // Saves the given function end instruction.
  inline void SetFunctionEnd(std::unique_ptr<Instruction> end_inst);

  iterator begin() { return iterator(&blocks_, blocks_.begin()); }
  iterator end() { return iterator(&blocks_, blocks_.end()); }
  const_iterator cbegin() { return const_iterator(&blocks_, blocks_.cbegin()); }
  const_iterator cend() { return const_iterator(&blocks_, blocks_.cend()); }

  // Runs the given function |f| on each instruction in this function, and
  // optionally on debug line instructions that might precede them.
  void ForEachInst(const std::function<void(Instruction*)>& f,
                   bool run_on_debug_line_insts = false);
  void ForEachInst(const std::function<void(const Instruction*)>& f,
                   bool run_on_debug_line_insts = false) const;

 private:
  // The enclosing module.
  Module* module_;
  // The OpFunction instruction that begins the definition of this function.
  std::unique_ptr<Instruction> def_inst_;
  // All parameters to this function.
  std::vector<std::unique_ptr<Instruction>> params_;
  // All basic blocks inside this function.
  std::vector<std::unique_ptr<BasicBlock>> blocks_;
  // The OpFunctionEnd instruction.
  std::unique_ptr<Instruction> end_inst_;
};

inline Function::Function(std::unique_ptr<Instruction> def_inst)
    : module_(nullptr), def_inst_(std::move(def_inst)), end_inst_() {}

inline void Function::AddParameter(std::unique_ptr<Instruction> p) {
  params_.emplace_back(std::move(p));
}

inline void Function::AddBasicBlock(std::unique_ptr<BasicBlock> b) {
  blocks_.emplace_back(std::move(b));
}

inline void Function::SetFunctionEnd(std::unique_ptr<Instruction> end_inst) {
  end_inst_ = std::move(end_inst);
}

}  // namespace ir
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_CONSTRUCTS_H_
