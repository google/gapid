/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "name_manager.h"
#include <assert.h>
#include "common.h"

namespace namemanager {

/***********************************************************************
 **************************** public ***********************************
 ***********************************************************************/
/**
 * Returns string of the variable with a given id and without offset
 **/
std::string NameManager::getStrName(uint32_t id) {
  return getStrName(id_offset(id, NONMEMBER_OFFSET));
}

/**
 * Returns string of the variable with a given id and offset
 **/
std::string NameManager::getStrName(id_offset id) {
  name_map::iterator it = id_to_inst.find(id);
  std::string res;

  assert(it != id_to_inst.end() &&
         "getStrName: not recognized instruction id with given offset.");
  if (it != id_to_inst.end())
    res = extractString(
        it->second->GetOperand(getStringPosition(it->second->opcode())).words);
  return res;
}

void NameManager::addName(Instruction* inst) {
  if ((inst->opcode() == SpvOpName && inst->NumOperands() >= 2) ||
      (inst->opcode() == SpvOpMemberName && inst->NumOperands() >= 3)) {
    uint32_t id = inst->GetSingleWordOperand(0);
    uint32_t offset = getNameOffset(inst);
    id_to_inst.insert(
        std::pair<id_offset, Instruction*>(id_offset(id, offset), inst));
    names_ids.insert(id);
  }
}

void NameManager::setIfName(id_offset id, std::string new_name) {
  name_map::iterator it = id_to_inst.find(id);

  if (it != id_to_inst.end()) {
    uint32_t string_pos = getStringPosition(it->second->opcode());
    it->second->SetInOperand(string_pos, makeVector(new_name.c_str()));
  }
}

/**
 * Check if given name is deprecated name.
 **/
bool NameManager::isDeprecatedBuiltInName(std::string name) const {
  if (name == "gl_FragColor" || name == "gl_FragData")
    return true;
  else
    return false;
}

/***********************************************************************
 **************************** private **********************************
 ***********************************************************************/
void NameManager::collectNames(Module* module) {
  // debugs2 returns OpName and OpMemeberName instructions.
  for (auto& inst : module->debugs2()) {
    addName(&inst);
  }
}

uint32_t NameManager::getStringPosition(SpvOp_ op) {
  uint32_t res = 0;
  switch (op) {
    case SpvOpName:
      res = 1;
      break;
    case SpvOpMemberName:
      res = 2;
      break;
    default:
      break;
  }
  assert(res != 0 && "getStringPosition: given opcode has to be name opcode.");
  return res;
}

uint32_t NameManager::getNameOffset(Instruction* inst) {
  uint32_t res = NONMEMBER_OFFSET;
  switch (inst->opcode()) {
    case SpvOpMemberName:
      res = inst->GetSingleWordOperand(1);
      break;
    case SpvOpName:
    default:
      break;
  }
  return res;
}
}  // namespace namemanager
