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

#include <memory>
#include <unordered_set>

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include "opt/def_use_manager.h"
#include "opt/libspirv.hpp"
#include "pass_utils.h"

namespace {

using ::testing::ElementsAre;

using namespace spvtools;
using spvtools::opt::analysis::DefUseManager;

// Disassembles the given |inst| and returns the disassembly.
std::string DisassembleInst(ir::Instruction* inst) {
  SpvTools tools(SPV_ENV_UNIVERSAL_1_1);

  std::vector<uint32_t> binary;
  // We need this to generate the necessary header in the binary.
  tools.Assemble("", &binary);
  inst->ToBinaryWithoutAttachedDebugInsts(&binary);

  std::string text;
  // We'll need to check the underlying id numbers.
  // So turn off friendly names for ids.
  tools.Disassemble(binary, &text, SPV_BINARY_TO_TEXT_OPTION_NO_HEADER);
  while (!text.empty() && text.back() == '\n') text.pop_back();
  return text;
}

// A struct for holding expected id defs and uses.
struct InstDefUse {
  using IdInstPair = std::pair<uint32_t, const char*>;
  using IdInstsPair = std::pair<uint32_t, std::vector<const char*>>;

  // Ids and their corresponding def instructions.
  std::vector<IdInstPair> defs;
  // Ids and their corresponding use instructions.
  std::vector<IdInstsPair> uses;
};

// Checks that the |actual_defs| and |actual_uses| are in accord with
// |expected_defs_uses|.
void CheckDef(const InstDefUse& expected_defs_uses,
              const DefUseManager::IdToDefMap& actual_defs) {
  // Check defs.
  ASSERT_EQ(expected_defs_uses.defs.size(), actual_defs.size());
  for (uint32_t i = 0; i < expected_defs_uses.defs.size(); ++i) {
    const auto id = expected_defs_uses.defs[i].first;
    const auto expected_def = expected_defs_uses.defs[i].second;
    ASSERT_EQ(1u, actual_defs.count(id)) << "expected to def id [" << id << "]";
    EXPECT_EQ(expected_def, DisassembleInst(actual_defs.at(id)));
  }
}

void CheckUse(const InstDefUse& expected_defs_uses,
              const DefUseManager::IdToUsesMap& actual_uses) {
  // Check uses.
  ASSERT_EQ(expected_defs_uses.uses.size(), actual_uses.size());
  for (uint32_t i = 0; i < expected_defs_uses.uses.size(); ++i) {
    const auto id = expected_defs_uses.uses[i].first;
    const auto& expected_uses = expected_defs_uses.uses[i].second;

    ASSERT_EQ(1u, actual_uses.count(id)) << "expected to use id [" << id << "]";
    const auto& uses = actual_uses.at(id);

    ASSERT_EQ(expected_uses.size(), uses.size())
        << "id [" << id << "] # uses: expected: " << expected_uses.size()
        << " actual: " << uses.size();
    auto it = uses.cbegin();
    for (const auto expected_use : expected_uses) {
      EXPECT_EQ(expected_use, DisassembleInst(it->inst))
          << "id [" << id << "] use instruction mismatch";
      ++it;
    }
  }
}

// The following test case mimics how LLVM handles induction variables.
// But, yeah, it's not very readable. However, we only care about the id
// defs and uses. So, no need to make sure this is valid OpPhi construct.
const char kOpPhiTestFunction[] =
    " %2 = OpFunction %1 None %3 "
    " %4 = OpLabel "
    "      OpBranch %5 "

    " %5 = OpLabel "
    " %7 = OpPhi %6 %8 %4 %9 %5 "
    "%11 = OpPhi %10 %12 %4 %13 %5 "
    " %9 = OpIAdd %6 %7 %14 "
    "%13 = OpFAdd %10 %11 %15 "
    "%17 = OpSLessThan %16 %7 %18 "
    "      OpLoopMerge %19 %5 None "
    "      OpBranchConditional %17 %5 %19 "

