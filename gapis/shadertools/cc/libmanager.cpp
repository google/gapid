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

#include "GlslangToSpv.h"
#include "spirv-tools/libspirv.hpp"

#include "libmanager.h"

#include <cstring>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

const TBuiltInResource DefaultTBuiltInResource = {
    /* .MaxLights = */ 32,
    /* .MaxClipPlanes = */ 6,
    /* .MaxTextureUnits = */ 32,
    /* .MaxTextureCoords = */ 32,
    /* .MaxVertexAttribs = */ 64,
    /* .MaxVertexUniformComponents = */ 4096,
    /* .MaxVaryingFloats = */ 64,
    /* .MaxVertexTextureImageUnits = */ 32,
    /* .MaxCombinedTextureImageUnits = */ 80,
    /* .MaxTextureImageUnits = */ 32,
    /* .MaxFragmentUniformComponents = */ 4096,
    /* .MaxDrawBuffers = */ 32,
    /* .MaxVertexUniformVectors = */ 128,
    /* .MaxVaryingVectors = */ 8,
    /* .MaxFragmentUniformVectors = */ 16,
    /* .MaxVertexOutputVectors = */ 16,
    /* .MaxFragmentInputVectors = */ 15,
    /* .MinProgramTexelOffset = */ -8,
    /* .MaxProgramTexelOffset = */ 7,
    /* .MaxClipDistances = */ 8,
    /* .MaxComputeWorkGroupCountX = */ 65535,
    /* .MaxComputeWorkGroupCountY = */ 65535,
    /* .MaxComputeWorkGroupCountZ = */ 65535,
    /* .MaxComputeWorkGroupSizeX = */ 1024,
    /* .MaxComputeWorkGroupSizeY = */ 1024,
    /* .MaxComputeWorkGroupSizeZ = */ 64,
    /* .MaxComputeUniformComponents = */ 1024,
    /* .MaxComputeTextureImageUnits = */ 16,
    /* .MaxComputeImageUniforms = */ 8,
    /* .MaxComputeAtomicCounters = */ 8,
    /* .MaxComputeAtomicCounterBuffers = */ 1,
    /* .MaxVaryingComponents = */ 60,
    /* .MaxVertexOutputComponents = */ 64,
    /* .MaxGeometryInputComponents = */ 64,
    /* .MaxGeometryOutputComponents = */ 128,
    /* .MaxFragmentInputComponents = */ 128,
    /* .MaxImageUnits = */ 8,
    /* .MaxCombinedImageUnitsAndFragmentOutputs = */ 8,
    /* .MaxCombinedShaderOutputResources = */ 8,
    /* .MaxImageSamples = */ 0,
    /* .MaxVertexImageUniforms = */ 0,
    /* .MaxTessControlImageUniforms = */ 0,
    /* .MaxTessEvaluationImageUniforms = */ 0,
    /* .MaxGeometryImageUniforms = */ 0,
    /* .MaxFragmentImageUniforms = */ 8,
    /* .MaxCombinedImageUniforms = */ 8,
    /* .MaxGeometryTextureImageUnits = */ 16,
    /* .MaxGeometryOutputVertices = */ 256,
    /* .MaxGeometryTotalOutputComponents = */ 1024,
    /* .MaxGeometryUniformComponents = */ 1024,
    /* .MaxGeometryVaryingComponents = */ 64,
    /* .MaxTessControlInputComponents = */ 128,
    /* .MaxTessControlOutputComponents = */ 128,
    /* .MaxTessControlTextureImageUnits = */ 16,
    /* .MaxTessControlUniformComponents = */ 1024,
    /* .MaxTessControlTotalOutputComponents = */ 4096,
    /* .MaxTessEvaluationInputComponents = */ 128,
    /* .MaxTessEvaluationOutputComponents = */ 128,
    /* .MaxTessEvaluationTextureImageUnits = */ 16,
    /* .MaxTessEvaluationUniformComponents = */ 1024,
    /* .MaxTessPatchComponents = */ 120,
    /* .MaxPatchVertices = */ 32,
    /* .MaxTessGenLevel = */ 64,
    /* .MaxViewports = */ 16,
    /* .MaxVertexAtomicCounters = */ 0,
    /* .MaxTessControlAtomicCounters = */ 0,
    /* .MaxTessEvaluationAtomicCounters = */ 0,
    /* .MaxGeometryAtomicCounters = */ 0,
    /* .MaxFragmentAtomicCounters = */ 8,
    /* .MaxCombinedAtomicCounters = */ 8,
    /* .MaxAtomicCounterBindings = */ 1,
    /* .MaxVertexAtomicCounterBuffers = */ 0,
    /* .MaxTessControlAtomicCounterBuffers = */ 0,
    /* .MaxTessEvaluationAtomicCounterBuffers = */ 0,
    /* .MaxGeometryAtomicCounterBuffers = */ 0,
    /* .MaxFragmentAtomicCounterBuffers = */ 1,
    /* .MaxCombinedAtomicCounterBuffers = */ 1,
    /* .MaxAtomicCounterBufferSize = */ 16384,
    /* .MaxTransformFeedbackBuffers = */ 4,
    /* .MaxTransformFeedbackInterleavedComponents = */ 64,
    /* .MaxCullDistances = */ 8,
    /* .MaxCombinedClipAndCullDistances = */ 8,
    /* .MaxSamples = */ 4,
    /* .maxMeshOutputVerticesNV = */ 256,
    /* .maxMeshOutputPrimitivesNV = */ 512,
    /* .maxMeshWorkGroupSizeX_NV = */ 32,
    /* .maxMeshWorkGroupSizeY_NV = */ 1,
    /* .maxMeshWorkGroupSizeZ_NV = */ 1,
    /* .maxTaskWorkGroupSizeX_NV = */ 32,
    /* .maxTaskWorkGroupSizeY_NV = */ 1,
    /* .maxTaskWorkGroupSizeZ_NV = */ 1,
    /* .maxMeshViewCountNV = */ 4,
    /* .maxDualSourceDrawBuffersEXT = */ 1,
    /* .limits = */
    {
        /* .nonInductiveForLoops = */ 1,
        /* .whileLoops = */ 1,
        /* .doWhileLoops = */ 1,
        /* .generalUniformIndexing = */ 1,
        /* .generalAttributeMatrixVectorIndexing = */ 1,
        /* .generalVaryingIndexing = */ 1,
        /* .generalSamplerIndexing = */ 1,
        /* .generalVariableIndexing = */ 1,
        /* .generalConstantMatrixVectorIndexing = */ 1,
    }};

