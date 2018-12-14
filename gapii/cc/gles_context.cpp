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

#include "gapii/cc/gles_spy.h"

#include <iostream>
#include <sstream>
#include <string>

#define GET_SHADER_PRECISION_FORMAT(shader_type, precision_type, format) \
  do {                                                                   \
    mImports.glGetShaderPrecisionFormat(shader_type, precision_type,     \
                                        &(format).mMinRange,             \
                                        &(format).mPrecision);           \
    if (uint32_t err = mImports.glGetError()) {                          \
      GAPID_WARNING("glGetShaderPrecisionFormat(" #shader_type           \
                    ", " #precision_type                                 \
                    ") "                                                 \
                    "gave error 0x%x",                                   \
                    err);                                                \
    }                                                                    \
  } while (false)

#define GET(func, name, ...)                                   \
  do {                                                         \
    mImports.func(name, __VA_ARGS__);                          \
    if (uint32_t err = mImports.glGetError()) {                \
      GAPID_WARNING(#func "(" #name ") gave error 0x%x", err); \
    }                                                          \
  } while (false)

#define GET_STRING(name, out)                                            \
  do {                                                                   \
    auto str = mImports.glGetString(name);                               \
    if (uint32_t err = mImports.glGetError()) {                          \
      GAPID_WARNING("glGetString(" #name ") gave error 0x%x", err);      \
    } else if (str == nullptr) {                                         \
      GAPID_WARNING("glGetString(" #name ") returned null w/o error");   \
    } else {                                                             \
      *out = gapil::String(arena(), reinterpret_cast<const char*>(str)); \
    }                                                                    \
  } while (false)

namespace gapii {

using namespace gapii::GLenum;

void GlesSpy::getContextConstants(Constants& out) {
  // Get essential constants which we always need regardless of version.
  GET_STRING(GL_RENDERER, &out.mRenderer);
  GET_STRING(GL_SHADING_LANGUAGE_VERSION, &out.mShadingLanguageVersion);
  GET_STRING(GL_VENDOR, &out.mVendor);
  GET_STRING(GL_VERSION, &out.mVersion);

  GLint major_version = 0;
  GLint minor_version = 0;
  if (out.mShadingLanguageVersion.length() == 0) {
    major_version = 1;
  } else {
    mImports.glGetError();  // Clear error state.
    mImports.glGetIntegerv(GL_MAJOR_VERSION, &major_version);
    mImports.glGetIntegerv(GL_MINOR_VERSION, &minor_version);
    if (mImports.glGetError() != GL_NO_ERROR) {
      // GL_MAJOR_VERSION/GL_MINOR_VERSION were introduced in GLES 3.0,
      // so if the commands returned error we assume it is GLES 2.0.
      major_version = 2;
      minor_version = 0;
    }
  }
  out.mMajorVersion = major_version;
  out.mMinorVersion = minor_version;

  if (major_version >= 3) {
    int32_t c;
    GET(glGetIntegerv, GL_NUM_EXTENSIONS, &c);
    for (int32_t i = 0; i < c; i++) {
      auto ext = reinterpret_cast<const char*>(
          mImports.glGetStringi(GL_EXTENSIONS, i));
      out.mExtensions[i] = gapil::String(arena(), ext);
      if (uint32_t err = mImports.glGetError()) {
        GAPID_WARNING("glGetStringi(GL_EXTENSIONS, %d) gave error 0x%x", i,
                      err);
      }
    }
  } else {
    std::string extensions =
        reinterpret_cast<const char*>(mImports.glGetString(GL_EXTENSIONS));
    if (uint32_t err = mImports.glGetError()) {
      GAPID_WARNING("glGetString(GL_EXTENSIONS) gave error 0x%x", err);
    }

    std::istringstream iss(extensions);
    std::string extension;
    while (std::getline(iss, extension, ' ')) {
      out.mExtensions[out.mExtensions.count()] =
          gapil::String(arena(), extension.c_str());
    }
  }

  bool gles20 = major_version >= 2;
  bool gles30 =
      (major_version > 3) || (major_version == 3 && minor_version >= 0);
  bool gles31 =
      (major_version > 3) || (major_version == 3 && minor_version >= 1);
  bool gles32 =
      (major_version > 3) || (major_version == 3 && minor_version >= 2);

  // Constants defined in version 2.0.25 (November 2, 2010)
  if (gles20) {
    GET(glGetFloatv, GL_ALIASED_LINE_WIDTH_RANGE, out.mAliasedLineWidthRange);
    GET(glGetFloatv, GL_ALIASED_POINT_SIZE_RANGE, out.mAliasedPointSizeRange);
    GET(glGetIntegerv, GL_MAX_COMBINED_TEXTURE_IMAGE_UNITS,
        &out.mMaxCombinedTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_CUBE_MAP_TEXTURE_SIZE,
        &out.mMaxCubeMapTextureSize);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_UNIFORM_VECTORS,
        &out.mMaxFragmentUniformVectors);
    GET(glGetIntegerv, GL_MAX_RENDERBUFFER_SIZE, &out.mMaxRenderbufferSize);
    GET(glGetIntegerv, GL_MAX_TEXTURE_IMAGE_UNITS, &out.mMaxTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_TEXTURE_SIZE, &out.mMaxTextureSize);
    GET(glGetIntegerv, GL_MAX_VARYING_VECTORS, &out.mMaxVaryingVectors);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATTRIBS, &out.mMaxVertexAttribs);
    GET(glGetIntegerv, GL_MAX_VERTEX_TEXTURE_IMAGE_UNITS,
        &out.mMaxVertexTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_VERTEX_UNIFORM_VECTORS,
        &out.mMaxVertexUniformVectors);
    GET(glGetIntegerv, GL_MAX_VIEWPORT_DIMS, out.mMaxViewportDims);
    GET(glGetBooleanv, GL_SHADER_COMPILER, &out.mShaderCompiler);
    GET(glGetIntegerv, GL_SUBPIXEL_BITS, &out.mSubpixelBits);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_LOW_FLOAT,
                                out.mVertexShaderPrecisionFormat.mLowFloat);
    GET_SHADER_PRECISION_FORMAT(GL_FRAGMENT_SHADER, GL_LOW_FLOAT,
                                out.mFragmentShaderPrecisionFormat.mLowFloat);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_MEDIUM_FLOAT,
                                out.mVertexShaderPrecisionFormat.mMediumFloat);
    GET_SHADER_PRECISION_FORMAT(
        GL_FRAGMENT_SHADER, GL_MEDIUM_FLOAT,
        out.mFragmentShaderPrecisionFormat.mMediumFloat);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_HIGH_FLOAT,
                                out.mVertexShaderPrecisionFormat.mHighFloat);
    GET_SHADER_PRECISION_FORMAT(GL_FRAGMENT_SHADER, GL_HIGH_FLOAT,
                                out.mFragmentShaderPrecisionFormat.mHighFloat);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_LOW_INT,
                                out.mVertexShaderPrecisionFormat.mLowInt);
    GET_SHADER_PRECISION_FORMAT(GL_FRAGMENT_SHADER, GL_LOW_INT,
                                out.mFragmentShaderPrecisionFormat.mLowInt);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_MEDIUM_INT,
                                out.mVertexShaderPrecisionFormat.mMediumInt);
    GET_SHADER_PRECISION_FORMAT(GL_FRAGMENT_SHADER, GL_MEDIUM_INT,
                                out.mFragmentShaderPrecisionFormat.mMediumInt);
    GET_SHADER_PRECISION_FORMAT(GL_VERTEX_SHADER, GL_HIGH_INT,
                                out.mVertexShaderPrecisionFormat.mHighInt);
    GET_SHADER_PRECISION_FORMAT(GL_FRAGMENT_SHADER, GL_HIGH_INT,
                                out.mFragmentShaderPrecisionFormat.mHighInt);

    GLint count = 0;
    std::vector<GLint> buf;
    GET(glGetIntegerv, GL_NUM_COMPRESSED_TEXTURE_FORMATS, &count);
    buf.resize(count);
    GET(glGetIntegerv, GL_COMPRESSED_TEXTURE_FORMATS, &buf[0]);
    for (GLint i = 0; i < count; i++) {
      out.mCompressedTextureFormats[i] = buf[i];
    }

    count = 0;
    GET(glGetIntegerv, GL_NUM_SHADER_BINARY_FORMATS, &count);
    buf.resize(count);
    GET(glGetIntegerv, GL_SHADER_BINARY_FORMATS, &buf[0]);
    for (GLint i = 0; i < count; i++) {
      out.mShaderBinaryFormats[i] = buf[i];
    }
  }

  // Constants defined in version 3.0.4 (August 27, 2014)
  if (gles30) {
    GET(glGetIntegerv, GL_MAX_3D_TEXTURE_SIZE, &out.mMax3dTextureSize);
    GET(glGetIntegerv, GL_MAX_ARRAY_TEXTURE_LAYERS,
        &out.mMaxArrayTextureLayers);
    GET(glGetIntegerv, GL_MAX_COLOR_ATTACHMENTS, &out.mMaxColorAttachments);
    GET(glGetInteger64v, GL_MAX_COMBINED_FRAGMENT_UNIFORM_COMPONENTS,
        &out.mMaxCombinedFragmentUniformComponents);
    GET(glGetIntegerv, GL_MAX_COMBINED_UNIFORM_BLOCKS,
        &out.mMaxCombinedUniformBlocks);
    GET(glGetInteger64v, GL_MAX_COMBINED_VERTEX_UNIFORM_COMPONENTS,
        &out.mMaxCombinedVertexUniformComponents);
    GET(glGetIntegerv, GL_MAX_DRAW_BUFFERS, &out.mMaxDrawBuffers);
    GET(glGetIntegerv, GL_MAX_ELEMENTS_INDICES, &out.mMaxElementsIndices);
    GET(glGetIntegerv, GL_MAX_ELEMENTS_VERTICES, &out.mMaxElementsVertices);
    GET(glGetInteger64v, GL_MAX_ELEMENT_INDEX, &out.mMaxElementIndex);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_INPUT_COMPONENTS,
        &out.mMaxFragmentInputComponents);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_UNIFORM_BLOCKS,
        &out.mMaxFragmentUniformBlocks);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_UNIFORM_COMPONENTS,
        &out.mMaxFragmentUniformComponents);
    GET(glGetIntegerv, GL_MAX_PROGRAM_TEXEL_OFFSET,
        &out.mMaxProgramTexelOffset);
    GET(glGetInteger64v, GL_MAX_SERVER_WAIT_TIMEOUT,
        &out.mMaxServerWaitTimeout);
    GET(glGetFloatv, GL_MAX_TEXTURE_LOD_BIAS, &out.mMaxTextureLodBias);
    GET(glGetIntegerv, GL_MAX_TRANSFORM_FEEDBACK_INTERLEAVED_COMPONENTS,
        &out.mMaxTransformFeedbackInterleavedComponents);
    GET(glGetIntegerv, GL_MAX_TRANSFORM_FEEDBACK_SEPARATE_ATTRIBS,
        &out.mMaxTransformFeedbackSeparateAttribs);
    GET(glGetIntegerv, GL_MAX_TRANSFORM_FEEDBACK_SEPARATE_COMPONENTS,
        &out.mMaxTransformFeedbackSeparateComponents);
    GET(glGetInteger64v, GL_MAX_UNIFORM_BLOCK_SIZE, &out.mMaxUniformBlockSize);
    GET(glGetIntegerv, GL_MAX_UNIFORM_BUFFER_BINDINGS,
        &out.mMaxUniformBufferBindings);
    GET(glGetIntegerv, GL_MAX_VARYING_COMPONENTS, &out.mMaxVaryingComponents);
    GET(glGetIntegerv, GL_MAX_VERTEX_OUTPUT_COMPONENTS,
        &out.mMaxVertexOutputComponents);
    GET(glGetIntegerv, GL_MAX_VERTEX_UNIFORM_BLOCKS,
        &out.mMaxVertexUniformBlocks);
    GET(glGetIntegerv, GL_MAX_VERTEX_UNIFORM_COMPONENTS,
        &out.mMaxVertexUniformComponents);
    GET(glGetIntegerv, GL_MIN_PROGRAM_TEXEL_OFFSET,
        &out.mMinProgramTexelOffset);
    GET(glGetIntegerv, GL_UNIFORM_BUFFER_OFFSET_ALIGNMENT,
        &out.mUniformBufferOffsetAlignment);

    GLint count = 0;
    std::vector<GLint> buf;
    GET(glGetIntegerv, GL_NUM_PROGRAM_BINARY_FORMATS, &count);
    buf.resize(count);
    GET(glGetIntegerv, GL_PROGRAM_BINARY_FORMATS, &buf[0]);
    for (GLint i = 0; i < count; i++) {
      out.mProgramBinaryFormats[i] = buf[i];
    }
  }

  // Constants defined in version 3.1 (April 29, 2015)
  if (gles31) {
    GET(glGetIntegerv, GL_MAX_ATOMIC_COUNTER_BUFFER_BINDINGS,
        &out.mMaxAtomicCounterBufferBindings);
    GET(glGetIntegerv, GL_MAX_ATOMIC_COUNTER_BUFFER_SIZE,
        &out.mMaxAtomicCounterBufferSize);
    GET(glGetIntegerv, GL_MAX_COLOR_TEXTURE_SAMPLES,
        &out.mMaxColorTextureSamples);
    GET(glGetIntegerv, GL_MAX_COMBINED_ATOMIC_COUNTERS,
        &out.mMaxCombinedAtomicCounters);
    GET(glGetIntegerv, GL_MAX_COMBINED_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxCombinedAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_COMBINED_COMPUTE_UNIFORM_COMPONENTS,
        &out.mMaxCombinedComputeUniformComponents);
    GET(glGetIntegerv, GL_MAX_COMBINED_IMAGE_UNIFORMS,
        &out.mMaxCombinedImageUniforms);
    GET(glGetIntegerv, GL_MAX_COMBINED_SHADER_OUTPUT_RESOURCES,
        &out.mMaxCombinedShaderOutputResources);
    GET(glGetIntegerv, GL_MAX_COMBINED_SHADER_STORAGE_BLOCKS,
        &out.mMaxCombinedShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_COMPUTE_ATOMIC_COUNTERS,
        &out.mMaxComputeAtomicCounters);
    GET(glGetIntegerv, GL_MAX_COMPUTE_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxComputeAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_COMPUTE_IMAGE_UNIFORMS,
        &out.mMaxComputeImageUniforms);
    GET(glGetIntegerv, GL_MAX_COMPUTE_SHADER_STORAGE_BLOCKS,
        &out.mMaxComputeShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_COMPUTE_SHARED_MEMORY_SIZE,
        &out.mMaxComputeSharedMemorySize);
    GET(glGetIntegerv, GL_MAX_COMPUTE_TEXTURE_IMAGE_UNITS,
        &out.mMaxComputeTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_COMPUTE_UNIFORM_BLOCKS,
        &out.mMaxComputeUniformBlocks);
    GET(glGetIntegerv, GL_MAX_COMPUTE_UNIFORM_COMPONENTS,
        &out.mMaxComputeUniformComponents);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_COUNT, 0,
        &out.mMaxComputeWorkGroupCount[0]);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_COUNT, 1,
        &out.mMaxComputeWorkGroupCount[1]);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_COUNT, 2,
        &out.mMaxComputeWorkGroupCount[2]);
    GET(glGetIntegerv, GL_MAX_COMPUTE_WORK_GROUP_INVOCATIONS,
        &out.mMaxComputeWorkGroupInvocations);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_SIZE, 0,
        &out.mMaxComputeWorkGroupSize[0]);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_SIZE, 1,
        &out.mMaxComputeWorkGroupSize[1]);
    GET(glGetIntegeri_v, GL_MAX_COMPUTE_WORK_GROUP_SIZE, 2,
        &out.mMaxComputeWorkGroupSize[2]);
    GET(glGetIntegerv, GL_MAX_DEPTH_TEXTURE_SAMPLES,
        &out.mMaxDepthTextureSamples);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_ATOMIC_COUNTERS,
        &out.mMaxFragmentAtomicCounters);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxFragmentAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_IMAGE_UNIFORMS,
        &out.mMaxFragmentImageUniforms);
    GET(glGetIntegerv, GL_MAX_FRAGMENT_SHADER_STORAGE_BLOCKS,
        &out.mMaxFragmentShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_FRAMEBUFFER_HEIGHT, &out.mMaxFramebufferHeight);
    GET(glGetIntegerv, GL_MAX_FRAMEBUFFER_SAMPLES, &out.mMaxFramebufferSamples);
    GET(glGetIntegerv, GL_MAX_FRAMEBUFFER_WIDTH, &out.mMaxFramebufferWidth);
    GET(glGetIntegerv, GL_MAX_IMAGE_UNITS, &out.mMaxImageUnits);
    GET(glGetIntegerv, GL_MAX_INTEGER_SAMPLES, &out.mMaxIntegerSamples);
    GET(glGetIntegerv, GL_MAX_PROGRAM_TEXTURE_GATHER_OFFSET,
        &out.mMaxProgramTextureGatherOffset);
    GET(glGetIntegerv, GL_MAX_SAMPLE_MASK_WORDS, &out.mMaxSampleMaskWords);
    GET(glGetInteger64v, GL_MAX_SHADER_STORAGE_BLOCK_SIZE,
        &out.mMaxShaderStorageBlockSize);
    GET(glGetIntegerv, GL_MAX_SHADER_STORAGE_BUFFER_BINDINGS,
        &out.mMaxShaderStorageBufferBindings);
    GET(glGetIntegerv, GL_MAX_UNIFORM_LOCATIONS, &out.mMaxUniformLocations);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATOMIC_COUNTERS,
        &out.mMaxVertexAtomicCounters);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxVertexAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATTRIB_BINDINGS,
        &out.mMaxVertexAttribBindings);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATTRIB_RELATIVE_OFFSET,
        &out.mMaxVertexAttribRelativeOffset);
    GET(glGetIntegerv, GL_MAX_VERTEX_ATTRIB_STRIDE,
        &out.mMaxVertexAttribStride);
    GET(glGetIntegerv, GL_MAX_VERTEX_IMAGE_UNIFORMS,
        &out.mMaxVertexImageUniforms);
    GET(glGetIntegerv, GL_MAX_VERTEX_SHADER_STORAGE_BLOCKS,
        &out.mMaxVertexShaderStorageBlocks);
    GET(glGetIntegerv, GL_MIN_PROGRAM_TEXTURE_GATHER_OFFSET,
        &out.mMinProgramTextureGatherOffset);
    GET(glGetIntegerv, GL_SHADER_STORAGE_BUFFER_OFFSET_ALIGNMENT,
        &out.mShaderStorageBufferOffsetAlignment);
  }

  // Constants defined in version 3.2 (June 15, 2016)
  if (gles32) {
    GET(glGetIntegerv, GL_CONTEXT_FLAGS, &out.mContextFlags);
    GET(glGetIntegerv, GL_FRAGMENT_INTERPOLATION_OFFSET_BITS,
        &out.mFragmentInterpolationOffsetBits);
    GET(glGetIntegerv, GL_LAYER_PROVOKING_VERTEX,
        reinterpret_cast<GLint*>(&out.mLayerProvokingVertex));
    GET(glGetIntegerv, GL_MAX_COMBINED_GEOMETRY_UNIFORM_COMPONENTS,
        &out.mMaxCombinedGeometryUniformComponents);
    GET(glGetIntegerv, GL_MAX_COMBINED_TESS_CONTROL_UNIFORM_COMPONENTS,
        &out.mMaxCombinedTessControlUniformComponents);
    GET(glGetIntegerv, GL_MAX_COMBINED_TESS_EVALUATION_UNIFORM_COMPONENTS,
        &out.mMaxCombinedTessEvaluationUniformComponents);
    GET(glGetIntegerv, GL_MAX_DEBUG_GROUP_STACK_DEPTH,
        &out.mMaxDebugGroupStackDepth);
    GET(glGetIntegerv, GL_MAX_DEBUG_LOGGED_MESSAGES,
        &out.mMaxDebugLoggedMessages);
    GET(glGetIntegerv, GL_MAX_DEBUG_MESSAGE_LENGTH,
        &out.mMaxDebugMessageLength);
    GET(glGetFloatv, GL_MAX_FRAGMENT_INTERPOLATION_OFFSET,
        &out.mMaxFragmentInterpolationOffset);
    GET(glGetIntegerv, GL_MAX_FRAMEBUFFER_LAYERS, &out.mMaxFramebufferLayers);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_ATOMIC_COUNTERS,
        &out.mMaxGeometryAtomicCounters);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxGeometryAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_IMAGE_UNIFORMS,
        &out.mMaxGeometryImageUniforms);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_INPUT_COMPONENTS,
        &out.mMaxGeometryInputComponents);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_OUTPUT_COMPONENTS,
        &out.mMaxGeometryOutputComponents);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_OUTPUT_VERTICES,
        &out.mMaxGeometryOutputVertices);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_SHADER_INVOCATIONS,
        &out.mMaxGeometryShaderInvocations);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_SHADER_STORAGE_BLOCKS,
        &out.mMaxGeometryShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_TEXTURE_IMAGE_UNITS,
        &out.mMaxGeometryTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_TOTAL_OUTPUT_COMPONENTS,
        &out.mMaxGeometryTotalOutputComponents);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_UNIFORM_BLOCKS,
        &out.mMaxGeometryUniformBlocks);
    GET(glGetIntegerv, GL_MAX_GEOMETRY_UNIFORM_COMPONENTS,
        &out.mMaxGeometryUniformComponents);
    GET(glGetIntegerv, GL_MAX_LABEL_LENGTH, &out.mMaxLabelLength);
    GET(glGetIntegerv, GL_MAX_PATCH_VERTICES, &out.mMaxPatchVertices);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_ATOMIC_COUNTERS,
        &out.mMaxTessControlAtomicCounters);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxTessControlAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_IMAGE_UNIFORMS,
        &out.mMaxTessControlImageUniforms);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_INPUT_COMPONENTS,
        &out.mMaxTessControlInputComponents);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_OUTPUT_COMPONENTS,
        &out.mMaxTessControlOutputComponents);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_SHADER_STORAGE_BLOCKS,
        &out.mMaxTessControlShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_TEXTURE_IMAGE_UNITS,
        &out.mMaxTessControlTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_TOTAL_OUTPUT_COMPONENTS,
        &out.mMaxTessControlTotalOutputComponents);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_UNIFORM_BLOCKS,
        &out.mMaxTessControlUniformBlocks);
    GET(glGetIntegerv, GL_MAX_TESS_CONTROL_UNIFORM_COMPONENTS,
        &out.mMaxTessControlUniformComponents);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_ATOMIC_COUNTERS,
        &out.mMaxTessEvaluationAtomicCounters);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_ATOMIC_COUNTER_BUFFERS,
        &out.mMaxTessEvaluationAtomicCounterBuffers);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_IMAGE_UNIFORMS,
        &out.mMaxTessEvaluationImageUniforms);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_INPUT_COMPONENTS,
        &out.mMaxTessEvaluationInputComponents);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_OUTPUT_COMPONENTS,
        &out.mMaxTessEvaluationOutputComponents);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_SHADER_STORAGE_BLOCKS,
        &out.mMaxTessEvaluationShaderStorageBlocks);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_TEXTURE_IMAGE_UNITS,
        &out.mMaxTessEvaluationTextureImageUnits);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_UNIFORM_BLOCKS,
        &out.mMaxTessEvaluationUniformBlocks);
    GET(glGetIntegerv, GL_MAX_TESS_EVALUATION_UNIFORM_COMPONENTS,
        &out.mMaxTessEvaluationUniformComponents);
    GET(glGetIntegerv, GL_MAX_TESS_GEN_LEVEL, &out.mMaxTessGenLevel);
    GET(glGetIntegerv, GL_MAX_TESS_PATCH_COMPONENTS,
        &out.mMaxTessPatchComponents);
    GET(glGetIntegerv, GL_MAX_TEXTURE_BUFFER_SIZE, &out.mMaxTextureBufferSize);
    GET(glGetFloatv, GL_MIN_FRAGMENT_INTERPOLATION_OFFSET,
        &out.mMinFragmentInterpolationOffset);
    GET(glGetFloatv, GL_MULTISAMPLE_LINE_WIDTH_GRANULARITY,
        &out.mMultisampleLineWidthGranularity);
    GET(glGetFloatv, GL_MULTISAMPLE_LINE_WIDTH_RANGE,
        out.mMultisampleLineWidthRange);
    GET(glGetBooleanv, GL_PRIMITIVE_RESTART_FOR_PATCHES_SUPPORTED,
        &out.mPrimitiveRestartForPatchesSupported);
    GET(glGetIntegerv, GL_TEXTURE_BUFFER_OFFSET_ALIGNMENT,
        &out.mTextureBufferOffsetAlignment);
    GET(glGetIntegerv, GL_RESET_NOTIFICATION_STRATEGY,
        &out.mResetNotificationStrategy);
  }

  // Constants defined in extensions
  GET(glGetFloatv, GL_MAX_TEXTURE_MAX_ANISOTROPY_EXT,
      &out.mMaxTextureMaxAnisotropyExt);
  GET(glGetIntegerv, GL_MAX_VIEWS_OVR, &out.mMaxViewsExt);
}

}  // namespace gapii
