// Copyright (c) 2015-2016 The Khronos Group Inc.
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

#include <cassert>
#include <cstdio>
#include <cstring>
#include <vector>

#include "source/spirv_target_env.h"
#include "spirv-tools/libspirv.h"
#include "tools/io.h"

void print_usage(char* argv0) {
  printf(
      R"(%s - Validate a SPIR-V binary file.

USAGE: %s [options] [<filename>]

The SPIR-V binary is read from <filename>. If no file is specified,
or if the filename is "-", then the binary is read from standard input.

NOTE: The validator is a work in progress.

Options:
  -h, --help   Print this help.
  --version    Display validator version information.
  --target-env {vulkan1.0|spv1.0|spv1.1}
               Use Vulkan1.0/SPIR-V1.0/SPIR-V1.1 validation rules.
)",
      argv0, argv0);
}

int main(int argc, char** argv) {
  const char* inFile = nullptr;
  spv_target_env target_env = SPV_ENV_UNIVERSAL_1_1;

  for (int argi = 1; argi < argc; ++argi) {
    const char* cur_arg = argv[argi];
    if ('-' == cur_arg[0]) {
      if (0 == strcmp(cur_arg, "--version")) {
        printf("%s\n", spvSoftwareVersionDetailsString());
        printf("Targets:\n  %s\n  %s\n",
               spvTargetEnvDescription(SPV_ENV_UNIVERSAL_1_1),
               spvTargetEnvDescription(SPV_ENV_VULKAN_1_0));
        return 0;
      } else if (0 == strcmp(cur_arg, "--help") || 0 == strcmp(cur_arg, "-h")) {
        print_usage(argv[0]);
        return 0;
      } else if (0 == strcmp(cur_arg, "--target-env")) {
        if (argi + 1 < argc) {
          const auto env_str = argv[++argi];
          if (!spvParseTargetEnv(env_str, &target_env)) {
            fprintf(stderr, "error: Unrecognized target env: %s\n", env_str);
            return 1;
          }
        } else {
          fprintf(stderr, "error: Missing argument to --target-env\n");
          return 1;
        }
      } else if (0 == cur_arg[1]) {
        // Setting a filename of "-" to indicate stdin.
        if (!inFile) {
          inFile = cur_arg;
        } else {
          fprintf(stderr, "error: More than one input file specified\n");
          return 1;
        }

      } else {
        print_usage(argv[0]);
        return 1;
      }
    } else {
      if (!inFile) {
        inFile = cur_arg;
      } else {
        fprintf(stderr, "error: More than one input file specified\n");
        return 1;
      }
    }
  }

  std::vector<uint32_t> contents;
  if (!ReadFile<uint32_t>(inFile, "rb", &contents)) return 1;

  spv_const_binary_t binary = {contents.data(), contents.size()};

  spv_diagnostic diagnostic = nullptr;
  spv_context context = spvContextCreate(target_env);
  spv_result_t error = spvValidate(context, &binary, &diagnostic);
  spvContextDestroy(context);
  if (error) {
    spvDiagnosticPrint(diagnostic);
    spvDiagnosticDestroy(diagnostic);
    return error;
  }

  return 0;
}
