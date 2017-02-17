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

#include <initializer_list>

#include "module_utils.h"
#include "opt/make_unique.h"
#include "pass_fixture.h"

namespace {

using namespace spvtools;
using spvtest::GetIdBound;
using ::testing::Eq;

// A null pass whose construtors accept arguments
class NullPassWithArgs : public opt::NullPass {
 public:
  NullPassWithArgs(uint32_t) : NullPass() {}
  NullPassWithArgs(std::string) : NullPass() {}
  NullPassWithArgs(const std::vector<int>&) : NullPass() {}
  NullPassWithArgs(const std::vector<int>&, uint32_t) : NullPass() {}

  const char* name() const override { return "null-with-args"; }
};

TEST(PassManager, Interface) {
  opt::PassManager manager;
  EXPECT_EQ(0u, manager.NumPasses());

  manager.AddPass<opt::StripDebugInfoPass>();
  EXPECT_EQ(1u, manager.NumPasses());
  EXPECT_STREQ("strip-debug", manager.GetPass(0)->name());

  manager.AddPass(MakeUnique<opt::NullPass>());
  EXPECT_EQ(2u, manager.NumPasses());
  EXPECT_STREQ("strip-debug", manager.GetPass(0)->name());
  EXPECT_STREQ("null", manager.GetPass(1)->name());

  manager.AddPass<opt::StripDebugInfoPass>();
  EXPECT_EQ(3u, manager.NumPasses());
  EXPECT_STREQ("strip-debug", manager.GetPass(0)->name());
  EXPECT_STREQ("null", manager.GetPass(1)->name());
  EXPECT_STREQ("strip-debug", manager.GetPass(2)->name());

  manager.AddPass<NullPassWithArgs>(1u);
  manager.AddPass<NullPassWithArgs>("null pass args");
  manager.AddPass<NullPassWithArgs>(std::initializer_list<int>{1, 2});
  manager.AddPass<NullPassWithArgs>(std::initializer_list<int>{1, 2}, 3);
  EXPECT_EQ(7u, manager.NumPasses());
  EXPECT_STREQ("strip-debug", manager.GetPass(0)->name());
  EXPECT_STREQ("null", manager.GetPass(1)->name());
  EXPECT_STREQ("strip-debug", manager.GetPass(2)->name());
  EXPECT_STREQ("null-with-args", manager.GetPass(3)->name());
  EXPECT_STREQ("null-with-args", manager.GetPass(4)->name());
  EXPECT_STREQ("null-with-args", manager.GetPass(5)->name());
  EXPECT_STREQ("null-with-args", manager.GetPass(6)->name());
}

// A pass that appends an OpNop instruction to the debug section.
class AppendOpNopPass : public opt::Pass {
  const char* name() const override { return "AppendOpNop"; }
  bool Process(ir::Module* module) override {
    auto inst = MakeUnique<ir::Instruction>();
    module->AddDebugInst(std::move(inst));
    return true;
  }
};

// A pass that appends specified number of OpNop instructions to the debug
// section.
class AppendMultipleOpNopPass : public opt::Pass {
 public:
  AppendMultipleOpNopPass(uint32_t num_nop) : num_nop_(num_nop) {}
  const char* name() const override { return "AppendOpNop"; }
  bool Process(ir::Module* module) override {
    for (uint32_t i = 0; i < num_nop_; i++) {
      auto inst = MakeUnique<ir::Instruction>();
      module->AddDebugInst(std::move(inst));
    }
    return true;
  }

 private:
  uint32_t num_nop_;
};

// A pass that duplicates the last instruction in the debug section.
class DuplicateInstPass : public opt::Pass {
  const char* name() const override { return "DuplicateInst"; }
  bool Process(ir::Module* module) override {
    auto inst = MakeUnique<ir::Instruction>(*(--module->debug_end()));
    module->AddDebugInst(std::move(inst));
    return true;
  }
};

using PassManagerTest = PassTest<::testing::Test>;

TEST_F(PassManagerTest, Run) {
  const std::string text = "OpMemoryModel Logical GLSL450\nOpSource ESSL 310\n";

  AddPass<AppendOpNopPass>();
  AddPass<AppendOpNopPass>();
  RunAndCheck(text.c_str(), (text + "OpNop\nOpNop\n").c_str());

  RenewPassManger();
  AddPass<AppendOpNopPass>();
  AddPass<DuplicateInstPass>();
  RunAndCheck(text.c_str(), (text + "OpNop\nOpNop\n").c_str());

  RenewPassManger();
  AddPass<DuplicateInstPass>();
  AddPass<AppendOpNopPass>();
  RunAndCheck(text.c_str(), (text + "OpSource ESSL 310\nOpNop\n").c_str());

  RenewPassManger();
  AddPass<AppendMultipleOpNopPass>(3);
  RunAndCheck(text.c_str(), (text + "OpNop\nOpNop\nOpNop\n").c_str());
}

// A pass that appends an OpTypeVoid instruction that uses a given id.
class AppendTypeVoidInstPass : public opt::Pass {
 public:
  AppendTypeVoidInstPass(uint32_t result_id) : result_id_(result_id) {}
  const char* name() const override { return "AppendTypeVoidInstPass"; }
  bool Process(ir::Module* module) override {
    auto inst = MakeUnique<ir::Instruction>(SpvOpTypeVoid, 0, result_id_,
                                            std::vector<ir::Operand>{});
    module->AddType(std::move(inst));
    return true;
  }

 private:
  uint32_t result_id_;
};

TEST(PassManager, RecomputeIdBoundAutomatically) {
  ir::Module module;
  EXPECT_THAT(GetIdBound(module), Eq(0u));

  opt::PassManager manager;
  manager.Run(&module);
  manager.AddPass<AppendOpNopPass>();
  // With no ID changes, the ID bound does not change.
  EXPECT_THAT(GetIdBound(module), Eq(0u));

  // Now we force an Id of 100 to be used.
  manager.AddPass(MakeUnique<AppendTypeVoidInstPass>(100));
  EXPECT_THAT(GetIdBound(module), Eq(0u));
  manager.Run(&module);
  // The Id has been updated automatically, even though the pass
  // did not update it.
  EXPECT_THAT(GetIdBound(module), Eq(101u));

  // Try one more time!
  manager.AddPass(MakeUnique<AppendTypeVoidInstPass>(200));
  manager.Run(&module);
  EXPECT_THAT(GetIdBound(module), Eq(201u));

  // Add another pass, but which uses a lower Id.
  manager.AddPass(MakeUnique<AppendTypeVoidInstPass>(10));
  manager.Run(&module);
  // The Id stays high.
  EXPECT_THAT(GetIdBound(module), Eq(201u));
}

}  // anonymous namespace
