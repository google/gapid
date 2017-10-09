/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "spirv2glsl.h"

// This will include copy of SPIRV headers form SPIRV-Cross.
// The version might not exactly match the one in SPRTV-Tools,
// so it is important we never include both at the same time.
#include "third_party/SPIRV-Cross/spirv_glsl.hpp"

std::string spirv2glsl(std::vector<uint32_t> spirv, bool strip_optimizations) {
  spirv_cross::CompilerGLSL glsl(std::move(spirv));
  spirv_cross::CompilerGLSL::Options cross_options;
  cross_options.version = 330;
  cross_options.es = false;
  cross_options.force_temporary = false;
  cross_options.vertex.fixup_clipspace = false;
  glsl.set_options(cross_options);
  if (strip_optimizations) {
    glsl.unset_execution_mode(spv::ExecutionModeEarlyFragmentTests);
  }
  return glsl.compile();
}
