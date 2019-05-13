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

#include "gapir/cc/gles_renderer.h"
#include "gapir/cc/gles_gfx_api.h"

#include "core/cc/gl/formats.h"
#include "core/cc/log.h"

#include <EGL/egl.h>
#include <EGL/eglext.h>

namespace gapir {
namespace {

class GlesRendererImpl : public GlesRenderer {
 public:
  GlesRendererImpl(GlesRendererImpl* shared);
  virtual ~GlesRendererImpl() override;

  virtual Api* api() override;
  virtual void setBackbuffer(Backbuffer backbuffer) override;
  virtual void bind(bool resetViewportScissor) override;
  virtual void unbind() override;
  virtual void* createExternalImage(uint32_t texture) override;
  virtual bool frameDelimiter() override;
  virtual const char* name() override;
  virtual const char* extensions() override;
  virtual const char* vendor() override;
  virtual const char* version() override;

 private:
  void reset();

  Backbuffer mBackbuffer;
  bool mNeedsResolve;
  Gles mApi;

  EGLConfig mConfig;
  EGLContext mContext;
  EGLContext mSharedContext;
  EGLSurface mSurface;
  EGLDisplay mDisplay;
};

#define EGL_CHECK_ERROR(FORMAT, ...)                    \
  do {                                                  \
    EGLint err = eglGetError();                         \
    if (err != EGL_SUCCESS) {                           \
      GAPID_FATAL(FORMAT ": 0x%x", ##__VA_ARGS__, err); \
    }                                                   \
  } while (false)

GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared)
    : mNeedsResolve(false),
      mContext(EGL_NO_CONTEXT),
      mSharedContext(shared == nullptr ? EGL_NO_CONTEXT : shared->mContext),
      mSurface(EGL_NO_SURFACE) {
  mDisplay = eglGetDisplay(EGL_DEFAULT_DISPLAY);
  EGL_CHECK_ERROR("Failed to get EGL display");

  eglInitialize(mDisplay, nullptr, nullptr);
  EGL_CHECK_ERROR("Failed to initialize EGL");

  eglBindAPI(EGL_OPENGL_ES_API);
  EGL_CHECK_ERROR("Failed to bind EGL API");
}

GlesRendererImpl::~GlesRendererImpl() {
  unbind();

  if (mContext != nullptr) {
    eglDestroyContext(mDisplay, mContext);
    EGL_CHECK_ERROR("Failed to destroy context %p", mContext);
  }

  if (mSurface != nullptr) {
    eglDestroySurface(mDisplay, mSurface);
    EGL_CHECK_ERROR("Failed to destroy surface %p", mSurface);
  }

  eglTerminate(mDisplay);
  EGL_CHECK_ERROR("Failed to terminate EGL");

  eglReleaseThread();
  EGL_CHECK_ERROR("Failed to release EGL thread");
}

Api* GlesRendererImpl::api() { return &mApi; }

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
  if (mBackbuffer == backbuffer) {
    return;  // No change
  }

  if (mBackbuffer.format == backbuffer.format) {
    // Only a resize is necessary
    GAPID_INFO("Resizing renderer: %dx%d -> %dx%d", mBackbuffer.width,
               mBackbuffer.height, backbuffer.width, backbuffer.height);
  } else {
    if (mContext != nullptr) {
      GAPID_WARNING(
          "Attempting to change format of renderer: [0x%x, 0x%x, 0x%x] -> "
          "[0x%x, 0x%x, 0x%x]",
          mBackbuffer.format.color, mBackbuffer.format.depth,
          mBackbuffer.format.stencil, backbuffer.format.color,
          backbuffer.format.depth, backbuffer.format.stencil);
    }

    // Find a supported EGL context config.
    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    const int configAttribList[] = {
        // clang-format off
        EGL_RED_SIZE, r,
        EGL_GREEN_SIZE, g,
        EGL_BLUE_SIZE, b,
        EGL_ALPHA_SIZE, a,
        EGL_BUFFER_SIZE, r+g+b+a,
        EGL_DEPTH_SIZE, d,
        EGL_STENCIL_SIZE, s,
        EGL_SURFACE_TYPE, EGL_PBUFFER_BIT,
        EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT,
        EGL_NONE
        // clang-format on
    };
    int one = 1;
    eglChooseConfig(mDisplay, configAttribList, &mConfig, 1, &one);
    EGL_CHECK_ERROR("Failed to choose EGL config");
  }

  // Delete existing surface.
  if (mSurface != EGL_NO_SURFACE) {
    eglDestroySurface(mDisplay, mSurface);
    EGL_CHECK_ERROR("Failed to destroy EGL surface %p", mSurface);
    mSurface = EGL_NO_SURFACE;
  }

  // Create an EGL surface for the read/draw framebuffer.
  const int surfaceAttribList[] = {
      // clang-format off
      EGL_WIDTH, backbuffer.width,
      EGL_HEIGHT, backbuffer.height,
      EGL_NONE
      // clang-format on
  };
  mSurface = eglCreatePbufferSurface(mDisplay, mConfig, surfaceAttribList);
  EGL_CHECK_ERROR("Failed to create EGL pbuffer surface");

  if (mContext == nullptr) {
    // Create an EGL context.
    const int contextAttribList[] = {
        // clang-format off
        EGL_CONTEXT_CLIENT_VERSION, 2,
        EGL_NONE
        // clang-format on
    };
    mContext =
        eglCreateContext(mDisplay, mConfig, mSharedContext, contextAttribList);
    EGL_CHECK_ERROR("Failed to create EGL context");
    mNeedsResolve = true;
  }

  mBackbuffer = backbuffer;
}

void GlesRendererImpl::bind(bool resetViewportScissor) {
  eglMakeCurrent(mDisplay, mSurface, mSurface, mContext);
  EGL_CHECK_ERROR("Failed to make context %p current", mContext);

  if (mNeedsResolve) {
    mNeedsResolve = false;
    mApi.resolve();
  }

  if (resetViewportScissor) {
    mApi.mFunctionStubs.glViewport(0, 0, mBackbuffer.width, mBackbuffer.height);
    mApi.mFunctionStubs.glScissor(0, 0, mBackbuffer.width, mBackbuffer.height);
  }
}

void GlesRendererImpl::unbind() {
  eglMakeCurrent(mDisplay, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
  EGL_CHECK_ERROR("Failed to release EGL context");
}

void* GlesRendererImpl::createExternalImage(uint32_t texture) {
  return mApi.mFunctionStubs.eglCreateImageKHR(
      mDisplay, mContext, EGL_GL_TEXTURE_2D_KHR,
      (EGLClientBuffer)(uint64_t)texture, nullptr);
}

bool GlesRendererImpl::frameDelimiter() {
  if (mSurface != nullptr) {
    eglSwapBuffers(mDisplay, mSurface);
    EGL_CHECK_ERROR("Failed to swap buffers");
    return true;
  }
  return false;
}

const char* GlesRendererImpl::name() {
  return reinterpret_cast<const char*>(
      mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_RENDERER));
}

const char* GlesRendererImpl::extensions() {
  return reinterpret_cast<const char*>(
      mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_EXTENSIONS));
}

const char* GlesRendererImpl::vendor() {
  return reinterpret_cast<const char*>(
      mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_VENDOR));
}

const char* GlesRendererImpl::version() {
  return reinterpret_cast<const char*>(
      mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_VERSION));
}

}  // anonymous namespace

GlesRenderer* GlesRenderer::create(GlesRenderer* sharedContext) {
  return new GlesRendererImpl(
      reinterpret_cast<GlesRendererImpl*>(sharedContext));
}

}  // namespace gapir
