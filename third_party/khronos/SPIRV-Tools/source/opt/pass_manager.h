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

#ifndef LIBSPIRV_OPT_PASS_MANAGER_H_
#define LIBSPIRV_OPT_PASS_MANAGER_H_

#include <cassert>
#include <memory>
#include <vector>

#include "module.h"
#include "passes.h"

namespace spvtools {
namespace opt {

// The pass manager, responsible for tracking and running passes.
// Clients should first call AddPass() to add passes and then call Run()
// to run on a module. Passes are executed in the exact order of added.
//
// TODO(antiagainst): The pass manager is fairly simple right now. Eventually it
// should support pass dependency, common functionality (like def-use analysis)
// sharing, etc.
class PassManager {
 public:
  // Adds a pass.
  void AddPass(std::unique_ptr<Pass> pass) {
    passes_.push_back(std::move(pass));
  }
  // Uses the argument to construct a pass instance of type PassT, and adds the
  // pass instance to this pass manger.
  template <typename PassT, typename... Args>
  void AddPass(Args&&... args) {
    passes_.emplace_back(new PassT(std::forward<Args>(args)...));
  }

  // Returns the number of passes added.
  uint32_t NumPasses() const { return static_cast<uint32_t>(passes_.size()); }
  // Returns a pointer to the |index|th pass added.
  Pass* GetPass(uint32_t index) const {
    assert(index < passes_.size() && "index out of bound");
    return passes_[index].get();
  }

  // Runs all passes on the given |module|.
  void Run(ir::Module* module) {
    bool modified = false;
    for (const auto& pass : passes_) {
      modified |= pass->Process(module);
    }
    // Set the Id bound in the header in case a pass forgot to do so.
    if (modified) module->SetIdBound(module->ComputeIdBound());
  }

 private:
  // A vector of passes. Order matters.
  std::vector<std::unique_ptr<Pass>> passes_;
};

}  // namespace opt
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_PASS_MANAGER_H_