std::vector<unsigned int> parseGlslang(const char* code, const char* preamble,
                                       std::string* err_msg,
                                       shader_type shader_ty,
                                       client_type client_ty,
                                       bool relaxed_errs) {
  std::vector<unsigned int> spirv;

  EShMessages messages = relaxed_errs ? EShMsgRelaxedErrors : EShMsgDefault;
  EShLanguage lang = EShLangVertex;
  switch (shader_ty) {
    case VERTEX: {
      lang = EShLangVertex;
      break;
    }
    case TESS_CONTROL: {
      lang = EShLangTessControl;
      break;
    }
    case TESS_EVALUATION: {
      lang = EShLangTessEvaluation;
      break;
    }
    case GEOMETRY: {
      lang = EShLangGeometry;
      break;
    }
    case FRAGMENT: {
      lang = EShLangFragment;
      break;
    }
    case COMPUTE: {
      lang = EShLangCompute;
      break;
    }
  }

  glslang::EShClient env_client = glslang::EShClientNone;
  int env_input_version = 100;
  glslang::EshTargetClientVersion env_client_version =
      glslang::EShTargetOpenGL_450;
  int parse_default_version = 100;
  EProfile parse_profile = ECoreProfile;
  // OpenGL:
  //  default version: 330
  //  profile: core
  // OpenGLES:
  //  default version: 100
  //  profile: es
  // Vulkan:
  //  default version: 110
  //  profile: no_profile
  switch (client_ty) {
    case OPENGL: {
      env_client = glslang::EShClientOpenGL;
      env_input_version = 100;
      env_client_version = glslang::EShTargetOpenGL_450;
      parse_default_version = 330;
      parse_profile = ECoreProfile;
      break;
    }
    case OPENGLES: {
      env_client = glslang::EShClientOpenGL;
      env_input_version = 100;
      env_client_version = glslang::EShTargetOpenGL_450;
      parse_default_version = 100;
      parse_profile = EEsProfile;
      break;
    }
    case VULKAN: {
      env_client = glslang::EShClientVulkan;
      env_input_version = 100;
      env_client_version = glslang::EShTargetVulkan_1_0;
      parse_default_version = 110;
      parse_profile = ENoProfile;
      break;
    }
  }

  glslang::InitializeProcess();
  glslang::TShader shader(lang);
  shader.setPreamble(preamble);
  shader.setStrings(&code, 1);
  shader.setAutoMapLocations(true);
  shader.setEnvInput(glslang::EShSourceGlsl, lang, env_client,
                     env_input_version);
  shader.setEnvClient(env_client, env_client_version);
  // HACK for ES: Disabled the call to setEnvTarget() as specifying the SPIRV
  // version currently causes the parser to fail with:
  //   ERROR: #version: ES shaders for OpenGL SPIR-V are not supported.
  //
  // A hacky workaround is to omit this call (which let's it parse) and add in
  // the SPIR-V version code (see below). This is terrible, but works in the
  // interim.
  //
  // Note that for Vulkan, setEnvTarget() must be called.
  if (client_ty == VULKAN) {
    shader.setEnvTarget(glslang::EShTargetSpv, glslang::EShTargetSpv_1_0);
  }

  bool parsed = shader.parse(
      &DefaultTBuiltInResource, parse_default_version, parse_profile,
      false /* force version and profile */, false, /* forward compatible */
      messages);

  if (!parsed) {
    *err_msg += "Compilation failed:\n" + std::string(shader.getInfoLog());
  } else {
    glslang::TProgram program;
    program.addShader(&shader);
    bool linked = program.link(messages);
    if (!linked) {
      *err_msg += "Linking failed:\n" + std::string(program.getInfoLog());
    }
    std::string warningsErrors;
    glslang::GlslangToSpv(*program.getIntermediate(lang), spirv);
  }

  // The compiler initialization is fairly expensive, so keep it initialized
  // indefinitely. glslang::FinalizeProcess();

  // Hack the SPIR-V to add a version to the header
  if (spirv.size() >= 2) {
    spirv[1] = glslang::EShTargetSpv_1_0;
  }

  return spirv;
}

