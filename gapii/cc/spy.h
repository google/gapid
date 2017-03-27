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

#ifndef GAPII_SPY_H
#define GAPII_SPY_H
#include "gapii/cc/core_spy.h"
#include "gapii/cc/vulkan_spy.h"
#include "gapii/cc/gles_spy.h"
#include "core/cc/thread.h"

#include <memory>
#include <unordered_map>

namespace gapii {

class Spy : public GlesSpy, public VulkanSpy, public CoreSpy {
public:
    // get lazily constructs and returns the singleton instance to the spy.
    static Spy* get();

    // resolve the imported functions. Call if the functions change due to
    // external factors.
    void resolveImports();

    EGLBoolean eglInitialize(CallObserver* observer, EGLDisplay dpy, EGLint* major, EGLint* minor);
    EGLContext eglCreateContext(CallObserver* observer, EGLDisplay display, EGLConfig config,
                                EGLContext share_context, EGLint* attrib_list);

    // Intercepted GLES methods to optionally fake no support for precompiled shaders.
    void glProgramBinary(CallObserver* observer, uint32_t program, uint32_t binary_format, void* binary, int32_t binary_size);
    void glProgramBinaryOES(CallObserver* observer, uint32_t program, uint32_t binary_format, void* binary, int32_t binary_size);
    void glShaderBinary(CallObserver* observer, int32_t count, uint32_t* shaders, uint32_t binary_format, void* binary, int32_t binary_size);
    void glGetInteger64v(CallObserver* observer, uint32_t param, int64_t* values);
    void glGetIntegerv(CallObserver* observer, uint32_t param, int32_t* values);
    GLubyte* glGetString(CallObserver* observer, uint32_t name);
    GLubyte* glGetStringi(CallObserver* observer, uint32_t name, GLuint index);

    void onPostDrawCall() override;
    void onPreEndOfFrame() override;
    void onPostEndOfFrame(CallObserver* observer) override;
    void onPostFence(CallObserver* observer) override;

    inline void RegisterSymbol(const std::string& name, void* symbol) {
        mSymbols.emplace(name, symbol);
    }

    inline void* LookupSymbol(const std::string& name) const {
        const auto symbol = mSymbols.find(name);
        return (symbol == mSymbols.end()) ? nullptr : symbol->second;
    }

    void setFakeGlError(GLenum_Error error);
    uint32_t glGetError(CallObserver* observer);

protected:
    virtual void onThreadSwitched(CallObserver* observer, uint64_t threadID) override;

private:
    Spy();

    // observeFramebuffer captures the currently bound framebuffer's color
    // buffer, and writes it to a FramebufferObservation atom.
    void observeFramebuffer();

    // getFramebufferAttachmentSize attempts to retrieve the currently bound
    // framebuffer's color buffer dimensions, returning true on success or
    // false if the dimensions could not be retrieved.
    bool getFramebufferAttachmentSize(uint32_t& width, uint32_t& height);

    // getDeviceProperties retrieves properties about the device, for example
    // by running getprop on Android device and parsing the output. The method
    // returns a map of properties to values.
    std::unordered_map<std::string, std::string> getDeviceProperties();

    std::shared_ptr<core::Encoder> mEncoder;
    std::unordered_map<std::string, void*> mSymbols;

    int mNumFrames;
    // The number of frames that we want to suspend capture for before
    // we start.
    std::atomic<int> mSuspendCaptureFrames;
    // The number of frames that we want to capture
    int mCaptureFrames;
    int mNumDraws;
    int mNumDrawsPerFrame;
    int mObserveFrameFrequency;
    int mObserveDrawFrequency;
    bool mDisablePrecompiledShaders;
    bool mRecordGLErrorState;

    std::unordered_map<ContextID, GLenum_Error> mFakeGlError;
    std::unique_ptr<core::AsyncJob> mDeferStartJob;
};

}  // namespace gapii

#endif  // GAPII_SPY_H
