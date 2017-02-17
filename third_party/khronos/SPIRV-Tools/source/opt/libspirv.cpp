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

#include "libspirv.hpp"

#include "ir_loader.h"
#include "make_unique.h"

namespace spvtools {

namespace {

// Sets the module header. Meets the interface requirement of spvBinaryParse().
spv_result_t SetSpvHeader(void* builder, spv_endianness_t, uint32_t magic,
                          uint32_t version, uint32_t generator,
                          uint32_t id_bound, uint32_t reserved) {
  reinterpret_cast<ir::IrLoader*>(builder)
      ->SetModuleHeader(magic, version, generator, id_bound, reserved);
  return SPV_SUCCESS;
};

// Processes a parsed instruction. Meets the interface requirement of
// spvBinaryParse().
spv_result_t SetSpvInst(void* builder, const spv_parsed_instruction_t* inst) {
  reinterpret_cast<ir::IrLoader*>(builder)->AddInstruction(inst);
  return SPV_SUCCESS;
};

}  // annoymous namespace

spv_result_t SpvTools::Assemble(const std::string& text,
                                std::vector<uint32_t>* binary) {
  spv_binary spvbinary = nullptr;
  spv_diagnostic diagnostic = nullptr;

  spv_result_t status = spvTextToBinary(context_, text.data(), text.size(),
                                        &spvbinary, &diagnostic);
  if (status == SPV_SUCCESS) {
    binary->assign(spvbinary->code, spvbinary->code + spvbinary->wordCount);
  }

  spvDiagnosticDestroy(diagnostic);
  spvBinaryDestroy(spvbinary);

  return status;
}

spv_result_t SpvTools::Disassemble(const std::vector<uint32_t>& binary,
                                   std::string* text, uint32_t options) {
  spv_text spvtext = nullptr;
  spv_diagnostic diagnostic = nullptr;

  spv_result_t status = spvBinaryToText(context_, binary.data(), binary.size(),
                                        options, &spvtext, &diagnostic);
  if (status == SPV_SUCCESS) {
    text->assign(spvtext->str, spvtext->str + spvtext->length);
  }

  spvDiagnosticDestroy(diagnostic);
  spvTextDestroy(spvtext);

  return status;
}

std::unique_ptr<ir::Module> SpvTools::BuildModule(
    const std::vector<uint32_t>& binary) {
  spv_diagnostic diagnostic = nullptr;

  auto module = MakeUnique<ir::Module>();
  ir::IrLoader loader(module.get());

  spv_result_t status =
      spvBinaryParse(context_, &loader, binary.data(), binary.size(),
                     SetSpvHeader, SetSpvInst, &diagnostic);

  spvDiagnosticDestroy(diagnostic);

  loader.EndModule();

  if (status == SPV_SUCCESS) return module;
  return nullptr;
}

std::unique_ptr<ir::Module> SpvTools::BuildModule(const std::string& text) {
  std::vector<uint32_t> binary;
  if (Assemble(text, &binary) != SPV_SUCCESS) return nullptr;
  return BuildModule(binary);
}

}  // namespace spvtools
