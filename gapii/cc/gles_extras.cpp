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

#include "gapis/api/gles/gles_pb/extras.pb.h"

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

    observer->encodeAndDelete(pi->toProto());
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
    observer->encodeAndDelete(extra->toProto());
    return extra;
#else
    return nullptr;
#endif  // TARGET_OS == GAPID_OS_ANDROID
}

// TODO: When gfx api macros produce functions instead of inlining, move this logic
// to the gles.api file.
bool GlesSpy::getFramebufferAttachmentSize(CallObserver* observer, uint32_t* width, uint32_t* height) {
    std::shared_ptr<Context> ctx = Contexts[observer->getCurrentThread()];
    if (ctx == nullptr) {
      return false;
    }

    auto framebuffer = ctx->mBound.mReadFramebuffer;
    if (framebuffer == nullptr) {
        return false;
    }

    return getFramebufferAttachmentSize(observer, framebuffer.get(), width, height);
}

bool GlesSpy::getFramebufferAttachmentSize(CallObserver* observer, Framebuffer* framebuffer, uint32_t* width, uint32_t* height) {
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

static bool ReadExternalPixels(GlesImports& imports, EGLImageKHR img, GLsizei width, GLsizei height, std::vector<uint8_t>* data) {
    using namespace GLenum;

    const char* vsSource =
        "precision highp float;\n"
        "attribute vec2 position;\n"
        "varying vec2 texcoord;\n"
        "void main() {\n"
        "  gl_Position = vec4(position, 0.5, 1.0);\n"
        "  texcoord = position * vec2(0.5) + vec2(0.5);\n"
        "}\n";

    const char* fsSource =
        "#extension GL_OES_EGL_image_external : require\n"
        "precision highp float;\n"
        "uniform samplerExternalOES tex;\n"
        "varying vec2 texcoord;\n"
        "void main() {\n"
        "  gl_FragColor = texture2D(tex, texcoord);\n"
        "}\n";

    GLint err;
    auto prog = imports.glCreateProgram();

    auto vs = imports.glCreateShader(GL_VERTEX_SHADER);
    imports.glShaderSource(vs, 1, const_cast<char**>(&vsSource), nullptr);
    imports.glCompileShader(vs);
    imports.glAttachShader(prog, vs);

    auto fs = imports.glCreateShader(GL_FRAGMENT_SHADER);
    imports.glShaderSource(fs, 1, const_cast<char**>(&fsSource), nullptr);
    imports.glCompileShader(fs);
    imports.glAttachShader(prog, fs);

    imports.glBindAttribLocation(prog, 0, "position");
    imports.glLinkProgram(prog);

    if ((err = imports.glGetError()) != 0) {
        GAPID_ERROR("ReadExternalPixels: Failed to create program: 0x%X", err);
        return false;
    }

    GLint linkStatus = 0;
    imports.glGetProgramiv(prog, GL_LINK_STATUS, &linkStatus);
    if (linkStatus == 0) {
        char log[1024];
        imports.glGetProgramInfoLog(prog, sizeof(log), nullptr, log);
        GAPID_ERROR("ReadExternalPixels: Failed to compile program:\n%s", log);
        return false;
    }

    GLuint srcTex = 0;
    imports.glGenTextures(1, &srcTex);
    imports.glBindTexture(GL_TEXTURE_EXTERNAL_OES, srcTex);
    imports.glEGLImageTargetTexture2DOES(GL_TEXTURE_EXTERNAL_OES, img);
    imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
    imports.glTexParameteri(GL_TEXTURE_EXTERNAL_OES, GL_TEXTURE_MAG_FILTER, GL_NEAREST);

    GLuint dstTex = 0;
    imports.glGenTextures(1, &dstTex);
    imports.glBindTexture(GL_TEXTURE_2D, dstTex);
    imports.glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, width, height, 0, GL_RGBA, GL_UNSIGNED_BYTE, nullptr);

    if ((err = imports.glGetError()) != 0) {
        GAPID_ERROR("ReadExternalPixels: Failed to create texture: 0x%X", err);
        return false;
    }

    GLuint fb = 0;
    imports.glGenFramebuffers(1, &fb);
    imports.glBindFramebuffer(GL_FRAMEBUFFER, fb);
    imports.glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, dstTex, 0);

    if ((err = imports.glGetError()) != 0) {
        GAPID_ERROR("ReadExternalPixels: Failed to create framebuffer: 0x%X", err);
        return false;
    }
    if ((err = imports.glCheckFramebufferStatus(GL_FRAMEBUFFER)) != GL_FRAMEBUFFER_COMPLETE) {
        GAPID_ERROR("ReadExternalPixels: Framebuffer incomplete: 0x%X", err);
        return false;
    }

    imports.glDisable(GL_CULL_FACE);
    imports.glDisable(GL_DEPTH_TEST);
    imports.glViewport(0, 0, width, height);
    imports.glClearColor(0.0, 0.0, 0.0, 0.0);
    imports.glClear(GLbitfield::GL_COLOR_BUFFER_BIT);
    imports.glUseProgram(prog);
    GLfloat vb[] = {
        -1.0f, +1.0f,  // 2--4
        -1.0f, -1.0f,  // |\ |
        +1.0f, +1.0f,  // | \|
        +1.0f, -1.0f,  // 1--3
    };
    imports.glEnableVertexAttribArray(0);
    imports.glVertexAttribPointer(0, 2, GL_FLOAT, 0, 0, vb);
    imports.glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
    if ((err = imports.glGetError()) != 0) {
        GAPID_ERROR("ReadExternalPixels: Failed to draw quad: 0x%X", err);
        return false;
    }

    data->resize(width * height * 4);
    imports.glReadPixels(0, 0, width, height, GL_RGBA, GL_UNSIGNED_BYTE, data->data());
    if ((err = imports.glGetError()) != 0) {
        GAPID_ERROR("ReadExternalPixels: Failed to read pixels: 0x%X", err);
        return false;
    }

    return true;
}

