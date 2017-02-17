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

#include <gmock/gmock.h>

#include "UnitSPIRV.h"

#include "source/spirv_target_env.h"

namespace {

using ::testing::AnyOf;
using ::testing::Eq;
using ::testing::ValuesIn;
using ::testing::StartsWith;

using TargetEnvTest = ::testing::TestWithParam<spv_target_env>;
TEST_P(TargetEnvTest, CreateContext) {
  spv_target_env env = GetParam();
  spv_context context = spvContextCreate(env);
  ASSERT_NE(nullptr, context);
  spvContextDestroy(context); // Avoid leaking
}

TEST_P(TargetEnvTest, ValidDescription) {
  const char* description = spvTargetEnvDescription(GetParam());
  ASSERT_NE(nullptr, description);
  ASSERT_THAT(description, StartsWith("SPIR-V "));
}

TEST_P(TargetEnvTest, ValidSpirvVersion) {
  auto spirv_version = spvVersionForTargetEnv(GetParam());
  ASSERT_THAT(spirv_version, AnyOf(0x10000, 0x10100));
}

INSTANTIATE_TEST_CASE_P(AllTargetEnvs, TargetEnvTest,
                        ValuesIn(spvtest::AllTargetEnvironments()));

TEST(GetContextTest, InvalidTargetEnvProducesNull) {
  spv_context context = spvContextCreate((spv_target_env)10);
  EXPECT_EQ(context, nullptr);
}

// A test case for parsing an environment string.
struct ParseCase {
  const char* input;
  bool success;        // Expect to successfully parse?
  spv_target_env env;  // The parsed environment, if successful.
};

using TargetParseTest = ::testing::TestWithParam<ParseCase>;

TEST_P(TargetParseTest, InvalidTargetEnvProducesNull) {
  spv_target_env env;
  bool parsed = spvParseTargetEnv(GetParam().input, &env);
  EXPECT_THAT(parsed, Eq(GetParam().success));
  EXPECT_THAT(env, Eq(GetParam().env));
}

INSTANTIATE_TEST_CASE_P(TargetParsing, TargetParseTest,
                        ValuesIn(std::vector<ParseCase>{
                            {"spv1.0", true, SPV_ENV_UNIVERSAL_1_0},
                            {"spv1.1", true, SPV_ENV_UNIVERSAL_1_1},
                            {"vulkan1.0", true, SPV_ENV_VULKAN_1_0},
                            {"opencl2.1", true, SPV_ENV_OPENCL_2_1},
                            {"opencl2.2", true, SPV_ENV_OPENCL_2_2},
                            {"opengl4.0", true, SPV_ENV_OPENGL_4_0},
                            {"opengl4.1", true, SPV_ENV_OPENGL_4_1},
                            {"opengl4.2", true, SPV_ENV_OPENGL_4_2},
                            {"opengl4.3", true, SPV_ENV_OPENGL_4_3},
                            {"opengl4.5", true, SPV_ENV_OPENGL_4_5},
                            {nullptr, false, SPV_ENV_UNIVERSAL_1_0},
                            {"", false, SPV_ENV_UNIVERSAL_1_0},
                            {"abc", false, SPV_ENV_UNIVERSAL_1_0},
                        }));

}  // anonymous namespace
