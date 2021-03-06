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

instruction_counters_t performStaticAnalysis(const uint32_t* spirv_binary,
                                             size_t length) {
  spirv_cross::Parser parser(spirv_binary, length);
  parser.parse();
  spirv_cross::ParsedIR pir = parser.get_parsed_ir();

  instruction_counters_t counters = instruction_counters_t{0, 0, 0, 0};

  for (auto& block : pir.ids_for_type[spirv_cross::Types::TypeBlock]) {
    if (pir.ids[block].get_type() ==
        static_cast<spirv_cross::Types>(spirv_cross::Types::TypeBlock)) {
      spirv_cross::SPIRBlock currentBlock =
          spirv_cross::variant_get<spirv_cross::SPIRBlock>(pir.ids[block]);
      for (auto& currentOp : currentBlock.ops) {
        uint32_t resultID = 0;

        switch (currentOp.op) {
          default:
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
            counters.alu_instructions++;

            resultID = pir.spirv[currentOp.offset + 1];
            if (pir.ids[resultID].get_type() == spirv_cross::TypeNone) {
              counters.temp_registers++;
            }
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

            resultID = pir.spirv[currentOp.offset + 1];
            if (pir.ids[resultID].get_type() == spirv_cross::TypeNone) {
              counters.temp_registers++;
            }
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
            resultID = pir.spirv[currentOp.offset + 1];
            if (pir.ids[resultID].get_type() == spirv_cross::TypeNone) {
              counters.temp_registers++;
            }
            break;
        }
      }

      switch (currentBlock.terminator) {
        default:
          break;

        // OpBranch
        case spirv_cross::SPIRBlock::Direct:
        // OpBranchConditional
        case spirv_cross::SPIRBlock::Select:
        // OpSwitch
        case spirv_cross::SPIRBlock::MultiSelect:
          counters.branch_instructions++;
          break;
      }
    }
  }

  return counters;
}