    "%19 = OpLabel "
    "      OpReturn "
    "      OpFunctionEnd";

struct ParseDefUseCase {
  const char* text;
  InstDefUse du;
};

using ParseDefUseTest = ::testing::TestWithParam<ParseDefUseCase>;

TEST_P(ParseDefUseTest, Case) {
  const auto& tc = GetParam();

  // Build module.
  const std::vector<const char*> text = {tc.text};
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(JoinAllInsts(text));
  ASSERT_NE(nullptr, module);

  // Analyze def and use.
  opt::analysis::DefUseManager manager(module.get());

  CheckDef(tc.du, manager.id_to_defs());
  CheckUse(tc.du, manager.id_to_uses());
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, ParseDefUseTest,
    ::testing::ValuesIn(std::vector<ParseDefUseCase>{
        {"", {{}, {}}},                              // no instruction
        {"OpMemoryModel Logical GLSL450", {{}, {}}}, // no def and use
        { // single def, no use
          "%1 = OpString \"wow\"",
          {
            {{1, "%1 = OpString \"wow\""}}, // defs
            {}                              // uses
          }
        },
        { // multiple def, no use
          "%1 = OpString \"hello\" "
          "%2 = OpString \"world\" "
          "%3 = OpTypeVoid",
          {
            {  // defs
              {1, "%1 = OpString \"hello\""},
              {2, "%2 = OpString \"world\""},
              {3, "%3 = OpTypeVoid"},
            },
            {} // uses
          }
        },
        { // single use, no def
          "OpTypeForwardPointer %1 Input",
          {
            {}, // defs
            {   // uses
              {1, {"OpTypeForwardPointer %1 Input"}},
            }
          }
        },
        { // multiple use, no def
          "OpEntryPoint Fragment %1 \"main\" "
          "OpTypeForwardPointer %2 Input "
          "OpTypeForwardPointer %3 Output",
          {
            {}, // defs
            {   // uses
              {1, {"OpEntryPoint Fragment %1 \"main\""}},
              {2, {"OpTypeForwardPointer %2 Input"}},
              {3, {"OpTypeForwardPointer %3 Output"}},
            }
          }
        },
        { // multiple def, multiple use
          "%1 = OpTypeBool "
          "%2 = OpTypeVector %1 3 "
          "%3 = OpTypeMatrix %2 3",
          {
            { // defs
              {1, "%1 = OpTypeBool"},
              {2, "%2 = OpTypeVector %1 3"},
              {3, "%3 = OpTypeMatrix %2 3"},
            },
            { // uses
              {1, {"%2 = OpTypeVector %1 3"}},
              {2, {"%3 = OpTypeMatrix %2 3"}},
            }
          }
        },
        { // multiple use of the same id
          "%1 = OpTypeBool "
          "%2 = OpTypeVector %1 2 "
          "%3 = OpTypeVector %1 3 "
          "%4 = OpTypeVector %1 4",
          {
            { // defs
              {1, "%1 = OpTypeBool"},
              {2, "%2 = OpTypeVector %1 2"},
              {3, "%3 = OpTypeVector %1 3"},
              {4, "%4 = OpTypeVector %1 4"},
            },
            { // uses
              {1,
                {
                  "%2 = OpTypeVector %1 2",
                  "%3 = OpTypeVector %1 3",
                  "%4 = OpTypeVector %1 4",
                }
              },
            }
          }
        },
        { // labels
          "%2 = OpFunction %1 None %3 "

          "%4 = OpLabel "
          "OpBranchConditional %5 %6 %7 "

          "%6 = OpLabel "
          "OpBranch %7 "

          "%7 = OpLabel "
          "OpReturn "

          "OpFunctionEnd",
          {
            { // defs
              {2, "%2 = OpFunction %1 None %3"},
              {4, "%4 = OpLabel"},
              {6, "%6 = OpLabel"},
              {7, "%7 = OpLabel"},
            },
            { // uses
              {1, {"%2 = OpFunction %1 None %3"}},
              {3, {"%2 = OpFunction %1 None %3"}},
              {5, {"OpBranchConditional %5 %6 %7"}},
              {6, {"OpBranchConditional %5 %6 %7"}},
              {7,
                {
                  "OpBranchConditional %5 %6 %7",
                  "OpBranch %7",
                }
              },
            }
          }
        },
        { // cross function
          "%1 = OpTypeBool "

          "%2 = OpFunction %1 None %3 "

          "%4 = OpLabel "
          "%5 = OpVariable %1 Function "
          "%6 = OpFunctionCall %1 %2 %5 "
          "OpReturnValue %6 "

          "OpFunctionEnd",
          {
            { // defs
              {1, "%1 = OpTypeBool"},
              {2, "%2 = OpFunction %1 None %3"},
              {4, "%4 = OpLabel"},
              {5, "%5 = OpVariable %1 Function"},
              {6, "%6 = OpFunctionCall %1 %2 %5"},
            },
            { // uses
              {1,
                {
                  "%2 = OpFunction %1 None %3",
                  "%5 = OpVariable %1 Function",
                  "%6 = OpFunctionCall %1 %2 %5",
                }
              },
              {2, {"%6 = OpFunctionCall %1 %2 %5"}},
              {5, {"%6 = OpFunctionCall %1 %2 %5"}},
              {3, {"%2 = OpFunction %1 None %3"}},
              {6, {"OpReturnValue %6"}},
            }
          }
        },
        { // selection merge and loop merge
          "%2 = OpFunction %1 None %3 "

          "%4 = OpLabel "
          "OpLoopMerge %5 %4 None "
          "OpBranch %6 "

          "%5 = OpLabel "
          "OpReturn "

          "%6 = OpLabel "
          "OpSelectionMerge %7 None "
          "OpBranchConditional %8 %9 %7 "

          "%7 = OpLabel "
          "OpReturn "

          "%9 = OpLabel "
          "OpReturn "

          "OpFunctionEnd",
          {
            { // defs
              {2, "%2 = OpFunction %1 None %3"},
              {4, "%4 = OpLabel"},
              {5, "%5 = OpLabel"},
              {6, "%6 = OpLabel"},
              {7, "%7 = OpLabel"},
              {9, "%9 = OpLabel"},
            },
            { // uses
              {1, {"%2 = OpFunction %1 None %3"}},
              {3, {"%2 = OpFunction %1 None %3"}},
              {4, {"OpLoopMerge %5 %4 None"}},
              {5, {"OpLoopMerge %5 %4 None"}},
              {6, {"OpBranch %6"}},
              {7,
                {
                  "OpSelectionMerge %7 None",
                  "OpBranchConditional %8 %9 %7",
                }
              },
              {8, {"OpBranchConditional %8 %9 %7"}},
              {9, {"OpBranchConditional %8 %9 %7"}},
            }
          }
        },
        { // Forward reference
          "OpDecorate %1 Block "
          "OpTypeForwardPointer %2 Input "
          "%3 = OpTypeInt 32 0 "
          "%1 = OpTypeStruct %3 "
          "%2 = OpTypePointer Input %3",
          {
            { // defs
              {1, "%1 = OpTypeStruct %3"},
              {2, "%2 = OpTypePointer Input %3"},
              {3, "%3 = OpTypeInt 32 0"},
            },
            { // uses
              {1, {"OpDecorate %1 Block"}},
              {2, {"OpTypeForwardPointer %2 Input"}},
              {3,
                {
                  "%1 = OpTypeStruct %3",
                  "%2 = OpTypePointer Input %3",
                }
              }
            },
          },
        },
        { // OpPhi
          kOpPhiTestFunction,
          {
            { // defs
              {2, "%2 = OpFunction %1 None %3"},
              {4, "%4 = OpLabel"},
              {5, "%5 = OpLabel"},
              {7, "%7 = OpPhi %6 %8 %4 %9 %5"},
              {9, "%9 = OpIAdd %6 %7 %14"},
              {11, "%11 = OpPhi %10 %12 %4 %13 %5"},
              {13, "%13 = OpFAdd %10 %11 %15"},
              {17, "%17 = OpSLessThan %16 %7 %18"},
              {19, "%19 = OpLabel"},
            },
            { // uses
              {1, {"%2 = OpFunction %1 None %3"}},
              {3, {"%2 = OpFunction %1 None %3"}},
              {4,
                {
                  "%7 = OpPhi %6 %8 %4 %9 %5",
                  "%11 = OpPhi %10 %12 %4 %13 %5",
                }
              },
              {5,
                {
                  "OpBranch %5",
                  "%7 = OpPhi %6 %8 %4 %9 %5",
                  "%11 = OpPhi %10 %12 %4 %13 %5",
                  "OpLoopMerge %19 %5 None",
                  "OpBranchConditional %17 %5 %19",
                }
              },
              {6,
                {
                  "%7 = OpPhi %6 %8 %4 %9 %5",
                  "%9 = OpIAdd %6 %7 %14",
                }
              },
              {7,
                {
                  "%9 = OpIAdd %6 %7 %14",
                  "%17 = OpSLessThan %16 %7 %18",
                }
              },
              {8, {"%7 = OpPhi %6 %8 %4 %9 %5"}},
              {9, {"%7 = OpPhi %6 %8 %4 %9 %5"}},
              {10,
                {
                  "%11 = OpPhi %10 %12 %4 %13 %5",
                  "%13 = OpFAdd %10 %11 %15",
                }
              },
              {11, {"%13 = OpFAdd %10 %11 %15"}},
              {12, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
              {13, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
              {14, {"%9 = OpIAdd %6 %7 %14"}},
              {15, {"%13 = OpFAdd %10 %11 %15"}},
              {16, {"%17 = OpSLessThan %16 %7 %18"}},
              {17, {"OpBranchConditional %17 %5 %19"}},
              {18, {"%17 = OpSLessThan %16 %7 %18"}},
              {19,
                {
                  "OpLoopMerge %19 %5 None",
                  "OpBranchConditional %17 %5 %19",
                }
              },
            },
          },
        },
        { // OpPhi defining and referencing the same id.
          "%1 = OpTypeBool "
          "%2 = OpConstantTrue %1 "

          "%4 = OpFunction %3 None %5 "
          "%6 = OpLabel "
          "     OpBranch %7 "
          "%7 = OpLabel "
          "%8 = OpPhi %1   %8 %7   %2 %6 " // both defines and uses %8
          "     OpBranch %7 "
          "     OpFunctionEnd",
          {
            { // defs
              {1, "%1 = OpTypeBool"},
              {2, "%2 = OpConstantTrue %1"},
              {4, "%4 = OpFunction %3 None %5"},
              {6, "%6 = OpLabel"},
              {7, "%7 = OpLabel"},
              {8, "%8 = OpPhi %1 %8 %7 %2 %6"},
            },
            { // uses
              {1,
                {
                  "%2 = OpConstantTrue %1",
                  "%8 = OpPhi %1 %8 %7 %2 %6",
                }
              },
              {2, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
              {3, {"%4 = OpFunction %3 None %5"}},
              {5, {"%4 = OpFunction %3 None %5"}},
              {6, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
              {7,
                {
                  "OpBranch %7",
                  "%8 = OpPhi %1 %8 %7 %2 %6",
                  "OpBranch %7",
                }
              },
              {8, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
            },
          },
        },
    })
);
// clang-format on

struct ReplaceUseCase {
  const char* before;
  std::vector<std::pair<uint32_t, uint32_t>> candidates;
  const char* after;
  InstDefUse du;
};

using ReplaceUseTest = ::testing::TestWithParam<ReplaceUseCase>;

// Disassembles the given |module| and returns the disassembly.
std::string DisassembleModule(ir::Module* module) {
  SpvTools tools(SPV_ENV_UNIVERSAL_1_1);

  std::vector<uint32_t> binary;
  module->ToBinary(&binary, /* skip_nop = */ false);

  std::string text;
  // We'll need to check the underlying id numbers.
  // So turn off friendly names for ids.
  tools.Disassemble(binary, &text, SPV_BINARY_TO_TEXT_OPTION_NO_HEADER);
  while (!text.empty() && text.back() == '\n') text.pop_back();
  return text;
}

TEST_P(ReplaceUseTest, Case) {
  const auto& tc = GetParam();

  // Build module.
  const std::vector<const char*> text = {tc.before};
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(JoinAllInsts(text));
  ASSERT_NE(nullptr, module);

  // Analyze def and use.
  opt::analysis::DefUseManager manager(module.get());

  // Do the substitution.
  for (const auto& candiate : tc.candidates) {
    manager.ReplaceAllUsesWith(candiate.first, candiate.second);
  }

  EXPECT_EQ(tc.after, DisassembleModule(module.get()));
  CheckDef(tc.du, manager.id_to_defs());
  CheckUse(tc.du, manager.id_to_uses());
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, ReplaceUseTest,
    ::testing::ValuesIn(std::vector<ReplaceUseCase>{
      { // no use, no replace request
        "", {}, "", {},
      },
      { // no use, some replace requests
        "OpMemoryModel Logical GLSL450",
        {{1, 2}, {3, 4}, {7, 8}, {7, 9}, {7, 10}, {2, 10}, {3, 10}},
        "OpMemoryModel Logical GLSL450",
        {},
      },
      { // replace one use
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 3",
        {{1, 3}},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %3 3",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %3 3"},
          },
          { // uses
            {3, {"%2 = OpTypeVector %3 3"}},
          },
        },
      },
      { // replace and then replace back
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 3",
        {{1, 3}, {3, 1}},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %1 3",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %1 3"},
          },
          { // uses
            {1, {"%2 = OpTypeVector %1 3"}},
          },
        },
      },
      { // replace with the same id
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 3",
        {{1, 1}, {2, 2}, {3, 3}},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %1 3",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %1 3"},
          },
          { // uses
            {1, {"%2 = OpTypeVector %1 3"}},
          },
        },
      },
      { // replace in sequence
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 3",
        {{1, 3}, {3, 4}, {4, 5}, {5, 100}},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %100 3",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %100 3"},
          },
          { // uses
            {100, {"%2 = OpTypeVector %100 3"}},
          },
        },
      },
      { // replace multiple uses
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 2 "
        "%3 = OpTypeVector %1 3 "
        "%4 = OpTypeVector %1 4 "
        "%5 = OpTypeMatrix %2 2 "
        "%6 = OpTypeMatrix %3 3 "
        "%7 = OpTypeMatrix %4 4",
        {{1, 10}, {2, 20}, {4, 40}},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %10 2\n"
        "%3 = OpTypeVector %10 3\n"
        "%4 = OpTypeVector %10 4\n"
        "%5 = OpTypeMatrix %20 2\n"
        "%6 = OpTypeMatrix %3 3\n"
        "%7 = OpTypeMatrix %40 4",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %10 2"},
            {3, "%3 = OpTypeVector %10 3"},
            {4, "%4 = OpTypeVector %10 4"},
            {5, "%5 = OpTypeMatrix %20 2"},
            {6, "%6 = OpTypeMatrix %3 3"},
            {7, "%7 = OpTypeMatrix %40 4"},
          },
          { // uses
            {10,
              {
                "%2 = OpTypeVector %10 2",
                "%3 = OpTypeVector %10 3",
                "%4 = OpTypeVector %10 4",
              }
            },
            {20, {"%5 = OpTypeMatrix %20 2"}},
            {3, {"%6 = OpTypeMatrix %3 3"}},
            {40, {"%7 = OpTypeMatrix %40 4"}},
          },
        },
      },
      { // OpPhi.
        kOpPhiTestFunction,
        // replace one id used by OpPhi, replace one id generated by OpPhi
        {{9, 9000}, {11, 9}},
         "%2 = OpFunction %1 None %3\n"
         "%4 = OpLabel\n"
               "OpBranch %5\n"

         "%5 = OpLabel\n"
         "%7 = OpPhi %6 %8 %4 %9000 %5\n" // %9 -> %9000
        "%11 = OpPhi %10 %12 %4 %13 %5\n"
         "%9 = OpIAdd %6 %7 %14\n"
        "%13 = OpFAdd %10 %9 %15\n"       // %11 -> %9
        "%17 = OpSLessThan %16 %7 %18\n"
              "OpLoopMerge %19 %5 None\n"
              "OpBranchConditional %17 %5 %19\n"

        "%19 = OpLabel\n"
              "OpReturn\n"
              "OpFunctionEnd",
        {
          { // defs.
            {2, "%2 = OpFunction %1 None %3"},
            {4, "%4 = OpLabel"},
            {5, "%5 = OpLabel"},
            {7, "%7 = OpPhi %6 %8 %4 %9000 %5"},
            {9, "%9 = OpIAdd %6 %7 %14"},
            {11, "%11 = OpPhi %10 %12 %4 %13 %5"},
            {13, "%13 = OpFAdd %10 %9 %15"},
            {17, "%17 = OpSLessThan %16 %7 %18"},
            {19, "%19 = OpLabel"},
          },
          { // uses
            {1, {"%2 = OpFunction %1 None %3"}},
            {3, {"%2 = OpFunction %1 None %3"}},
            {4,
              {
                "%7 = OpPhi %6 %8 %4 %9000 %5",
                "%11 = OpPhi %10 %12 %4 %13 %5",
              }
            },
            {5,
              {
                "OpBranch %5",
                "%7 = OpPhi %6 %8 %4 %9000 %5",
                "%11 = OpPhi %10 %12 %4 %13 %5",
                "OpLoopMerge %19 %5 None",
                "OpBranchConditional %17 %5 %19",
              }
            },
            {6,
              {
                "%7 = OpPhi %6 %8 %4 %9000 %5",
                "%9 = OpIAdd %6 %7 %14",
              }
            },
            {7,
              {
                "%9 = OpIAdd %6 %7 %14",
                "%17 = OpSLessThan %16 %7 %18",
              }
            },
            {8, {"%7 = OpPhi %6 %8 %4 %9000 %5"}},
            {9, {"%13 = OpFAdd %10 %9 %15"}}, // uses of %9 changed from %7 to %13
            {10,
              {
                "%11 = OpPhi %10 %12 %4 %13 %5",
                "%13 = OpFAdd %10 %9 %15",
              }
            },
            // no more uses of %11
            {12, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
            {13, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
            {14, {"%9 = OpIAdd %6 %7 %14"}},
            {15, {"%13 = OpFAdd %10 %9 %15"}},
            {16, {"%17 = OpSLessThan %16 %7 %18"}},
            {17, {"OpBranchConditional %17 %5 %19"}},
            {18, {"%17 = OpSLessThan %16 %7 %18"}},
            {19,
              {
                "OpLoopMerge %19 %5 None",
                "OpBranchConditional %17 %5 %19",
              }
            },
            // new uses of %9000
            {9000, {"%7 = OpPhi %6 %8 %4 %9000 %5"}},
          },
        },
      },
      { // OpPhi defining and referencing the same id.
        "%1 = OpTypeBool "
        "%2 = OpConstantTrue %1 "

        "%4 = OpFunction %3 None %5 "
        "%6 = OpLabel "
        "     OpBranch %7 "
        "%7 = OpLabel "
        "%8 = OpPhi %1   %8 %7   %2 %6 " // both defines and uses %8
        "     OpBranch %7 "
        "     OpFunctionEnd",
        {{8, 2}},
        "%1 = OpTypeBool\n"
        "%2 = OpConstantTrue %1\n"

        "%4 = OpFunction %3 None %5\n"
        "%6 = OpLabel\n"
             "OpBranch %7\n"
        "%7 = OpLabel\n"
        "%8 = OpPhi %1 %2 %7 %2 %6\n" // use of %8 changed to %2
             "OpBranch %7\n"
             "OpFunctionEnd",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpConstantTrue %1"},
            {4, "%4 = OpFunction %3 None %5"},
            {6, "%6 = OpLabel"},
            {7, "%7 = OpLabel"},
            {8, "%8 = OpPhi %1 %2 %7 %2 %6"},
          },
          { // uses
            {1,
              {
                "%2 = OpConstantTrue %1",
                "%8 = OpPhi %1 %2 %7 %2 %6",
              }
            },
            {2,
              {
                // TODO(antiagainst): address this.
                // We have duplication here because we didn't check existence
                // before inserting uses.
                "%8 = OpPhi %1 %2 %7 %2 %6",
                "%8 = OpPhi %1 %2 %7 %2 %6",
              }
            },
            {3, {"%4 = OpFunction %3 None %5"}},
            {5, {"%4 = OpFunction %3 None %5"}},
            {6, {"%8 = OpPhi %1 %2 %7 %2 %6"}},
            {7,
              {
                "OpBranch %7",
                "%8 = OpPhi %1 %2 %7 %2 %6",
                "OpBranch %7",
              }
            },
            // {8, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
          },
        },
      },
    })
);
// clang-format on

