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
#include "gapii/cc/gles_exports.h"
#include "gapii/cc/gles_types.h"


#define ANDROID_NATIVE_MAKE_CONSTANT(a,b,c,d) \
    (((unsigned)(a)<<24)|((unsigned)(b)<<16)|((unsigned)(c)<<8)|(unsigned)(d))

#define ANDROID_NATIVE_WINDOW_MAGIC \
    ANDROID_NATIVE_MAKE_CONSTANT('_','w','n','d')

#define ANDROID_NATIVE_BUFFER_MAGIC \
    ANDROID_NATIVE_MAKE_CONSTANT('_','b','f','r')

namespace gapii {

// getProgramInfo returns a ProgramInfo, populated with the details of all the
// attributes and uniforms exposed by program.
std::shared_ptr<ProgramInfo> GlesSpy::GetProgramInfoExtra(CallObserver* observer, std::shared_ptr<Context> ctx, ProgramId program) {
    bool gles30 = ctx->mConstants.mMajorVersion >= 3;

    // Allocate temporary buffer large enough to hold any of the returned strings.
    int32_t infoLogLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_INFO_LOG_LENGTH, &infoLogLength);
    int32_t activeAttributeMaxLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_ATTRIBUTE_MAX_LENGTH, &activeAttributeMaxLength);
    int32_t activeUniformMaxLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORM_MAX_LENGTH, &activeUniformMaxLength);
    int32_t activeUniformBlockMaxNameLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORM_BLOCK_MAX_NAME_LENGTH, &activeUniformBlockMaxNameLength);
    const int strSize = std::max(
      std::max(infoLogLength, activeAttributeMaxLength),
      std::max(activeUniformMaxLength, activeUniformBlockMaxNameLength)) +
      16 /* extra space for sprintf */ + 1 /* null-terminator */;
    char* str = observer->getScratch()->create<char>(strSize);

    auto pi = std::shared_ptr<ProgramInfo>(new ProgramInfo());

    int32_t linkStatus = 0;
    mImports.glGetProgramiv(program, GLenum::GL_LINK_STATUS, &linkStatus);
    pi->mLinkStatus = linkStatus;

    mImports.glGetProgramInfoLog(program, strSize, &infoLogLength, str);
    pi->mInfoLog = std::string(str, infoLogLength);

    if (linkStatus == GLbooleanLabels::GL_TRUE) {

      int32_t activeUniforms = 0;
      mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORMS, &activeUniforms);
      for (uint32_t i = 0; i < activeUniforms; i++) {
          ActiveUniform au{};

          int32_t nameLength = 0;
          mImports.glGetActiveUniform(program, i, strSize, &nameLength, &au.mArraySize, &au.mType, str);
          au.mName = std::string(str, nameLength);

          au.mBlockIndex = -1;
          if (gles30) {
              int32_t blockIndex = -1;
              mImports.glGetActiveUniformsiv(program, 1, &i, GLenum::GL_UNIFORM_BLOCK_INDEX, &blockIndex);
              au.mBlockIndex = blockIndex;

              mImports.glGetActiveUniformsiv(program, 1, &i, GLenum::GL_UNIFORM_OFFSET, &au.mOffset);
              mImports.glGetActiveUniformsiv(program, 1, &i, GLenum::GL_UNIFORM_ARRAY_STRIDE, &au.mArrayStride);
              mImports.glGetActiveUniformsiv(program, 1, &i, GLenum::GL_UNIFORM_MATRIX_STRIDE, &au.mMatrixStride);
              mImports.glGetActiveUniformsiv(program, 1, &i, GLenum::GL_UNIFORM_IS_ROW_MAJOR, &au.mIsRowMajor);
          }

          if (au.mBlockIndex == -1) {
              au.mLocation = mImports.glGetUniformLocation(program, str);
              au.mLocations[0] = mImports.glGetUniformLocation(program, str);
              if (nameLength >= 3 && strcmp(str + nameLength - 3, "[0]") == 0) {
                nameLength -= 3; // Remove the "[0]" suffix of array
              }
              for (int32_t j = 1; j < au.mArraySize; j++) {
                sprintf(str + nameLength, "[%i]", j); // Append array suffix
                au.mLocations[j] = mImports.glGetUniformLocation(program, str);
              }
          }

          pi->mActiveUniforms[i] = au;
      }

      int32_t activeAttributes = 0;
      mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_ATTRIBUTES, &activeAttributes);
      for (int32_t i = 0; i < activeAttributes; i++) {
          ActiveAttribute aa{};
          int32_t nameLength = 0;
          mImports.glGetActiveAttrib(program, i, strSize, &nameLength, &aa.mArraySize, &aa.mType, str);
          aa.mName = std::string(str, nameLength);
          aa.mLocation = mImports.glGetAttribLocation(program, str);
          pi->mActiveAttributes[i] = aa;
      }

      int32_t activeUniformBlocks = 0;
      if (gles30) {
          mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORM_BLOCKS, &activeUniformBlocks);
          for (int32_t i = 0; i < activeUniformBlocks; i++) {
              ActiveUniformBlock aub{};

              int32_t nameLength = 0;
              mImports.glGetActiveUniformBlockName(program, i, strSize, &nameLength, str);
              aub.mName = std::string(str, nameLength);

              mImports.glGetActiveUniformBlockiv(program, i, GLenum::GL_UNIFORM_BLOCK_BINDING, &aub.mBinding);

              mImports.glGetActiveUniformBlockiv(program, i, GLenum::GL_UNIFORM_BLOCK_DATA_SIZE, &aub.mDataSize);

              int32_t referencedByVS = 0;
              mImports.glGetActiveUniformBlockiv(program, i, GLenum::GL_UNIFORM_BLOCK_REFERENCED_BY_VERTEX_SHADER, &referencedByVS);
              aub.mReferencedByVertexShader = referencedByVS != 0;

              int32_t referencedByFS = 0;
              mImports.glGetActiveUniformBlockiv(program, i, GLenum::GL_UNIFORM_BLOCK_REFERENCED_BY_FRAGMENT_SHADER, &referencedByFS);
              aub.mReferencedByFragmentShader = referencedByFS != 0;

              pi->mActiveUniformBlocks[i] = aub;
          }
      }

      GAPID_DEBUG("Created ProgramInfo: LinkStatus=GL_TRUE ActiveUniforms=%i ActiveAttributes=%i ActiveUniformBlocks=%i",
          activeUniforms, activeAttributes, activeUniformBlocks);
    } else {
      GAPID_DEBUG("Created ProgramInfo: LinkStatus=GL_FALSE InfoLog=\"%s\"", pi->mInfoLog.data());
    }

    observer->addExtra(pi->toProto());
    return pi;
}

