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

#define ANDROID_NATIVE_MAKE_CONSTANT(a,b,c,d) \
    (((unsigned)(a)<<24)|((unsigned)(b)<<16)|((unsigned)(c)<<8)|(unsigned)(d))

#define ANDROID_NATIVE_WINDOW_MAGIC \
    ANDROID_NATIVE_MAKE_CONSTANT('_','w','n','d')

#define ANDROID_NATIVE_BUFFER_MAGIC \
    ANDROID_NATIVE_MAKE_CONSTANT('_','b','f','r')

namespace gapii {

// getProgramInfo returns a ProgramInfo, populated with the details of all the
// attributes and uniforms exposed by program.
std::shared_ptr<ProgramInfo> GlesSpy::GetProgramInfoExtra(CallObserver* observer, ProgramId program) {
    // Allocate temporary buffer large enough to hold any of the returned strings.
    int32_t infoLogLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_INFO_LOG_LENGTH, &infoLogLength);
    int32_t activeAttributeMaxLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_ATTRIBUTE_MAX_LENGTH, &activeAttributeMaxLength);
    int32_t activeUniformMaxLength = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORM_MAX_LENGTH, &activeUniformMaxLength);
    const int strSize = std::max(infoLogLength, std::max(activeAttributeMaxLength, activeUniformMaxLength));
    char* str = observer->getScratch()->create<char>(strSize);
    int32_t strLen = 0;

    auto pi = std::shared_ptr<ProgramInfo>(new ProgramInfo());

    int32_t linkStatus = 0;
    mImports.glGetProgramiv(program, GLenum::GL_LINK_STATUS, &linkStatus);
    pi->mLinkStatus = linkStatus;

    mImports.glGetProgramInfoLog(program, strSize, &strLen, str);
    pi->mInfoLog = std::string(str, strLen);

    int32_t activeUniforms = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_UNIFORMS, &activeUniforms);
    for (int32_t i = 0; i < activeUniforms; i++) {
        ActiveUniform au;
        mImports.glGetActiveUniform(program, i, strSize, &strLen, &au.mArraySize, &au.mType, str);
        au.mName = std::string(str, strLen);
        au.mLocation = mImports.glGetUniformLocation(program, str);
        pi->mActiveUniforms[i] = au;
    }

    int32_t activeAttributes = 0;
    mImports.glGetProgramiv(program, GLenum::GL_ACTIVE_ATTRIBUTES, &activeAttributes);
    for (int32_t i = 0; i < activeAttributes; i++) {
        ActiveAttribute aa;
        mImports.glGetActiveAttrib(program, i, strSize, &strLen, &aa.mArraySize, &aa.mType, str);
        aa.mName = std::string(str, strLen);
        aa.mLocation = mImports.glGetAttribLocation(program, str);
        pi->mActiveAttributes[i] = aa;
    }

    GAPID_DEBUG("Created ProgramInfo: LinkStatus=%i ActiveUniforms=%i ActiveAttributes=%i",
        linkStatus, activeUniforms, activeAttributes);

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

}

#undef ANDROID_NATIVE_MAKE_CONSTANT
#undef ANDROID_NATIVE_WINDOW_MAGIC
#undef ANDROID_NATIVE_BUFFER_MAGIC