struct KillDefCase {
  const char* before;
  std::vector<uint32_t> ids_to_kill;
  const char* after;
  InstDefUse du;
};

using KillDefTest = ::testing::TestWithParam<KillDefCase>;

TEST_P(KillDefTest, Case) {
  const auto& tc = GetParam();

  // Build module.
  const std::vector<const char*> text = {tc.before};
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(JoinAllInsts(text));
  ASSERT_NE(nullptr, module);

  // Analyze def and use.
  opt::analysis::DefUseManager manager(module.get());

  // Do the substitution.
  for (const auto id : tc.ids_to_kill) manager.KillDef(id);

  EXPECT_EQ(tc.after, DisassembleModule(module.get()));
  CheckDef(tc.du, manager.id_to_defs());
  CheckUse(tc.du, manager.id_to_uses());
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, KillDefTest,
    ::testing::ValuesIn(std::vector<KillDefCase>{
      { // no def, no use, no kill
        "", {}, "", {}
      },
      { // kill nothing
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 2 "
        "%3 = OpTypeVector %1 3 ",
        {},
        "%1 = OpTypeBool\n"
        "%2 = OpTypeVector %1 2\n"
        "%3 = OpTypeVector %1 3",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpTypeVector %1 2"},
            {3, "%3 = OpTypeVector %1 3"},
          },
          { // uses
            {1,
              {
                "%2 = OpTypeVector %1 2",
                "%3 = OpTypeVector %1 3",
              }
            },
          },
        },
      },
      { // kill id used, kill id not used, kill id not defined
        "%1 = OpTypeBool "
        "%2 = OpTypeVector %1 2 "
        "%3 = OpTypeVector %1 3 "
        "%4 = OpTypeVector %1 4 "
        "%5 = OpTypeMatrix %3 3 "
        "%6 = OpTypeMatrix %2 3",
        {1, 3, 5, 10}, // ids to kill
        "OpNop\n"
        "%2 = OpTypeVector %1 2\n"
        "OpNop\n"
        "%4 = OpTypeVector %1 4\n"
        "OpNop\n"
        "%6 = OpTypeMatrix %2 3",
        {
          { // defs
            {2, "%2 = OpTypeVector %1 2"},
            {4, "%4 = OpTypeVector %1 4"},
            {6, "%6 = OpTypeMatrix %2 3"},
          },
          { // uses. %1 and %3 are both killed, so no uses
            // recorded for them anymore.
            {2, {"%6 = OpTypeMatrix %2 3"}},
          }
        },
      },
      { // OpPhi.
        kOpPhiTestFunction,
        {9, 11}, // kill one id used by OpPhi, kill one id generated by OpPhi
         "%2 = OpFunction %1 None %3\n"
         "%4 = OpLabel\n"
               "OpBranch %5\n"

         "%5 = OpLabel\n"
         "%7 = OpPhi %6 %8 %4 %9 %5\n"
              "OpNop\n"
              "OpNop\n"
        "%13 = OpFAdd %10 %11 %15\n"
        "%17 = OpSLessThan %16 %7 %18\n"
              "OpLoopMerge %19 %5 None\n"
              "OpBranchConditional %17 %5 %19\n"

        "%19 = OpLabel\n"
              "OpReturn\n"
              "OpFunctionEnd",
        {
          { // defs. %9 & %11 are killed.
            {2, "%2 = OpFunction %1 None %3"},
            {4, "%4 = OpLabel"},
            {5, "%5 = OpLabel"},
            {7, "%7 = OpPhi %6 %8 %4 %9 %5"},
            {13, "%13 = OpFAdd %10 %11 %15"},
            {17, "%17 = OpSLessThan %16 %7 %18"},
            {19, "%19 = OpLabel"},
          },
          { // uses
            {1, {"%2 = OpFunction %1 None %3"}},
            {3, {"%2 = OpFunction %1 None %3"}},
            {4,
              {
                "%7 = OpPhi %6 %8 %4 %9 %5",
                // "%11 = OpPhi %10 %12 %4 %13 %5",
              }
            },
            {5,
              {
                "OpBranch %5",
                "%7 = OpPhi %6 %8 %4 %9 %5",
                // "%11 = OpPhi %10 %12 %4 %13 %5",
                "OpLoopMerge %19 %5 None",
                "OpBranchConditional %17 %5 %19",
              }
            },
            {6,
              {
                "%7 = OpPhi %6 %8 %4 %9 %5",
                // "%9 = OpIAdd %6 %7 %14",
              }
            },
            {7,
              {
                // "%9 = OpIAdd %6 %7 %14",
                "%17 = OpSLessThan %16 %7 %18",
              }
            },
            {8, {"%7 = OpPhi %6 %8 %4 %9 %5"}},
            // {9, {"%7 = OpPhi %6 %8 %4 %9 %5"}},
            {10,
              {
                // "%11 = OpPhi %10 %12 %4 %13 %5",
                "%13 = OpFAdd %10 %11 %15",
              }
            },
            // {11, {"%13 = OpFAdd %10 %11 %15"}},
            // {12, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
            // {13, {"%11 = OpPhi %10 %12 %4 %13 %5"}},
            // {14, {"%9 = OpIAdd %6 %7 %14"}},
            {15, {"%13 = OpFAdd %10 %11 %15"}},
            {16, {"%17 = OpSLessThan %16 %7 %18"}},
            {17, {"OpBranchConditional %17 %5 %19"}},
            {18, {"%17 = OpSLessThan %16 %7 %18"}},
            {19,
              {
                "OpLoopMerge %19 %5 None",
                "OpBranchConditional %17 %5 %19",
              }
            },
          },
        },
      },
      { // OpPhi defining and referencing the same id.
        "%1 = OpTypeBool "
        "%2 = OpConstantTrue %1 "

        "%4 = OpFunction %3 None %5 "
        "%6 = OpLabel "
        "     OpBranch %7 "
        "%7 = OpLabel "
        "%8 = OpPhi %1   %8 %7   %2 %6 " // both defines and uses %8
        "     OpBranch %7 "
        "     OpFunctionEnd",
        {8},
        "%1 = OpTypeBool\n"
        "%2 = OpConstantTrue %1\n"

        "%4 = OpFunction %3 None %5\n"
        "%6 = OpLabel\n"
             "OpBranch %7\n"
        "%7 = OpLabel\n"
             "OpNop\n"
             "OpBranch %7\n"
             "OpFunctionEnd",
        {
          { // defs
            {1, "%1 = OpTypeBool"},
            {2, "%2 = OpConstantTrue %1"},
            {4, "%4 = OpFunction %3 None %5"},
            {6, "%6 = OpLabel"},
            {7, "%7 = OpLabel"},
            // {8, "%8 = OpPhi %1 %8 %7 %2 %6"},
          },
          { // uses
            {1,
              {
                "%2 = OpConstantTrue %1",
                // "%8 = OpPhi %1 %8 %7 %2 %6",
              }
            },
            // {2, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
            {3, {"%4 = OpFunction %3 None %5"}},
            {5, {"%4 = OpFunction %3 None %5"}},
            // {6, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
            {7,
              {
                "OpBranch %7",
                // "%8 = OpPhi %1 %8 %7 %2 %6",
                "OpBranch %7",
              }
            },
            // {8, {"%8 = OpPhi %1 %8 %7 %2 %6"}},
          },
        },
      },
    })
);
// clang-format on
//
TEST(DefUseTest, OpSwitch) {
  // Because disassembler has basic type check for OpSwitch's selector, we
  // cannot use the DisassembleInst() in the above. Thus, this special spotcheck
  // test case.

  const char original_text[] =
      // int64 f(int64 v) {
      //   switch (v) {
      //     case 1:                   break;
      //     case -4294967296:         break;
      //     case 9223372036854775807: break;
      //     default:                  break;
      //   }
      //   return v;
      // }
      " %1 = OpTypeInt 64 1 "
      " %2 = OpFunction %1 None %3 "  // %3 is int64(int64)*
      " %4 = OpFunctionParameter %1 "
      " %5 = OpLabel "
      " %6 = OpLoad %1 %4 "  // selector value
      "      OpSelectionMerge %7 None "
      "      OpSwitch %6 %8 "
      "                  1                    %9 "  // 1
      "                  -4294967296         %10 "  // -2^32
      "                  9223372036854775807 %11 "  // 2^63-1
      " %8 = OpLabel "                              // default
      "      OpBranch %7 "
      " %9 = OpLabel "
      "      OpBranch %7 "
      "%10 = OpLabel "
      "      OpBranch %7 "
      "%11 = OpLabel "
      "      OpBranch %7 "
      " %7 = OpLabel "
      "      OpReturnValue %6 "
      "      OpFunctionEnd";

  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(original_text);
  ASSERT_NE(nullptr, module);

  // Analyze def and use.
  opt::analysis::DefUseManager manager(module.get());

  // Do a bunch replacements.
  manager.ReplaceAllUsesWith(9, 900);    // to unused id
  manager.ReplaceAllUsesWith(10, 1000);  // to unused id
  manager.ReplaceAllUsesWith(11, 7);     // to existing id

  // clang-format off
  const char modified_text[] =
       "%1 = OpTypeInt 64 1\n"
       "%2 = OpFunction %1 None %3\n" // %3 is int64(int64)*
       "%4 = OpFunctionParameter %1\n"
       "%5 = OpLabel\n"
       "%6 = OpLoad %1 %4\n" // selector value
            "OpSelectionMerge %7 None\n"
            "OpSwitch %6 %8 1 %900 -4294967296 %1000 9223372036854775807 %7\n" // changed!
       "%8 = OpLabel\n"      // default
            "OpBranch %7\n"
       "%9 = OpLabel\n"
            "OpBranch %7\n"
      "%10 = OpLabel\n"
            "OpBranch %7\n"
      "%11 = OpLabel\n"
            "OpBranch %7\n"
       "%7 = OpLabel\n"
            "OpReturnValue %6\n"
            "OpFunctionEnd";
  // clang-format on

  EXPECT_EQ(modified_text, DisassembleModule(module.get()));

  InstDefUse def_uses = {};
  def_uses.defs = {
      {1, "%1 = OpTypeInt 64 1"},
      {2, "%2 = OpFunction %1 None %3"},
      {4, "%4 = OpFunctionParameter %1"},
      {5, "%5 = OpLabel"},
      {6, "%6 = OpLoad %1 %4"},
      {7, "%7 = OpLabel"},
      {8, "%8 = OpLabel"},
      {9, "%9 = OpLabel"},
      {10, "%10 = OpLabel"},
      {11, "%11 = OpLabel"},
  };
  CheckDef(def_uses, manager.id_to_defs());

  {
    auto* use_list = manager.GetUses(6);
    ASSERT_NE(nullptr, use_list);
    EXPECT_EQ(2u, use_list->size());
    EXPECT_EQ(SpvOpSwitch, use_list->front().inst->opcode());
    EXPECT_EQ(SpvOpReturnValue, use_list->back().inst->opcode());
  }
  {
    auto* use_list = manager.GetUses(7);
    ASSERT_NE(nullptr, use_list);
    EXPECT_EQ(6u, use_list->size());
    std::vector<SpvOp> opcodes;
    for (const auto& use : *use_list) {
      opcodes.push_back(use.inst->opcode());
    }
    // OpSwitch is now a user of %7.
    EXPECT_THAT(opcodes,
                ElementsAre(SpvOpSelectionMerge, SpvOpBranch, SpvOpBranch,
                            SpvOpBranch, SpvOpBranch, SpvOpSwitch));
  }
  // Check all ids only used by OpSwitch after replacement.
  for (const auto id : {8, 900, 1000}) {
    auto* use_list = manager.GetUses(id);
    ASSERT_NE(nullptr, use_list);
    EXPECT_EQ(1u, use_list->size());
    EXPECT_EQ(SpvOpSwitch, use_list->front().inst->opcode());
  }
}

