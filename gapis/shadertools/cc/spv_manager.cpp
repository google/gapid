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

#include "spv_manager.h"
#include <assert.h>

namespace spvmanager {

#define PRINT_NAME "print"
#define LABEL_PRINT_NAME "label"
#define PRINT_PARAM_NAME "value"
#define WIDTH 32
#define RESULT_VEC_SIZE 4
#define COORDINATE_SIZE 2
#define FIRST_CHAIN_INDEX_OPERAND 3

/***********************************************************************
 **************************** public ***********************************
 ***********************************************************************/
/**
* Changes spv handled by module to prepare debug instructions.
* Firstly, changes all non-build-in names to avoid versions incompatybility.
* Secondly, for every input creates output variable to show value of each input.
* Finally, insert print functions and call instructions to appropriate print
* after each 'store' inst.
**/
void SpvManager::makeSpvDebuggable() {
  declareDebugVariables();
  declarePrints();
  insertPrintCallsIntoFunctions();
}

/**
 * Adds output variable for each input.
 * Assigns value of input value to appropriate output in the beginning of main function.
 * Output variable has the same name as input with added prefix (option 'outs_pref').
 **/
void SpvManager::addOutputForInputs(std::string outs_pref) {
  Variable out_var;
  curr_block_insts.clear();
  std::unordered_set<uint32_t> names = name_mgr->getNamedIds();
  for (auto& id : names) {
    Instruction* def_inst = def_use_mgr->GetDef(id);
    if (isInputVariable(*def_inst)) {
      std::string in_name = name_mgr->getStrName(id);
      out_var.name = outs_pref + in_name;

      const Type* type = getPointeeIfPointer(def_inst->GetSingleWordOperand(0));
      out_var.type_id = TypeToId(type);
      addGlobalVariable(spv::StorageClassOutput, &out_var);
      // instructions stored in curr_block_insts
      uint32_t ref_id =
          collectInstWithResult(SpvOpLoad, {{def_inst->GetSingleWordOperand(1)}}, out_var.type_id);
      collectInstWithoutResult(SpvOpStore, {{out_var.ref_id}, {ref_id}});
    }
  }
  // add curr_block_insts to first block in main
  auto it = module->begin()->begin()->begin();
  for (auto& curr_block_inst : curr_block_insts) {
    it = it.InsertBefore(std::move(curr_block_inst));
    it++;
  }
  curr_block_insts.clear();
}

/**
 * Changes all declared names with adding name_pref.
 **/
void SpvManager::mapDeclarationNames(std::string name_pref) {
  std::unordered_set<uint32_t> builtIns;
  for (auto& annotations_it : module->annotations())
    if (isBuiltInDecoration(annotations_it)) builtIns.insert(annotations_it.GetSingleWordOperand(0));

  namemanager::name_map::iterator it;
  for (it = name_mgr->begin(); it != name_mgr->end(); it++) {
    std::string name = name_mgr->getStrName(it->first);
    if (name != "main")  // special word
      if (name_mgr->isDeprecatedBuiltInName(name) || builtIns.find(it->first.first) == builtIns.end()) {
        std::string out_name = name_pref + name;
        name_mgr->setIfName(it->first, out_name);
      }
  }
}

// Replace the built-in gl_ViewID_OVR with custom uniform.
void SpvManager::renameViewIndex() {
  for (auto& id : name_mgr->getNamedIds()) {
    if (name_mgr->getStrName(id) == "gl_ViewID_OVR") {
      Instruction* inst = def_use_mgr->GetDef(id);
      globals.viewID.type_id = addTypeInst(SpvOpTypeInt, {{WIDTH}, {0}});
      addGlobalVariable(spv::StorageClassUniformConstant, &globals.viewID);
      for (auto& use : *def_use_mgr->GetUses(id)) {
        if (use.inst->opcode() == SpvOpLoad) {
          use.inst->SetInOperand(0, {globals.viewID.ref_id});
        }
      }
      return;
    }
  }
}

// Explicitly initialize locals. This is a work around for SPIR-V cross problem
// where it may generate code which reads locals before initializing them.
// For example, "v.x = 42.0;" becomes "v = vec4(42.0, v.y, v.z, v.w);"
void SpvManager::initLocals() {
  for (auto& function : *module) {
    std::map<Instruction*, std::unique_ptr<Instruction>> replacement;
    std::unordered_set<Instruction*> seen;
    function.ForEachInst([this,&seen,&replacement](Instruction* inst) {
      if (inst->opcode() == SpvOpLoad || inst->opcode() == SpvOpStore) {
        Instruction* var = def_use_mgr->GetDef(inst->GetSingleWordInOperand(0));
        if (seen.insert(var).second) {
          if (inst->opcode() == SpvOpLoad &&
              var->opcode() == SpvOpVariable &&
              (var->GetSingleWordInOperand(0) == spv::StorageClassFunction ||
               var->GetSingleWordInOperand(0) == spv::StorageClassPrivate) &&
              var->NumInOperands() == 1 /* No initializer */) {
            // We have found Load before Store to a local variable (function scope)
            auto* vecType = getPointeeIfPointer(var->type_id())->AsVector();
            if (vecType && vecType->element_count() == 4) {
              // TODO: Handle more then just vec4 types, or fix the issue in SPIRV-Cross
              uint32_t elem_type_id = TypeToId(vecType->element_type());
              uint32_t elem_id = addConstant(elem_type_id, {0});
              uint32_t init_value_id = getUnique();
              module->AddGlobalValue(makeInstruction(SpvOpConstantComposite, TypeToId(vecType), init_value_id,
                                     {{elem_id, elem_id, elem_id, elem_id}}));
              replacement[var] = makeInstruction(var->opcode(), var->type_id(), var->result_id(),
                                                 {{var->GetSingleWordInOperand(0)}, {init_value_id}});
            }
          }
        }
      }
    });
    for (auto& basic_block : function) {
      for (auto it = basic_block.begin(); it != basic_block.end(); it++) {
        std::unique_ptr<Instruction> newInst(std::move(replacement[&*it]));
        if (newInst != nullptr) {
          it = it.Erase().InsertBefore(std::move(newInst));
        }
      }
    }
    for (auto it = module->types_values_begin(); it != module->types_values_end(); it++) {
      std::unique_ptr<Instruction> newInst(std::move(replacement[&*it]));
      if (newInst != nullptr) {
        it = it.Erase().InsertBefore(std::move(newInst));
      }
    }
  }
}

// Remove all layout(location = ...) qualifiers.
// For most use cases we should be binding/remapping locations regardless whether they
// are assigned explicit location or compiler generated one.
// TODO: Handle separate shader objects - we just hope the game uses consistent block names for now.
void SpvManager::removeLayoutLocations() {
  for (auto& it : module->annotations()) {
    if (it.opcode() == SpvOpDecorate && it.GetSingleWordOperand(1) == SpvDecorationLocation) {
      it.ToNop();
    }
  }
}

/**
 * Return binary currently handled by module
 **/
std::vector<unsigned int> SpvManager::getSpvBinary() {
  std::vector<unsigned int> res;
  module->ToBinary(&res, false);

  return res;
}

/**
 * Returns set of debug instructions searching through the module.
 **/
debug_instructions_t* SpvManager::getDebugInstructions() {
  debug_instructions_t* result = new debug_instructions_t{};
  std::vector<instruction_t> insts;
  module->ForEachInst(
      std::bind(&SpvManager::appendDebugInstruction, this, &insts, std::placeholders::_1), true);

  // Little bit of reordering to improve debug readability of the disassembly.
  std::multimap<uint32_t, instruction_t*> names;

  result->insts = new instruction_t[insts.size()];
  for (auto& inst : insts) {
    // Push names for later...
    if (inst.opcode == SpvOpName || inst.opcode == SpvOpMemberName) {
      names.emplace(inst.words[0], &inst);
      continue;
    }

    result->insts[result->insts_num++] = inst;

    // Pop any names related to this instruction...
    auto range = names.equal_range(inst.id);
    for (auto it = range.first; it != range.second; it++ ) {
      result->insts[result->insts_num++] = *it->second;
    }
    names.erase(range.first, range.second);
  }
  // Pop any left over names, just in case...
  for (auto& it : names) {
      result->insts[result->insts_num++] = *it.second;
  }

  return result;
}

/***********************************************************************
 **************************** private **********************************
 ***********************************************************************/
std::vector<spvtools::ir::Operand> SpvManager::makeOperands(
    spv_opcode_desc& op_desc, std::initializer_list<std::initializer_list<uint32_t>>& words,
    const char* literal_string) {
  std::vector<spvtools::ir::Operand> operands;
  std::initializer_list<std::initializer_list<uint32_t>>::iterator it = words.begin();
  for (int i = 0; i < op_desc->numTypes; i++) {
    spv_operand_type_t operand_type = op_desc->operandTypes[i];

    switch (operand_type) {
      case SPV_OPERAND_TYPE_TYPE_ID:
      case SPV_OPERAND_TYPE_RESULT_ID:
        break;

      case SPV_OPERAND_TYPE_LITERAL_STRING: {
        assert(literal_string && "makeOperands: SPV_OPERAND_TYPE_LITERAL_STRING is missing.");
        operands.emplace_back(operand_type, std::move(makeVector(literal_string)));
        break;
      }
      case SPV_OPERAND_TYPE_OPTIONAL_LITERAL_STRING: {
        if (literal_string) operands.emplace_back(operand_type, std::move(makeVector(literal_string)));
        break;
      }
      default: {
        assert((it != words.end() || spvOperandIsOptional(operand_type) ||
                spvOperandIsVariable(operand_type)) &&
               "makeOperands: too few operands to make vector of operands.");

        if (it != words.end()) {
          operands.emplace_back(operand_type, std::move(makeVector(*it)));
          it++;
        }
        break;
      }
    }
  }
  assert((it == words.end() || it->size() == 0) &&
         "makeOperands: too many operands to make vector of operands.");
  return operands;
}

std::unique_ptr<Instruction> SpvManager::makeInstruction(
    SpvOp_ op, uint32_t type_id, uint32_t result_id,
    std::initializer_list<std::initializer_list<uint32_t>> words, const char* literal_string) {
  spv_opcode_desc op_desc;
  spv_result_t res;
  std::unique_ptr<Instruction> inst;
  res = grammar->lookupOpcode(op, &op_desc);
  assert(res == SPV_SUCCESS && "makeInstruction: cannot find opcode description");
  if (res == SPV_SUCCESS) {
    inst = spvtools::MakeUnique<Instruction>(op, type_id, result_id,
                                             makeOperands(op_desc, words, literal_string));
    def_use_mgr->AnalyzeInstDefUse(inst.get());
  }
  return inst;
}

/**
 * Returns pointer to created BasicBlock (body of appropriate print function).
 **/
std::unique_ptr<BasicBlock> SpvManager::makeBasicBlock(uint32_t label_id, Function* parent,
                                                       std::vector<std::unique_ptr<Instruction>>&& body) {
  auto label_inst = makeInstruction(SpvOpLabel, 0, label_id, {{}});
  std::unique_ptr<BasicBlock> bb = spvtools::MakeUnique<BasicBlock>(std::move(label_inst));

  for(auto& inst : body) {
    bb->AddInstruction(std::move(inst));
  }

  bb->SetParent(parent);
  return bb;
}

/**
 * Functions that create Instructions and add them directly to the module.
**/
uint32_t SpvManager::addName(const char* name) {
  uint32_t ref_id = getUnique();
  auto inst = makeInstruction(SpvOpName, 0, 0, {{ref_id}}, name);
  name_mgr->addName(inst.get());
  module->AddDebugInst(std::move(inst));
  return ref_id;
}

/**
 * OpConstant has SPV_OPERAND_TYPE_TYPED_LITERAL_NUMBER operand.
 * This operand is a literal number whose format and size are determined by a previous operand
 * in this instruction. It's a signed integer, an unsigned integer, or a floating point number,
 * so can occupy multiple SPIR-V words
 **/
uint32_t SpvManager::addConstant(uint32_t type_id, std::initializer_list<uint32_t> num) {
  uint32_t result_id = getUnique();
  auto inst = makeInstruction(SpvOpConstant, type_id, result_id, {num});
  module->AddGlobalValue(std::move(inst));
  return result_id;
}

uint32_t SpvManager::addTypeInst(SpvOp_ op, std::initializer_list<std::initializer_list<uint32_t>> words,
                                 uint32_t type_id) {
  uint32_t result_id = getUnique();
  auto inst = makeInstruction(op, type_id, result_id, words);
  // Kill type-manager, we will lazily rebuild it.
  // TODO: Add functions to SPRV-Tool to make sure it stays up to date.
  type_mgr.reset();
  module->AddType(std::move(inst));
  return result_id;
}

uint32_t SpvManager::TypeToId(const Type* type) {
  if (type_mgr == nullptr) {
    auto print_msg_to_stderr = [](spv_message_level_t, const char*, const spv_position_t&, const char* m) {
      std::cerr << "error: " << m << std::endl;
    };
    type_mgr.reset(new TypeManager(print_msg_to_stderr, *module));
  }
  return type_mgr->GetId(type);
}

Type* SpvManager::IdToType(uint32_t id) {
  if (type_mgr == nullptr) {
    auto print_msg_to_stderr = [](spv_message_level_t, const char*, const spv_position_t&, const char* m) {
      std::cerr << "error: " << m << std::endl;
    };
    type_mgr.reset(new TypeManager(print_msg_to_stderr, *module));
  }
  return type_mgr->GetType(id);
}

void SpvManager::addVariable(uint32_t type_id, uint32_t ref_id, spv::StorageClass storage_class) {
  auto inst = makeInstruction(SpvOpVariable, type_id, ref_id, {{static_cast<uint32_t>(storage_class)}});
  if (storage_class == spv::StorageClassFunction)
    curr_block_insts.emplace_back(std::move(inst));
  else
    module->AddGlobalValue(std::move(inst));
}

void SpvManager::addGlobalVariable(spv::StorageClass storage_class, Variable* const var) {
  if (!var->name.empty() && var->type_id) {
    var->ref_id = addName(var->name.c_str());
    uint32_t ptr_id = addTypeInst(SpvOpTypePointer, {{static_cast<uint32_t>(storage_class)}, {var->type_id}});
    addVariable(ptr_id, var->ref_id, storage_class);
  }
}

/**
 * Adds function (with exactly one parameter) to the module
 **/
uint32_t SpvManager::addFunction(const char* name, uint32_t result_type_id, uint32_t param_type) {
  uint32_t name_id = addName(name);
  uint32_t param_type_id = addTypeInst(SpvOpTypePointer, {{spv::StorageClassFunction}, {param_type}});
  uint32_t fun_type_id = addTypeInst(SpvOpTypeFunction, {{globals.void_id}, {param_type_id}});

  auto fun_inst = makeInstruction(SpvOpFunction, result_type_id, name_id,
                                  {{spv::FunctionControlMaskNone}, {fun_type_id}});
  std::unique_ptr<Function> fun = spvtools::MakeUnique<Function>(std::move(fun_inst));

  uint32_t param_name_id = addName(PRINT_PARAM_NAME);
  auto param_inst = makeInstruction(SpvOpFunctionParameter, param_type_id, param_name_id, {{}});
  fun->AddParameter(std::move(param_inst));

  name_type param(param_name_id, param_type);

  curr_block_insts.clear();
  if (param_type == globals.result.type_id) {
    uint32_t true_label_id = getUnique();
    uint32_t false_label_id = getUnique();

    // condition block
    uint32_t load_id =
        collectInstWithResult(SpvOpLoad, {{globals.curr_step.ref_id}}, globals.curr_step.type_id);
    uint32_t sub_id =
        collectInstWithResult(SpvOpISub, {{load_id}, {getConstId(1)}}, globals.curr_step.type_id);
    collectInstWithoutResult(SpvOpStore, {{globals.curr_step.ref_id}, {sub_id}});
    collectCondition(true_label_id, false_label_id);
    fun->AddBasicBlock(std::move(makeBasicBlock(getUnique(), fun.get(), std::move(curr_block_insts))));
    curr_block_insts.clear();

    // true block
    load_id = collectInstWithResult(SpvOpLoad, {{param.first}}, globals.result.type_id);
    collectInstWithoutResult(SpvOpStore, {{globals.result.ref_id}, {load_id}});
    collectInstWithoutResult(SpvOpBranch, {{false_label_id}});
    fun->AddBasicBlock(std::move(makeBasicBlock(true_label_id, fun.get(), std::move(curr_block_insts))));
    curr_block_insts.clear();

    // after-if-statement block
    collectInstWithoutResult(SpvOpReturn);
    fun->AddBasicBlock(std::move(makeBasicBlock(false_label_id, fun.get(), std::move(curr_block_insts))));
    curr_block_insts.clear();

  } else {
    // call-another-print block
    const Type* arg_type = IdToType(param.second);
    uint32_t type_to_convert = getTypeToConvert(arg_type);
    collectPrintCall(param, type_to_convert);
    collectInstWithoutResult(SpvOpReturn);
    fun->AddBasicBlock(std::move(makeBasicBlock(getUnique(), fun.get(), std::move(curr_block_insts))));
    curr_block_insts.clear();
  }

  auto inst_end = makeInstruction(SpvOpFunctionEnd, 0, 0, {{}});
  fun->SetFunctionEnd(std::move(inst_end));
  module->AddFunction(std::move(fun));
  return name_id;
}

/**
 * Functions that create Instructions and add them to currently constructed block instructions.
**/
uint32_t SpvManager::collectInstWithResult(SpvOp_ op,
                                           std::initializer_list<std::initializer_list<uint32_t>> data,
                                           uint32_t type_id) {
  uint32_t result_id = getUnique();
  auto inst = makeInstruction(op, type_id, result_id, data);
  curr_block_insts.emplace_back(std::move(inst));
  return result_id;
}

void SpvManager::collectInstWithoutResult(SpvOp_ op,
                                          std::initializer_list<std::initializer_list<uint32_t>> data,
                                          uint32_t type_id) {
  auto inst = makeInstruction(op, 0, type_id, data);
  curr_block_insts.emplace_back(std::move(inst));
}

uint32_t SpvManager::collectCompositeConstruct(
    std::initializer_list<std::initializer_list<uint32_t>> data, uint32_t type_id) {
  const spvtools::opt::analysis::Vector* vec = IdToType(type_id)->AsVector();
  assert((vec != nullptr && vec->element_count() == data.begin()->size()) &&
         "collectCompositeConstruct: wrong data size to construct vector");

  return collectInstWithResult(SpvOpCompositeConstruct, data, type_id);
}

void SpvManager::collectCondition(uint32_t true_label_id, uint32_t false_label_id) {
  uint32_t load_id =
      collectInstWithResult(SpvOpLoad, {{globals.curr_step.ref_id}}, globals.curr_step.type_id);
  uint32_t bool_id = addTypeInst(SpvOpTypeBool, {{}});
  uint32_t cond_id = collectInstWithResult(SpvOpIEqual, {{load_id}, {getConstId(0)}}, bool_id);

  collectInstWithoutResult(SpvOpSelectionMerge, {{false_label_id}, {SpvSelectionControlMaskNone}});
  collectInstWithoutResult(SpvOpBranchConditional, {{cond_id}, {true_label_id}, {false_label_id}});
}

/**
 * Converts given type 'from' to 'to_type' type and returns 'ref_id' to new converted element.
 * Type conversion only for:
 * bool, float, sint --> uint
 * uint, vec --> uvec4
 **/
uint32_t SpvManager::collectTypeConversion(name_type from, uint32_t to_type) {
  const Type* from_type = IdToType(from.second);

  uint32_t ref_id = 0;
  if (from_type->AsBool()) {
    ref_id =
        collectInstWithResult(SpvOpSelect, {{from.first}, {getConstId(1)}, {getConstId(0)}}, to_type);
  } else if (from_type->AsFloat()) {
    ref_id = collectInstWithResult(SpvOpBitcast, {{from.first}}, to_type);
  } else if (from_type->AsInteger()) {
    if (from_type->AsInteger()->IsSigned()) {
      ref_id = collectInstWithResult(SpvOpBitcast, {{from.first}}, to_type);
    } else {
      ref_id = collectCompositeConstruct({{from.first, getConstId(0), getConstId(0), getConstId(0)}},
                                         to_type);
    }
  } else if (from_type->AsVector()) {
    uint32_t elem_type_id = TypeToId(from_type->AsVector()->element_type());
    uint32_t elem_count = from_type->AsVector()->element_count();
    std::vector<uint32_t> components(RESULT_VEC_SIZE);
    for (uint32_t i = 0; i < RESULT_VEC_SIZE; i++) {
      if (i < elem_count) {
        uint32_t elem_id =
            collectInstWithResult(SpvOpAccessChain, {{from.first}, {getConstId(i)}}, elem_type_id);
        components[i] =
            collectTypeConversion(std::make_pair(elem_id, elem_type_id), globals.uint_type_id);
      } else {
        components[i] = getConstId(0);
      }
    }
    assert(RESULT_VEC_SIZE == 4 && "collectTypeConversion: assumption that resulting vec size is 4.");
    ref_id = collectCompositeConstruct({{components[0], components[1], components[2], components[3]}},
                                       to_type);
  }

  assert(ref_id != 0 &&
         "collectTypeConversion: conversion only from types: bool, float, sint, uint, vec.");
  return ref_id;
}

/**
 * Collects FunctionCall instruction with appropriate fun_id and arg.
 * If fun_param_type_id is given then argument type should be different then fun_param_type_id.
 * By default fun_param_type_id = 0 and that means that conversion shouldn't be needed.
 **/
void SpvManager::collectPrintCall(name_type arg, uint32_t fun_param_type_id) {
  uint32_t fun_id;
  uint32_t arg_name_id;

  if (fun_param_type_id && fun_param_type_id != arg.second) {
    fun_id = getPrintFunction(fun_param_type_id);
    uint32_t source = collectInstWithResult(SpvOpLoad, {{arg.first}}, arg.second);
    arg_name_id = collectTypeConversion(std::make_pair(source, arg.second), fun_param_type_id);

  } else {
    fun_id = getPrintFunction(arg.second);
    arg_name_id = arg.first;
  }

  collectInstWithResult(SpvOpFunctionCall, {{fun_id}, {arg_name_id}}, globals.void_id);
}

void SpvManager::collectPrintChain(name_type arg) {
  const Type* arg_type = IdToType(arg.second);

  if (isConvertedType(arg_type)) {
    collectPrintCall(arg);
    return;
  }

  uint32_t print_elem_type;
  uint32_t elem_id;
  uint32_t elem_count;

  if (arg_type->AsMatrix()) {
    const spvtools::opt::analysis::Matrix* matrix = arg_type->AsMatrix();
    print_elem_type = TypeToId(matrix->element_type());
    elem_count = matrix->element_count();
    for (uint32_t i = 0; i < elem_count; i++) {
      elem_id = collectInstWithResult(SpvOpAccessChain, {{arg.first}, {getConstId(i)}}, print_elem_type);
      collectPrintChain(std::make_pair(elem_id, print_elem_type));
    }
  }
  if (arg_type->AsStruct()) {
    const std::vector<Type*>& element_types = arg_type->AsStruct()->element_types();
    elem_count = element_types.size();
    for (uint32_t i = 0; i < elem_count; i++) {
      print_elem_type = TypeToId(element_types[i]);
      elem_id = collectInstWithResult(SpvOpAccessChain, {{arg.first}, {getConstId(i)}}, print_elem_type);
      collectPrintChain(std::make_pair(elem_id, print_elem_type));
    }
  }
  if (arg_type->AsArray()) {
    print_elem_type = TypeToId(arg_type->AsArray()->element_type());
    elem_count = getArrayLength(arg_type);
    for (uint32_t i = 0; i < elem_count; i++) {
      elem_id = collectInstWithResult(SpvOpAccessChain, {{arg.first}, {getConstId(i)}}, print_elem_type);
      collectPrintChain(std::make_pair(elem_id, print_elem_type));
    }
  }
}

/**
 * Declares global debug variables: result, sampler, coordinate, curr_step.
 **/
void SpvManager::declareDebugVariables() {
  globals.uint_type_id = addTypeInst(SpvOpTypeInt, {{WIDTH}, {0}});

  globals.result.type_id = addTypeInst(SpvOpTypeVector, {{globals.uint_type_id}, {RESULT_VEC_SIZE}});
  addGlobalVariable(spv::StorageClassOutput, &globals.result);

  uint32_t float_type_id = addTypeInst(SpvOpTypeFloat, {{WIDTH}});
  globals.coordinate.type_id = addTypeInst(SpvOpTypeVector, {{float_type_id}, {COORDINATE_SIZE}});
  addGlobalVariable(spv::StorageClassInput, &globals.coordinate);

  uint32_t image_type = addTypeInst(
      SpvOpTypeImage, {{globals.uint_type_id}, {SpvDim2D}, {0}, {0}, {0}, {1}, {SpvImageFormatUnknown}});
  uint32_t sampler_type = addTypeInst(SpvOpTypeSampledImage, {{image_type}});
  globals.sampler.type_id = sampler_type;
  addGlobalVariable(spv::StorageClassUniformConstant, &globals.sampler);

  globals.curr_step.type_id = globals.uint_type_id;
  addGlobalVariable(spv::StorageClassPrivate, &globals.curr_step);
}

/**
 * Adds print functions (declarations) to module.
 * For every different type that type_mgr holds prepares print function with this parameter type.
 **/
void SpvManager::declarePrints() {
  globals.void_id = addTypeInst(SpvOpTypeVoid, {{}});
  // declares those two functions first, because other print functions call them
  insertPrintDeclaration(globals.result.type_id);
  insertPrintDeclaration(globals.uint_type_id);
  // special 'print' for labels printing
  globals.label_print_id = addFunction(LABEL_PRINT_NAME, globals.void_id, globals.uint_type_id);

  // makes a copy, because iterating through all types insertPrintDeclaration adds new types
  std::set<uint32_t> type_ids;
  for (auto const& it : *type_mgr) {
    type_ids.emplace(it.first);
  }

  for (uint32_t type_id : type_ids) {
    insertPrintDeclaration(type_id);
  }
}

/**
 * Variable 'curr_step' is stored in Sampler2D.
 * Retrieves that data and assign to 'curr_step'.
 **/
void SpvManager::setStepVariable() {
  uint32_t sampler_id =
      collectInstWithResult(SpvOpLoad, {{globals.sampler.ref_id}}, globals.sampler.type_id);
  uint32_t coor_id =
      collectInstWithResult(SpvOpLoad, {{globals.coordinate.ref_id}}, globals.coordinate.type_id);
  uint32_t float_type_id = addTypeInst(SpvOpTypeFloat, {{WIDTH}});
  uint32_t vec = addTypeInst(SpvOpTypeVector, {{float_type_id}, {4}});
  uint32_t texture_res =
      collectInstWithResult(SpvOpImageSampleImplicitLod, {{sampler_id}, {coor_id}}, vec);
  uint32_t float_val =
      collectInstWithResult(SpvOpAccessChain, {{texture_res}, {getConstId(0)}}, float_type_id);
  collectInstWithoutResult(SpvOpStore, {{globals.curr_step.ref_id}, {float_val}});
}

/**
 * Traverses function blocks to insert print function calls.
 **/
void SpvManager::insertPrintCallsIntoFunctions() {
  // main function is first, add insts to the first block
  setStepVariable();

  for (Module::iterator it = module->begin(); it != module->end(); it++) {
    if (isDebugFunction(*it)) continue;

    for (Function::iterator fun_it = it->begin(); fun_it != it->end(); fun_it++)
      insertPrintCallsIntoBlock(*fun_it);
  }
}

void SpvManager::moveCollectedBlockInsts(BasicBlock::iterator& it) {
  for (auto& i : curr_block_insts) {
    it = it.InsertBefore(std::move(i));
    it++;
  }
  curr_block_insts.clear();
}

/**
 * Inserts FunctionCall instruction after Store instructions.
 **/
void SpvManager::insertPrintCallsIntoBlock(BasicBlock& bb) {
  BasicBlock::iterator it = bb.begin();
  uint32_t label_id = bb.Label().result_id();
  // print label id to indicate current BasicBlock
  assert(globals.label_print_id != 0 &&
         "insertPrintCallsIntoBlock: label_print_id has to be bigger then zero.");
  collectInstWithResult(SpvOpFunctionCall, {{globals.label_print_id}, {getConstId(label_id)}},
                        globals.void_id);
  moveCollectedBlockInsts(it);

  while (it != bb.end()) {
    if (it->opcode() == SpvOpStore && !isArgStoreInst(it, bb.end())) {
      uint32_t pointer = it->GetSingleWordOperand(0);
      Instruction* pointer_def = def_use_mgr->GetDef(pointer);

      const Type* pointee_type = getPointeeIfPointer(pointer_def->type_id());
      assert(pointee_type != nullptr && "insertPrintCallsIntoBlock: not recognized pointee type.");
      if (pointee_type) {
        uint32_t opcode = pointer_def->opcode();
        if (opcode == SpvOpAccessChain || opcode == SpvOpInBoundsAccessChain) {
          uint32_t offset_id;
          for (uint32_t i = FIRST_CHAIN_INDEX_OPERAND; i < pointer_def->NumOperands(); i++) {
            offset_id = pointer_def->GetSingleWordOperand(i);
            collectPrintCall(std::make_pair(offset_id, getVariableTypeId(offset_id)));
          }
        }

        uint32_t pointee_type_id = TypeToId(pointee_type);
        collectPrintChain(std::make_pair(pointer, pointee_type_id));
      }
    }

    it++;
    moveCollectedBlockInsts(it);
  }
}

/**
 * Extends typeid_to_printid by inserting new pair <type_id, fun_id>,
 * where fun_id is function made for type_id type.
 * If function for this type already exists, return fun_id.
 * Attention! addFunction uses curr_block_insts vector to collect function body.
 **/
uint32_t SpvManager::insertPrintDeclaration(uint32_t type_id) {
  const Type* type = IdToType(type_id);

  if (isConvertedType(type)) {
    for (map_uint::iterator it = typeid_to_printid.begin(); it != typeid_to_printid.end(); ++it) {
      if (type->IsSame(IdToType(it->first))) {
        return it->second;
      }
    }

    if (type->AsVector() && type_id != globals.result.type_id) {
      uint32_t elem_type_id = TypeToId(type->AsVector()->element_type());
      insertPrintDeclaration(elem_type_id);
    }

    typeid_to_printid[type_id] = addFunction(PRINT_NAME, globals.void_id, type_id);
    return typeid_to_printid[type_id];
  }
  return 0;
}

uint32_t SpvManager::getVariableTypeId(uint32_t var_id) {
  Instruction* var_inst = def_use_mgr->GetDef(var_id);
  uint32_t type_id = var_inst->type_id();

  assert(type_id != 0 && "getVariableTypeId: variable type not found.");
  return type_id;
}

/**
 * Returns type_id of element which we want to print for given type.
 **/
uint32_t SpvManager::getTypeToConvert(const Type* type) {
  uint32_t res_id = 0;
  if ((type->AsInteger() && !type->AsInteger()->IsSigned()) || type->AsVector()) {
    res_id = globals.result.type_id;
  } else if (type->AsBool() || type->AsInteger() || type->AsFloat()) {
    res_id = globals.uint_type_id;
  }

  assert(res_id != 0 && "getTypeToConvert: type id to convert not found.");
  return res_id;
}

/**
 * Determines length of array if given type represents array.
 **/
uint32_t SpvManager::getArrayLength(const Type* type) {
  uint32_t length = 0;

  if (type->AsArray()) {
    const spvtools::opt::analysis::Array* array = type->AsArray();
    Instruction* const_inst = def_use_mgr->GetDef(array->LengthId());
    assert((const_inst->opcode() == SpvOpConstant && const_inst->NumOperands() == 3) &&
           "getArrayLength: array length must come from constant instruction.");
    length = const_inst->GetSingleWordOperand(2);
  }

  assert(length != 0 && "getArrayLength: array length must be at least 1.");
  return length;
}

/**
 * Returns pointer to pointee_type if given id defines Pointer type.
 **/
const Type* SpvManager::getPointeeIfPointer(uint32_t id) {
  const Type* type = IdToType(id);
  if (type) {
    const spvtools::opt::analysis::Pointer* pointer = type->AsPointer();
    if (pointer) {
      return pointer->pointee_type();
    }
  }
  return nullptr;
}

/**
 * Returns id to function declaration with the same parameter type
 * as given type_id.
 **/
uint32_t SpvManager::getPrintFunction(uint32_t type_id) {
  uint32_t fun_id = 0;

  map_uint::iterator it = typeid_to_printid.find(type_id);

  if (it != typeid_to_printid.end()) {
    fun_id = it->second;
  } else {
    const Type* type = IdToType(type_id);
    it = typeid_to_printid.begin();
    while (it != typeid_to_printid.end()) {
      if (type->IsSame(IdToType(it->first))) {
        fun_id = it->second;
        break;
      }
      it++;
    }
  }

  assert(fun_id != 0 && "getPrintFunction: function id must be bigger then 0.");
  return fun_id;
}

bool SpvManager::isConvertedType(const Type* type) {
  return (type->AsBool() || type->AsInteger() || type->AsFloat() || type->AsVector());
}

/**
 * Checks if given function is print function added for debugging.
 **/
bool SpvManager::isDebugFunction(Function& f) {
  std::string name = name_mgr->getStrName(f.DefInst().result_id());
  return name == PRINT_NAME || name == LABEL_PRINT_NAME;
}

/**
 * Checks if given inst is Store instruction and
 * stores variable on function argument.
 **/
bool SpvManager::isArgStoreInst(BasicBlock::iterator bb_curr, BasicBlock::iterator bb_end) {
  if (bb_curr->NumOperands() >= 2) {
    uint32_t dest_id = bb_curr->GetSingleWordOperand(0);

    while (bb_curr->opcode() == SpvOpStore && bb_curr != bb_end) {
      bb_curr++;
    }

    if (bb_curr != bb_end)
      if (bb_curr->opcode() == SpvOpFunctionCall && bb_curr->NumOperands() >= 4)
        for (auto i = 3; i < bb_curr->NumOperands(); i++)
          if (dest_id == bb_curr->GetSingleWordOperand(i)) return true;
  }

  return false;
}

/**
 * Checks if given instruction is decoration of built in variable.
 **/
bool SpvManager::isBuiltInDecoration(const Instruction& inst) const {
  spv_opcode_desc op_decoration_desc;

  if (grammar->lookupOpcode(inst.opcode(), &op_decoration_desc) == SPV_SUCCESS) {
    switch (inst.opcode()) {
      case SpvOpDecorate:
        if (op_decoration_desc->numTypes >= 2 && inst.GetSingleWordOperand(1) == SpvDecorationBuiltIn)
          return true;
      case SpvOpMemberDecorate:
        if (op_decoration_desc->numTypes >= 3 && inst.GetSingleWordOperand(2) == SpvDecorationBuiltIn)
          return true;
      default:
        break;
    }
  }

  return false;
}

/**
 * Checks if instuction represents some input variable.
 **/
bool SpvManager::isInputVariable(const Instruction& inst) const {
  return (inst.opcode() == SpvOpVariable && inst.NumOperands() >= 3 &&
          inst.GetSingleWordOperand(2) == spv::StorageClassInput);
}

/**
 * Returns id to const instruction with a given value.
 * Adds new constant instruction if not added before.
 **/
uint32_t SpvManager::getConstId(uint32_t val) {
  if (consts.find(val) == consts.end()) {
    consts[val] = addConstant(globals.uint_type_id, {val});
  }
  return consts[val];
}

/**
 * Returns first free unique id.
 **/
uint32_t SpvManager::getUnique() {
  uint32_t res;
  res = module->IdBound();
  module->SetIdBound(res + 1);
  return res;
}

/**
 * Checks if instruction gives useful information in debugging process.
 **/
bool SpvManager::isDebugInstruction(Instruction* inst) {
  if (spvtools::ir::IsTypeInst(inst->opcode())) {
    return true;
  }

  switch (inst->opcode()) {
    case SpvOpName:
    case SpvOpMemberName:
    case SpvOpLine:
    case SpvOpVariable:
    case SpvOpLabel:
    case SpvOpAccessChain:
    case SpvOpInBoundsAccessChain:
      return true;
    case SpvOpFunctionCall: {
      uint32_t ref_id = inst->GetSingleWordOperand(2);
      Instruction* def_inst = def_use_mgr->GetDef(ref_id);
      if (def_inst->opcode() == SpvOpFunction) {
        std::string name = name_mgr->getStrName(def_inst->result_id());
        return name == PRINT_NAME || name == LABEL_PRINT_NAME;
      }
      return false;
    }
    default:
      return false;
  }
}

/**
 * Adds instruction to 'debugs' vector if it is usuful in debugging process.
 * If instruction is a Name or MemberName instruction remember name as a char array.
 **/
void SpvManager::appendDebugInstruction(std::vector<instruction_t>* debugs, Instruction* inst) {
  SpvOp_ opcode = inst->opcode();

  if (isDebugInstruction(inst)) {
    instruction_t i{};
    i.id = inst->result_id();
    i.opcode = opcode;

    if (opcode == SpvOpName || opcode == SpvOpMemberName) {
      std::string str_name;
      if (opcode == SpvOpName)
        str_name = name_mgr->getStrName(inst->GetSingleWordOperand(0));
      else
        str_name = name_mgr->getStrName(
            namemanager::id_offset(inst->GetSingleWordOperand(0), inst->GetSingleWordOperand(1)));

      i.name = new char[str_name.length() + 1];
      std::strcpy(i.name, str_name.c_str());
    }

    // copy operands words
    i.words = new uint32_t[inst->NumOperands()];
    for (int op = 0; op < inst->NumOperands(); op++) {
      switch (inst->GetOperand(op).type) {
        case SPV_OPERAND_TYPE_RESULT_ID:
        case SPV_OPERAND_TYPE_LITERAL_STRING:
          break; // Skip
        default:
          i.words[i.words_num++] = inst->GetSingleWordOperand(op);
      }
    }

    debugs->emplace_back(std::move(i));
  }
}

}  // namespace spvmanager
