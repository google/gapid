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

#include "ext_inst.h"

#include <cassert>
#include <cstring>

#include "spirv/1.0/GLSL.std.450.h"
#include "spirv/1.0/OpenCL.std.h"
#include "spirv_definition.h"

#include "macro.h"

static const spv_ext_inst_desc_t glslStd450Entries_1_0[] = {
#include "glsl.std.450.insts-1.0.inc"
};

static const spv_ext_inst_desc_t openclEntries_1_0[] = {
#include "opencl.std.insts-1.0.inc"
};

spv_result_t spvExtInstTableGet(spv_ext_inst_table* pExtInstTable,
                                spv_target_env env) {
  if (!pExtInstTable) return SPV_ERROR_INVALID_POINTER;

  static const spv_ext_inst_group_t groups_1_0[] = {
      {SPV_EXT_INST_TYPE_GLSL_STD_450, ARRAY_SIZE(glslStd450Entries_1_0),
       glslStd450Entries_1_0},
      {SPV_EXT_INST_TYPE_OPENCL_STD, ARRAY_SIZE(openclEntries_1_0),
       openclEntries_1_0},
  };

  static const spv_ext_inst_table_t table_1_0 = {ARRAY_SIZE(groups_1_0),
                                                 groups_1_0};

  switch (env) {
    // The extended instruction sets are all version 1.0 so far.
    case SPV_ENV_UNIVERSAL_1_0:
    case SPV_ENV_VULKAN_1_0:
    case SPV_ENV_UNIVERSAL_1_1:
    case SPV_ENV_OPENCL_2_1:
    case SPV_ENV_OPENCL_2_2:
    case SPV_ENV_OPENGL_4_0:
    case SPV_ENV_OPENGL_4_1:
    case SPV_ENV_OPENGL_4_2:
    case SPV_ENV_OPENGL_4_3:
    case SPV_ENV_OPENGL_4_5:
      *pExtInstTable = &table_1_0;
      return SPV_SUCCESS;
    default:
      assert(0 && "Unknown spv_target_env in spvExtInstTableGet()");
      return SPV_ERROR_INVALID_TABLE;
  }
}

spv_ext_inst_type_t spvExtInstImportTypeGet(const char* name) {
  // The names are specified by the respective extension instruction
  // specifications.
  if (!strcmp("GLSL.std.450", name)) {
    return SPV_EXT_INST_TYPE_GLSL_STD_450;
  }
  if (!strcmp("OpenCL.std", name)) {
    return SPV_EXT_INST_TYPE_OPENCL_STD;
  }
  return SPV_EXT_INST_TYPE_NONE;
}

spv_result_t spvExtInstTableNameLookup(const spv_ext_inst_table table,
                                       const spv_ext_inst_type_t type,
                                       const char* name,
                                       spv_ext_inst_desc* pEntry) {
  if (!table) return SPV_ERROR_INVALID_TABLE;
  if (!pEntry) return SPV_ERROR_INVALID_POINTER;

  for (uint32_t groupIndex = 0; groupIndex < table->count; groupIndex++) {
    const auto& group = table->groups[groupIndex];
    if (type != group.type) continue;
    for (uint32_t index = 0; index < group.count; index++) {
      const auto& entry = group.entries[index];
      if (!strcmp(name, entry.name)) {
        *pEntry = &entry;
        return SPV_SUCCESS;
      }
    }
  }

  return SPV_ERROR_INVALID_LOOKUP;
}

spv_result_t spvExtInstTableValueLookup(const spv_ext_inst_table table,
                                        const spv_ext_inst_type_t type,
                                        const uint32_t value,
                                        spv_ext_inst_desc* pEntry) {
  if (!table) return SPV_ERROR_INVALID_TABLE;
  if (!pEntry) return SPV_ERROR_INVALID_POINTER;

  for (uint32_t groupIndex = 0; groupIndex < table->count; groupIndex++) {
    const auto& group = table->groups[groupIndex];
    if (type != group.type) continue;
    for (uint32_t index = 0; index < group.count; index++) {
      const auto& entry = group.entries[index];
      if (value == entry.ext_inst) {
        *pEntry = &entry;
        return SPV_SUCCESS;
      }
    }
  }

  return SPV_ERROR_INVALID_LOOKUP;
}
