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

#ifndef SPIRV_TOOLS_LIBSPIRV_HPP_
#define SPIRV_TOOLS_LIBSPIRV_HPP_

#include <memory>
#include <string>
#include <vector>

#include "module.h"
#include "spirv-tools/libspirv.h"

namespace spvtools {

// C++ interface for SPIRV-Tools functionalities. It wraps the context
// (including target environment and the corresponding SPIR-V grammar) and
// provides methods for assembling, disassembling, validating, and optimizing.
//
// Instances of this class are thread-safe.
class SpvTools {
 public:
  // Creates an instance targeting the given environment |env|.
  SpvTools(spv_target_env env) : context_(spvContextCreate(env)) {}

  ~SpvTools() { spvContextDestroy(context_); }

  // TODO(antiagainst): handle error message in the following APIs.

  // Assembles the given assembly |text| and writes the result to |binary|.
  // Returns SPV_SUCCESS on successful assembling.
  spv_result_t Assemble(const std::string& text, std::vector<uint32_t>* binary);

  // Disassembles the given SPIR-V |binary| with the given options and returns
  // the assembly. By default the options are set to generate assembly with
  // friendly variable names and no SPIR-V assembly header. Returns SPV_SUCCESS
  // on successful disassembling.
  spv_result_t Disassemble(
      const std::vector<uint32_t>& binary, std::string* text,
      uint32_t options = SPV_BINARY_TO_TEXT_OPTION_NO_HEADER |
                         SPV_BINARY_TO_TEXT_OPTION_FRIENDLY_NAMES);

  // Builds and returns a Module from the given SPIR-V |binary|.
  std::unique_ptr<ir::Module> BuildModule(const std::vector<uint32_t>& binary);

  // Builds and returns a Module from the given SPIR-V assembly |text|.
  std::unique_ptr<ir::Module> BuildModule(const std::string& text);

 private:
  // Context for the current invocation. Thread-safety of this class depends on
  // the constness of this field.
  spv_context context_;
};

}  // namespace spvtools

#endif  // SPIRV_TOOLS_LIBSPIRV_HPP_
