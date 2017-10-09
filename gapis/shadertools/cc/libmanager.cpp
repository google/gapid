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

#include "third_party/glslang/SPIRV/GlslangToSpv.h"
#include "third_party/glslang/SPIRV/disassemble.h"
#include "third_party/glslang/glslang/Public/ShaderLang.h"

#include "spv_manager.h"
#include "libmanager.h"
#include "spirv2glsl.h"

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
    /* .limits = */ {
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

void set_error_msg(code_with_debug_info_t* x, std::string msg) {
  x->ok = false;
  x->message = new char[msg.length() + 1];
  strcpy(x->message, msg.c_str());
}

std::vector<unsigned int> parseGlslang(const char* code, const char* preamble,
    std::string* err_msg, shader_type type, bool es_profile, bool relaxed_errs) {
  std::vector<unsigned int> spirv;

  EShMessages messages = relaxed_errs ? EShMsgRelaxedErrors : EShMsgDefault;
  EShLanguage lang = EShLangVertex;
  switch (type) {
    case VERTEX: { lang = EShLangVertex; break; }
    case TESS_CONTROL: { lang = EShLangTessControl; break; }
    case TESS_EVALUATION: { lang = EShLangTessEvaluation; break; }
    case GEOMETRY: { lang = EShLangGeometry; break; }
    case FRAGMENT: { lang = EShLangFragment; break; }
    case COMPUTE: { lang = EShLangCompute; break; }
  }

  glslang::InitializeProcess();
  glslang::TShader shader(lang);
  shader.setPreamble(preamble);
  shader.setStrings(&code, 1);
  // use 100 for ES environment, 330 for desktop
  int default_version = es_profile ? 100 : 330;
  EProfile profile = es_profile ? EEsProfile : ECoreProfile;
  bool parsed = shader.parse(&DefaultTBuiltInResource, default_version, profile,
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
    spv::SpvBuildLogger logger;
    glslang::GlslangToSpv(*program.getIntermediate(lang), spirv, &logger);
  }

  // The compiler initialization is fairly expensive, so keep it initialized indefinitely.
  // glslang::FinalizeProcess();

  return spirv;
}

/**
 * Only Vertex and Fragment shaders are supported.
 * 1. Compiles source code to spirv using glslang,
 * 2. Changes spirv code to insert debug information using SpvManager,
 * 3. Decompiles changed spirv to source code using spirv-cross,
 * 4. Check, if changed source code correctly compiles.
 **/
code_with_debug_info_t* convertGlsl(const char* input, size_t length, const options_t* options) {
  code_with_debug_info_t* result = new code_with_debug_info_t{};
  std::string err_msg;

  std::vector<unsigned int> spirv = parseGlslang(
      input, options->preamble, &err_msg, options->shader_type, true, options->relaxed);

  if (!err_msg.empty()) {
    set_error_msg(result, "Failed to parse original source code:\n" + err_msg);
    return result;
  }

  // makes changes
  spvmanager::SpvManager my_manager(spirv);
  if (options->prefix_names) {
    if (options->names_prefix) {
      my_manager.mapDeclarationNames(options->names_prefix);
    } else {
      my_manager.mapDeclarationNames();
    }
  }
  if (options->add_outputs_for_inputs) {
    if (options->output_prefix) {
      my_manager.addOutputForInputs(options->output_prefix);
    } else {
      my_manager.addOutputForInputs();
    }
  }
  if (options->make_debuggable) {
    my_manager.makeSpvDebuggable();
  }
  my_manager.renameViewIndex();
  my_manager.removeLayoutLocations();
  my_manager.initLocals();

  std::vector<unsigned int> spirv_new = my_manager.getSpvBinary();

  if (spirv_new.empty()) {
    set_error_msg(result, "SpvManager did not produce any code.");
    return result;
  }

  if (options->disassemble) {
    std::stringstream disassembly_stream;
    spv::Disassemble(disassembly_stream, spirv_new);
    const std::string& tmp = disassembly_stream.str();
    result->disassembly_string = new char[tmp.length() + 1];
    strcpy(result->disassembly_string, tmp.c_str());
  }

  std::string source = spirv2glsl(std::move(spirv_new), options->strip_optimizations);

  result->source_code = new char[source.length() + 1];
  strcpy(result->source_code, source.c_str());
  result->source_code[source.length()] = '\0';

  // check if changed source code compiles again
  if (options->check_after_changes) {
    parseGlslang(result->source_code, nullptr, &err_msg, options->shader_type, false, false);
  }

  if (!err_msg.empty()) {
    set_error_msg(result, "Failed to parse modified source code:\n" + err_msg);
    return result;
  }

  result->info = my_manager.getDebugInstructions();
  result->ok = true;

  return result;
}

/**
 * Releses memory allocated by SpvManager.
 * May needs update after changes.
 **/
void deleteGlslCodeWithDebug(code_with_debug_info_t* debug) {
  delete debug->message;
  delete[] debug->source_code;
  delete[] debug->disassembly_string;

  if (debug->info) {
    for (int i = 0; i < debug->info->insts_num; i++) {
      delete[] debug->info->insts[i].words;
      delete[] debug->info->insts[i].name;
    }
    delete[] debug->info->insts;
    delete debug->info;
  }

  delete debug;
}

/**
 * Returns pointer to disassemble text.
 **/
const char* getDisassembleText(uint32_t* spirv_binary, size_t length) {
  std::vector<uint32_t> spirv_vec(length);
  for (int i = 0; i < length; i++){
    spirv_vec[i] = spirv_binary[i];
  }

  spvtools::SpirvTools tools(SPV_ENV_VULKAN_1_0);
  std::string disassembly;
  const bool result = tools.Disassemble(
    spirv_vec, &disassembly,
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
  if (text)
    delete[] text;
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

const char* opcodeToString(uint32_t opcode) {
  return spvOpcodeString(static_cast<SpvOp>(opcode));
}
