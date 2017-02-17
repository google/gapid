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

// A pass turning all none debug line instructions into Nop.
class NopifyPass : public opt::Pass {
 public:
  const char* name() const override { return "NopifyPass"; }
  bool Process(ir::Module* module) override {
    module->ForEachInst([](ir::Instruction* inst) { inst->ToNop(); },
                        /* run_on_debug_line_insts = */ false);
    return true;
  }
};

using PassTestForLineDebugInfo = PassTest<::testing::Test>;

// This test's purpose to show our implementation choice: line debug info is
// preserved even if the following instruction is killed. It serves as a guard
// of potential behavior changes.
TEST_F(PassTestForLineDebugInfo, KeepLineDebugInfo) {
  // clang-format off
  const char* text =
               "OpCapability Shader "
          "%1 = OpExtInstImport \"GLSL.std.450\" "
               "OpMemoryModel Logical GLSL450 "
               "OpEntryPoint Vertex %2 \"main\" "
          "%3 = OpString \"minimal.vert\" "
               "OpNoLine "
               "OpLine %3 10 10 "
       "%void = OpTypeVoid "
               "OpLine %3 100 100 "
          "%5 = OpTypeFunction %void "
          "%2 = OpFunction %void None %5 "
               "OpLine %3 1 1 "
               "OpNoLine "
               "OpLine %3 2 2 "
               "OpLine %3 3 3 "
          "%6 = OpLabel "
               "OpLine %3 4 4 "
               "OpNoLine "
               "OpReturn "
               "OpLine %3 4 4 "
               "OpNoLine "
               "OpFunctionEnd ";
  // clang-format on

  const char* result_keep_nop =
      "OpNop\n"
      "OpNop\n"
      "OpNop\n"
      "OpNop\n"
      "OpNop\n"
      "OpNoLine\n"
      "OpLine %3 10 10\n"
      "OpNop\n"
      "OpLine %3 100 100\n"
      "OpNop\n"
      "OpNop\n"
      "OpLine %3 1 1\n"
      "OpNoLine\n"
      "OpLine %3 2 2\n"
      "OpLine %3 3 3\n"
      "OpNop\n"
      "OpLine %3 4 4\n"
      "OpNoLine\n"
      "OpNop\n"
      "OpLine %3 4 4\n"
      "OpNoLine\n"
      "OpNop\n";
  SinglePassRunAndCheck<NopifyPass>(text, result_keep_nop,
                                    /* skip_nop = */ false);
  const char* result_skip_nop =
      "OpNoLine\n"
      "OpLine %3 10 10\n"
      "OpLine %3 100 100\n"
      "OpLine %3 1 1\n"
      "OpNoLine\n"
      "OpLine %3 2 2\n"
      "OpLine %3 3 3\n"
      "OpLine %3 4 4\n"
      "OpNoLine\n"
      "OpLine %3 4 4\n"
      "OpNoLine\n";
  SinglePassRunAndCheck<NopifyPass>(text, result_skip_nop,
                                    /* skip_nop = */ true);
}

}  // anonymous namespace
