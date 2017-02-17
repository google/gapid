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

#include "module.h"

#include <algorithm>

#include "operand.h"
#include "reflect.h"

namespace spvtools {
namespace ir {

std::vector<Instruction*> Module::GetTypes() {
  std::vector<Instruction*> insts;
  for (uint32_t i = 0; i < types_values_.size(); ++i) {
    if (IsTypeInst(types_values_[i]->opcode()))
      insts.push_back(types_values_[i].get());
  }
  return insts;
};

std::vector<const Instruction*> Module::GetTypes() const {
  std::vector<const Instruction*> insts;
  for (uint32_t i = 0; i < types_values_.size(); ++i) {
    if (IsTypeInst(types_values_[i]->opcode()))
      insts.push_back(types_values_[i].get());
  }
  return insts;
};

std::vector<Instruction*> Module::GetConstants() {
  std::vector<Instruction*> insts;
  for (uint32_t i = 0; i < types_values_.size(); ++i) {
    if (IsConstantInst(types_values_[i]->opcode()))
      insts.push_back(types_values_[i].get());
  }
  return insts;
};

std::vector<const Instruction*> Module::GetConstants() const {
  std::vector<const Instruction*> insts;
  for (uint32_t i = 0; i < types_values_.size(); ++i) {
    if (IsConstantInst(types_values_[i]->opcode()))
      insts.push_back(types_values_[i].get());
  }
  return insts;
};

void Module::ForEachInst(const std::function<void(Instruction*)>& f,
                         bool run_on_debug_line_insts) {
#define DELEGATE(i) i->ForEachInst(f, run_on_debug_line_insts)
  for (auto& i : capabilities_) DELEGATE(i);
  for (auto& i : extensions_) DELEGATE(i);
  for (auto& i : ext_inst_imports_) DELEGATE(i);
  if (memory_model_) DELEGATE(memory_model_);
  for (auto& i : entry_points_) DELEGATE(i);
  for (auto& i : execution_modes_) DELEGATE(i);
  for (auto& i : debugs_) DELEGATE(i);
  for (auto& i : annotations_) DELEGATE(i);
  for (auto& i : types_values_) DELEGATE(i);
  for (auto& i : functions_) DELEGATE(i);
#undef DELEGATE
}

void Module::ForEachInst(const std::function<void(const Instruction*)>& f,
                         bool run_on_debug_line_insts) const {
#define DELEGATE(i)                                      \
  static_cast<const Instruction*>(i.get())->ForEachInst( \
      f, run_on_debug_line_insts)
  for (auto& i : capabilities_) DELEGATE(i);
  for (auto& i : extensions_) DELEGATE(i);
  for (auto& i : ext_inst_imports_) DELEGATE(i);
  if (memory_model_) DELEGATE(memory_model_);
  for (auto& i : entry_points_) DELEGATE(i);
  for (auto& i : execution_modes_) DELEGATE(i);
  for (auto& i : debugs_) DELEGATE(i);
  for (auto& i : annotations_) DELEGATE(i);
  for (auto& i : types_values_) DELEGATE(i);
  for (auto& i : functions_) {
    static_cast<const Function*>(i.get())->ForEachInst(f,
                                                       run_on_debug_line_insts);
  }
#undef DELEGATE
}

void Module::ToBinary(std::vector<uint32_t>* binary, bool skip_nop) const {
  binary->push_back(header_.magic_number);
  binary->push_back(header_.version);
  // TODO(antiagainst): should we change the generator number?
  binary->push_back(header_.generator);
  binary->push_back(header_.bound);
  binary->push_back(header_.reserved);

  auto write_inst = [this, binary, skip_nop](const Instruction* i) {
    if (!(skip_nop && i->IsNop())) i->ToBinaryWithoutAttachedDebugInsts(binary);
  };
  ForEachInst(write_inst, true);
}

uint32_t Module::ComputeIdBound() const {
  uint32_t highest = 0;

  ForEachInst(
      [&highest](const Instruction* inst) {
        for (const auto& operand : *inst) {
          if (spvIsIdType(operand.type)) {
            highest = std::max(highest, operand.words[0]);
          }
        }
      },
      true /* scan debug line insts as well */);

  return highest + 1;
}

}  // namespace ir
}  // namespace spvtools
