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

#ifndef NAME_MANAGER_H_
#define NAME_MANAGER_H_

#include "third_party/SPIRV-Tools/include/spirv-tools/libspirv.h"
#include "third_party/SPIRV-Tools/include/spirv-tools/libspirv.hpp"
#include "third_party/SPIRV-Tools/source/opt/instruction.h"
#include "third_party/SPIRV-Tools/source/opt/iterator.h"
#include "third_party/SPIRV-Tools/source/opt/module.h"

#include <map>
#include <unordered_set>

using spvtools::ir::Instruction;
using spvtools::ir::Module;

namespace namemanager {

#define NONMEMBER_OFFSET ~0u

typedef std::pair<uint32_t, uint32_t> id_offset;
typedef std::map<id_offset, Instruction*> name_map;

class NameManager {
 public:
  NameManager(Module* module) { collectNames(module); }

  void addName(Instruction*);
  std::string getStrName(uint32_t);
  std::string getStrName(id_offset);
  std::unordered_set<uint32_t> getNamedIds() { return names_ids; };

  name_map::iterator begin() { return id_to_inst.begin(); };
  name_map::iterator end() { return id_to_inst.end(); };
  void setIfName(id_offset, std::string);
  bool isDeprecatedBuiltInName(std::string) const;

 private:
  name_map id_to_inst;
  std::unordered_set<uint32_t> names_ids;
  void collectNames(Module*);
  uint32_t getStringPosition(SpvOp_);
  uint32_t getNameOffset(Instruction*);
};
}  // namespace namemanager

#endif  // NAME_MANAGER_H_
