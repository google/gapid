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

#include "def_use_manager.h"

#include <cassert>
#include <functional>

#include "instruction.h"
#include "module.h"
#include "reflect.h"

namespace spvtools {
namespace opt {
namespace analysis {

void DefUseManager::AnalyzeInstDefUse(ir::Instruction* inst) {
  const uint32_t def_id = inst->result_id();
  if (def_id != 0) {
    auto iter = id_to_def_.find(def_id);
    if (iter != id_to_def_.end()) {
      // Clear the original instruction that defining the same result id of the
      // new instruction.
      ClearInst(iter->second);
    }
    id_to_def_[def_id] = inst;
  } else {
    ClearInst(inst);
  }

  // Create entry for the given instruction. Note that the instruction may
  // not have any in-operands. In such cases, we still need a entry for those
  // instructions so this manager knows it has seen the instruction later.
  inst_to_used_ids_[inst] = {};

  for (uint32_t i = 0; i < inst->NumOperands(); ++i) {
    switch (inst->GetOperand(i).type) {
      // For any id type but result id type
      case SPV_OPERAND_TYPE_ID:
      case SPV_OPERAND_TYPE_TYPE_ID:
      case SPV_OPERAND_TYPE_MEMORY_SEMANTICS_ID:
      case SPV_OPERAND_TYPE_SCOPE_ID: {
        uint32_t use_id = inst->GetSingleWordOperand(i);
        // use_id is used by the instruction generating def_id.
        id_to_uses_[use_id].push_back({inst, i});
        inst_to_used_ids_[inst].push_back(use_id);
      } break;
      default:
        break;
    }
  }
}

ir::Instruction* DefUseManager::GetDef(uint32_t id) {
  auto iter = id_to_def_.find(id);
  if (iter == id_to_def_.end()) return nullptr;
  return iter->second;
}

UseList* DefUseManager::GetUses(uint32_t id) {
  auto iter = id_to_uses_.find(id);
  if (iter == id_to_uses_.end()) return nullptr;
  return &iter->second;
}

const UseList* DefUseManager::GetUses(uint32_t id) const {
  const auto iter = id_to_uses_.find(id);
  if (iter == id_to_uses_.end()) return nullptr;
  return &iter->second;
}

std::vector<ir::Instruction*> DefUseManager::GetAnnotations(
    uint32_t id) const {
  std::vector<ir::Instruction*> annos;
  const auto* uses = GetUses(id);
  if (!uses)  return annos;
  for (const auto& c : *uses) {
    if (ir::IsAnnotationInst(c.inst->opcode())) {
      annos.push_back(c.inst);
    }
  }
  return annos;
}

std::vector<ir::Instruction*> DefUseManager::GetVariables() const {
  std::vector<ir::Instruction*> insts;
  for(auto iter = id_to_def_.begin(); iter != id_to_def_.end(); iter++) {
    if (ir::IsVariableInst(iter->second->opcode())) {
      insts.push_back(iter->second);
    }
  }
  return insts;
};

bool DefUseManager::KillDef(uint32_t id) {
  auto iter = id_to_def_.find(id);
  if (iter == id_to_def_.end()) return false;
  KillInst(iter->second);
  return true;
}

void DefUseManager::KillInst(ir::Instruction* inst) {
  if (!inst) return;
  ClearInst(inst);
  inst->ToNop();
}

bool DefUseManager::ReplaceAllUsesWith(uint32_t before, uint32_t after) {
  if (before == after) return false;
  if (id_to_uses_.count(before) == 0) return false;

  for (auto it = id_to_uses_[before].cbegin(); it != id_to_uses_[before].cend();
       ++it) {
    const uint32_t type_result_id_count =
        (it->inst->result_id() != 0) + (it->inst->type_id() != 0);

    if (it->operand_index < type_result_id_count) {
      // Update the type_id. Note that result id is immutable so it should
      // never be updated.
      if (it->inst->type_id() != 0 && it->operand_index == 0) {
        it->inst->SetResultType(after);
      } else if (it->inst->type_id() == 0) {
        assert(false &&
               "Result type id considered as using while the instruction "
               "doesn't have a result type id.");
      } else {
        assert(false && "Trying Setting the result id which is immutable.");
      }
    } else {
      // Update an in-operand.
      uint32_t in_operand_pos = it->operand_index - type_result_id_count;
      // Make the modification in the instruction.
      it->inst->SetInOperand(in_operand_pos, {after});
    }
    // Register the use of |after| id into id_to_uses_.
    // TODO(antiagainst): de-duplication.
    id_to_uses_[after].push_back({it->inst, it->operand_index});
  }
  id_to_uses_.erase(before);
  return true;
}

void DefUseManager::AnalyzeDefUse(ir::Module* module) {
  if (!module) return;
  module->ForEachInst(std::bind(&DefUseManager::AnalyzeInstDefUse, this,
                                std::placeholders::_1));
}

void DefUseManager::ClearInst(ir::Instruction* inst) {
  auto iter = inst_to_used_ids_.find(inst);
  if (iter != inst_to_used_ids_.end()) {
    EraseUseRecordsOfOperandIds(inst);
    if (inst->result_id() != 0) {
      id_to_uses_.erase(inst->result_id());  // Remove all uses of this id.
      id_to_def_.erase(inst->result_id());
    }
  }
}

void DefUseManager::EraseUseRecordsOfOperandIds(const ir::Instruction* inst) {
  // Go through all ids used by this instruction, remove this instruction's
  // uses of them.
  auto iter = inst_to_used_ids_.find(inst);
  if (iter != inst_to_used_ids_.end()) {
    for (const auto use_id : iter->second) {
      auto uses_iter = id_to_uses_.find(use_id);
      if (uses_iter == id_to_uses_.end()) continue;
      auto& uses = uses_iter->second;
      for (auto it = uses.begin(); it != uses.end();) {
        if (it->inst == inst) {
          it = uses.erase(it);
        } else {
          ++it;
        }
      }
      if (uses.empty()) id_to_uses_.erase(use_id);
    }
    inst_to_used_ids_.erase(inst);
  }
}

}  // namespace analysis
}  // namespace opt
}  // namespace spvtools
