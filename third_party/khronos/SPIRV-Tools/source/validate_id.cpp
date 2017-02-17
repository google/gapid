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

#include "validate.h"

#include <cassert>

#include <algorithm>
#include <iostream>
#include <unordered_set>
#include <utility>
#include <vector>

#include "diagnostic.h"
#include "instruction.h"
#include "opcode.h"
#include "spirv-tools/libspirv.h"
#include "val/Function.h"
#include "val/ValidationState.h"

#define spvCheck(condition, action) \
  if (condition) {                  \
    action;                         \
  }

using libspirv::ValidationState_t;
using std::function;
using std::ignore;
using std::make_pair;
using std::pair;
using std::unordered_set;
using std::vector;

namespace {

class idUsage {
 public:
  idUsage(const spv_opcode_table opcodeTableArg,
          const spv_operand_table operandTableArg,
          const spv_ext_inst_table extInstTableArg,
          const spv_instruction_t* pInsts, const uint64_t instCountArg,
          const SpvMemoryModel memoryModelArg,
          const SpvAddressingModel addressingModelArg,
          const ValidationState_t& module, const vector<uint32_t>& entry_points,
          spv_position positionArg, spv_diagnostic* pDiagnosticArg)
      : opcodeTable(opcodeTableArg),
        operandTable(operandTableArg),
        extInstTable(extInstTableArg),
        firstInst(pInsts),
        instCount(instCountArg),
        memoryModel(memoryModelArg),
        addressingModel(addressingModelArg),
        position(positionArg),
        pDiagnostic(pDiagnosticArg),
        module_(module),
        entry_points_(entry_points) {}

  bool isValid(const spv_instruction_t* inst);

  template <SpvOp>
  bool isValid(const spv_instruction_t* inst, const spv_opcode_desc);

