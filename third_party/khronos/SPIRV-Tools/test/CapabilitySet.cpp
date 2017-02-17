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

#include <vector>
#include "gmock/gmock.h"

#include "enum_set.h"
#include "UnitSPIRV.h"

namespace {

using libspirv::CapabilitySet;
using spvtest::ElementsIn;
using ::testing::Eq;
using ::testing::ValuesIn;

TEST(CapabilitySet, DefaultIsEmpty) {
  CapabilitySet c;
  for (uint32_t i = 0; i < 1000; ++i) {
    EXPECT_FALSE(c.Contains(i));
    EXPECT_FALSE(c.Contains(static_cast<SpvCapability>(i)));
  }
}

TEST(CapabilitySet, ConstructSingleMemberMatrix) {
  CapabilitySet s(SpvCapabilityMatrix);
  EXPECT_TRUE(s.Contains(SpvCapabilityMatrix));
  EXPECT_FALSE(s.Contains(SpvCapabilityShader));
  EXPECT_FALSE(s.Contains(1000));
}

TEST(CapabilitySet, ConstructSingleMemberMaxInMask) {
  CapabilitySet s(static_cast<SpvCapability>(63));
  EXPECT_FALSE(s.Contains(SpvCapabilityMatrix));
  EXPECT_FALSE(s.Contains(SpvCapabilityShader));
  EXPECT_TRUE(s.Contains(63));
  EXPECT_FALSE(s.Contains(64));
  EXPECT_FALSE(s.Contains(1000));
}

TEST(CapabilitySet, ConstructSingleMemberMinOverflow) {
  // Check the first one that forces overflow beyond the mask.
  CapabilitySet s(static_cast<SpvCapability>(64));
  EXPECT_FALSE(s.Contains(SpvCapabilityMatrix));
  EXPECT_FALSE(s.Contains(SpvCapabilityShader));
  EXPECT_FALSE(s.Contains(63));
  EXPECT_TRUE(s.Contains(64));
  EXPECT_FALSE(s.Contains(1000));
}

TEST(CapabilitySet, ConstructSingleMemberMaxOverflow) {
  // Check the max 32-bit signed int.
  CapabilitySet s(SpvCapability(0x7fffffffu));
  EXPECT_FALSE(s.Contains(SpvCapabilityMatrix));
  EXPECT_FALSE(s.Contains(SpvCapabilityShader));
  EXPECT_FALSE(s.Contains(1000));
  EXPECT_TRUE(s.Contains(0x7fffffffu));
}

TEST(CapabilitySet, AddEnum) {
  CapabilitySet s(SpvCapabilityShader);
  s.Add(SpvCapabilityKernel);
  EXPECT_FALSE(s.Contains(SpvCapabilityMatrix));
  EXPECT_TRUE(s.Contains(SpvCapabilityShader));
  EXPECT_TRUE(s.Contains(SpvCapabilityKernel));
}

TEST(CapabilitySet, AddInt) {
  CapabilitySet s(SpvCapabilityShader);
  s.Add(42);
  EXPECT_FALSE(s.Contains(SpvCapabilityMatrix));
  EXPECT_TRUE(s.Contains(SpvCapabilityShader));
  EXPECT_TRUE(s.Contains(42));
  EXPECT_TRUE(s.Contains(static_cast<SpvCapability>(42)));
}

TEST(CapabilitySet, InitializerListEmpty) {
  CapabilitySet s{};
  for (uint32_t i = 0; i < 1000; i++) {
    EXPECT_FALSE(s.Contains(i));
  }
}

struct ForEachCase {
  CapabilitySet capabilities;
  std::vector<SpvCapability> expected;
};

using CapabilitySetForEachTest = ::testing::TestWithParam<ForEachCase>;

TEST_P(CapabilitySetForEachTest, CallsAsExpected) {
  EXPECT_THAT(ElementsIn(GetParam().capabilities), Eq(GetParam().expected));
}

TEST_P(CapabilitySetForEachTest, CopyConstructor) {
  CapabilitySet copy(GetParam().capabilities);
  EXPECT_THAT(ElementsIn(copy), Eq(GetParam().expected));
}

TEST_P(CapabilitySetForEachTest, MoveConstructor) {
  // We need a writable copy to move from.
  CapabilitySet copy(GetParam().capabilities);
  CapabilitySet moved(std::move(copy));
  EXPECT_THAT(ElementsIn(moved), Eq(GetParam().expected));

  // The moved-from set is empty.
  EXPECT_THAT(ElementsIn(copy), Eq(std::vector<SpvCapability>{}));
}

TEST_P(CapabilitySetForEachTest, OperatorEquals) {
  CapabilitySet assigned = GetParam().capabilities;
  EXPECT_THAT(ElementsIn(assigned), Eq(GetParam().expected));
}

TEST_P(CapabilitySetForEachTest, OperatorEqualsSelfAssign) {
  CapabilitySet assigned{GetParam().capabilities};
  assigned = assigned;
  EXPECT_THAT(ElementsIn(assigned), Eq(GetParam().expected));
}

INSTANTIATE_TEST_CASE_P(Samples, CapabilitySetForEachTest,
                        ValuesIn(std::vector<ForEachCase>{
                            {{}, {}},
                            {{SpvCapabilityMatrix}, {SpvCapabilityMatrix}},
                            {{SpvCapabilityKernel, SpvCapabilityShader},
                             {SpvCapabilityShader, SpvCapabilityKernel}},
                            {{static_cast<SpvCapability>(999)},
                             {static_cast<SpvCapability>(999)}},
                            {{static_cast<SpvCapability>(0x7fffffff)},
                             {static_cast<SpvCapability>(0x7fffffff)}},
                            // Mixture and out of order
                            {{static_cast<SpvCapability>(0x7fffffff),
                              static_cast<SpvCapability>(100),
                              SpvCapabilityShader, SpvCapabilityMatrix},
                             {SpvCapabilityMatrix, SpvCapabilityShader,
                              static_cast<SpvCapability>(100),
                              static_cast<SpvCapability>(0x7fffffff)}},
                        }), );

}  // anonymous namespace
