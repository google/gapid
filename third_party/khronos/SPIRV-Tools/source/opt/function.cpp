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

#include "function.h"

namespace spvtools {
namespace ir {

void Function::ForEachInst(const std::function<void(Instruction*)>& f,
                           bool run_on_debug_line_insts) {
  if (def_inst_) def_inst_->ForEachInst(f, run_on_debug_line_insts);
  for (auto& param : params_) param->ForEachInst(f, run_on_debug_line_insts);
  for (auto& bb : blocks_) bb->ForEachInst(f, run_on_debug_line_insts);
  if (end_inst_) end_inst_->ForEachInst(f, run_on_debug_line_insts);
}

void Function::ForEachInst(const std::function<void(const Instruction*)>& f,
                           bool run_on_debug_line_insts) const {
  if (def_inst_)
    static_cast<const Instruction*>(def_inst_.get())
        ->ForEachInst(f, run_on_debug_line_insts);

  for (const auto& param : params_)
    static_cast<const Instruction*>(param.get())
        ->ForEachInst(f, run_on_debug_line_insts);

  for (const auto& bb : blocks_)
    static_cast<const BasicBlock*>(bb.get())->ForEachInst(
        f, run_on_debug_line_insts);

  if (end_inst_)
    static_cast<const Instruction*>(end_inst_.get())
        ->ForEachInst(f, run_on_debug_line_insts);
}

}  // namespace ir
}  // namespace spvtools