 private:
  const spv_opcode_table opcodeTable;
  const spv_operand_table operandTable;
  const spv_ext_inst_table extInstTable;
  const spv_instruction_t* const firstInst;
  const uint64_t instCount;
  const SpvMemoryModel memoryModel;
  const SpvAddressingModel addressingModel;
  spv_position position;
  spv_diagnostic* pDiagnostic;
  const ValidationState_t& module_;
  vector<uint32_t> entry_points_;
};

#define DIAG(INDEX)         \
  position->index += INDEX; \
  DIAGNOSTIC

#if 0
template <>
bool idUsage::isValid<SpvOpUndef>(const spv_instruction_t *inst,
                                  const spv_opcode_desc) {
  assert(0 && "Unimplemented!");
  return false;
}
#endif  // 0

template <>
bool idUsage::isValid<SpvOpMemberName>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  auto typeIndex = 1;
  auto type = module_.FindDef(inst->words[typeIndex]);
  if (!type || SpvOpTypeStruct != type->opcode()) {
    DIAG(typeIndex) << "OpMemberName Type <id> '" << inst->words[typeIndex]
                    << "' is not a struct type.";
    return false;
  }
  auto memberIndex = 2;
  auto member = inst->words[memberIndex];
  auto memberCount = (uint32_t)(type->words().size() - 2);
  spvCheck(memberCount <= member, DIAG(memberIndex)
                                      << "OpMemberName Member <id> '"
                                      << inst->words[memberIndex]
                                      << "' index is larger than Type <id> '"
                                      << type->id() << "'s member count.";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpLine>(const spv_instruction_t* inst,
                                 const spv_opcode_desc) {
  auto fileIndex = 1;
  auto file = module_.FindDef(inst->words[fileIndex]);
  if (!file || SpvOpString != file->opcode()) {
    DIAG(fileIndex) << "OpLine Target <id> '" << inst->words[fileIndex]
                    << "' is not an OpString.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpMemberDecorate>(const spv_instruction_t* inst,
                                           const spv_opcode_desc) {
  auto structTypeIndex = 1;
  auto structType = module_.FindDef(inst->words[structTypeIndex]);
  if (!structType || SpvOpTypeStruct != structType->opcode()) {
    DIAG(structTypeIndex) << "OpMemberDecorate Structure type <id> '"
                          << inst->words[structTypeIndex]
                          << "' is not a struct type.";
    return false;
  }
  auto memberIndex = 2;
  auto member = inst->words[memberIndex];
  auto memberCount = static_cast<uint32_t>(structType->words().size() - 2);
  spvCheck(memberCount < member, DIAG(memberIndex)
                                     << "OpMemberDecorate Structure type <id> '"
                                     << inst->words[memberIndex]
                                     << "' member count is less than Member";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpGroupDecorate>(const spv_instruction_t* inst,
                                          const spv_opcode_desc) {
  auto decorationGroupIndex = 1;
  auto decorationGroup = module_.FindDef(inst->words[decorationGroupIndex]);
  if (!decorationGroup || SpvOpDecorationGroup != decorationGroup->opcode()) {
    DIAG(decorationGroupIndex) << "OpGroupDecorate Decoration group <id> '"
                               << inst->words[decorationGroupIndex]
                               << "' is not a decoration group.";
    return false;
  }
  return true;
}

#if 0
template <>
bool idUsage::isValid<SpvOpGroupMemberDecorate>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif  // 0

#if 0
template <>
bool idUsage::isValid<SpvOpExtInst>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif  // 0

template <>
bool idUsage::isValid<SpvOpEntryPoint>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  auto entryPointIndex = 2;
  auto entryPoint = module_.FindDef(inst->words[entryPointIndex]);
  if (!entryPoint || SpvOpFunction != entryPoint->opcode()) {
    DIAG(entryPointIndex) << "OpEntryPoint Entry Point <id> '"
                          << inst->words[entryPointIndex]
                          << "' is not a function.";
    return false;
  }
  // don't check kernel function signatures
  auto executionModel = inst->words[1];
  if (executionModel != SpvExecutionModelKernel) {
    // TODO: Check the entry point signature is void main(void), may be subject
    // to change
    auto entryPointType = module_.FindDef(entryPoint->words()[4]);
    if (!entryPointType || 3 != entryPointType->words().size()) {
      DIAG(entryPointIndex) << "OpEntryPoint Entry Point <id> '"
                            << inst->words[entryPointIndex]
                            << "'s function parameter count is not zero.";
      return false;
    }
  }
  auto returnType = module_.FindDef(entryPoint->type_id());
  if (!returnType || SpvOpTypeVoid != returnType->opcode()) {
    DIAG(entryPointIndex) << "OpEntryPoint Entry Point <id> '"
                          << inst->words[entryPointIndex]
                          << "'s function return type is not void.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpExecutionMode>(const spv_instruction_t* inst,
                                          const spv_opcode_desc) {
  auto entryPointIndex = 1;
  auto entryPointID = inst->words[entryPointIndex];
  auto found =
      std::find(entry_points_.cbegin(), entry_points_.cend(), entryPointID);
  if (found == entry_points_.cend()) {
    DIAG(entryPointIndex) << "OpExecutionMode Entry Point <id> '"
                          << inst->words[entryPointIndex]
                          << "' is not the Entry Point "
                             "operand of an OpEntryPoint.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeVector>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  auto componentIndex = 2;
  auto componentType = module_.FindDef(inst->words[componentIndex]);
  if (!componentType || !spvOpcodeIsScalarType(componentType->opcode())) {
    DIAG(componentIndex) << "OpTypeVector Component Type <id> '"
                         << inst->words[componentIndex]
                         << "' is not a scalar type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeMatrix>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  auto columnTypeIndex = 2;
  auto columnType = module_.FindDef(inst->words[columnTypeIndex]);
  if (!columnType || SpvOpTypeVector != columnType->opcode()) {
    DIAG(columnTypeIndex) << "OpTypeMatrix Column Type <id> '"
                          << inst->words[columnTypeIndex]
                          << "' is not a vector.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeSampler>(const spv_instruction_t*,
                                        const spv_opcode_desc) {
  // OpTypeSampler takes no arguments in Rev31 and beyond.
  return true;
}

// True if the integer constant is > 0. constWords are words of the
// constant-defining instruction (either OpConstant or
// OpSpecConstant). typeWords are the words of the constant's-type-defining
// OpTypeInt.
bool aboveZero(const vector<uint32_t>& constWords,
               const vector<uint32_t>& typeWords) {
  const uint32_t width = typeWords[2];
  const bool is_signed = typeWords[3] > 0;
  const uint32_t loWord = constWords[3];
  if (width > 32) {
    // The spec currently doesn't allow integers wider than 64 bits.
    const uint32_t hiWord = constWords[4];  // Must exist, per spec.
    if (is_signed && (hiWord >> 31)) return false;
    return (loWord | hiWord) > 0;
  } else {
    if (is_signed && (loWord >> 31)) return false;
    return loWord > 0;
  }
}

template <>
bool idUsage::isValid<SpvOpTypeArray>(const spv_instruction_t* inst,
                                      const spv_opcode_desc) {
  auto elementTypeIndex = 2;
  auto elementType = module_.FindDef(inst->words[elementTypeIndex]);
  if (!elementType || !spvOpcodeGeneratesType(elementType->opcode())) {
    DIAG(elementTypeIndex) << "OpTypeArray Element Type <id> '"
                           << inst->words[elementTypeIndex]
                           << "' is not a type.";
    return false;
  }
  auto lengthIndex = 3;
  auto length = module_.FindDef(inst->words[lengthIndex]);
  if (!length || !spvOpcodeIsConstant(length->opcode())) {
    DIAG(lengthIndex) << "OpTypeArray Length <id> '" << inst->words[lengthIndex]
                      << "' is not a scalar constant type.";
    return false;
  }

  // NOTE: Check the initialiser value of the constant
  auto constInst = length->words();
  auto constResultTypeIndex = 1;
  auto constResultType = module_.FindDef(constInst[constResultTypeIndex]);
  if (!constResultType || SpvOpTypeInt != constResultType->opcode()) {
    DIAG(lengthIndex) << "OpTypeArray Length <id> '" << inst->words[lengthIndex]
                      << "' is not a constant integer type.";
    return false;
  }

  switch (length->opcode()) {
    case SpvOpSpecConstant:
    case SpvOpConstant:
      if (aboveZero(length->words(), constResultType->words())) break;
    // Else fall through!
    case SpvOpConstantNull: {
      DIAG(lengthIndex) << "OpTypeArray Length <id> '"
                        << inst->words[lengthIndex]
                        << "' default value must be at least 1.";
      return false;
    }
    case SpvOpSpecConstantOp:
      // Assume it's OK, rather than try to evaluate the operation.
      break;
    default:
      assert(0 && "bug in spvOpcodeIsConstant() or result type isn't int");
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeRuntimeArray>(const spv_instruction_t* inst,
                                             const spv_opcode_desc) {
  auto elementTypeIndex = 2;
  auto elementType = module_.FindDef(inst->words[elementTypeIndex]);
  if (!elementType || !spvOpcodeGeneratesType(elementType->opcode())) {
    DIAG(elementTypeIndex) << "OpTypeRuntimeArray Element Type <id> '"
                           << inst->words[elementTypeIndex]
                           << "' is not a type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeStruct>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  for (size_t memberTypeIndex = 2; memberTypeIndex < inst->words.size();
       ++memberTypeIndex) {
    auto memberType = module_.FindDef(inst->words[memberTypeIndex]);
    if (!memberType || !spvOpcodeGeneratesType(memberType->opcode())) {
      DIAG(memberTypeIndex) << "OpTypeStruct Member Type <id> '"
                            << inst->words[memberTypeIndex]
                            << "' is not a type.";
      return false;
    }
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypePointer>(const spv_instruction_t* inst,
                                        const spv_opcode_desc) {
  auto typeIndex = 3;
  auto type = module_.FindDef(inst->words[typeIndex]);
  if (!type || !spvOpcodeGeneratesType(type->opcode())) {
    DIAG(typeIndex) << "OpTypePointer Type <id> '" << inst->words[typeIndex]
                    << "' is not a type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypeFunction>(const spv_instruction_t* inst,
                                         const spv_opcode_desc) {
  auto returnTypeIndex = 2;
  auto returnType = module_.FindDef(inst->words[returnTypeIndex]);
  if (!returnType || !spvOpcodeGeneratesType(returnType->opcode())) {
    DIAG(returnTypeIndex) << "OpTypeFunction Return Type <id> '"
                          << inst->words[returnTypeIndex] << "' is not a type.";
    return false;
  }
  for (size_t paramTypeIndex = 3; paramTypeIndex < inst->words.size();
       ++paramTypeIndex) {
    auto paramType = module_.FindDef(inst->words[paramTypeIndex]);
    if (!paramType || !spvOpcodeGeneratesType(paramType->opcode())) {
      DIAG(paramTypeIndex) << "OpTypeFunction Parameter Type <id> '"
                           << inst->words[paramTypeIndex] << "' is not a type.";
      return false;
    }
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpTypePipe>(const spv_instruction_t*,
                                     const spv_opcode_desc) {
  // OpTypePipe has no ID arguments.
  return true;
}

template <>
bool idUsage::isValid<SpvOpConstantTrue>(const spv_instruction_t* inst,
                                         const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypeBool != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpConstantTrue Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a boolean type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpConstantFalse>(const spv_instruction_t* inst,
                                          const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypeBool != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpConstantFalse Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a boolean type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpConstantComposite>(const spv_instruction_t* inst,
                                              const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || !spvOpcodeIsComposite(resultType->opcode())) {
    DIAG(resultTypeIndex) << "OpConstantComposite Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a composite type.";
    return false;
  }

  auto constituentCount = inst->words.size() - 3;
  switch (resultType->opcode()) {
    case SpvOpTypeVector: {
      auto componentCount = resultType->words()[3];
      spvCheck(
          componentCount != constituentCount,
          // TODO: Output ID's on diagnostic
          DIAG(inst->words.size() - 1)
              << "OpConstantComposite Constituent <id> count does not match "
                 "Result Type <id> '"
              << resultType->id() << "'s vector component count.";
          return false);
      auto componentType = module_.FindDef(resultType->words()[2]);
      assert(componentType);
      for (size_t constituentIndex = 3; constituentIndex < inst->words.size();
           constituentIndex++) {
        auto constituent = module_.FindDef(inst->words[constituentIndex]);
        if (!constituent || !spvOpcodeIsConstant(constituent->opcode())) {
          DIAG(constituentIndex) << "OpConstantComposite Constituent <id> '"
                                 << inst->words[constituentIndex]
                                 << "' is not a constant.";
          return false;
        }
        auto constituentResultType = module_.FindDef(constituent->type_id());
        if (!constituentResultType ||
            componentType->opcode() != constituentResultType->opcode()) {
          DIAG(constituentIndex) << "OpConstantComposite Constituent <id> '"
                                 << inst->words[constituentIndex]
                                 << "'s type does not match Result Type <id> '"
                                 << resultType->id()
                                 << "'s vector element type.";
          return false;
        }
      }
    } break;
    case SpvOpTypeMatrix: {
      auto columnCount = resultType->words()[3];
      spvCheck(
          columnCount != constituentCount,
          // TODO: Output ID's on diagnostic
          DIAG(inst->words.size() - 1)
              << "OpConstantComposite Constituent <id> count does not match "
                 "Result Type <id> '"
              << resultType->id() << "'s matrix column count.";
          return false);

      auto columnType = module_.FindDef(resultType->words()[2]);
      assert(columnType);
      auto componentCount = columnType->words()[3];
      auto componentType = module_.FindDef(columnType->words()[2]);
      assert(componentType);

      for (size_t constituentIndex = 3; constituentIndex < inst->words.size();
           constituentIndex++) {
        auto constituent = module_.FindDef(inst->words[constituentIndex]);
        if (!constituent || SpvOpConstantComposite != constituent->opcode()) {
          DIAG(constituentIndex) << "OpConstantComposite Constituent <id> '"
                                 << inst->words[constituentIndex]
                                 << "' is not a constant composite.";
          return false;
        }
        auto vector = module_.FindDef(constituent->type_id());
        assert(vector);
        spvCheck(columnType->opcode() != vector->opcode(),
                 DIAG(constituentIndex)
                     << "OpConstantComposite Constituent <id> '"
                     << inst->words[constituentIndex]
                     << "' type does not match Result Type <id> '"
                     << resultType->id() << "'s matrix column type.";
                 return false);
        auto vectorComponentType = module_.FindDef(vector->words()[2]);
        assert(vectorComponentType);
        spvCheck(componentType->id() != vectorComponentType->id(),
                 DIAG(constituentIndex)
                     << "OpConstantComposite Constituent <id> '"
                     << inst->words[constituentIndex]
                     << "' component type does not match Result Type <id> '"
                     << resultType->id() << "'s matrix column component type.";
                 return false);
        spvCheck(
            componentCount != vector->words()[3],
            DIAG(constituentIndex)
                << "OpConstantComposite Constituent <id> '"
                << inst->words[constituentIndex]
                << "' vector component count does not match Result Type <id> '"
                << resultType->id() << "'s vector component count.";
            return false);
      }
    } break;
    case SpvOpTypeArray: {
      auto elementType = module_.FindDef(resultType->words()[2]);
      assert(elementType);
      auto length = module_.FindDef(resultType->words()[3]);
      assert(length);
      spvCheck(length->words()[3] != constituentCount,
               DIAG(inst->words.size() - 1)
                   << "OpConstantComposite Constituent count does not match "
                      "Result Type <id> '"
                   << resultType->id() << "'s array length.";
               return false);
      for (size_t constituentIndex = 3; constituentIndex < inst->words.size();
           constituentIndex++) {
        auto constituent = module_.FindDef(inst->words[constituentIndex]);
        if (!constituent || !spvOpcodeIsConstant(constituent->opcode())) {
          DIAG(constituentIndex) << "OpConstantComposite Constituent <id> '"
                                 << inst->words[constituentIndex]
                                 << "' is not a constant.";
          return false;
        }
        auto constituentType = module_.FindDef(constituent->type_id());
        assert(constituentType);
        spvCheck(elementType->id() != constituentType->id(),
                 DIAG(constituentIndex)
                     << "OpConstantComposite Constituent <id> '"
                     << inst->words[constituentIndex]
                     << "'s type does not match Result Type <id> '"
                     << resultType->id() << "'s array element type.";
                 return false);
      }
    } break;
    case SpvOpTypeStruct: {
      auto memberCount = resultType->words().size() - 2;
      spvCheck(memberCount != constituentCount,
               DIAG(resultTypeIndex)
                   << "OpConstantComposite Constituent <id> '"
                   << inst->words[resultTypeIndex]
                   << "' count does not match Result Type <id> '"
                   << resultType->id() << "'s struct member count.";
               return false);
      for (uint32_t constituentIndex = 3, memberIndex = 2;
           constituentIndex < inst->words.size();
           constituentIndex++, memberIndex++) {
        auto constituent = module_.FindDef(inst->words[constituentIndex]);
        if (!constituent || !spvOpcodeIsConstant(constituent->opcode())) {
          DIAG(constituentIndex) << "OpConstantComposite Constituent <id> '"
                                 << inst->words[constituentIndex]
                                 << "' is not a constant.";
          return false;
        }
        auto constituentType = module_.FindDef(constituent->type_id());
        assert(constituentType);

        auto memberType = module_.FindDef(resultType->words()[memberIndex]);
        assert(memberType);
        spvCheck(memberType->id() != constituentType->id(),
                 DIAG(constituentIndex)
                     << "OpConstantComposite Constituent <id> '"
                     << inst->words[constituentIndex]
                     << "' type does not match the Result Type <id> '"
                     << resultType->id() << "'s member type.";
                 return false);
      }
    } break;
    default: { assert(0 && "Unreachable!"); } break;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpConstantSampler>(const spv_instruction_t* inst,
                                            const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypeSampler != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpConstantSampler Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a sampler type.";
    return false;
  }
  return true;
}

// True if instruction defines a type that can have a null value, as defined by
// the SPIR-V spec.  Tracks composite-type components through module to check
// nullability transitively.
bool IsTypeNullable(const vector<uint32_t>& instruction,
                    const ValidationState_t& module) {
  uint16_t opcode;
  uint16_t word_count;
  spvOpcodeSplit(instruction[0], &word_count, &opcode);
  switch (static_cast<SpvOp>(opcode)) {
    case SpvOpTypeBool:
    case SpvOpTypeInt:
    case SpvOpTypeFloat:
    case SpvOpTypePointer:
    case SpvOpTypeEvent:
    case SpvOpTypeDeviceEvent:
    case SpvOpTypeReserveId:
    case SpvOpTypeQueue:
      return true;
    case SpvOpTypeArray:
    case SpvOpTypeMatrix:
    case SpvOpTypeVector: {
      auto base_type = module.FindDef(instruction[2]);
      return base_type && IsTypeNullable(base_type->words(), module);
    }
    case SpvOpTypeStruct: {
      for (size_t elementIndex = 2; elementIndex < instruction.size();
           ++elementIndex) {
        auto element = module.FindDef(instruction[elementIndex]);
        if (!element || !IsTypeNullable(element->words(), module)) return false;
      }
      return true;
    }
    default:
      return false;
  }
}

template <>
bool idUsage::isValid<SpvOpConstantNull>(const spv_instruction_t* inst,
                                         const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || !IsTypeNullable(resultType->words(), module_)) {
    DIAG(resultTypeIndex) << "OpConstantNull Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' cannot have a null value.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpSpecConstantTrue>(const spv_instruction_t* inst,
                                             const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypeBool != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpSpecConstantTrue Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a boolean type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpSpecConstantFalse>(const spv_instruction_t* inst,
                                              const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypeBool != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpSpecConstantFalse Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a boolean type.";
    return false;
  }
  return true;
}

#if 0
template <>
bool idUsage::isValid<SpvOpSpecConstantComposite>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<SpvOpSpecConstantOp>(const spv_instruction_t *inst) {}
#endif

template <>
bool idUsage::isValid<SpvOpVariable>(const spv_instruction_t* inst,
                                     const spv_opcode_desc opcodeEntry) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType || SpvOpTypePointer != resultType->opcode()) {
    DIAG(resultTypeIndex) << "OpVariable Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' is not a pointer type.";
    return false;
  }
  if (opcodeEntry->numTypes < inst->words.size()) {
    auto initialiserIndex = 4;
    auto initialiser = module_.FindDef(inst->words[initialiserIndex]);
    if (!initialiser || !spvOpcodeIsConstant(initialiser->opcode())) {
      DIAG(initialiserIndex) << "OpVariable Initializer <id> '"
                             << inst->words[initialiserIndex]
                             << "' is not a constant.";
      return false;
    }
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpLoad>(const spv_instruction_t* inst,
                                 const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  spvCheck(!resultType, DIAG(resultTypeIndex) << "OpLoad Result Type <id> '"
                                              << inst->words[resultTypeIndex]
                                              << "' is not defind.";
           return false);
  auto pointerIndex = 3;
  auto pointer = module_.FindDef(inst->words[pointerIndex]);
  if (!pointer || (addressingModel == SpvAddressingModelLogical &&
                   !spvOpcodeReturnsLogicalPointer(pointer->opcode()))) {
    DIAG(pointerIndex) << "OpLoad Pointer <id> '" << inst->words[pointerIndex]
                       << "' is not a pointer.";
    return false;
  }
  auto pointerType = module_.FindDef(pointer->type_id());
  if (!pointerType || pointerType->opcode() != SpvOpTypePointer) {
    DIAG(pointerIndex) << "OpLoad type for pointer <id> '"
                       << inst->words[pointerIndex]
                       << "' is not a pointer type.";
    return false;
  }
  auto pointeeType = module_.FindDef(pointerType->words()[3]);
  if (!pointeeType || resultType->id() != pointeeType->id()) {
    DIAG(resultTypeIndex) << "OpLoad Result Type <id> '"
                          << inst->words[resultTypeIndex]
                          << "' does not match Pointer <id> '" << pointer->id()
                          << "'s type.";
    return false;
  }
  return true;
}

template <>
bool idUsage::isValid<SpvOpStore>(const spv_instruction_t* inst,
                                  const spv_opcode_desc) {
  auto pointerIndex = 1;
  auto pointer = module_.FindDef(inst->words[pointerIndex]);
  if (!pointer || (addressingModel == SpvAddressingModelLogical &&
                   !spvOpcodeReturnsLogicalPointer(pointer->opcode()))) {
    DIAG(pointerIndex) << "OpStore Pointer <id> '" << inst->words[pointerIndex]
                       << "' is not a pointer.";
    return false;
  }
  auto pointerType = module_.FindDef(pointer->type_id());
  if (!pointer || pointerType->opcode() != SpvOpTypePointer) {
    DIAG(pointerIndex) << "OpStore type for pointer <id> '"
                       << inst->words[pointerIndex]
                       << "' is not a pointer type.";
    return false;
  }
  auto type = module_.FindDef(pointerType->words()[3]);
  assert(type);
  spvCheck(SpvOpTypeVoid == type->opcode(), DIAG(pointerIndex)
                                                << "OpStore Pointer <id> '"
                                                << inst->words[pointerIndex]
                                                << "'s type is void.";
           return false);

  auto objectIndex = 2;
  auto object = module_.FindDef(inst->words[objectIndex]);
  if (!object || !object->type_id()) {
    DIAG(objectIndex) << "OpStore Object <id> '" << inst->words[objectIndex]
                      << "' is not an object.";
    return false;
  }
  auto objectType = module_.FindDef(object->type_id());
  assert(objectType);
  spvCheck(SpvOpTypeVoid == objectType->opcode(),
           DIAG(objectIndex) << "OpStore Object <id> '"
                             << inst->words[objectIndex] << "'s type is void.";
           return false);

  spvCheck(type->id() != objectType->id(),
           DIAG(pointerIndex)
               << "OpStore Pointer <id> '" << inst->words[pointerIndex]
               << "'s type does not match Object <id> '" << objectType->id()
               << "'s type.";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpCopyMemory>(const spv_instruction_t* inst,
                                       const spv_opcode_desc) {
  auto targetIndex = 1;
  auto target = module_.FindDef(inst->words[targetIndex]);
  if (!target) return false;
  auto sourceIndex = 2;
  auto source = module_.FindDef(inst->words[sourceIndex]);
  if (!source) return false;
  auto targetPointerType = module_.FindDef(target->type_id());
  assert(targetPointerType);
  auto targetType = module_.FindDef(targetPointerType->words()[3]);
  assert(targetType);
  auto sourcePointerType = module_.FindDef(source->type_id());
  assert(sourcePointerType);
  auto sourceType = module_.FindDef(sourcePointerType->words()[3]);
  assert(sourceType);
  spvCheck(targetType->id() != sourceType->id(),
           DIAG(sourceIndex)
               << "OpCopyMemory Target <id> '" << inst->words[sourceIndex]
               << "'s type does not match Source <id> '" << sourceType->id()
               << "'s type.";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpCopyMemorySized>(const spv_instruction_t* inst,
                                            const spv_opcode_desc) {
  auto targetIndex = 1;
  auto target = module_.FindDef(inst->words[targetIndex]);
  if (!target) return false;
  auto sourceIndex = 2;
  auto source = module_.FindDef(inst->words[sourceIndex]);
  if (!source) return false;
  auto sizeIndex = 3;
  auto size = module_.FindDef(inst->words[sizeIndex]);
  if (!size) return false;
  auto targetPointerType = module_.FindDef(target->type_id());
  spvCheck(
      !targetPointerType || SpvOpTypePointer != targetPointerType->opcode(),
      DIAG(targetIndex) << "OpCopyMemorySized Target <id> '"
                        << inst->words[targetIndex] << "' is not a pointer.";
      return false);
  auto sourcePointerType = module_.FindDef(source->type_id());
  spvCheck(
      !sourcePointerType || SpvOpTypePointer != sourcePointerType->opcode(),
      DIAG(sourceIndex) << "OpCopyMemorySized Source <id> '"
                        << inst->words[sourceIndex] << "' is not a pointer.";
      return false);
  switch (size->opcode()) {
    // TODO: The following opcode's are assumed to be valid, refer to the
    // following bug https://cvs.khronos.org/bugzilla/show_bug.cgi?id=13871 for
    // clarification
    case SpvOpConstant:
    case SpvOpSpecConstant: {
      auto sizeType = module_.FindDef(size->type_id());
      assert(sizeType);
      spvCheck(SpvOpTypeInt != sizeType->opcode(),
               DIAG(sizeIndex) << "OpCopyMemorySized Size <id> '"
                               << inst->words[sizeIndex]
                               << "'s type is not an integer type.";
               return false);
    } break;
    case SpvOpVariable: {
      auto pointerType = module_.FindDef(size->type_id());
      assert(pointerType);
      auto sizeType = module_.FindDef(pointerType->type_id());
      spvCheck(!sizeType || SpvOpTypeInt != sizeType->opcode(),
               DIAG(sizeIndex) << "OpCopyMemorySized Size <id> '"
                               << inst->words[sizeIndex]
                               << "'s variable type is not an integer type.";
               return false);
    } break;
    default:
      DIAG(sizeIndex) << "OpCopyMemorySized Size <id> '"
                      << inst->words[sizeIndex]
                      << "' is not a constant or variable.";
      return false;
  }
  // TODO: Check that consant is a least size 1, see the same bug as above for
  // clarification?
  return true;
}

#if 0
template <>
bool idUsage::isValid<SpvOpAccessChain>(const spv_instruction_t *inst,
                                        const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<SpvOpInBoundsAccessChain>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<SpvOpArrayLength>(const spv_instruction_t *inst,
                                        const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<SpvOpImagePointer>(const spv_instruction_t *inst,
                                         const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<SpvOpGenericPtrMemSemantics>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

template <>
bool idUsage::isValid<SpvOpFunction>(const spv_instruction_t* inst,
                                     const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType) return false;
  auto functionTypeIndex = 4;
  auto functionType = module_.FindDef(inst->words[functionTypeIndex]);
  if (!functionType || SpvOpTypeFunction != functionType->opcode()) {
    DIAG(functionTypeIndex) << "OpFunction Function Type <id> '"
                            << inst->words[functionTypeIndex]
                            << "' is not a function type.";
    return false;
  }
  auto returnType = module_.FindDef(functionType->words()[2]);
  assert(returnType);
  spvCheck(returnType->id() != resultType->id(),
           DIAG(resultTypeIndex) << "OpFunction Result Type <id> '"
                                 << inst->words[resultTypeIndex]
                                 << "' does not match the Function Type <id> '"
                                 << resultType->id() << "'s return type.";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpFunctionParameter>(const spv_instruction_t* inst,
                                              const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType) return false;
  // NOTE: Find OpFunction & ensure OpFunctionParameter is not out of place.
  size_t paramIndex = 0;
  assert(firstInst < inst && "Invalid instruction pointer");
  while (firstInst != --inst) {
    if (SpvOpFunction == inst->opcode) {
      break;
    } else if (SpvOpFunctionParameter == inst->opcode) {
      paramIndex++;
    }
  }
  auto functionType = module_.FindDef(inst->words[4]);
  assert(functionType);
  if (paramIndex >= functionType->words().size() - 3) {
    DIAG(0) << "Too many OpFunctionParameters for " << inst->words[2]
            << ": expected " << functionType->words().size() - 3
            << " based on the function's type";
    return false;
  }
  auto paramType = module_.FindDef(functionType->words()[paramIndex + 3]);
  assert(paramType);
  spvCheck(resultType->id() != paramType->id(),
           DIAG(resultTypeIndex)
               << "OpFunctionParameter Result Type <id> '"
               << inst->words[resultTypeIndex]
               << "' does not match the OpTypeFunction parameter "
                  "type of the same index.";
           return false);
  return true;
}

template <>
bool idUsage::isValid<SpvOpFunctionCall>(const spv_instruction_t* inst,
                                         const spv_opcode_desc) {
  auto resultTypeIndex = 1;
  auto resultType = module_.FindDef(inst->words[resultTypeIndex]);
  if (!resultType) return false;
  auto functionIndex = 3;
  auto function = module_.FindDef(inst->words[functionIndex]);
  if (!function || SpvOpFunction != function->opcode()) {
    DIAG(functionIndex) << "OpFunctionCall Function <id> '"
                        << inst->words[functionIndex] << "' is not a function.";
    return false;
  }
  auto returnType = module_.FindDef(function->type_id());
  assert(returnType);
  spvCheck(returnType->id() != resultType->id(),
           DIAG(resultTypeIndex) << "OpFunctionCall Result Type <id> '"
                                 << inst->words[resultTypeIndex]
                                 << "'s type does not match Function <id> '"
                                 << returnType->id() << "'s return type.";
           return false);
  auto functionType = module_.FindDef(function->words()[4]);
  assert(functionType);
  auto functionCallArgCount = inst->words.size() - 4;
  auto functionParamCount = functionType->words().size() - 3;
  spvCheck(
      functionParamCount != functionCallArgCount,
      DIAG(inst->words.size() - 1)
          << "OpFunctionCall Function <id>'s parameter count does not match "
             "the argument count.";
      return false);
  for (size_t argumentIndex = 4, paramIndex = 3;
       argumentIndex < inst->words.size(); argumentIndex++, paramIndex++) {
    auto argument = module_.FindDef(inst->words[argumentIndex]);
    if (!argument) return false;
    auto argumentType = module_.FindDef(argument->type_id());
    assert(argumentType);
    auto parameterType = module_.FindDef(functionType->words()[paramIndex]);
    assert(parameterType);
    spvCheck(argumentType->id() != parameterType->id(),
             DIAG(argumentIndex) << "OpFunctionCall Argument <id> '"
                                 << inst->words[argumentIndex]
                                 << "'s type does not match Function <id> '"
                                 << parameterType->id() << "'s parameter type.";
             return false);
  }
  return true;
}

#if 0
template <>
bool idUsage::isValid<OpConvertUToF>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpConvertFToS>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpConvertSToF>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpConvertUToF>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUConvert>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSConvert>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFConvert>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpConvertPtrToU>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpConvertUToPtr>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpPtrCastToGeneric>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGenericCastToPtr>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBitcast>(const spv_instruction_t *inst,
                                 const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGenericCastToPtrExplicit>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSatConvertSToU>(const spv_instruction_t *inst) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSatConvertUToS>(const spv_instruction_t *inst) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpVectorExtractDynamic>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpVectorInsertDynamic>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpVectorShuffle>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpCompositeConstruct>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCompositeExtract>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCompositeInsert>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCopyObject>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpTranspose>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSNegate>(const spv_instruction_t *inst,
                                 const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFNegate>(const spv_instruction_t *inst,
                                 const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpNot>(const spv_instruction_t *inst,
                             const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIAdd>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFAdd>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpISub>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFSub>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIMul>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFMul>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUDiv>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSDiv>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFDiv>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUMod>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSRem>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSMod>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFRem>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFMod>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpVectorTimesScalar>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpMatrixTimesScalar>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpVectorTimesMatrix>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpMatrixTimesVector>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpMatrixTimesMatrix>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpOuterProduct>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDot>(const spv_instruction_t *inst,
                             const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpShiftRightLogical>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpShiftRightArithmetic>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpShiftLeftLogical>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBitwiseOr>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBitwiseXor>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBitwiseAnd>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAny>(const spv_instruction_t *inst,
                             const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAll>(const spv_instruction_t *inst,
                             const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsNan>(const spv_instruction_t *inst,
                               const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsInf>(const spv_instruction_t *inst,
                               const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsFinite>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsNormal>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSignBitSet>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpLessOrGreater>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpOrdered>(const spv_instruction_t *inst,
                                 const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUnordered>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpLogicalOr>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpLogicalXor>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpLogicalAnd>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSelect>(const spv_instruction_t *inst,
                                const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIEqual>(const spv_instruction_t *inst,
                                const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdEqual>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordEqual>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpINotEqual>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdNotEqual>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordNotEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpULessThan>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSLessThan>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdLessThan>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordLessThan>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUGreaterThan>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSGreaterThan>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdGreaterThan>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordGreaterThan>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpULessThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSLessThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdLessThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordLessThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpUGreaterThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSGreaterThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFOrdGreaterThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFUnordGreaterThanEqual>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdx>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdy>(const spv_instruction_t *inst,
                              const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFWidth>(const spv_instruction_t *inst,
                                const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdxFine>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdyFine>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFwidthFine>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdxCoarse>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpDPdyCoarse>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpFwidthCoarse>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpPhi>(const spv_instruction_t *inst,
                             const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpLoopMerge>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSelectionMerge>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBranch>(const spv_instruction_t *inst,
                                const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBranchConditional>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSwitch>(const spv_instruction_t *inst,
                                const spv_opcode_desc opcodeEntry) {}
#endif

template <>
bool idUsage::isValid<SpvOpReturnValue>(const spv_instruction_t* inst,
                                        const spv_opcode_desc) {
  auto valueIndex = 1;
  auto value = module_.FindDef(inst->words[valueIndex]);
  if (!value || !value->type_id()) {
    DIAG(valueIndex) << "OpReturnValue Value <id> '" << inst->words[valueIndex]
                     << "' does not represent a value.";
    return false;
  }
  auto valueType = module_.FindDef(value->type_id());
  if (!valueType || SpvOpTypeVoid == valueType->opcode()) {
    DIAG(valueIndex) << "OpReturnValue value's type <id> '" << value->type_id()
                     << "' is missing or void.";
    return false;
  }
  if (addressingModel == SpvAddressingModelLogical &&
      SpvOpTypePointer == valueType->opcode()) {
    DIAG(valueIndex)
        << "OpReturnValue value's type <id> '" << value->type_id()
        << "' is a pointer, which is invalid in the Logical addressing model.";
    return false;
  }
  // NOTE: Find OpFunction
  const spv_instruction_t* function = inst - 1;
  while (firstInst != function) {
    spvCheck(SpvOpFunction == function->opcode, break);
    function--;
  }
  spvCheck(SpvOpFunction != function->opcode,
           DIAG(valueIndex) << "OpReturnValue is not in a basic block.";
           return false);
  auto returnType = module_.FindDef(function->words[1]);
  spvCheck(!returnType || returnType->id() != valueType->id(),
           DIAG(valueIndex)
               << "OpReturnValue Value <id> '" << inst->words[valueIndex]
               << "'s type does not match OpFunction's return type.";
           return false);
  return true;
}

#if 0
template <>
bool idUsage::isValid<OpLifetimeStart>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpLifetimeStop>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicInit>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicLoad>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicStore>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicExchange>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicCompareExchange>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicCompareExchangeWeak>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicIIncrement>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicIDecrement>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicIAdd>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicISub>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicUMin>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicUMax>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicAnd>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicOr>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicXor>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicIMin>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpAtomicIMax>(const spv_instruction_t *inst,
                                    const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpEmitStreamVertex>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpEndStreamPrimitive>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupAsyncCopy>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupWaitEvents>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupAll>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupAny>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupBroadcast>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupIAdd>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupFAdd>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupFMin>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupUMin>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupSMin>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupFMax>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupUMax>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupSMax>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpEnqueueMarker>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpEnqueueKernel>(const spv_instruction_t *inst,
                                       const spv_opcode_desc opcodeEntry) {
}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetKernelNDrangeSubGroupCount>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetKernelNDrangeMaxSubGroupSize>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetKernelWorkGroupSize>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetKernelPreferredWorkGroupSizeMultiple>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpRetainEvent>(const spv_instruction_t *inst,
                                     const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReleaseEvent>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCreateUserEvent>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsValidEvent>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpSetUserEventStatus>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCaptureEventProfilingInfo>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetDefaultQueue>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpBuildNDRange>(const spv_instruction_t *inst,
                                      const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReadPipe>(const spv_instruction_t *inst,
                                  const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpWritePipe>(const spv_instruction_t *inst,
                                   const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReservedReadPipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReservedWritePipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReserveReadPipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpReserveWritePipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCommitReadPipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpCommitWritePipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpIsValidReserveId>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetNumPipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGetMaxPipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupReserveReadPipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupReserveWritePipePackets>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupCommitReadPipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#if 0
template <>
bool idUsage::isValid<OpGroupCommitWritePipe>(
    const spv_instruction_t *inst, const spv_opcode_desc opcodeEntry) {}
#endif

#undef DIAG

bool idUsage::isValid(const spv_instruction_t* inst) {
  spv_opcode_desc opcodeEntry = nullptr;
  spvCheck(spvOpcodeTableValueLookup(opcodeTable, inst->opcode, &opcodeEntry),
           return false);
#define CASE(OpCode) \
  case Spv##OpCode:  \
    return isValid<Spv##OpCode>(inst, opcodeEntry);
#define TODO(OpCode) \
  case Spv##OpCode:  \
    return true;
  switch (inst->opcode) {
    TODO(OpUndef)
    CASE(OpMemberName)
    CASE(OpLine)
    CASE(OpMemberDecorate)
    CASE(OpGroupDecorate)
    TODO(OpGroupMemberDecorate)
    TODO(OpExtInst)
    CASE(OpEntryPoint)
    CASE(OpExecutionMode)
    CASE(OpTypeVector)
    CASE(OpTypeMatrix)
    CASE(OpTypeSampler)
    CASE(OpTypeArray)
    CASE(OpTypeRuntimeArray)
    CASE(OpTypeStruct)
    CASE(OpTypePointer)
    CASE(OpTypeFunction)
    CASE(OpTypePipe)
    CASE(OpConstantTrue)
    CASE(OpConstantFalse)
    CASE(OpConstantComposite)
    CASE(OpConstantSampler)
    CASE(OpConstantNull)
    CASE(OpSpecConstantTrue)
    CASE(OpSpecConstantFalse)
    TODO(OpSpecConstantComposite)
    TODO(OpSpecConstantOp)
    CASE(OpVariable)
    CASE(OpLoad)
    CASE(OpStore)
    CASE(OpCopyMemory)
    CASE(OpCopyMemorySized)
    TODO(OpAccessChain)
    TODO(OpInBoundsAccessChain)
    TODO(OpArrayLength)
    TODO(OpGenericPtrMemSemantics)
    CASE(OpFunction)
    CASE(OpFunctionParameter)
    CASE(OpFunctionCall)
    TODO(OpConvertUToF)
    TODO(OpConvertFToS)
    TODO(OpConvertSToF)
    TODO(OpUConvert)
    TODO(OpSConvert)
    TODO(OpFConvert)
    TODO(OpConvertPtrToU)
    TODO(OpConvertUToPtr)
    TODO(OpPtrCastToGeneric)
    TODO(OpGenericCastToPtr)
    TODO(OpBitcast)
    TODO(OpGenericCastToPtrExplicit)
    TODO(OpSatConvertSToU)
    TODO(OpSatConvertUToS)
    TODO(OpVectorExtractDynamic)
    TODO(OpVectorInsertDynamic)
    TODO(OpVectorShuffle)
    TODO(OpCompositeConstruct)
    TODO(OpCompositeExtract)
    TODO(OpCompositeInsert)
    TODO(OpCopyObject)
    TODO(OpTranspose)
    TODO(OpSNegate)
    TODO(OpFNegate)
    TODO(OpNot)
    TODO(OpIAdd)
    TODO(OpFAdd)
    TODO(OpISub)
    TODO(OpFSub)
    TODO(OpIMul)
    TODO(OpFMul)
    TODO(OpUDiv)
    TODO(OpSDiv)
    TODO(OpFDiv)
    TODO(OpUMod)
    TODO(OpSRem)
    TODO(OpSMod)
    TODO(OpFRem)
    TODO(OpFMod)
    TODO(OpVectorTimesScalar)
    TODO(OpMatrixTimesScalar)
    TODO(OpVectorTimesMatrix)
    TODO(OpMatrixTimesVector)
    TODO(OpMatrixTimesMatrix)
    TODO(OpOuterProduct)
    TODO(OpDot)
    TODO(OpShiftRightLogical)
    TODO(OpShiftRightArithmetic)
    TODO(OpShiftLeftLogical)
    TODO(OpBitwiseOr)
    TODO(OpBitwiseXor)
    TODO(OpBitwiseAnd)
    TODO(OpAny)
    TODO(OpAll)
    TODO(OpIsNan)
    TODO(OpIsInf)
    TODO(OpIsFinite)
    TODO(OpIsNormal)
    TODO(OpSignBitSet)
    TODO(OpLessOrGreater)
    TODO(OpOrdered)
    TODO(OpUnordered)
    TODO(OpLogicalOr)
    TODO(OpLogicalAnd)
    TODO(OpSelect)
    TODO(OpIEqual)
    TODO(OpFOrdEqual)
    TODO(OpFUnordEqual)
    TODO(OpINotEqual)
    TODO(OpFOrdNotEqual)
    TODO(OpFUnordNotEqual)
    TODO(OpULessThan)
    TODO(OpSLessThan)
    TODO(OpFOrdLessThan)
    TODO(OpFUnordLessThan)
    TODO(OpUGreaterThan)
    TODO(OpSGreaterThan)
    TODO(OpFOrdGreaterThan)
    TODO(OpFUnordGreaterThan)
    TODO(OpULessThanEqual)
    TODO(OpSLessThanEqual)
    TODO(OpFOrdLessThanEqual)
    TODO(OpFUnordLessThanEqual)
    TODO(OpUGreaterThanEqual)
    TODO(OpSGreaterThanEqual)
    TODO(OpFOrdGreaterThanEqual)
    TODO(OpFUnordGreaterThanEqual)
    TODO(OpDPdx)
    TODO(OpDPdy)
    TODO(OpFwidth)
    TODO(OpDPdxFine)
    TODO(OpDPdyFine)
    TODO(OpFwidthFine)
    TODO(OpDPdxCoarse)
    TODO(OpDPdyCoarse)
    TODO(OpFwidthCoarse)
    TODO(OpPhi)
    TODO(OpLoopMerge)
    TODO(OpSelectionMerge)
    TODO(OpBranch)
    TODO(OpBranchConditional)
    TODO(OpSwitch)
    CASE(OpReturnValue)
    TODO(OpLifetimeStart)
    TODO(OpLifetimeStop)
    TODO(OpAtomicLoad)
    TODO(OpAtomicStore)
    TODO(OpAtomicExchange)
    TODO(OpAtomicCompareExchange)
    TODO(OpAtomicCompareExchangeWeak)
    TODO(OpAtomicIIncrement)
    TODO(OpAtomicIDecrement)
    TODO(OpAtomicIAdd)
    TODO(OpAtomicISub)
    TODO(OpAtomicUMin)
    TODO(OpAtomicUMax)
    TODO(OpAtomicAnd)
    TODO(OpAtomicOr)
    TODO(OpAtomicSMin)
    TODO(OpAtomicSMax)
    TODO(OpEmitStreamVertex)
    TODO(OpEndStreamPrimitive)
    TODO(OpGroupAsyncCopy)
    TODO(OpGroupWaitEvents)
    TODO(OpGroupAll)
    TODO(OpGroupAny)
    TODO(OpGroupBroadcast)
    TODO(OpGroupIAdd)
    TODO(OpGroupFAdd)
    TODO(OpGroupFMin)
    TODO(OpGroupUMin)
    TODO(OpGroupSMin)
    TODO(OpGroupFMax)
    TODO(OpGroupUMax)
    TODO(OpGroupSMax)
    TODO(OpEnqueueMarker)
    TODO(OpEnqueueKernel)
    TODO(OpGetKernelNDrangeSubGroupCount)
    TODO(OpGetKernelNDrangeMaxSubGroupSize)
    TODO(OpGetKernelWorkGroupSize)
    TODO(OpGetKernelPreferredWorkGroupSizeMultiple)
    TODO(OpRetainEvent)
    TODO(OpReleaseEvent)
    TODO(OpCreateUserEvent)
    TODO(OpIsValidEvent)
    TODO(OpSetUserEventStatus)
    TODO(OpCaptureEventProfilingInfo)
    TODO(OpGetDefaultQueue)
    TODO(OpBuildNDRange)
    TODO(OpReadPipe)
    TODO(OpWritePipe)
    TODO(OpReservedReadPipe)
    TODO(OpReservedWritePipe)
    TODO(OpReserveReadPipePackets)
    TODO(OpReserveWritePipePackets)
    TODO(OpCommitReadPipe)
    TODO(OpCommitWritePipe)
    TODO(OpIsValidReserveId)
    TODO(OpGetNumPipePackets)
    TODO(OpGetMaxPipePackets)
    TODO(OpGroupReserveReadPipePackets)
    TODO(OpGroupReserveWritePipePackets)
    TODO(OpGroupCommitReadPipe)
    TODO(OpGroupCommitWritePipe)
    default:
      return true;
  }
#undef TODO
#undef CASE
}
// This function takes the opcode of an instruction and returns
// a function object that will return true if the index
// of the operand can be forwarad declared. This function will
// used in the SSA validation stage of the pipeline
function<bool(unsigned)> getCanBeForwardDeclaredFunction(SpvOp opcode) {
  function<bool(unsigned index)> out;
  switch (opcode) {
    case SpvOpExecutionMode:
    case SpvOpEntryPoint:
    case SpvOpName:
    case SpvOpMemberName:
    case SpvOpSelectionMerge:
    case SpvOpDecorate:
    case SpvOpMemberDecorate:
    case SpvOpBranch:
    case SpvOpLoopMerge:
      out = [](unsigned) { return true; };
      break;
    case SpvOpGroupDecorate:
    case SpvOpGroupMemberDecorate:
    case SpvOpBranchConditional:
    case SpvOpSwitch:
      out = [](unsigned index) { return index != 0; };
      break;

    case SpvOpFunctionCall:
      out = [](unsigned index) { return index == 2; };
      break;

    case SpvOpPhi:
      out = [](unsigned index) { return index > 1; };
      break;

    case SpvOpEnqueueKernel:
      out = [](unsigned index) { return index == 8; };
      break;

    case SpvOpGetKernelNDrangeSubGroupCount:
    case SpvOpGetKernelNDrangeMaxSubGroupSize:
      out = [](unsigned index) { return index == 3; };
      break;

    case SpvOpGetKernelWorkGroupSize:
    case SpvOpGetKernelPreferredWorkGroupSizeMultiple:
      out = [](unsigned index) { return index == 2; };
      break;

    default:
      out = [](unsigned) { return false; };
      break;
  }
  return out;
}
}  // anonymous namespace

namespace libspirv {

spv_result_t UpdateIdUse(ValidationState_t& _) {
  for (const auto& inst : _.ordered_instructions()) {
    for (auto& operand : inst.operands()) {
      const spv_operand_type_t& type = operand.type;
      const uint32_t operand_id = inst.word(operand.offset);
      if (spvIsIdType(type) && type != SPV_OPERAND_TYPE_RESULT_ID) {
        if (auto def = _.FindDef(operand_id))
          def->RegisterUse(&inst, operand.offset);
      }
    }
  }
  return SPV_SUCCESS;
}

/// This function checks all ID definitions dominate their use in the CFG.
///
/// This function will iterate over all ID definitions that are defined in the
/// functions of a module and make sure that the definitions appear in a
/// block that dominates their use.
///
/// NOTE: This function does NOT check module scoped functions which are
/// checked during the initial binary parse in the IdPass below
spv_result_t CheckIdDefinitionDominateUse(const ValidationState_t& _) {
  unordered_set<const Instruction*> phi_instructions;
  for (const auto& definition : _.all_definitions()) {
    // Check only those definitions defined in a function
    if (const Function* func = definition.second->function()) {
      if (const BasicBlock* block = definition.second->block()) {
        if (!block->reachable()) continue;
        // If the Id is defined within a block then make sure all references to
        // that Id appear in a blocks that are dominated by the defining block
        for (auto& use_index_pair : definition.second->uses()) {
          const Instruction* use = use_index_pair.first;
          if (const BasicBlock* use_block = use->block()) {
            if (use_block->reachable() == false) continue;
            if (use->opcode() == SpvOpPhi) {
              phi_instructions.insert(use);
            } else if (!block->dominates(*use->block())) {
              return _.diag(SPV_ERROR_INVALID_ID)
                     << "ID " << _.getIdName(definition.first)
                     << " defined in block " << _.getIdName(block->id())
                     << " does not dominate its use in block "
                     << _.getIdName(use_block->id());
            }
          }
        }
      } else {
        // If the Ids defined within a function but not in a block(i.e. function
        // parameters, block ids), then make sure all references to that Id
        // appear within the same function
        for (auto use : definition.second->uses()) {
          const Instruction* inst = use.first;
          if (inst->function() && inst->function() != func) {
            return _.diag(SPV_ERROR_INVALID_ID)
                   << "ID " << _.getIdName(definition.first)
                   << " used in function "
                   << _.getIdName(inst->function()->id())
                   << " is used outside of it's defining function "
                   << _.getIdName(func->id());
          }
        }
      }
    }
    // NOTE: Ids defined outside of functions must appear before they are used
    // This check is being performed in the IdPass function
  }

  // Check all OpPhi parent blocks are dominated by the variable's defining
  // blocks
  for (const Instruction* phi : phi_instructions) {
    if (phi->block()->reachable() == false) continue;
    for (size_t i = 3; i < phi->operands().size(); i += 2) {
      const Instruction* variable = _.FindDef(phi->word(i));
      const BasicBlock* parent =
          phi->function()->GetBlock(phi->word(i + 1)).first;
      if (variable->block() && !variable->block()->dominates(*parent)) {
        return _.diag(SPV_ERROR_INVALID_ID)
               << "In OpPhi instruction " << _.getIdName(phi->id()) << ", ID "
               << _.getIdName(variable->id())
               << " definition does not dominate its parent "
               << _.getIdName(parent->id());
      }
    }
  }

  return SPV_SUCCESS;
}

// Performs SSA validation on the IDs of an instruction. The
// can_have_forward_declared_ids  functor should return true if the
// instruction operand's ID can be forward referenced.
spv_result_t IdPass(ValidationState_t& _,
                    const spv_parsed_instruction_t* inst) {
  auto can_have_forward_declared_ids =
      getCanBeForwardDeclaredFunction(static_cast<SpvOp>(inst->opcode));

  for (unsigned i = 0; i < inst->num_operands; i++) {
    const spv_parsed_operand_t& operand = inst->operands[i];
    const spv_operand_type_t& type = operand.type;
    const uint32_t* operand_ptr = inst->words + operand.offset;

    auto ret = SPV_ERROR_INTERNAL;
    switch (type) {
      case SPV_OPERAND_TYPE_RESULT_ID:
        // NOTE: Multiple Id definitions are being checked by the binary parser
        // NOTE: result Id is added *after* all of the other Ids have been
        // checked to avoid premature use in the same instruction
        _.RemoveIfForwardDeclared(*operand_ptr);
        ret = SPV_SUCCESS;
        break;
      case SPV_OPERAND_TYPE_ID:
      case SPV_OPERAND_TYPE_TYPE_ID:
      case SPV_OPERAND_TYPE_MEMORY_SEMANTICS_ID:
      case SPV_OPERAND_TYPE_SCOPE_ID:
        if (_.IsDefinedId(*operand_ptr)) {
          ret = SPV_SUCCESS;
        } else if (can_have_forward_declared_ids(i)) {
          ret = _.ForwardDeclareId(*operand_ptr);
        } else {
          ret = _.diag(SPV_ERROR_INVALID_ID) << "ID "
                                             << _.getIdName(*operand_ptr)
                                             << " has not been defined";
        }
        break;
      default:
        ret = SPV_SUCCESS;
        break;
    }
    if (SPV_SUCCESS != ret) {
      return ret;
    }
  }
  _.RegisterInstruction(*inst);
  return SPV_SUCCESS;
}
}  // namespace libspirv

spv_result_t spvValidateInstructionIDs(const spv_instruction_t* pInsts,
                                       const uint64_t instCount,
                                       const spv_opcode_table opcodeTable,
                                       const spv_operand_table operandTable,
                                       const spv_ext_inst_table extInstTable,
                                       const libspirv::ValidationState_t& state,
                                       spv_position position,
                                       spv_diagnostic* pDiag) {
  idUsage idUsage(opcodeTable, operandTable, extInstTable, pInsts, instCount,
                  state.memory_model(), state.addressing_model(), state,
                  state.entry_points(), position, pDiag);
  for (uint64_t instIndex = 0; instIndex < instCount; ++instIndex) {
    spvCheck(!idUsage.isValid(&pInsts[instIndex]), return SPV_ERROR_INVALID_ID);
    position->index += pInsts[instIndex].words.size();
  }
  return SPV_SUCCESS;
}
