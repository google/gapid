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

#ifndef LIBSPIRV_OPT_UNIFY_CONSTANT_PASS_H_
#define LIBSPIRV_OPT_UNIFY_CONSTANT_PASS_H_

#include "module.h"
#include "pass.h"

namespace spvtools {
namespace opt {

// The optimization pass to de-duplicate the constants. Constants with exactly
// same values and identical form will be unified and only one constant will be
// kept for each unique pair of type and value.
// There are several cases not handled by this pass:
//  1) Constants defined by OpConstantNull instructions (null constants) and
//  constants defined by OpConstantFalse, OpConstant or OpConstantComposite
//  with value(s) 0 (zero-valued normal constants) are not considered
//  equivalent. So null constants won't be used to replace zero-valued normal
//  constants, and other constants won't replace the null constants either.
//  2) Whenever there are decorations to the constant's result id or its type
//  id, the constants won't be handled, which means, it won't be used to
//  replace any other constants, neither can other constants replace it.
//  3) NaN in float point format with different bit patterns are not unified.
class UnifyConstantPass : public Pass {
 public:
  const char* name() const override { return "unify-const"; }
  bool Process(ir::Module*) override;
};

}  // namespace opt
}  // namespace spvtools

#endif // LIBSPIRV_OPT_UNIFY_CONSTANT_PASS_H_
