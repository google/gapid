/*
 * Copyright (C) 2021 Google Inc.
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

#include "spirv-tools/libspirv.hpp"
#include "spirv_parser.hpp"

#include "staticanalysis.h"

#include <cstring>
#include <map>
#include <set>
#include <vector>

struct AnalysisBlock {
  std::set<uint32_t> def;
  std::set<uint32_t> use;
  std::set<uint32_t> in;
  std::set<uint32_t> out;
  std::vector<uint32_t> successors;
  std::map<uint32_t, uint32_t> firstMade;
  std::map<uint32_t, uint32_t> lastUse;
};

bool isValidIDForPressure(spirv_cross::ParsedIR pir, uint32_t id,
                          spirv_cross::Instruction op, uint32_t offset) {
  if (pir.ids[id].get_type() == spirv_cross::TypeNone) {
    switch (op.op) {
      case spv::OpExtInst:
        return offset != 3;

      case spv::OpVectorShuffle:
        return offset < 4;

      case spv::OpArrayLength:
        return offset != 3;

      case spv::OpCompositeExtract:
        return offset < 3;

      case spv::OpCompositeInsert:
        return offset < 4;

      default:
        return true;
    }
  }

  return false;
}

void processInstructionForPressure(spirv_cross::ParsedIR pir,
                                   spirv_cross::Instruction currentOp,
                                   AnalysisBlock& analysisBlock,
                                   uint32_t instructionCounter,
                                   std::map<uint32_t, uint32_t>& resultSizes,
                                   bool hasResultID) {
  if (hasResultID) {
    uint32_t resultID = pir.spirv[currentOp.offset + 1];
    if (isValidIDForPressure(pir, resultID, currentOp, 1)) {
      analysisBlock.def.insert(resultID);
      analysisBlock.lastUse[resultID] = instructionCounter;
      analysisBlock.firstMade[resultID] = instructionCounter;

      uint32_t typeID = pir.spirv[currentOp.offset];
      auto typeInfo =
          spirv_cross::variant_get<spirv_cross::SPIRType>(pir.ids[typeID]);
      resultSizes[resultID] = typeInfo.vecsize * typeInfo.columns;
    }

    for (uint32_t i = 2; i < currentOp.length; ++i) {
      uint32_t id = pir.spirv[currentOp.offset + i];
      if (isValidIDForPressure(pir, id, currentOp, i)) {
        if (analysisBlock.def.find(id) == analysisBlock.def.end()) {
          analysisBlock.use.insert(id);
        }
        analysisBlock.lastUse[id] = instructionCounter;
      }
    }
  } else {
    for (uint32_t i = 0; i < currentOp.length; ++i) {
      uint32_t id = pir.spirv[currentOp.offset + i];
      if (isValidIDForPressure(pir, id, currentOp, i)) {
        if (analysisBlock.def.find(id) == analysisBlock.def.end()) {
          analysisBlock.use.insert(id);
        }
        analysisBlock.lastUse[id] = instructionCounter;
      }
    }
  }
}

instruction_counters_t performStaticAnalysisInternal(
    const uint32_t* spirv_binary, size_t length) {
  spirv_cross::Parser parser(spirv_binary, length);
  parser.parse();
  spirv_cross::ParsedIR pir = parser.get_parsed_ir();

  instruction_counters_t counters = instruction_counters_t{0, 0, 0, 0};

  std::map<uint32_t, AnalysisBlock> analysisBlocks;

  std::map<uint32_t, uint32_t> idRegSizes;

  for (auto& block : pir.ids_for_type[spirv_cross::Types::TypeBlock]) {
    if (pir.ids[block].get_type() ==
        static_cast<spirv_cross::Types>(spirv_cross::Types::TypeBlock)) {
      spirv_cross::SPIRBlock currentBlock =
          spirv_cross::variant_get<spirv_cross::SPIRBlock>(pir.ids[block]);

      AnalysisBlock currentAnalysisBlock;
      uint32_t instructionCounter = 0;
      for (auto& currentOp : currentBlock.ops) {
        switch (currentOp.op) {
          default:
            processInstructionForPressure(pir, currentOp, currentAnalysisBlock,
                                          instructionCounter, idRegSizes,
                                          false);
            break;

          // ALU Instructions
          case spv::OpSizeOf:
          case spv::OpConvertFToU:
          case spv::OpConvertFToS:
          case spv::OpConvertSToF:
          case spv::OpConvertUToF:
          case spv::OpUConvert:
          case spv::OpSConvert:
          case spv::OpFConvert:
          case spv::OpQuantizeToF16:
          case spv::OpConvertPtrToU:
          case spv::OpSatConvertSToU:
          case spv::OpSatConvertUToS:
          case spv::OpConvertUToPtr:
          case spv::OpPtrCastToGeneric:
          case spv::OpGenericCastToPtr:
          case spv::OpGenericCastToPtrExplicit:
          case spv::OpBitcast:
          case spv::OpPtrEqual:
          case spv::OpPtrNotEqual:
          case spv::OpPtrDiff:
          case spv::OpSNegate:
          case spv::OpFNegate:
          case spv::OpIAdd:
          case spv::OpFAdd:
          case spv::OpISub:
          case spv::OpFSub:
          case spv::OpIMul:
          case spv::OpFMul:
          case spv::OpUDiv:
          case spv::OpSDiv:
          case spv::OpFDiv:
          case spv::OpUMod:
          case spv::OpSRem:
          case spv::OpSMod:
          case spv::OpFRem:
          case spv::OpFMod:
          case spv::OpVectorTimesScalar:
          case spv::OpMatrixTimesScalar:
          case spv::OpVectorTimesMatrix:
          case spv::OpMatrixTimesVector:
          case spv::OpMatrixTimesMatrix:
          case spv::OpOuterProduct:
          case spv::OpDot:
          case spv::OpIAddCarry:
          case spv::OpISubBorrow:
          case spv::OpUMulExtended:
          case spv::OpSMulExtended:
          case spv::OpShiftRightLogical:
          case spv::OpShiftRightArithmetic:
          case spv::OpBitwiseOr:
          case spv::OpBitwiseXor:
          case spv::OpBitwiseAnd:
          case spv::OpNot:
          case spv::OpBitFieldInsert:
          case spv::OpBitFieldSExtract:
          case spv::OpBitFieldUExtract:
          case spv::OpBitReverse:
          case spv::OpBitCount:
          case spv::OpAny:
          case spv::OpAll:
          case spv::OpIsNan:
          case spv::OpIsInf:
          case spv::OpIsFinite:
          case spv::OpIsNormal:
          case spv::OpSignBitSet:
          case spv::OpLessOrGreater:
          case spv::OpOrdered:
          case spv::OpUnordered:
          case spv::OpLogicalEqual:
          case spv::OpLogicalNotEqual:
          case spv::OpLogicalOr:
          case spv::OpLogicalAnd:
          case spv::OpLogicalNot:
          case spv::OpSelect:
          case spv::OpIEqual:
          case spv::OpINotEqual:
          case spv::OpUGreaterThan:
          case spv::OpSGreaterThan:
          case spv::OpUGreaterThanEqual:
          case spv::OpSGreaterThanEqual:
          case spv::OpULessThan:
          case spv::OpSLessThan:
          case spv::OpULessThanEqual:
          case spv::OpSLessThanEqual:
          case spv::OpFOrdEqual:
          case spv::OpFUnordEqual:
          case spv::OpFOrdNotEqual:
          case spv::OpFUnordNotEqual:
          case spv::OpFOrdLessThan:
          case spv::OpFUnordLessThan:
          case spv::OpFOrdGreaterThan:
          case spv::OpFUnordGreaterThan:
          case spv::OpFOrdLessThanEqual:
          case spv::OpFUnordLessThanEqual:
          case spv::OpFOrdGreaterThanEqual:
          case spv::OpFUnordGreaterThanEqual:
          case spv::OpDPdx:
          case spv::OpDPdy:
          case spv::OpFwidth:
          case spv::OpDPdxFine:
          case spv::OpDPdyFine:
          case spv::OpFwidthFine:
          case spv::OpDPdxCoarse:
          case spv::OpDPdyCoarse:
          case spv::OpFwidthCoarse:
          case spv::OpAtomicIIncrement:
          case spv::OpAtomicIDecrement:
          case spv::OpAtomicIAdd:
          case spv::OpAtomicISub:
          case spv::OpAtomicSMin:
          case spv::OpAtomicUMin:
          case spv::OpAtomicSMax:
          case spv::OpAtomicUMax:
          case spv::OpAtomicAnd:
          case spv::OpAtomicOr:
          case spv::OpAtomicXor:
          case spv::OpAtomicFlagTestAndSet:
          case spv::OpAtomicFlagClear:
          case spv::OpGroupIAdd:
          case spv::OpGroupFAdd:
          case spv::OpGroupFMin:
          case spv::OpGroupUMin:
          case spv::OpGroupSMin:
          case spv::OpGroupFMax:
          case spv::OpGroupUMax:
          case spv::OpGroupSMax:
          case spv::OpGroupIAddNonUniformAMD:
          case spv::OpGroupFAddNonUniformAMD:
          case spv::OpGroupFMinNonUniformAMD:
          case spv::OpGroupUMinNonUniformAMD:
          case spv::OpGroupSMinNonUniformAMD:
          case spv::OpGroupFMaxNonUniformAMD:
          case spv::OpGroupUMaxNonUniformAMD:
          case spv::OpGroupSMaxNonUniformAMD:
          case spv::OpGroupNonUniformIAdd:
          case spv::OpGroupNonUniformFAdd:
          case spv::OpGroupNonUniformIMul:
          case spv::OpGroupNonUniformFMul:
          case spv::OpGroupNonUniformSMin:
          case spv::OpGroupNonUniformUMin:
          case spv::OpGroupNonUniformFMin:
          case spv::OpGroupNonUniformSMax:
          case spv::OpGroupNonUniformUMax:
          case spv::OpGroupNonUniformFMax:
          case spv::OpGroupNonUniformBitwiseAnd:
          case spv::OpGroupNonUniformBitwiseOr:
          case spv::OpGroupNonUniformBitwiseXor:
          case spv::OpGroupNonUniformLogicalAnd:
          case spv::OpGroupNonUniformLogicalOr:
          case spv::OpGroupNonUniformLogicalXor:
          case spv::OpExtInst:  // Treat OpExtInst as arithmetic for now
            counters.alu_instructions++;
            processInstructionForPressure(pir, currentOp, currentAnalysisBlock,
                                          instructionCounter, idRegSizes, true);
            break;

          case spv::OpSampledImage:
          case spv::OpImageSampleImplicitLod:
          case spv::OpImageSampleExplicitLod:
          case spv::OpImageSampleDrefImplicitLod:
          case spv::OpImageSampleDrefExplicitLod:
          case spv::OpImageSampleProjImplicitLod:
          case spv::OpImageSampleProjExplicitLod:
          case spv::OpImageSampleProjDrefImplicitLod:
          case spv::OpImageSampleProjDrefExplicitLod:
          case spv::OpImageFetch:
          case spv::OpImageGather:
          case spv::OpImageDrefGather:
          case spv::OpImageRead:
          case spv::OpImageWrite:
          case spv::OpImage:
          case spv::OpImageQueryFormat:
          case spv::OpImageQueryOrder:
          case spv::OpImageQuerySizeLod:
          case spv::OpImageQuerySize:
          case spv::OpImageQueryLod:
          case spv::OpImageQueryLevels:
          case spv::OpImageQuerySamples:
          case spv::OpImageSparseSampleImplicitLod:
          case spv::OpImageSparseSampleExplicitLod:
          case spv::OpImageSparseSampleDrefImplicitLod:
          case spv::OpImageSparseSampleDrefExplicitLod:
          case spv::OpImageSparseSampleProjImplicitLod:
          case spv::OpImageSparseSampleProjExplicitLod:
          case spv::OpImageSparseSampleProjDrefImplicitLod:
          case spv::OpImageSparseSampleProjDrefExplicitLod:
          case spv::OpImageSparseFetch:
          case spv::OpImageSparseGather:
          case spv::OpImageSparseDrefGather:
          case spv::OpImageSparseTexelsResident:
          case spv::OpImageSparseRead:
          case spv::OpImageSampleFootprintNV:
            counters.texture_instructions++;
            processInstructionForPressure(pir, currentOp, currentAnalysisBlock,
                                          instructionCounter, idRegSizes, true);
            break;

          // Deal with other instructions that have a result ID.
          // Ignore ones like OpString that are already known to not be
          // TypeNone.
          case spv::OpLoad:
          case spv::OpAccessChain:
          case spv::OpInBoundsAccessChain:
          case spv::OpPtrAccessChain:
          case spv::OpArrayLength:
          case spv::OpGenericPtrMemSemantics:
          case spv::OpInBoundsPtrAccessChain:
          case spv::OpFunctionParameter:
          case spv::OpFunctionCall:
          case spv::OpVectorExtractDynamic:
          case spv::OpVectorInsertDynamic:
          case spv::OpVectorShuffle:
          case spv::OpCompositeConstruct:
          case spv::OpCompositeExtract:
          case spv::OpCompositeInsert:
          case spv::OpCopyObject:
          case spv::OpTranspose:
          case spv::OpCopyLogical:
          case spv::OpPhi:
          case spv::OpLabel:
          case spv::OpAtomicLoad:
          case spv::OpAtomicExchange:
          case spv::OpAtomicCompareExchange:
          case spv::OpAtomicCompareExchangeWeak:
          case spv::OpNamedBarrierInitialize:
          case spv::OpGroupAsyncCopy:
          case spv::OpGroupAll:
          case spv::OpGroupAny:
          case spv::OpGroupBroadcast:
          case spv::OpSubgroupBallotKHR:
          case spv::OpSubgroupFirstInvocationKHR:
          case spv::OpSubgroupAllKHR:
          case spv::OpSubgroupAnyKHR:
          case spv::OpSubgroupAllEqualKHR:
          case spv::OpSubgroupReadInvocationKHR:
          case spv::OpSubgroupShuffleINTEL:
          case spv::OpSubgroupShuffleDownINTEL:
          case spv::OpSubgroupShuffleUpINTEL:
          case spv::OpSubgroupShuffleXorINTEL:
          case spv::OpSubgroupBlockReadINTEL:
          case spv::OpSubgroupBlockWriteINTEL:
          case spv::OpSubgroupImageBlockReadINTEL:
          case spv::OpSubgroupImageBlockWriteINTEL:
          case spv::OpSubgroupImageMediaBlockReadINTEL:
          case spv::OpSubgroupImageMediaBlockWriteINTEL:
          case spv::OpEnqueueMarker:
          case spv::OpEnqueueKernel:
          case spv::OpGetKernelNDrangeSubGroupCount:
          case spv::OpGetKernelNDrangeMaxSubGroupSize:
          case spv::OpGetKernelWorkGroupSize:
          case spv::OpGetKernelPreferredWorkGroupSizeMultiple:
          case spv::OpCreateUserEvent:
          case spv::OpIsValidEvent:
          case spv::OpGetDefaultQueue:
          case spv::OpBuildNDRange:
          case spv::OpGetKernelLocalSizeForSubgroupCount:
          case spv::OpGetKernelMaxNumSubgroups:
          case spv::OpReadPipe:
          case spv::OpWritePipe:
          case spv::OpReservedReadPipe:
          case spv::OpReservedWritePipe:
          case spv::OpReserveReadPipePackets:
          case spv::OpReserveWritePipePackets:
          case spv::OpIsValidReserveId:
          case spv::OpGetNumPipePackets:
          case spv::OpGetMaxPipePackets:
          case spv::OpGroupReserveReadPipePackets:
          case spv::OpGroupReserveWritePipePackets:
          case spv::OpConstantPipeStorage:
          case spv::OpCreatePipeFromPipeStorage:
          case spv::OpGroupNonUniformElect:
          case spv::OpGroupNonUniformAll:
          case spv::OpGroupNonUniformAny:
          case spv::OpGroupNonUniformAllEqual:
          case spv::OpGroupNonUniformBroadcast:
          case spv::OpGroupNonUniformBroadcastFirst:
          case spv::OpGroupNonUniformBallot:
          case spv::OpGroupNonUniformInverseBallot:
          case spv::OpGroupNonUniformBallotBitExtract:
          case spv::OpGroupNonUniformBallotBitCount:
          case spv::OpGroupNonUniformBallotFindLSB:
          case spv::OpGroupNonUniformBallotFindMSB:
          case spv::OpGroupNonUniformShuffle:
          case spv::OpGroupNonUniformShuffleXor:
          case spv::OpGroupNonUniformShuffleUp:
          case spv::OpGroupNonUniformShuffleDown:
          case spv::OpGroupNonUniformQuadBroadcast:
          case spv::OpGroupNonUniformQuadSwap:
          case spv::OpGroupNonUniformPartitionNV:
            processInstructionForPressure(pir, currentOp, currentAnalysisBlock,
                                          instructionCounter, idRegSizes, true);
            break;
        }

        instructionCounter++;
      }

      switch (currentBlock.terminator) {
        default:
          break;

        // OpBranch
        case spirv_cross::SPIRBlock::Direct:
          counters.branch_instructions++;
          currentAnalysisBlock.successors.push_back(currentBlock.next_block);
          break;
        // OpBranchConditional
        case spirv_cross::SPIRBlock::Select:
          counters.branch_instructions++;
          currentAnalysisBlock.successors.push_back(currentBlock.true_block);
          currentAnalysisBlock.successors.push_back(currentBlock.false_block);
          break;
        // OpSwitch
        case spirv_cross::SPIRBlock::MultiSelect:
          counters.branch_instructions++;
          for (auto& spirvCase : currentBlock.cases) {
            currentAnalysisBlock.successors.push_back(spirvCase.block);
          }
          break;
      }

      analysisBlocks[block] = currentAnalysisBlock;
    }
  }

  // Perform live-range analysis
  bool inSetChanged = true;
  while (inSetChanged) {
    inSetChanged = false;

    for (auto& block : pir.ids_for_type[spirv_cross::Types::TypeBlock]) {
      AnalysisBlock& currentAnalysisBlock = analysisBlocks[block];

      currentAnalysisBlock.out.clear();

      // out = union of successor ins
      for (uint32_t i = 0; i < currentAnalysisBlock.successors.size(); ++i) {
        AnalysisBlock& currentSuccessor =
            analysisBlocks[currentAnalysisBlock.successors[i]];
        currentAnalysisBlock.out.insert(currentSuccessor.in.begin(),
                                        currentSuccessor.in.end());
      }

      // diff = out - def
      std::set<uint32_t> diff;
      for (std::set<uint32_t>::iterator it = currentAnalysisBlock.out.begin();
           it != currentAnalysisBlock.out.end(); ++it) {
        if (currentAnalysisBlock.def.find(*it) ==
            currentAnalysisBlock.def.end()) {
          diff.insert(*it);
        }
      }

      // in = union of use and diff
      std::set<uint32_t> newIn;
      newIn.insert(currentAnalysisBlock.use.begin(),
                   currentAnalysisBlock.use.end());
      newIn.insert(diff.begin(), diff.end());

      // check for changes
      for (std::set<uint32_t>::iterator it = newIn.begin(); it != newIn.end();
           ++it) {
        if (currentAnalysisBlock.in.find(*it) ==
            currentAnalysisBlock.in.end()) {
          inSetChanged = true;
          currentAnalysisBlock.in.insert(*it);
        }
      }
    }
  }

  uint32_t maxPressure = 0;
  for (auto& block : pir.ids_for_type[spirv_cross::Types::TypeBlock]) {
    if (pir.ids[block].get_type() ==
        static_cast<spirv_cross::Types>(spirv_cross::Types::TypeBlock)) {
      auto currentBlock =
          spirv_cross::variant_get<spirv_cross::SPIRBlock>(pir.ids[block]);
      const AnalysisBlock& currentAnalysisBlock = analysisBlocks[block];

      uint32_t pressure = 0;
      for (std::set<uint32_t>::iterator it = currentAnalysisBlock.in.begin();
           it != currentAnalysisBlock.in.end(); ++it) {
        pressure += idRegSizes.find(*it)->second;
      }

      uint32_t instructionCounter = 0;
      uint32_t pressureToDelete = 0;
      for (auto& currentOp : currentBlock.ops) {
        std::set<uint32_t> deletedIDs;  // prevents multi-deletion if an
                                        // instruction uses an id more than once
        for (uint32_t i = 0; i < currentOp.length; ++i) {
          uint32_t currentID = pir.spirv[currentOp.offset + i];

          if (isValidIDForPressure(pir, currentID, currentOp, i)) {
            std::map<uint32_t, uint32_t>::const_iterator it =
                currentAnalysisBlock.firstMade.find(currentID);
            std::map<uint32_t, uint32_t>::const_iterator it2 =
                currentAnalysisBlock.lastUse.find(currentID);

            if (it != currentAnalysisBlock.firstMade.end() &&
                it->second == instructionCounter) {
              pressure += idRegSizes.find(currentID)->second;
            }

            if (it2 != currentAnalysisBlock.lastUse.end() &&
                it2->second == instructionCounter &&
                currentAnalysisBlock.out.find(currentID) ==
                    currentAnalysisBlock.out.end() &&
                deletedIDs.find(currentID) == deletedIDs.end()) {
              pressureToDelete += idRegSizes.find(currentID)->second;
              deletedIDs.insert(currentID);
            }
          }
        }

        if (pressure > maxPressure) {
          maxPressure = pressure;
        }

        pressure -= pressureToDelete;
        pressureToDelete = 0;

        instructionCounter++;
      }
    }
  }

  counters.temp_registers = maxPressure;

  return counters;
}

instruction_counters_t performStaticAnalysis(const uint32_t* spirv_binary,
                                             size_t length) {
  try {
    return performStaticAnalysisInternal(spirv_binary, length);
  } catch (...) {
    // TODO: do proper error handling and reporting.
    return instruction_counters_t{0, 0, 0, 0};
  }
}
