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
#include "gtest/gtest.h"

#include "opt/libspirv.hpp"
#include "opt/module.h"

#include "module_utils.h"

namespace {

using spvtools::ir::Module;
using spvtest::GetIdBound;
using ::testing::Eq;

TEST(ModuleTest, SetIdBound) {
  Module m;
  // It's initialized to 0.
  EXPECT_EQ(0u, GetIdBound(m));

  m.SetIdBound(19);
  EXPECT_EQ(19u, GetIdBound(m));

  m.SetIdBound(102);
  EXPECT_EQ(102u, GetIdBound(m));
}

// Returns a module formed by assembling the given text,
// then loading the result.
std::unique_ptr<Module> BuildModule(std::string text) {
  spvtools::SpvTools t(SPV_ENV_UNIVERSAL_1_1);
  return t.BuildModule(text);
}

TEST(ModuleTest, ComputeIdBound) {
  // Emtpy module case.
  EXPECT_EQ(1u, BuildModule("")->ComputeIdBound());
  // Sensitive to result id
  EXPECT_EQ(2u, BuildModule("%void = OpTypeVoid")->ComputeIdBound());
  // Sensitive to type id
  EXPECT_EQ(1000u, BuildModule("%a = OpTypeArray !999 3")->ComputeIdBound());
  // Sensitive to a regular Id parameter
  EXPECT_EQ(2000u, BuildModule("OpDecorate !1999 0")->ComputeIdBound());
  // Sensitive to a scope Id parameter.
  EXPECT_EQ(3000u,
            BuildModule("%f = OpFunction %void None %fntype %a = OpLabel "
                        "OpMemoryBarrier !2999 %b\n")
                ->ComputeIdBound());
  // Sensitive to a semantics Id parameter
  EXPECT_EQ(4000u,
            BuildModule("%f = OpFunction %void None %fntype %a = OpLabel "
                        "OpMemoryBarrier %b !3999\n")
                ->ComputeIdBound());
}

}  // anonymous namespace

