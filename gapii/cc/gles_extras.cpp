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

#include "gapii/cc/gles_exports.h"
#include "gapii/cc/gles_spy.h"
#include "gapii/cc/gles_types.h"

#include "gapis/api/gles/gles_pb/extras.pb.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif  // __STDC_FORMAT_MACROS
#include <inttypes.h>

#define ANDROID_NATIVE_MAKE_CONSTANT(a, b, c, d)                          \
  (((unsigned)(a) << 24) | ((unsigned)(b) << 16) | ((unsigned)(c) << 8) | \
   (unsigned)(d))

#define ANDROID_NATIVE_WINDOW_MAGIC \
  ANDROID_NATIVE_MAKE_CONSTANT('_', 'w', 'n', 'd')

#define ANDROID_NATIVE_BUFFER_MAGIC \
  ANDROID_NATIVE_MAKE_CONSTANT('_', 'b', 'f', 'r')

namespace gapii {

// Handles GLES 2.0 and GLES 3.0 (the old reflection API)
static void GetProgramReflectionInfo_GLES20(GlesSpy* spy,
                                            LinkProgramExtra* extra,
                                            Program* p) {
  using namespace GLenum;
  auto resources = gapil::Ref<ActiveProgramResources>::create(spy->arena());

  const GLuint program = extra->mID;
  const bool gles30 =
      spy->mState.Version != nullptr && spy->mState.Version->mGLES30;
  const auto& imports = spy->imports();

  // Helper method to get property of program
  auto getProgramiv = [&](uint32_t pname) {
    GLint value = 0;
    imports.glGetProgramiv(program, pname, &value);
    return value;
  };

  // Allocate temporary buffer large enough to hold any of the returned strings.
  int32_t maxLength = 0;
  maxLength = std::max(maxLength, getProgramiv(GL_ACTIVE_ATTRIBUTE_MAX_LENGTH));
  maxLength = std::max(maxLength, getProgramiv(GL_ACTIVE_UNIFORM_MAX_LENGTH));
  if (gles30) {
    maxLength = std::max(maxLength,
                         getProgramiv(GL_ACTIVE_UNIFORM_BLOCK_MAX_NAME_LENGTH));
  }
  maxLength += 16;  // extra space for appending of array suffix
  std::vector<char> buffer(maxLength);

  auto getActiveUniformsiv = [&](GLuint i, uint32_t pname) {
    GLint value = 0;
    imports.glGetActiveUniformsiv(program, 1, &i, pname, &value);
    return value;
  };

  int32_t activeUniforms = getProgramiv(GL_ACTIVE_UNIFORMS);
  for (int32_t i = 0; i < activeUniforms; i++) {
    auto res = gapil::Ref<ProgramResource>::create(spy->arena());

    int32_t nameLength = 0;
    imports.glGetActiveUniform(program, i, buffer.size(), &nameLength,
                               &res->mArraySize, &res->mType, buffer.data());
    res->mName = gapil::String(spy->arena(), buffer.data(), nameLength);

    if (gles30) {
      res->mBlockIndex = getActiveUniformsiv(i, GL_UNIFORM_BLOCK_INDEX);
    } else {
      res->mBlockIndex = -1;
    }

    if (res->mBlockIndex == -1) {
      res->mLocations[0] = imports.glGetUniformLocation(program, buffer.data());
      if (nameLength >= 3 &&
          strcmp(buffer.data() + nameLength - 3, "[0]") == 0) {
        nameLength -= 3;  // Remove the "[0]" suffix of array
      }
      for (int32_t j = 1; j < res->mArraySize; j++) {
        sprintf(buffer.data() + nameLength, "[%i]", j);  // Append array suffix
        res->mLocations[j] =
            imports.glGetUniformLocation(program, buffer.data());
      }
    } else {
      auto layout = gapil::Ref<ProgramResourceLayout>::create(spy->arena());
      layout->mOffset = getActiveUniformsiv(i, GL_UNIFORM_OFFSET);
      layout->mArrayStride = getActiveUniformsiv(i, GL_UNIFORM_ARRAY_STRIDE);
      layout->mMatrixStride = getActiveUniformsiv(i, GL_UNIFORM_MATRIX_STRIDE);
      layout->mIsRowMajor = getActiveUniformsiv(i, GL_UNIFORM_IS_ROW_MAJOR);
      res->mLayout = std::move(layout);
    }

    resources->mUniforms[i] = std::move(res);
  }

  int32_t activeAttributes = 0;
  activeAttributes = getProgramiv(GL_ACTIVE_ATTRIBUTES);
  for (int32_t i = 0; i < activeAttributes; i++) {
    auto res = gapil::Ref<ProgramResource>::create(spy->arena());

    int32_t nameLength = 0;
    imports.glGetActiveAttrib(program, i, buffer.size(), &nameLength,
                              &res->mArraySize, &res->mType, buffer.data());
    res->mName = gapil::String(spy->arena(), buffer.data(), nameLength);
    res->mLocations[0] = imports.glGetAttribLocation(program, buffer.data());

    resources->mProgramInputs[i] = std::move(res);
  }

  int32_t activeUniformBlocks = 0;
  if (gles30) {
    auto getUniformBlockiv = [&](GLuint i, uint32_t pname) {
      GLint value = 0;
      imports.glGetActiveUniformBlockiv(program, i, pname, &value);
      return value;
    };

    activeUniformBlocks = getProgramiv(GL_ACTIVE_UNIFORM_BLOCKS);
    for (int32_t i = 0; i < activeUniformBlocks; i++) {
      auto block = gapil::Ref<ProgramResourceBlock>::create(spy->arena());

      int32_t nameLength = 0;
      imports.glGetActiveUniformBlockName(program, i, buffer.size(),
                                          &nameLength, buffer.data());
      block->mName = gapil::String(spy->arena(), buffer.data(), nameLength);

      block->mBinding = getUniformBlockiv(i, GL_UNIFORM_BLOCK_BINDING);
      block->mDataSize = getUniformBlockiv(i, GL_UNIFORM_BLOCK_DATA_SIZE);

      auto usedBy = gapil::Ref<ProgramResourceUses>::create(spy->arena());
      usedBy->mVertexShader =
          getUniformBlockiv(i, GL_UNIFORM_BLOCK_REFERENCED_BY_VERTEX_SHADER);
      usedBy->mFragmentShader =
          getUniformBlockiv(i, GL_UNIFORM_BLOCK_REFERENCED_BY_FRAGMENT_SHADER);
      block->mReferencedBy = std::move(usedBy);

      resources->mUniformBlocks[i] = std::move(block);
    }
  }

  extra->mActiveResources = std::move(resources);
}

// Handles GLES 3.1 and GLES 3.2 (the new reflection API)
static void GetProgramReflectionInfo_GLES31(GlesSpy* spy,
                                            LinkProgramExtra* extra,
                                            Program* p) {
  using namespace GLenum;

  const GLuint program = extra->mID;
  const auto& imports = spy->imports();

  const bool hasGeometryShader = p->mShaders.contains(GL_GEOMETRY_SHADER);
  const bool hasTessControlShader =
      p->mShaders.contains(GL_TESS_CONTROL_SHADER);
  const bool hasTessEvaluationShader =
      p->mShaders.contains(GL_TESS_EVALUATION_SHADER);
  const bool hasComputeShader = p->mShaders.contains(GL_COMPUTE_SHADER);

  std::vector<char> buffer;  // Temporary buffer for getting string.
  const int bufferSuffixSize =
      16;  // Allocate a bit more extra space so we can append integer to name.

  // Helper method to get property of program
  auto getProgramiv = [&](uint32_t pname) {
    GLint value = 0;
    imports.glGetProgramiv(program, pname, &value);
    return value;
  };

  // Helper method to get property of program interface
  auto getInterfaceiv = [&](uint32_t interface, uint32_t pname) {
    GLint value = 0;
    imports.glGetProgramInterfaceiv(program, interface, pname, &value);
    return value;
  };

  // Helper method to get property of program resource
  auto getResourceiv = [&](uint32_t interface, GLuint i, uint32_t pname) {
    GLint value = 0;
    imports.glGetProgramResourceiv(program, interface, i, 1, &pname, 1, nullptr,
                                   &value);
    return value;
  };

  // Helper method to get name of program resource
  auto getResourceName = [&](uint32_t interface, GLuint i) {
    GAPID_ASSERT(getResourceiv(interface, i, GL_NAME_LENGTH) <=
                 (int32_t)buffer.size());
    GLsizei length = 0;
    imports.glGetProgramResourceName(program, interface, i, buffer.size(),
                                     &length, buffer.data());
    return gapil::String(spy->arena(), buffer.data(), length);
  };

  // Helper method to get all locations of program resource
  auto getResourceLocations = [&](uint32_t interface, const gapil::String& name,
                                  GLint arraySize) {
    U32ToGLint locations(spy->arena());
    locations[0] = imports.glGetProgramResourceLocation(
        program, interface, reinterpret_cast<const GLchar*>(name.c_str()));
    if (arraySize > 1) {
      // Copy the array base name (without the [0] suffix) to the temporary
      // buffer
      size_t baseLength = name.length();
      if (baseLength >= 3 &&
          strcmp(name.c_str() + baseLength - 3, "[0]") == 0) {
        baseLength -= 3;  // Remove the "[0]" suffix of array
      }
      GAPID_ASSERT(baseLength + bufferSuffixSize <= buffer.size());
      memcpy(buffer.data(), name.c_str(), baseLength);
      // Get location for each array element.
      for (int32_t j = 1; j < arraySize; j++) {
        snprintf(buffer.data() + baseLength, buffer.size(), "[%i]",
                 j);  // Append array suffix
        locations[j] = imports.glGetProgramResourceLocation(program, interface,
                                                            buffer.data());
      }
    }
    return locations;
  };

  // Helper method to get all referenced-by properties
  auto getResourceUses = [&](uint32_t interface, GLuint i) {
    auto usedBy = gapil::Ref<ProgramResourceUses>::create(spy->arena());
    usedBy->mVertexShader =
        getResourceiv(interface, i, GL_REFERENCED_BY_VERTEX_SHADER) != 0;
    if (hasTessControlShader) {
      usedBy->mTessControlShader =
          getResourceiv(interface, i, GL_REFERENCED_BY_TESS_CONTROL_SHADER) !=
          0;
    }
    if (hasTessEvaluationShader) {
      usedBy->mTessEvaluationShader =
          getResourceiv(interface, i,
                        GL_REFERENCED_BY_TESS_EVALUATION_SHADER) != 0;
    }
    if (hasGeometryShader) {
      usedBy->mGeometryShader =
          getResourceiv(interface, i, GL_REFERENCED_BY_GEOMETRY_SHADER) != 0;
    }
    usedBy->mFragmentShader =
        getResourceiv(interface, i, GL_REFERENCED_BY_FRAGMENT_SHADER) != 0;
    usedBy->mComputeShader =
        getResourceiv(interface, i, GL_REFERENCED_BY_COMPUTE_SHADER) != 0;
    return usedBy;
  };

  // Helper method to get all resource blocks of given type
  auto getResourceBlocks = [&](uint32_t interface) {
    U32ToProgramResourceBlock__R blocks(spy->arena());
    GLint count = getInterfaceiv(interface, GL_ACTIVE_RESOURCES);
    if (interface != GL_ATOMIC_COUNTER_BUFFER) {
      buffer.resize(getInterfaceiv(interface, GL_MAX_NAME_LENGTH) +
                    bufferSuffixSize);
    }
    for (int i = 0; i < count; i++) {
      auto block = gapil::Ref<ProgramResourceBlock>::create(spy->arena());
      if (interface != GL_ATOMIC_COUNTER_BUFFER) {
        block->mName = getResourceName(interface, i);
      }
      block->mBinding = getResourceiv(interface, i, GL_BUFFER_BINDING);
      block->mDataSize = getResourceiv(interface, i, GL_BUFFER_DATA_SIZE);
      block->mReferencedBy = getResourceUses(interface, i);
      blocks[i] = std::move(block);
    }
    return blocks;
  };

  // Helper method to get all resources of given type
  auto getResources = [&](uint32_t interface) {
    // Helper flags for determining if property is applicable to this interface.
    // Trying to get a property on the wrong interface will result in GL error.
    const bool pi = (interface == GL_PROGRAM_INPUT);
    const bool po = (interface == GL_PROGRAM_OUTPUT);
    const bool u = (interface == GL_UNIFORM);
    const bool bv = (interface == GL_BUFFER_VARIABLE);

    U32ToProgramResource__R resources(spy->arena());
    GLint count = getInterfaceiv(interface, GL_ACTIVE_RESOURCES);
    buffer.resize(getInterfaceiv(interface, GL_MAX_NAME_LENGTH) +
                  bufferSuffixSize);
    for (int i = 0; i < count; i++) {
      auto resource = gapil::Ref<ProgramResource>::create(spy->arena());
      resource->mName = getResourceName(interface, i);
      resource->mType = getResourceiv(interface, i, GL_TYPE);
      resource->mArraySize = getResourceiv(interface, i, GL_ARRAY_SIZE);

      bool backedByBufferObject = false;
      if (bv || u) {
        resource->mBlockIndex = getResourceiv(interface, i, GL_BLOCK_INDEX);
        backedByBufferObject |= (resource->mBlockIndex != -1);
      }
      if (u) {
        resource->mAtomicCounterBufferIndex =
            getResourceiv(interface, i, GL_ATOMIC_COUNTER_BUFFER_INDEX);
        backedByBufferObject |= (resource->mAtomicCounterBufferIndex != -1);
      }
      if (bv || pi || po || u) {
        resource->mReferencedBy = getResourceUses(interface, i);
      }
      if (backedByBufferObject) {
        auto layout = gapil::Ref<ProgramResourceLayout>::create(spy->arena());
        if (bv || u) {
          layout->mOffset = getResourceiv(interface, i, GL_OFFSET);
          layout->mArrayStride = getResourceiv(interface, i, GL_ARRAY_STRIDE);
          layout->mMatrixStride = getResourceiv(interface, i, GL_MATRIX_STRIDE);
          layout->mIsRowMajor = getResourceiv(interface, i, GL_IS_ROW_MAJOR);
        }
        if (bv) {
          layout->mTopLevelArraySize =
              getResourceiv(interface, i, GL_TOP_LEVEL_ARRAY_SIZE);
          layout->mTopLevelArrayStride =
              getResourceiv(interface, i, GL_TOP_LEVEL_ARRAY_STRIDE);
        }
        resource->mLayout = std::move(layout);
      } else {
        if (pi || po || u) {
          resource->mLocations = getResourceLocations(
              interface, resource->mName, resource->mArraySize);
        }
      }
      if ((pi || po) && (hasTessControlShader || hasTessEvaluationShader)) {
        resource->mIsPerPatch = getResourceiv(interface, i, GL_IS_PER_PATCH);
      }

      resources[i] = std::move(resource);
    }
    return resources;
  };

  ///////////////////////////////////////////////////////////////////
  // Get the program state using the helper methods above          //
  ///////////////////////////////////////////////////////////////////

  // Get all active resources.
  {
    auto resources = gapil::Ref<ActiveProgramResources>::create(spy->arena());
    resources->mProgramInputs = getResources(GL_PROGRAM_INPUT);
    resources->mProgramOutputs = getResources(GL_PROGRAM_OUTPUT);
    resources->mUniforms = getResources(GL_UNIFORM);
    resources->mUniformBlocks = getResourceBlocks(GL_UNIFORM_BLOCK);
    resources->mAtomicCounterBuffers =
        getResourceBlocks(GL_ATOMIC_COUNTER_BUFFER);
    resources->mBufferVariables = getResources(GL_BUFFER_VARIABLE);
    resources->mShaderStorageBlocks =
        getResourceBlocks(GL_SHADER_STORAGE_BLOCK);
    resources->mTransformFeedbackVaryings =
        getResources(GL_TRANSFORM_FEEDBACK_VARYING);
    extra->mActiveResources = std::move(resources);
  }

  // Get global layout qualifiers from shaders.
  {
    auto layout = gapil::Ref<ShaderLayoutQualifiers>::create(spy->arena());

    if (hasGeometryShader) {
      layout->mGeometryVerticesOut = getProgramiv(GL_GEOMETRY_VERTICES_OUT);
      layout->mGeometryInputType = getProgramiv(GL_GEOMETRY_INPUT_TYPE);
      layout->mGeometryOutputType = getProgramiv(GL_GEOMETRY_OUTPUT_TYPE);
      layout->mGeometryShaderInvocations =
          getProgramiv(GL_GEOMETRY_SHADER_INVOCATIONS);
    }
    if (hasTessControlShader) {
      layout->mTessControlOutputVertices =
          getProgramiv(GL_TESS_CONTROL_OUTPUT_VERTICES);
    }
    if (hasTessEvaluationShader) {
      layout->mTessGenMode = getProgramiv(GL_TESS_GEN_MODE);
      layout->mTessGenSpacing = getProgramiv(GL_TESS_GEN_SPACING);
      layout->mTessGenVertexOrder = getProgramiv(GL_TESS_GEN_VERTEX_ORDER);
      layout->mTessGenPointMode = getProgramiv(GL_TESS_GEN_POINT_MODE);
    }
    if (hasComputeShader) {
      GLint computeWorkGroupSize[3];
      imports.glGetProgramiv(program, GL_COMPUTE_WORK_GROUP_SIZE,
                             computeWorkGroupSize);
      layout->mComputeWorkGroupSize[0] = computeWorkGroupSize[0];
      layout->mComputeWorkGroupSize[1] = computeWorkGroupSize[1];
      layout->mComputeWorkGroupSize[2] = computeWorkGroupSize[2];
    }

    extra->mShaderLayout = std::move(layout);
  }
}

// GetLinkProgramExtra is called by glLinkProgram and glProgramBinary
gapil::Ref<LinkProgramExtra> GlesSpy::GetLinkProgramExtra(
    CallObserver* observer, gapil::Ref<Context> ctx, gapil::Ref<Program> p,
    gapil::Ref<BinaryExtra> binary) {
  using namespace GLenum;

  // TODO: It is kind of evil to call glGetError, as it modifies the driver
  // state.
  GlesSpy::mImports.glGetError();  // Clear error.

  const GLuint program = p->mID;
  const bool gles31 = mState.Version != nullptr && mState.Version->mGLES31;

  // Helper method to get property of program
  auto getProgramiv = [&](uint32_t pname) {
    GLint value = 0;
    mImports.glGetProgramiv(program, pname, &value);
    return value;
  };

  auto extra = gapil::Ref<LinkProgramExtra>::create(arena());
  extra->mID = program;
  extra->mLinkStatus = getProgramiv(GL_LINK_STATUS);

  // Get info log string
  std::vector<char> buffer;  // Temporary buffer for getting string.
  buffer.resize(getProgramiv(
      GL_INFO_LOG_LENGTH));  // Returned length includes null-terminator.
  GLint infoLogLength =
      0;  // Returned length by the command below excludes null-terminator.
  mImports.glGetProgramInfoLog(program, buffer.size(), &infoLogLength,
                               buffer.data());
  extra->mInfoLog = gapil::String(arena(), buffer.data(), infoLogLength);

  // Get meta-data about the active resources generated by the compiler.
  if (extra->mLinkStatus) {
    // The API changed radically in GLES 3.1, so we need two distinct versions.
    if (gles31) {
      GetProgramReflectionInfo_GLES31(this, extra.get(), p.get());
    } else {
      GetProgramReflectionInfo_GLES20(this, extra.get(), p.get());
    }

    // Add resources to the resource blocks that own them, for convenience.
    auto* resources = extra->mActiveResources.get();
    for (auto& kvp : resources->mUniforms) {
      auto& id = kvp.first;
      auto& u = kvp.second;
      if (u->mBlockIndex != -1) {
        GAPID_ASSERT(resources->mUniformBlocks.contains(u->mBlockIndex));
        resources->mUniformBlocks[u->mBlockIndex]->mResources[id] = u;
      } else {
        resources->mDefaultUniformBlock[id] = u;
      }
      if (u->mAtomicCounterBufferIndex != -1) {
        GAPID_ASSERT(resources->mAtomicCounterBuffers.contains(
            u->mAtomicCounterBufferIndex));
        resources->mAtomicCounterBuffers[u->mAtomicCounterBufferIndex]
            ->mResources[id] = u;
      }
    }
    for (auto& kvp : resources->mBufferVariables) {
      auto& id = kvp.first;
      auto& u = kvp.second;
      if (u->mBlockIndex != -1) {
        GAPID_ASSERT(resources->mShaderStorageBlocks.contains(u->mBlockIndex));
        resources->mShaderStorageBlocks[u->mBlockIndex]->mResources[id] = u;
      }
    }
  }

  // TODO: It is kind of evil to call glGetError, as it modifies the driver
  // state.
  //       But if we omit it, and cause an error, it would be even more
  //       confusing. The ideal solution is probably to create shared context
  //       sibling, and query all the state from there (maybe even in parallel
  //       on other thread).
  auto err = GlesSpy::mImports.glGetError();
  if (err) {
    GAPID_ERROR("Failed to get reflection data for program %i: Error 0x%x",
                program, err);
  }

  // Include snapshot of the current state (i.e. the inputs of the operation)
  for (auto it : p->mShaders) {
    if (it.second != nullptr) {
      extra->mShaders[it.first] = it.second->mCompileExtra;
    }
  }
  extra->mBinary = binary;
  extra->mAttributeBindings = p->mAttributeBindings.clone();
  extra->mTransformFeedbackVaryings = p->mTransformFeedbackVaryings.clone();
  extra->mTransformFeedbackBufferMode = p->mTransformFeedbackBufferMode;
  extra->mSeparable = p->mSeparable;
  extra->mBinaryRetrievableHint = p->mBinaryRetrievableHint;

  observer->encode(*extra.get());
  return extra;
}

// GetCompileShaderExtra is called by glCompileShader and glShaderBinary
gapil::Ref<CompileShaderExtra> GlesSpy::GetCompileShaderExtra(
    CallObserver* observer, gapil::Ref<Context> ctx, gapil::Ref<Shader> p,
    gapil::Ref<BinaryExtra> binary) {
  using namespace GLenum;
  auto extra = gapil::Ref<CompileShaderExtra>::create(arena());
  GLuint shader = p->mID;
  extra->mID = shader;

  GLint compileStatus = 0;
  mImports.glGetShaderiv(shader, GL_COMPILE_STATUS, &compileStatus);
  extra->mCompileStatus = compileStatus;

  GLint logLength = 0;
  mImports.glGetShaderiv(shader, GL_INFO_LOG_LENGTH, &logLength);
  std::vector<char> buffer(logLength + 1);
  mImports.glGetShaderInfoLog(shader, buffer.size(), &logLength, buffer.data());
  extra->mInfoLog = gapil::String(arena(), buffer.data(), logLength);

  // Make snapshot of the inputs.
  extra->mSource = p->mSource;
  extra->mBinary = binary;

  observer->encode(*extra.get());
  return extra;
}

// GetValidateProgramExtra is called by glValidateProgram
gapil::Ref<ValidateProgramExtra> GlesSpy::GetValidateProgramExtra(
    CallObserver* observer, gapil::Ref<Context> ctx, gapil::Ref<Program> p) {
  using namespace GLenum;
  auto extra = gapil::Ref<ValidateProgramExtra>::create(arena());
  GLuint program = p->mID;
  extra->mID = program;

  GLint validateStatus = 0;
  mImports.glGetProgramiv(program, GL_VALIDATE_STATUS, &validateStatus);
  extra->mValidateStatus = validateStatus;

  GLint infoLogLength = 0;
  mImports.glGetProgramiv(program, GL_INFO_LOG_LENGTH, &infoLogLength);
  std::vector<char> buffer(infoLogLength + 1);
  mImports.glGetProgramInfoLog(program, buffer.size(), &infoLogLength,
                               buffer.data());
  extra->mInfoLog = gapil::String(arena(), buffer.data(), infoLogLength);

  observer->encode(*extra.get());
  return extra;
}

// GetValidateProgramPipelineExtra is called by glValidateProgramPipeline
gapil::Ref<ValidateProgramPipelineExtra>
GlesSpy::GetValidateProgramPipelineExtra(CallObserver* observer,
                                         gapil::Ref<Context> ctx,
                                         gapil::Ref<Pipeline> p) {
  using namespace GLenum;
  auto extra = gapil::Ref<ValidateProgramPipelineExtra>::create(arena());
  GLuint pipe = p->mID;
  extra->mID = pipe;

  GLint validateStatus = 0;
  mImports.glGetProgramPipelineiv(pipe, GL_VALIDATE_STATUS, &validateStatus);
  extra->mValidateStatus = validateStatus;

  GLint infoLogLength = 0;
  mImports.glGetProgramPipelineiv(pipe, GL_INFO_LOG_LENGTH, &infoLogLength);
  std::vector<char> buffer(infoLogLength + 1);
  mImports.glGetProgramPipelineInfoLog(pipe, buffer.size(), &infoLogLength,
                                       buffer.data());
  extra->mInfoLog = gapil::String(arena(), buffer.data(), infoLogLength);

  observer->encode(*extra.get());

  return extra;
}

gapil::Ref<AndroidNativeBufferExtra> GlesSpy::GetAndroidNativeBufferExtra(
    CallObserver* observer, void* ptr) {
#if TARGET_OS == GAPID_OS_ANDROID
  struct android_native_base_t {
    int magic;
    int version;
    void* reserved[4];
    void (*incRef)(android_native_base_t* base);
    void (*decRef)(android_native_base_t* base);
  };

  struct ANativeWindowBuffer {
    android_native_base_t common;
    int width;
    int height;
    int stride;
    int format;
    int usage;
    uintptr_t layer_count;
    void* reserved;
    void* handle;
    void* reserved_proc[8];
  };

  auto buffer = reinterpret_cast<ANativeWindowBuffer*>(ptr);

  if (buffer->common.magic != ANDROID_NATIVE_BUFFER_MAGIC) {
    GAPID_WARNING("Unknown EGLClientBuffer with magic: 0x%x",
                  buffer->common.magic);
    return nullptr;
  }

  auto android_version_major =
      device_instance()->configuration().os().major_version();

  bool use_layer_count = android_version_major >= 8;  // Android O

  auto extra = gapil::Ref<AndroidNativeBufferExtra>::create(
      arena(), buffer->width, buffer->height, buffer->stride, buffer->format,
      buffer->usage, use_layer_count ? buffer->layer_count : 0);

  GAPID_INFO(
      "Created AndroidNativeBufferExtra: os_version:%i, width=%i, height=%i, "
      "layers=%" PRIx64,
      (int)android_version_major, buffer->width, buffer->height,
      (uint64_t)buffer->layer_count);

  observer->encode(*extra.get());
  return extra;
#else
  return nullptr;
#endif  // TARGET_OS == GAPID_OS_ANDROID
}

// TODO: When gfx api macros produce functions instead of inlining, move this
// logic to the gles.api file.
bool GlesSpy::getFramebufferAttachmentSize(CallObserver* observer,
                                           uint32_t* width, uint32_t* height) {
  gapil::Ref<Context> ctx = mState.Contexts[observer->getCurrentThread()];
  if (ctx == nullptr) {
    return false;
  }

  auto framebuffer = ctx->mBound.mReadFramebuffer;
  if (framebuffer == nullptr) {
    return false;
  }

  return getFramebufferAttachmentSize(observer, framebuffer.get(), width,
                                      height);
}

bool GlesSpy::getFramebufferAttachmentSize(CallObserver* observer,
                                           Framebuffer* framebuffer,
                                           uint32_t* width, uint32_t* height) {
  auto attachment = framebuffer->mColorAttachments.find(0);
  if (attachment == framebuffer->mColorAttachments.end()) {
    return false;
  }

  switch (attachment->second.mType) {
    case GLenum::GL_TEXTURE: {
      auto t = attachment->second.mTexture;
      auto level = t->mLevels.find(attachment->second.mTextureLevel);
      if (level == t->mLevels.end()) {
        return false;
      }
      auto layer = level->second.mLayers.find(attachment->second.mTextureLayer);
      if (layer == level->second.mLayers.end()) {
        return false;
      }
      auto image = layer->second;
      if (image == nullptr) {
        return false;
      }
      *width = uint32_t(image->mWidth);
      *height = uint32_t(image->mHeight);
      return true;
    }
    case GLenum::GL_RENDERBUFFER: {
      auto image = attachment->second.mRenderbuffer->mImage;
      if (image == nullptr) {
        return false;
      }
      *width = uint32_t(image->mWidth);
      *height = uint32_t(image->mHeight);
      return true;
    }
  }
  return false;
}

bool GlesSpy::observeFramebuffer(CallObserver* observer, uint32_t* w,
                                 uint32_t* h, std::vector<uint8_t>* data) {
  if (!getFramebufferAttachmentSize(observer, w, h)) {
    return false;  // Could not get the framebuffer size.
  }
  data->resize((*w) * (*h) * 4);
  GlesSpy::mImports.glReadPixels(0, 0, int32_t(*w), int32_t(*h),
                                 GLenum::GL_RGBA, GLenum::GL_UNSIGNED_BYTE,
                                 data->data());
  return true;
}

}  // namespace gapii

#undef ANDROID_NATIVE_MAKE_CONSTANT
#undef ANDROID_NATIVE_WINDOW_MAGIC
#undef ANDROID_NATIVE_BUFFER_MAGIC