std::shared_ptr<AndroidNativeBufferExtra> GlesSpy::GetAndroidNativeBufferExtra(CallObserver* observer, void* ptr) {
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
        void* reserved[2];
        void* handle;
        void* reserved_proc[8];
    };

    auto buffer = reinterpret_cast<ANativeWindowBuffer*>(ptr);

    if (buffer->common.magic != ANDROID_NATIVE_BUFFER_MAGIC) {
        GAPID_WARNING("Unknown EGLClientBuffer with magic: 0x%x", buffer->common.magic);
        return nullptr;
    }

    std::shared_ptr<AndroidNativeBufferExtra> extra(new AndroidNativeBufferExtra(
        buffer->width,
        buffer->height,
        buffer->stride,
        buffer->format,
        buffer->usage
    ));
    GAPID_DEBUG("Created AndroidNativeBufferExtra: width=%i, height=%i", buffer->width, buffer->height);
    observer->addExtra(extra->toProto());
    return extra;
#else
    return nullptr;
#endif  // TARGET_OS == GAPID_OS_ANDROID
}

// TODO: When gfx api macros produce functions instead of inlining, move this logic
// to the gles.api file.
bool GlesSpy::getFramebufferAttachmentSize(CallObserver* observer, uint32_t* width, uint32_t* height) {
    std::shared_ptr<Context> ctx = Contexts[mCurrentThread];
    if (ctx == nullptr) {
      return false;
    }

    auto framebuffer = ctx->mBound.mReadFramebuffer;
    if (framebuffer == nullptr) {
        return false;
    }

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
            auto r = attachment->second.mRenderbuffer;
            *width = uint32_t(r->mWidth);
            *height = uint32_t(r->mHeight);
            return true;
        }
    }
    return false;
}

bool GlesSpy::observeFramebuffer(CallObserver* observer, uint32_t* w, uint32_t* h, std::vector<uint8_t>* data) {
    if (!getFramebufferAttachmentSize(observer, w, h)) {
        return false; // Could not get the framebuffer size.
    }
    data->resize((*w) * (*h) * 4);
    GlesSpy::mImports.glReadPixels(0, 0, int32_t(*w), int32_t(*h),
            GLenum::GL_RGBA, GLenum::GL_UNSIGNED_BYTE, data->data());
    return true;
}

}

#undef ANDROID_NATIVE_MAKE_CONSTANT
#undef ANDROID_NATIVE_WINDOW_MAGIC
#undef ANDROID_NATIVE_BUFFER_MAGIC
