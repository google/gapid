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

#ifndef SPV_MANAGER_H_
#define SPV_MANAGER_H_

#include "third_party/SPIRV-Headers/include/spirv/1.2/spirv.hpp"
#include "third_party/SPIRV-Tools/include/spirv-tools/libspirv.h"
#include "third_party/SPIRV-Tools/source/assembly_grammar.h"
#include "third_party/SPIRV-Tools/source/opcode.h"
#include "third_party/SPIRV-Tools/source/operand.h"
#include "third_party/SPIRV-Tools/source/opt/def_use_manager.h"
#include "third_party/SPIRV-Tools/include/spirv-tools/libspirv.hpp"
#include "third_party/SPIRV-Tools/source/opt/build_module.h"
#include "third_party/SPIRV-Tools/source/opt/make_unique.h"
#include "third_party/SPIRV-Tools/source/opt/reflect.h"
#include "third_party/SPIRV-Tools/source/opt/type_manager.h"
#include "third_party/SPIRV-Tools/source/opt/types.h"

#include "name_manager.h"
#include "common.h"

#include "libmanager.h"

#include <cstring>
#include <iostream>
#include <map>
#include <set>
#include <stdint.h>
#include <string>
#include <utility>  // std::pair, std::make_pair
#include <vector>

namespace spvmanager {

using spvtools::ir::Module;
using spvtools::ir::Instruction;
using spvtools::ir::BasicBlock;
using spvtools::ir::Function;
using spvtools::opt::analysis::TypeManager;
using spvtools::opt::analysis::Type;
using spvtools::opt::analysis::DefUseManager;

#define MANAGER_SPV_ENV SPV_ENV_UNIVERSAL_1_1  // from spirv-tools

typedef std::map<uint32_t, uint32_t> map_uint;
typedef std::pair<uint32_t, uint32_t> name_type;

class SpvManager {
 public:
  SpvManager(const std::vector<uint32_t>& spv_binary) {
    globals.result.name = "gapid_result";
    globals.sampler.name = "gapid_sampler";
    globals.coordinate.name = "gapid_coor";
    globals.curr_step.name = "gapid_curr_step";
    globals.viewID.name = "gapid_gl_ViewID_OVR";
    globals.uint_type_id = 0;
    globals.void_id = 0;
    globals.label_print_id = 0;

    auto print_msg_to_stderr = [](spv_message_level_t, const char*, const spv_position_t&, const char* m) {
      std::cerr << "error: " << m << std::endl;
    };

    std::unique_ptr<spv_context_t> context(spvContextCreate(MANAGER_SPV_ENV));
    grammar.reset(new libspirv::AssemblyGrammar(context.get()));
    // init module
    module = spvtools::BuildModule(MANAGER_SPV_ENV, print_msg_to_stderr, spv_binary.data(), spv_binary.size());
    type_mgr.reset(new TypeManager(print_msg_to_stderr, *module));
    def_use_mgr.reset(new DefUseManager(print_msg_to_stderr, module.get()));
    name_mgr.reset(new namemanager::NameManager(module.get()));
  }

  void addOutputForInputs(std::string = "_out");
  void mapDeclarationNames(std::string = "x");
  void renameViewIndex();
  void removeLayoutLocations();
  void initLocals();
  void makeSpvDebuggable();
  std::vector<unsigned int> getSpvBinary();
  debug_instructions_t* getDebugInstructions();

 private:
  struct Variable {
    std::string name;
    uint32_t ref_id = 0;
    uint32_t type_id = 0;
  };

  struct ManagerGlobals {
    Variable result;
    Variable sampler;
    Variable coordinate;
    Variable curr_step;
    Variable viewID;
    uint32_t uint_type_id;
    uint32_t void_id;
    uint32_t label_print_id;
  };

  std::unique_ptr<libspirv::AssemblyGrammar> grammar;
  std::unique_ptr<Module> module;
  std::unique_ptr<TypeManager> type_mgr;
  std::unique_ptr<DefUseManager> def_use_mgr;
  std::unique_ptr<namemanager::NameManager> name_mgr;

  ManagerGlobals globals;

  // accumulator
  std::vector<std::unique_ptr<Instruction>> curr_block_insts;
  map_uint typeid_to_printid;
  map_uint consts;

  std::vector<spvtools::ir::Operand> makeOperands(
      spv_opcode_desc&, std::initializer_list<std::initializer_list<uint32_t>>&, const char* = nullptr);
  std::unique_ptr<Instruction> makeInstruction(SpvOp_, uint32_t, uint32_t,
                                               std::initializer_list<std::initializer_list<uint32_t>>,
                                               const char* = nullptr);
  std::unique_ptr<BasicBlock> makeBasicBlock(uint32_t, Function*,
                                             std::vector<std::unique_ptr<Instruction>>&&);

  uint32_t addName(const char*);
  uint32_t addConstant(uint32_t, std::initializer_list<uint32_t>);
  uint32_t addTypeInst(SpvOp_, std::initializer_list<std::initializer_list<uint32_t>>, uint32_t = 0);
  void addVariable(uint32_t, uint32_t, spv::StorageClass);
  void addGlobalVariable(spv::StorageClass, Variable*);
  uint32_t addFunction(const char*, uint32_t, uint32_t);

  uint32_t collectInstWithResult(SpvOp_, std::initializer_list<std::initializer_list<uint32_t>> = {{}},
                                 uint32_t = 0);
  void collectInstWithoutResult(SpvOp_, std::initializer_list<std::initializer_list<uint32_t>> = {{}},
                                uint32_t = 0);
  uint32_t collectCompositeConstruct(std::initializer_list<std::initializer_list<uint32_t>>, uint32_t);
  void collectCondition(uint32_t, uint32_t);
  uint32_t collectTypeConversion(name_type, uint32_t);
  void collectPrintCall(name_type, uint32_t = 0);
  void collectPrintChain(name_type);

  void declareDebugVariables();
  void declarePrints();
  void setStepVariable();
  void insertPrintCallsIntoFunctions();
  void moveCollectedBlockInsts(BasicBlock::iterator&);
  void insertPrintCallsIntoBlock(BasicBlock&);
  uint32_t insertPrintDeclaration(uint32_t);

  uint32_t getVariableTypeId(uint32_t);
  uint32_t getTypeToConvert(const Type*);
  uint32_t getArrayLength(const Type*);
  const Type* getPointeeIfPointer(uint32_t);
  uint32_t getPrintFunction(uint32_t);

  bool isConvertedType(const Type*);
  bool isDebugFunction(Function&);
  bool isDebugInstruction(Instruction*);
  bool isArgStoreInst(BasicBlock::iterator, BasicBlock::iterator);
  bool isBuiltInDecoration(const Instruction&) const;
  bool isInputVariable(const Instruction&) const;
  uint32_t getConstId(uint32_t);
  uint32_t getUnique();

  uint32_t TypeToId(const Type*);
  Type* IdToType(uint32_t);

  void appendDebugInstruction(std::vector<instruction_t>*, Instruction*);
};

}  // namespace spvmanager

#endif  // SPV_MANAGER_H_