void GlesSpy::GetEGLImageData(CallObserver* observer, EGLImageKHR img, GLsizei width, GLsizei height) {
    using namespace EGLenum;

    GAPID_DEBUG("Get EGLImage data: 0x%x %xx%x", img, width, height);

    // Save old state.
    auto display = mImports.eglGetCurrentDisplay();
    auto draw = mImports.eglGetCurrentSurface(EGL_DRAW);
    auto read = mImports.eglGetCurrentSurface(EGL_READ);
    auto oldCtx = mImports.eglGetCurrentContext();

    // Find an EGL config.
    EGLConfig cfg;
    EGLint cfgAttribs[] = { EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT, EGL_NONE };
    int one = 1;
    if (mImports.eglChooseConfig(display, cfgAttribs, &cfg, 1, &one) == EGL_FALSE || one != 1) {
        GAPID_ERROR("Failed to choose EGL config");
        return;
    }

    // Create an EGL context.
    EGLContext ctx;
    EGLint ctxAttribs[] = { EGL_CONTEXT_CLIENT_VERSION, 2, EGL_NONE };
    if ((ctx = mImports.eglCreateContext(display, cfg, nullptr, ctxAttribs)) == nullptr) {
        GAPID_ERROR("Failed to create EGL context");
        return;
    }

    // Create an EGL surface.
    EGLSurface surface;
    EGLint surfaceAttribs[] = { EGL_WIDTH, 16, EGL_HEIGHT, 16, EGL_NONE };
    if ((surface = mImports.eglCreatePbufferSurface(display, cfg, surfaceAttribs)) == nullptr) {
        GAPID_ERROR("Failed to create EGL surface");
        return;
    }

    // Bind the EGL context.
    if (mImports.eglMakeCurrent(display, surface, surface, ctx) == EGL_FALSE) {
        GAPID_ERROR("Failed to bind new EGL context");
        return;
    }

    std::vector<uint8_t> data;
    if (ReadExternalPixels(mImports, img, width, height, &data)) {
        core::Id id = core::Id::Hash(data.data(), data.size());
        if (getResources().count(id) == 0) {
            capture::Resource resource;
            resource.set_id(reinterpret_cast<const char*>(id.data), sizeof(id.data));
            resource.set_data(data.data(), data.size());
            getEncoder(kApiIndex)->object(&resource);
            getResources().emplace(id);
        }

        auto extra = new gles_pb::EGLImageData();
        extra->set_id(reinterpret_cast<const char*>(id.data), sizeof(id.data));
        extra->set_size(data.size());
        extra->set_width(width);
        extra->set_height(height);
        extra->set_format(GLenum::GL_RGBA);
        extra->set_type(GLenum::GL_UNSIGNED_BYTE);
        observer->encodeAndDelete(extra);
    }

    if (mImports.eglMakeCurrent(display, draw, read, oldCtx) == EGL_FALSE) {
        GAPID_FATAL("Failed to restore old EGL context");
    }

    mImports.eglDestroySurface(display, surface);
    mImports.eglDestroyContext(display, ctx);
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
