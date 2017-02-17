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

#include "UnitSPIRV.h"

#include "enum_set.h"

namespace {

using libspirv::CapabilitySet;
using spvtest::ElementsIn;

// Capabilities required by an Opcode.
struct ExpectedOpCodeCapabilities {
  SpvOp opcode;
  CapabilitySet capabilities;
};

using OpcodeTableCapabilitiesTest =
    ::testing::TestWithParam<ExpectedOpCodeCapabilities>;

TEST_P(OpcodeTableCapabilitiesTest, TableEntryMatchesExpectedCapabilities) {
  spv_opcode_table opcodeTable;
  ASSERT_EQ(SPV_SUCCESS,
            spvOpcodeTableGet(&opcodeTable, SPV_ENV_UNIVERSAL_1_1));
  spv_opcode_desc entry;
  ASSERT_EQ(SPV_SUCCESS,
            spvOpcodeTableValueLookup(opcodeTable, GetParam().opcode, &entry));
  EXPECT_EQ(ElementsIn(GetParam().capabilities),
            ElementsIn(entry->capabilities));
}

INSTANTIATE_TEST_CASE_P(
    TableRowTest, OpcodeTableCapabilitiesTest,
    // Spot-check a few opcodes.
    ::testing::Values(
        ExpectedOpCodeCapabilities{
            SpvOpImageQuerySize,
            CapabilitySet{SpvCapabilityKernel, SpvCapabilityImageQuery}},
        ExpectedOpCodeCapabilities{
            SpvOpImageQuerySizeLod,
            CapabilitySet{SpvCapabilityKernel, SpvCapabilityImageQuery}},
        ExpectedOpCodeCapabilities{
            SpvOpImageQueryLevels,
            CapabilitySet{SpvCapabilityKernel, SpvCapabilityImageQuery}},
        ExpectedOpCodeCapabilities{
            SpvOpImageQuerySamples,
            CapabilitySet{SpvCapabilityKernel, SpvCapabilityImageQuery}},
        ExpectedOpCodeCapabilities{SpvOpImageSparseSampleImplicitLod,
                                   CapabilitySet{SpvCapabilitySparseResidency}},
        ExpectedOpCodeCapabilities{SpvOpCopyMemorySized,
                                   CapabilitySet{SpvCapabilityAddresses}},
        ExpectedOpCodeCapabilities{SpvOpArrayLength,
                                   CapabilitySet{SpvCapabilityShader}},
        ExpectedOpCodeCapabilities{SpvOpFunction, CapabilitySet()},
        ExpectedOpCodeCapabilities{SpvOpConvertFToS, CapabilitySet()},
        ExpectedOpCodeCapabilities{SpvOpEmitStreamVertex,
                                   CapabilitySet{SpvCapabilityGeometryStreams}},
        ExpectedOpCodeCapabilities{SpvOpTypeNamedBarrier,
                                   CapabilitySet{SpvCapabilityNamedBarrier}},
        ExpectedOpCodeCapabilities{
            SpvOpGetKernelMaxNumSubgroups,
            CapabilitySet{SpvCapabilitySubgroupDispatch}}), );

}  // anonymous namespace
