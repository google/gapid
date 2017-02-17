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

#ifndef LIBSPIRV_TEST_OPT_PASS_FIXTURE_H_
#define LIBSPIRV_TEST_OPT_PASS_FIXTURE_H_

#include <string>
#include <tuple>
#include <vector>

#include <gtest/gtest.h>

#include "opt/libspirv.hpp"
#include "opt/make_unique.h"
#include "opt/pass_manager.h"
#include "opt/passes.h"

namespace spvtools {

// Template class for testing passes. It contains some handy utility methods for
// running passes and checking results.
//
// To write value-Parameterized tests:
//   using ValueParamTest = PassTest<::testing::TestWithParam<std::string>>;
// To use as normal fixture:
//   using FixtureTest = PassTest<::testing::Test>;
template <typename TestT>
class PassTest : public TestT {
 public:
  PassTest()
      : tools_(SPV_ENV_UNIVERSAL_1_1), manager_(new opt::PassManager()) {}

  // Runs the given |pass| on the binary assembled from the |assembly|, and
  // disassebles the optimized binary. Returns a tuple of disassembly string
  // and the boolean value returned from pass Process() function.
  std::tuple<std::string, bool> OptimizeAndDisassemble(
      opt::Pass* pass, const std::string& original, bool skip_nop) {
    std::unique_ptr<ir::Module> module = tools_.BuildModule(original);
    EXPECT_NE(nullptr, module) << "Assembling failed for shader:\n"
                               << original << std::endl;
    if (!module) {
      return std::make_tuple(std::string(), false);
    }

    const bool modified = pass->Process(module.get());

    std::vector<uint32_t> binary;
    module->ToBinary(&binary, skip_nop);
    std::string optimized;
    EXPECT_EQ(SPV_SUCCESS, tools_.Disassemble(binary, &optimized))
        << "Disassembling failed for shader:\n"
        << original << std::endl;
    return std::make_tuple(optimized, modified);
  }

  // Runs a single pass of class |PassT| on the binary assembled from the
  // |assembly|, disassembles the optimized binary. Returns a tuple of
  // disassembly string and the boolean value from the pass Process() function.
  template <typename PassT, typename... Args>
  std::tuple<std::string, bool> SinglePassRunAndDisassemble(
      const std::string& assembly, bool skip_nop, Args&&... args) {
    auto pass = MakeUnique<PassT>(std::forward<Args>(args)...);
    return OptimizeAndDisassemble(pass.get(), assembly, skip_nop);
  }

  // Runs a single pass of class |PassT| on the binary assembled from the
  // |original| assembly, and checks whether the optimized binary can be
  // disassembled to the |expected| assembly. This does *not* involve pass
  // manager. Callers are suggested to use SCOPED_TRACE() for better messages.
  template <typename PassT, typename... Args>
  void SinglePassRunAndCheck(const std::string& original,
                             const std::string& expected, bool skip_nop,
                             Args&&... args) {
    std::string optimized;
    bool modified = false;
    std::tie(optimized, modified) = SinglePassRunAndDisassemble<PassT>(
        original, skip_nop, std::forward<Args>(args)...);
    // Check whether the pass returns the correct modification indication.
    EXPECT_EQ(original != expected, modified);
    EXPECT_EQ(expected, optimized);
  }

  // Adds a pass to be run.
  template <typename PassT, typename... Args>
  void AddPass(Args&&... args) {
    manager_->AddPass<PassT>(std::forward<Args>(args)...);
  }

  // Renews the pass manager, including clearing all previously added passes.
  void RenewPassManger() { manager_.reset(new opt::PassManager()); }

  // Runs the passes added thus far using a pass manager on the binary assembled
  // from the |original| assembly, and checks whether the optimized binary can
  // be disassembled to the |expected| assembly. Callers are suggested to use
  // SCOPED_TRACE() for better messages.
  void RunAndCheck(const std::string& original, const std::string& expected) {
    assert(manager_->NumPasses());

    std::unique_ptr<ir::Module> module = tools_.BuildModule(original);
    ASSERT_NE(nullptr, module);

    manager_->Run(module.get());

    std::vector<uint32_t> binary;
    module->ToBinary(&binary, /* skip_nop = */ false);

    std::string optimized;
    EXPECT_EQ(SPV_SUCCESS, tools_.Disassemble(binary, &optimized));
    EXPECT_EQ(expected, optimized);
  }

 private:
  SpvTools tools_;  // An instance for calling SPIRV-Tools functionalities.
  std::unique_ptr<opt::PassManager> manager_;  // The pass manager.
};

}  // namespace spvtools

#endif  // LIBSPIRV_TEST_OPT_PASS_FIXTURE_H_
