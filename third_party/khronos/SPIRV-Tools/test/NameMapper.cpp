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

#include "gmock/gmock.h"

#include "TestFixture.h"
#include "UnitSPIRV.h"

#include "source/name_mapper.h"

using libspirv::NameMapper;
using libspirv::FriendlyNameMapper;
using spvtest::ScopedContext;
using ::testing::Eq;

namespace {

TEST(TrivialNameTest, Samples) {
  auto mapper = libspirv::GetTrivialNameMapper();
  EXPECT_EQ(mapper(1), "1");
  EXPECT_EQ(mapper(1999), "1999");
  EXPECT_EQ(mapper(1024), "1024");
}

// A test case for the name mappers that actually look at an assembled module.
struct NameIdCase {
  std::string assembly;  // Input assembly text
  uint32_t id;
  std::string expected_name;
};

using FriendlyNameTest =
    spvtest::TextToBinaryTestBase<::testing::TestWithParam<NameIdCase>>;

TEST_P(FriendlyNameTest, SingleMapping) {
  ScopedContext context(SPV_ENV_UNIVERSAL_1_1);
  auto words = CompileSuccessfully(GetParam().assembly, SPV_ENV_UNIVERSAL_1_1);
  auto friendly_mapper =
      FriendlyNameMapper(context.context, words.data(), words.size());
  NameMapper mapper = friendly_mapper.GetNameMapper();
  EXPECT_THAT(mapper(GetParam().id), Eq(GetParam().expected_name))
      << GetParam().assembly << std::endl
      << " for id " << GetParam().id;
}

INSTANTIATE_TEST_CASE_P(ScalarType, FriendlyNameTest,
                        ::testing::ValuesIn(std::vector<NameIdCase>{
                            {"%1 = OpTypeVoid", 1, "void"},
                            {"%1 = OpTypeBool", 1, "bool"},
                            {"%1 = OpTypeInt 8 0", 1, "uchar"},
                            {"%1 = OpTypeInt 8 1", 1, "char"},
                            {"%1 = OpTypeInt 16 0", 1, "ushort"},
                            {"%1 = OpTypeInt 16 1", 1, "short"},
                            {"%1 = OpTypeInt 32 0", 1, "uint"},
                            {"%1 = OpTypeInt 32 1", 1, "int"},
                            {"%1 = OpTypeInt 64 0", 1, "ulong"},
                            {"%1 = OpTypeInt 64 1", 1, "long"},
                            {"%1 = OpTypeInt 1 0", 1, "u1"},
                            {"%1 = OpTypeInt 1 1", 1, "i1"},
                            {"%1 = OpTypeInt 33 0", 1, "u33"},
                            {"%1 = OpTypeInt 33 1", 1, "i33"},

                            {"%1 = OpTypeFloat 16", 1, "half"},
                            {"%1 = OpTypeFloat 32", 1, "float"},
                            {"%1 = OpTypeFloat 64", 1, "double"},
                            {"%1 = OpTypeFloat 10", 1, "fp10"},
                            {"%1 = OpTypeFloat 55", 1, "fp55"},
                        }), );

INSTANTIATE_TEST_CASE_P(
    VectorType, FriendlyNameTest,
    ::testing::ValuesIn(std::vector<NameIdCase>{
        {"%1 = OpTypeBool %2 = OpTypeVector %1 1", 2, "v1bool"},
        {"%1 = OpTypeBool %2 = OpTypeVector %1 2", 2, "v2bool"},
        {"%1 = OpTypeBool %2 = OpTypeVector %1 3", 2, "v3bool"},
        {"%1 = OpTypeBool %2 = OpTypeVector %1 4", 2, "v4bool"},

        {"%1 = OpTypeInt 8 0 %2 = OpTypeVector %1 2", 2, "v2uchar"},
        {"%1 = OpTypeInt 16 1 %2 = OpTypeVector %1 3", 2, "v3short"},
        {"%1 = OpTypeInt 32 0 %2 = OpTypeVector %1 4", 2, "v4uint"},
        {"%1 = OpTypeInt 64 1 %2 = OpTypeVector %1 3", 2, "v3long"},
        {"%1 = OpTypeInt 20 0 %2 = OpTypeVector %1 4", 2, "v4u20"},
        {"%1 = OpTypeInt 21 1 %2 = OpTypeVector %1 3", 2, "v3i21"},

        {"%1 = OpTypeFloat 32 %2 = OpTypeVector %1 2", 2, "v2float"},
        // OpName overrides the element name.
        {"OpName %1 \"time\" %1 = OpTypeFloat 32 %2 = OpTypeVector %1 2", 2,
         "v2time"},
    }), );

INSTANTIATE_TEST_CASE_P(
    MatrixType, FriendlyNameTest,
    ::testing::ValuesIn(std::vector<NameIdCase>{
        {"%1 = OpTypeBool %2 = OpTypeVector %1 2 %3 = OpTypeMatrix %2 2", 3,
         "mat2v2bool"},
        {"%1 = OpTypeFloat 32 %2 = OpTypeVector %1 2 %3 = OpTypeMatrix %2 3", 3,
         "mat3v2float"},
        {"%1 = OpTypeFloat 32 %2 = OpTypeVector %1 2 %3 = OpTypeMatrix %2 4", 3,
         "mat4v2float"},
        {"OpName %1 \"time\" %1 = OpTypeFloat 32 %2 = OpTypeVector %1 2 %3 = "
         "OpTypeMatrix %2 4",
         3, "mat4v2time"},
        {"OpName %2 \"lat_long\" %1 = OpTypeFloat 32 %2 = OpTypeVector %1 2 %3 "
         "= OpTypeMatrix %2 4",
         3, "mat4lat_long"},
    }), );

INSTANTIATE_TEST_CASE_P(
    OpName, FriendlyNameTest,
    ::testing::ValuesIn(std::vector<NameIdCase>{
        {"OpName %1 \"abcdefg\"", 1, "abcdefg"},
        {"OpName %1 \"Hello world!\"", 1, "Hello_world_"},
        {"OpName %1 \"0123456789\"", 1, "0123456789"},
        {"OpName %1 \"_\"", 1, "_"},
        // An empty string is not valid for SPIR-V assembly IDs.
        {"OpName %1 \"\"", 1, "_"},
        // Test uniqueness when presented with things mapping to "_"
        {"OpName %1 \"\" OpName %2 \"\"", 1, "_"},
        {"OpName %1 \"\" OpName %2 \"\"", 2, "__0"},
        {"OpName %1 \"\" OpName %2 \"\" OpName %3 \"_\"", 3, "__1"},
        // Test uniqueness of names that are forced to be
        // numbers.
        {"OpName %1 \"2\" OpName %2 \"2\"", 1, "2"},
        {"OpName %1 \"2\" OpName %2 \"2\"", 2, "2_0"},
        // Test uniqueness in the face of forward references
        // for Ids that don't already have friendly names.
        // In particular, the first OpDecorate assigns the name, and
        // the second one can't override it.
        {"OpDecorate %1 Volatile OpDecorate %1 Restrict", 1, "1"},
        // But a forced name can override the name that
        // would have been assigned via the OpDecorate
        // forward reference.
        {"OpName %1 \"mememe\" OpDecorate %1 Volatile OpDecorate %1 Restrict",
         1, "mememe"},
        // OpName can override other inferences.  We assume valid instruction
        // ordering, where OpName precedes type definitions.
        {"OpName %1 \"myfloat\" %1 = OpTypeFloat 32", 1, "myfloat"},
    }), );

INSTANTIATE_TEST_CASE_P(
    UniquenessHeuristic, FriendlyNameTest,
    ::testing::ValuesIn(std::vector<NameIdCase>{
        {"%1 = OpTypeVoid %2 = OpTypeVoid %3 = OpTypeVoid", 1, "void"},
        {"%1 = OpTypeVoid %2 = OpTypeVoid %3 = OpTypeVoid", 2, "void_0"},
        {"%1 = OpTypeVoid %2 = OpTypeVoid %3 = OpTypeVoid", 3, "void_1"},
    }), );

INSTANTIATE_TEST_CASE_P(Arrays, FriendlyNameTest,
                        ::testing::ValuesIn(std::vector<NameIdCase>{
                            {"OpName %2 \"FortyTwo\" %1 = OpTypeFloat 32 "
                             "%2 = OpConstant %1 42 %3 = OpTypeArray %1 %2",
                             3, "_arr_float_FortyTwo"},
                            {"%1 = OpTypeInt 32 0 "
                             "%2 = OpTypeRuntimeArray %1",
                             2, "_runtimearr_uint"},
                        }), );

INSTANTIATE_TEST_CASE_P(Structs, FriendlyNameTest,
                        ::testing::ValuesIn(std::vector<NameIdCase>{
                            {"%1 = OpTypeBool "
                             "%2 = OpTypeStruct %1 %1 %1",
                             2, "_struct_2"},
                            {"%1 = OpTypeBool "
                             "%2 = OpTypeStruct %1 %1 %1 "
                             "%3 = OpTypeStruct %2 %2",
                             3, "_struct_3"},
                        }), );

INSTANTIATE_TEST_CASE_P(
    Pointer, FriendlyNameTest,
    ::testing::ValuesIn(std::vector<NameIdCase>{
        {"%1 = OpTypeFloat 32 %2 = OpTypePointer Workgroup %1", 2,
         "_ptr_Workgroup_float"},
        {"%1 = OpTypeBool %2 = OpTypePointer Private %1", 2,
         "_ptr_Private_bool"},
        // OpTypeForwardPointer doesn't force generation of the name for its
        // target type.
        {"%1 = OpTypeBool OpTypeForwardPointer %2 Private %2 = OpTypePointer "
         "Private %1",
         2, "_ptr_Private_bool"},
    }), );

INSTANTIATE_TEST_CASE_P(ExoticTypes, FriendlyNameTest,
                        ::testing::ValuesIn(std::vector<NameIdCase>{
                            {"%1 = OpTypeEvent", 1, "Event"},
                            {"%1 = OpTypeDeviceEvent", 1, "DeviceEvent"},
                            {"%1 = OpTypeReserveId", 1, "ReserveId"},
                            {"%1 = OpTypeQueue", 1, "Queue"},
                            {"%1 = OpTypeOpaque \"hello world!\"", 1, "Opaque_hello_world_"},
                            {"%1 = OpTypePipe ReadOnly", 1, "PipeReadOnly"},
                            {"%1 = OpTypePipe WriteOnly", 1, "PipeWriteOnly"},
                            {"%1 = OpTypePipe ReadWrite", 1, "PipeReadWrite"},
                            {"%1 = OpTypePipeStorage", 1, "PipeStorage"},
                            {"%1 = OpTypeNamedBarrier", 1, "NamedBarrier"},
                        }), );

}  // anonymous namespace