// Creates an |result_id| = OpTypeInt 32 1 instruction.
ir::Instruction Int32TypeInstruction(uint32_t result_id) {
  return ir::Instruction(SpvOp::SpvOpTypeInt, 0, result_id,
                         {ir::Operand(SPV_OPERAND_TYPE_LITERAL_INTEGER, {32}),
                          ir::Operand(SPV_OPERAND_TYPE_LITERAL_INTEGER, {1})});
}

// Creates an |result_id| = OpConstantTrue/Flase |type_id| instruction.
ir::Instruction ConstantBoolInstruction(bool value, uint32_t type_id,
                                        uint32_t result_id) {
  return ir::Instruction(
      value ? SpvOp::SpvOpConstantTrue : SpvOp::SpvOpConstantFalse, type_id,
      result_id, {});
}

// Creates an |result_id| = OpLabel instruction.
ir::Instruction LabelInstruction(uint32_t result_id) {
  return ir::Instruction(SpvOp::SpvOpLabel, 0, result_id, {});
}

// Creates an OpBranch |target_id| instruction.
ir::Instruction BranchInstruction(uint32_t target_id) {
  return ir::Instruction(SpvOp::SpvOpBranch, 0, 0,
                         {
                             ir::Operand(SPV_OPERAND_TYPE_ID, {target_id}),
                         });
}

