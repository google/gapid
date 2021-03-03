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

  instruction_counters_t counters;
  counters.alu_instructions = 0;
  counters.texture_instructions = 0;
  counters.branch_instructions = 0;

  for (auto& block : pir.ids_for_type[spirv_cross::Types::TypeBlock]) {
    if (pir.ids[block].get_type() ==
        static_cast<spirv_cross::Types>(spirv_cross::Types::TypeBlock)) {
      spirv_cross::SPIRBlock currentBlock =
          spirv_cross::variant_get<spirv_cross::SPIRBlock>(pir.ids[block]);
      for (auto& currentOp : currentBlock.ops) {
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
          case spv::OpShiftLeftLogical:
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
            counters.alu_instructions++;
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
