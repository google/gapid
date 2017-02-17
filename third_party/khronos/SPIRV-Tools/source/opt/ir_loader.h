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

#ifndef LIBSPIRV_OPT_IR_LOADER_H_
#define LIBSPIRV_OPT_IR_LOADER_H_

#include <memory>

#include "basic_block.h"
#include "instruction.h"
#include "module.h"
#include "spirv-tools/libspirv.h"

namespace spvtools {
namespace ir {

// Loader class for constructing SPIR-V in-memory IR representation. Methods in
// this class are designed to work with the interface for spvBinaryParse() in
// libspirv.h so that we can leverage the syntax checks implemented behind it.
//
// The user is expected to call SetModuleHeader() to fill in the module's
// header, and then AddInstruction() for each decoded instruction, and finally
// EndModule() to finalize the module. The instructions processed in sequence
// by AddInstruction() should comprise a valid SPIR-V module.
class IrLoader {
 public:
  // Instantiates a builder to construct the given |module| gradually.
  IrLoader(Module* module) : module_(module) {}

  // Sets the fields in the module's header to the given parameters.
  void SetModuleHeader(uint32_t magic, uint32_t version, uint32_t generator,
                       uint32_t bound, uint32_t reserved) {
    module_->SetHeader({magic, version, generator, bound, reserved});
  }
  // Adds an instruction to the module. This method will properly capture and
  // store the data provided in |inst| so that |inst| is no longer needed after
  // returning.
  void AddInstruction(const spv_parsed_instruction_t* inst);
  // Finalizes the module construction. This must be called after the module
  // header has been set and all instructions have been added.  This is
  // forgiving in the case of a missing terminator instruction on a basic block,
  // or a missing OpFunctionEnd.  Resolves internal bookkeeping.
  void EndModule();

 private:
  // The module to be built.
  Module* module_;
  // The current Function under construction.
  std::unique_ptr<Function> function_;
  // The current BasicBlock under construction.
  std::unique_ptr<BasicBlock> block_;
  // Line related debug instructions accumulated thus far.
  std::vector<Instruction> dbg_line_info_;
};

}  // namespace ir
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_IR_LOADER_H_