// Test case for analyzing individual instructions.
struct AnalyzeInstDefUseTestCase {
  std::vector<ir::Instruction> insts;  // instrutions to be analyzed in order.
  const char* module_text;
  InstDefUse expected_define_use;
};

using AnalyzeInstDefUseTest =
    ::testing::TestWithParam<AnalyzeInstDefUseTestCase>;

// Test the analyzing result for individual instructions.
TEST_P(AnalyzeInstDefUseTest, Case) {
  auto tc = GetParam();

  // Build module.
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(tc.module_text);
  ASSERT_NE(nullptr, module);

  // Analyze the instructions.
  opt::analysis::DefUseManager manager(module.get());
  for (ir::Instruction& inst : tc.insts) {
    manager.AnalyzeInstDefUse(&inst);
  }

  CheckDef(tc.expected_define_use, manager.id_to_defs());
  CheckUse(tc.expected_define_use, manager.id_to_uses());
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, AnalyzeInstDefUseTest,
    ::testing::ValuesIn(std::vector<AnalyzeInstDefUseTestCase>{
      { // A type declaring instruction.
        {Int32TypeInstruction(1)},
        "",
        {
          // defs
          {{1, "%1 = OpTypeInt 32 1"}},
          {}, // no uses
        },
      },
      { // A type declaring instruction and a constant value.
        {
          Int32TypeInstruction(1),
          ConstantBoolInstruction(true, 1, 2),
        },
        "",
        {
          { // defs
            {1, "%1 = OpTypeInt 32 1"},
            {2, "%2 = OpConstantTrue %1"}, // It is fine the SPIR-V code here is invalid.
          },
          { // uses
            {1, {"%2 = OpConstantTrue %1"}},
          },
        },
      },
      { // Analyze two instrutions that have same result id. The def use info
        // of the result id from the first instruction should be overwritten by
        // the second instruction.
        {
          ConstantBoolInstruction(true, 1, 2),
          // The def-use info of the following instruction should overwrite the
          // records of the above one.
          ConstantBoolInstruction(false, 3, 2),
        },
        "",
        {
          // defs
          {{2, "%2 = OpConstantFalse %3"}},
          // uses
          {{3, {"%2 = OpConstantFalse %3"}}}
        }
      },
      { // Analyze forward reference instruction, also instruction that does
        // not have result id.
        {
          BranchInstruction(2),
          LabelInstruction(2),
        },
        "",
        {
          // defs
          {{2, "%2 = OpLabel"}},
          // uses
          {{2, {"OpBranch %2"}}},
        }
      },
      { // Analyzing an additional instruction with new result id to an
        // existing module.
        {
          ConstantBoolInstruction(true, 1, 2),
        },
        "%1 = OpTypeInt 32 1 ",
        {
          { // defs
            {1, "%1 = OpTypeInt 32 1"},
            {2, "%2 = OpConstantTrue %1"},
          },
          { // uses
            {1, {"%2 = OpConstantTrue %1"}},
          },
        }
      },
      { // Analyzing an additional instruction with existing result id to an
        // existing module.
        {
          ConstantBoolInstruction(true, 1, 2),
        },
        "%1 = OpTypeInt 32 1 "
        "%2 = OpTypeBool ",
        {
          { // defs
            {1, "%1 = OpTypeInt 32 1"},
            {2, "%2 = OpConstantTrue %1"},
          },
          { // uses
            {1, {"%2 = OpConstantTrue %1"}},
          },
        }
      },
      }));
