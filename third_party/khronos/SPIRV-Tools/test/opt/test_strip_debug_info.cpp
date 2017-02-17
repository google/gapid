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

#include "pass_fixture.h"
#include "pass_utils.h"

namespace {

using namespace spvtools;

using StripLineDebugInfoTest = PassTest<::testing::Test>;

TEST_F(StripLineDebugInfoTest, LineNoLine) {
  std::vector<const char*> text = {
      // clang-format off
               "OpCapability Shader",
          "%1 = OpExtInstImport \"GLSL.std.450\"",
               "OpMemoryModel Logical GLSL450",
               "OpEntryPoint Vertex %2 \"main\"",
          "%3 = OpString \"minimal.vert\"",
               "OpNoLine",
               "OpLine %3 10 10",
       "%void = OpTypeVoid",
               "OpLine %3 100 100",
          "%5 = OpTypeFunction %void",
          "%2 = OpFunction %void None %5",
               "OpLine %3 1 1",
               "OpNoLine",
               "OpLine %3 2 2",
               "OpLine %3 3 3",
          "%6 = OpLabel",
               "OpLine %3 4 4",
               "OpNoLine",
               "OpReturn",
               "OpLine %3 4 4",
               "OpNoLine",
               "OpFunctionEnd",
      // clang-format on
  };
  SinglePassRunAndCheck<opt::StripDebugInfoPass>(JoinAllInsts(text),
                                                 JoinNonDebugInsts(text),
                                                 /* skip_nop = */ false);

  // Let's add more debug instruction before the "OpString" instruction.
  const std::vector<const char*> more_text = {
      "OpSourceContinued \"I'm a happy shader! Yay! ;)\"",
      "OpSourceContinued \"wahahaha\"",
      "OpSource ESSL 310",
      "OpSource ESSL 310",
      "OpSourceContinued \"wahahaha\"",
      "OpSourceContinued \"wahahaha\"",
      "OpSourceExtension \"save-the-world-extension\"",
      "OpName %2 \"main\"",
      "OpModuleProcessed \"42\"",
      "OpModuleProcessed \"43\"",
      "OpModuleProcessed \"44\"",
  };
  text.insert(text.begin() + 4, more_text.cbegin(), more_text.cend());
  SinglePassRunAndCheck<opt::StripDebugInfoPass>(JoinAllInsts(text),
                                                 JoinNonDebugInsts(text),
                                                 /* skip_nop = */ false);
}

using StripDebugInfoTest = PassTest<::testing::TestWithParam<const char*>>;

TEST_P(StripDebugInfoTest, Kind) {
  std::vector<const char*> text = {
      "OpCapability Shader", "OpMemoryModel Logical GLSL450", GetParam(),
  };
  SinglePassRunAndCheck<opt::StripDebugInfoPass>(JoinAllInsts(text),
                                                 JoinNonDebugInsts(text),
                                                 /* skip_nop = */ false);
}

// Test each possible non-line debug instruction.
// clang-format off
INSTANTIATE_TEST_CASE_P(
    SingleKindDebugInst, StripDebugInfoTest,
    ::testing::ValuesIn(std::vector<const char*>({
        "OpSourceContinued \"I'm a happy shader! Yay! ;)\"",
        "OpSource ESSL 310",
        "OpSourceExtension \"save-the-world-extension\"",
        "OpName %main \"main\"",
        "OpMemberName %struct 0 \"field\"",
        "%1 = OpString \"name.vert\"",
        "OpModuleProcessed \"42\"",
    })));
// clang-format on

}  // anonymous namespace