/**
 * Returns pointer to disassemble text.
 **/
const char* getDisassembleText(uint32_t* spirv_binary, size_t length) {
  std::vector<uint32_t> spirv_vec(length);
  for (size_t i = 0; i < length; i++) {
    spirv_vec[i] = spirv_binary[i];
  }

  spvtools::SpirvTools tools(SPV_ENV_VULKAN_1_0);
  std::string disassembly;
  const bool result =
      tools.Disassemble(spirv_vec, &disassembly,
                        (SPV_BINARY_TO_TEXT_OPTION_FRIENDLY_NAMES |
                         SPV_BINARY_TO_TEXT_OPTION_INDENT));

  if (!result) {
    return nullptr;
  }

  char* chars = new char[disassembly.size() + 1];
  strcpy(chars, disassembly.c_str());
  return chars;
}

void deleteDisassembleText(const char* text) {
  if (text) delete[] text;
}

spirv_binary_t* assembleToBinary(const char* text) {
  if (!text) {
    return nullptr;
  }
  spirv_binary_t* binary = new spirv_binary_t{nullptr, 0};
  std::string disassembly(text);
  std::vector<uint32_t> words;
  spvtools::SpirvTools tools(SPV_ENV_VULKAN_1_0);
  const auto result = tools.Assemble(disassembly, &words);
  if (!result) {
    return nullptr;
  }
  binary->words_num = words.size();
  binary->words = new uint32_t[words.size()];
  for (size_t i = 0; i < words.size(); i++) {
    binary->words[i] = words[i];
  }
  return binary;
}

void deleteBinary(spirv_binary_t* binary) {
  if (binary) {
    delete[] binary->words;
  }
  delete binary;
}

glsl_compile_result_t* compileGlsl(const char* code,
                                   const compile_options_t* options) {
  glsl_compile_result_t* result =
      new glsl_compile_result_t{true, nullptr, spirv_binary_t{nullptr, 0}};

  std::string err_msg;
  std::vector<unsigned int> spirv =
      parseGlslang(code, options->preamble, &err_msg, options->shader_type,
                   options->client_type, false);
  if (!err_msg.empty()) {
    result->ok = false;
    result->message = new char[err_msg.length() + 1];
    strcpy(result->message, err_msg.c_str());
  }
  if (spirv.size() > 0) {
    result->binary.words_num = spirv.size();
    result->binary.words = new uint32_t[spirv.size()];
    for (size_t i = 0; i < spirv.size(); i++) {
      result->binary.words[i] = spirv[i];
    }
  }
  return result;
}

void deleteCompileResult(glsl_compile_result_t* result) {
  if (result) {
    delete[] result->message;
    delete[] result->binary.words;
  }
  delete result;
}