// clang-format on

struct KillInstTestCase {
  const char* before;
  std::unordered_set<uint32_t> indices_for_inst_to_kill;
  const char* after;
  InstDefUse expected_define_use;
};

using KillInstTest = ::testing::TestWithParam<KillInstTestCase>;

TEST_P(KillInstTest, Case) {
  auto tc = GetParam();

  // Build module.
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(tc.before);
  ASSERT_NE(nullptr, module);

  // KillInst
  uint32_t index = 0;
  opt::analysis::DefUseManager manager(module.get());
  module->ForEachInst([&index, &tc, &manager](ir::Instruction* inst) {
    if (tc.indices_for_inst_to_kill.count(index) != 0) {
      manager.KillInst(inst);
    }
    index++;
  });

  EXPECT_EQ(tc.after, DisassembleModule(module.get()));
  CheckDef(tc.expected_define_use, manager.id_to_defs());
  CheckUse(tc.expected_define_use, manager.id_to_uses());
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, KillInstTest,
    ::testing::ValuesIn(std::vector<KillInstTestCase>{
      // Kill id defining instructions.
      {
        "%2 = OpFunction %1 None %3 "
        "%4 = OpLabel "
        "     OpBranch %5 "
        "%5 = OpLabel "
        "     OpBranch %6 "
        "%6 = OpLabel "
        "     OpBranch %4 "
        "%7 = OpLabel "
        "     OpReturn "
        "     OpFunctionEnd",
        {0, 3, 5, 7},
        "OpNop\n"
        "%4 = OpLabel\n"
        "OpBranch %5\n"
        "OpNop\n"
        "OpBranch %6\n"
        "OpNop\n"
        "OpBranch %4\n"
        "OpNop\n"
        "OpReturn\n"
        "OpFunctionEnd",
        {
          // defs
          {{4, "%4 = OpLabel"}},
          // uses
          {{4, {"OpBranch %4"}}}
        }
      },
      // Kill instructions that do not have result ids.
      {
        "%2 = OpFunction %1 None %3 "
        "%4 = OpLabel "
        "     OpBranch %5 "
        "%5 = OpLabel "
        "     OpBranch %6 "
        "%6 = OpLabel "
        "     OpBranch %4 "
        "%7 = OpLabel "
        "     OpReturn "
        "     OpFunctionEnd",
        {2, 4},
        "%2 = OpFunction %1 None %3\n"
        "%4 = OpLabel\n"
             "OpNop\n"
        "%5 = OpLabel\n"
             "OpNop\n"
        "%6 = OpLabel\n"
             "OpBranch %4\n"
        "%7 = OpLabel\n"
             "OpReturn\n"
             "OpFunctionEnd",
        {
          // defs
          {
            {2, "%2 = OpFunction %1 None %3"},
            {4, "%4 = OpLabel"},
            {5, "%5 = OpLabel"},
            {6, "%6 = OpLabel"},
            {7, "%7 = OpLabel"},
          },
          // uses
          {
            {1, {"%2 = OpFunction %1 None %3"}},
            {3, {"%2 = OpFunction %1 None %3"}},
            {4, {"OpBranch %4"}},
          }
        }
      },
      }));
