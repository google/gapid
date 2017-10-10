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

#include "core/cc/thread.h"
#include "gapii/cc/gles_spy.h"
#include "gapii/cc/gvr_spy.h"
#include "gapii/cc/vulkan_spy.h"

#include <atomic>
#include <memory>
#include <unordered_map>

namespace gapii {
class ConnectionStream;
class Spy : public GlesSpy, public GvrSpy, public VulkanSpy {
 public:
  // get lazily constructs and returns the singleton instance to the spy.
  static Spy* get();

  // writeHeader encodes the capture header to the encoder.
  void writeHeader();

  // resolve the imported functions. Call if the functions change due to
  // external factors.
  void resolveImports();

  CallObserver* enter(const char* name, uint32_t api);
  void exit();

  EGLBoolean eglInitialize(CallObserver* observer, EGLDisplay dpy,
                           EGLint* major, EGLint* minor);
  EGLContext eglCreateContext(CallObserver* observer, EGLDisplay display,
                              EGLConfig config, EGLContext share_context,
                              EGLint* attrib_list);
  EGLBoolean eglMakeCurrent(CallObserver* observer, EGLDisplay display, EGLSurface draw, EGLSurface read, EGLContext context);

  // Intercepted GLES methods to optionally fake no support for precompiled
  // shaders.
  void glProgramBinary(CallObserver* observer, uint32_t program,
                       uint32_t binary_format, void* binary,
                       int32_t binary_size);
  void glProgramBinaryOES(CallObserver* observer, uint32_t program,
                          uint32_t binary_format, void* binary,
                          int32_t binary_size);
  void glShaderBinary(CallObserver* observer, int32_t count, uint32_t* shaders,
                      uint32_t binary_format, void* binary,
                      int32_t binary_size);
  void glGetInteger64v(CallObserver* observer, uint32_t param, int64_t* values);
  void glGetIntegerv(CallObserver* observer, uint32_t param, int32_t* values);
  GLubyte* glGetString(CallObserver* observer, uint32_t name);
  GLubyte* glGetStringi(CallObserver* observer, uint32_t name, GLuint index);

  void gvr_frame_submit(CallObserver* observer, gvr_frame** frame, gvr_buffer_viewport_list* list, gvr_mat4_abi head_space_from_start_space);

  void onPostDrawCall(CallObserver* observer, uint8_t api) override;
  void onPreStartOfFrame(CallObserver* observer, uint8_t api) override;
  void onPostStartOfFrame() override;
  void onPreEndOfFrame(CallObserver* observer, uint8_t api) override;
  void onPostEndOfFrame() override;
  void onPostFence(CallObserver* observer) override;

  inline void RegisterSymbol(const std::string& name, void* symbol) {
    mSymbols.emplace(name, symbol);
  }

  inline void* LookupSymbol(const std::string& name) const {
    const auto symbol = mSymbols.find(name);
    return (symbol == mSymbols.end()) ? nullptr : symbol->second;
  }

  void setFakeGlError(CallObserver* observer, GLenum_Error error);
  uint32_t glGetError(CallObserver* observer);

 private:
  Spy();

  // observeFramebuffer captures the currently bound framebuffer's color
  // buffer, and writes it to a FramebufferObservation message.
  void observeFramebuffer(CallObserver* observer, uint8_t api);

  // getFramebufferAttachmentSize attempts to retrieve the currently bound
  // framebuffer's color buffer dimensions, returning true on success or
  // false if the dimensions could not be retrieved.
  bool getFramebufferAttachmentSize(CallObserver* observer, uint32_t& width, uint32_t& height);

  std::unordered_map<std::string, void*> mSymbols;

  int mNumFrames;
  // The number of frames that we want to suspend capture for before
  // we start.
  std::atomic<int> mSuspendCaptureFrames;

  // The connection stream to the server
  std::shared_ptr<ConnectionStream> mConnection;
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
