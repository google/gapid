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

#ifndef LIBSPIRV_OPT_FREEZE_SPEC_CONSTANT_VALUE_PASS_H_
#define LIBSPIRV_OPT_FREEZE_SPEC_CONSTANT_VALUE_PASS_H_

#include "module.h"
#include "pass.h"

namespace spvtools {
namespace opt {

// The transformation pass that specializes the value of spec constants to
// their default values. This pass only processes the spec constants that have
// Spec ID decorations (defined by OpSpecConstant, OpSpecConstantTrue and
// OpSpecConstantFalse instructions) and replaces them with their front-end
// version counterparts (OpConstant, OpConstantTrue and OpConstantFalse). The
// corresponding Spec ID annotation instructions will also be removed. This
// pass does not fold the newly added front-end constants and does not process
// other spec constants defined by OpSpecConstantComposite or OpSpecConstantOp.
class FreezeSpecConstantValuePass : public Pass {
 public:
  const char* name() const override { return "freeze-spec-const"; }
  bool Process(ir::Module*) override;
};

}  // namespace opt
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_FREEZE_SPEC_CONSTANT_VALUE_PASS_H_