// clang-format on

struct GetAnnotationsTestCase {
  const char* code;
  uint32_t id;
  std::vector<std::string> annotations;
};

using GetAnnotationsTest = ::testing::TestWithParam<GetAnnotationsTestCase>;

TEST_P(GetAnnotationsTest, Case) {
  const GetAnnotationsTestCase& tc = GetParam();

  // Build module.
  std::unique_ptr<ir::Module> module =
      SpvTools(SPV_ENV_UNIVERSAL_1_1).BuildModule(tc.code);
  ASSERT_NE(nullptr, module);

  // Get annotations
  opt::analysis::DefUseManager manager(module.get());
  auto insts = manager.GetAnnotations(tc.id);

  // Check
  ASSERT_EQ(tc.annotations.size(), insts.size())
      << "wrong number of annotation instructions";
  auto inst_iter = insts.begin();
  for (const std::string& expected_anno_inst : tc.annotations) {
    EXPECT_EQ(expected_anno_inst, DisassembleInst(*inst_iter))
        << "annotation instruction mismatch";
    inst_iter++;
  }
}

// clang-format off
INSTANTIATE_TEST_CASE_P(
    TestCase, GetAnnotationsTest,
    ::testing::ValuesIn(std::vector<GetAnnotationsTestCase>{
      // empty
      {"", 0, {}},
      // basic
      {
        // code
        "OpDecorate %1 Block "
        "OpDecorate %1 RelaxedPrecision "
        "%3 = OpTypeInt 32 0 "
        "%1 = OpTypeStruct %3",
        // id
        1,
        // annotations
        {
          "OpDecorate %1 Block",
          "OpDecorate %1 RelaxedPrecision",
        },
      },
      // with debug instructions
      {
        // code
        "OpName %1 \"struct_type\" "
        "OpName %3 \"int_type\" "
        "OpDecorate %1 Block "
        "OpDecorate %1 RelaxedPrecision "
        "%3 = OpTypeInt 32 0 "
        "%1 = OpTypeStruct %3",
        // id
        1,
        // annotations
        {
          "OpDecorate %1 Block",
          "OpDecorate %1 RelaxedPrecision",
        },
      },
      // no annotations
      {
        // code
        "OpName %1 \"struct_type\" "
        "OpName %3 \"int_type\" "
        "OpDecorate %1 Block "
        "OpDecorate %1 RelaxedPrecision "
        "%3 = OpTypeInt 32 0 "
        "%1 = OpTypeStruct %3",
        // id
        3,
        // annotations
        {},
      },
      // decoration group
      {
        // code
        "OpDecorate %1 Block "
        "OpDecorate %1 RelaxedPrecision "
        "%1 = OpDecorationGroup "
        "OpGroupDecorate %1 %2 %3 "
        "%4 = OpTypeInt 32 0 "
        "%2 = OpTypeStruct %4 "
        "%3 = OpTypeStruct %4 %4",
        // id
        3,
        // annotations
        {
          "OpGroupDecorate %1 %2 %3",
        },
      },
      // memeber decorate
      {
        // code
        "OpMemberDecorate %1 0 RelaxedPrecision "
        "%2 = OpTypeInt 32 0 "
        "%1 = OpTypeStruct %2 %2",
        // id
        1,
        // annotations
        {
          "OpMemberDecorate %1 0 RelaxedPrecision",
        },
      },
      }));
// clang-format on
}  // anonymous namespace
