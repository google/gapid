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

#include <cstring>
#include <iostream>
#include <vector>

#include "source/opt/ir_loader.h"
#include "source/opt/libspirv.hpp"
#include "source/opt/pass_manager.h"
#include "tools/io.h"

using namespace spvtools;

void PrintUsage(const char* program) {
  printf(
      R"(%s - Optimize a SPIR-V binary file.

USAGE: %s [options] [<input>] -o <output>

The SPIR-V binary is read from <input>. If no file is specified,
or if <input> is "-", then the binary is read from standard input.
if <output> is "-", then the optimized output is written to
standard output.

NOTE: The optimizer is a work in progress.

Options:
  --strip-debug
               Remove all debug instructions.
  --freeze-spec-const
               Freeze the values of specialization constants to their default
               values.
  --eliminate-dead-const
               Eliminate dead constants.
  --fold-spec-const-op-composite
               Fold the spec constants defined by OpSpecConstantOp or
               OpSpecConstantComposite instructions to front-end constants
               when possible.
  --unify-const
               Remove the duplicated constants.
  -h, --help   Print this help.
  --version    Display optimizer version information.
)",
      program, program);
}

int main(int argc, char** argv) {
  const char* in_file = nullptr;
  const char* out_file = nullptr;

  spv_target_env target_env = SPV_ENV_UNIVERSAL_1_1;

  opt::PassManager pass_manager;

  for (int argi = 1; argi < argc; ++argi) {
    const char* cur_arg = argv[argi];
    if ('-' == cur_arg[0]) {
      if (0 == strcmp(cur_arg, "--version")) {
        printf("%s\n", spvSoftwareVersionDetailsString());
        return 0;
      } else if (0 == strcmp(cur_arg, "--help") || 0 == strcmp(cur_arg, "-h")) {
        PrintUsage(argv[0]);
        return 0;
      } else if (0 == strcmp(cur_arg, "-o")) {
        if (!out_file && argi + 1 < argc) {
          out_file = argv[++argi];
        } else {
          PrintUsage(argv[0]);
          return 1;
        }
      } else if (0 == strcmp(cur_arg, "--strip-debug")) {
        pass_manager.AddPass<opt::StripDebugInfoPass>();
      } else if (0 == strcmp(cur_arg, "--freeze-spec-const")) {
        pass_manager.AddPass<opt::FreezeSpecConstantValuePass>();
      } else if (0 == strcmp(cur_arg, "--eliminate-dead-const")) {
        pass_manager.AddPass<opt::EliminateDeadConstantPass>();
      } else if (0 == strcmp(cur_arg, "--fold-spec-const-op-composite")) {
        pass_manager.AddPass<opt::FoldSpecConstantOpAndCompositePass>();
      } else if (0 == strcmp(cur_arg, "--unify-const")) {
        pass_manager.AddPass<opt::UnifyConstantPass>();
      } else if ('\0' == cur_arg[1]) {
        // Setting a filename of "-" to indicate stdin.
        if (!in_file) {
          in_file = cur_arg;
        } else {
          fprintf(stderr, "error: More than one input file specified\n");
          return 1;
        }
      } else {
        PrintUsage(argv[0]);
        return 1;
      }
    } else {
      if (!in_file) {
        in_file = cur_arg;
      } else {
        fprintf(stderr, "error: More than one input file specified\n");
        return 1;
      }
    }
  }

  if (out_file == nullptr) {
    fprintf(stderr, "error: -o required\n");
    return 1;
  }

  std::vector<uint32_t> source;
  if (!ReadFile<uint32_t>(in_file, "rb", &source)) return 1;

  // Let's do validation first.
  spv_context context = spvContextCreate(target_env);
  spv_diagnostic diagnostic = nullptr;
  spv_const_binary_t binary = {source.data(), source.size()};
  spv_result_t error = spvValidate(context, &binary, &diagnostic);
  if (error) {
    spvDiagnosticPrint(diagnostic);
    spvDiagnosticDestroy(diagnostic);
    spvContextDestroy(context);
    return error;
  }
  spvDiagnosticDestroy(diagnostic);
  spvContextDestroy(context);

  std::unique_ptr<ir::Module> module = SpvTools(target_env).BuildModule(source);
  pass_manager.Run(module.get());

  std::vector<uint32_t> target;
  module->ToBinary(&target, /* skip_nop = */ true);

  if (!WriteFile<uint32_t>(out_file, "wb", target.data(), target.size())) {
    return 1;
  }

  return 0;
}
