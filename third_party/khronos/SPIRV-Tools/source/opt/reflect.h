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

#ifndef LIBSPIRV_OPT_REFLECT_H_
#define LIBSPIRV_OPT_REFLECT_H_

#include "spirv/1.1/spirv.h"

namespace spvtools {
namespace ir {

// Note that as SPIR-V evolves over time, new opcodes may appear. So the
// following functions tend to be outdated and should be updated when SPIR-V
// version bumps.

inline bool IsDebugInst(SpvOp opcode) {
  return (opcode >= SpvOpSourceContinued && opcode <= SpvOpLine) ||
         opcode == SpvOpNoLine || opcode == SpvOpModuleProcessed;
}
inline bool IsDebugLineInst(SpvOp opcode) {
  return opcode == SpvOpLine || opcode == SpvOpNoLine;
}
inline bool IsAnnotationInst(SpvOp opcode) {
  return opcode >= SpvOpDecorate && opcode <= SpvOpGroupMemberDecorate;
}
inline bool IsTypeInst(SpvOp opcode) {
  return (opcode >= SpvOpTypeVoid && opcode <= SpvOpTypeForwardPointer) ||
         opcode == SpvOpTypePipeStorage || opcode == SpvOpTypeNamedBarrier;
}
inline bool IsConstantInst(SpvOp opcode) {
  return opcode >= SpvOpConstantTrue && opcode <= SpvOpSpecConstantOp;
}
inline bool IsTerminatorInst(SpvOp opcode) {
  return opcode >= SpvOpBranch && opcode <= SpvOpUnreachable;
}
inline bool IsVariableInst(SpvOp opcode) {
	return opcode == SpvOpVariable;
}

}  // namespace ir
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_REFLECT_H_
