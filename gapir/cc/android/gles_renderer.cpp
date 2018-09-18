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

namespace gapir {
namespace {

class GlesRendererImpl : public GlesRenderer {
 public:
  GlesRendererImpl(GlesRendererImpl* shared);
  virtual ~GlesRendererImpl() override;

  virtual Api* api() override;
  virtual void setBackbuffer(Backbuffer backbuffer) override;
  virtual void bind() override;
  virtual void unbind() override;
  virtual const char* name() override;
  virtual const char* extensions() override;
  virtual const char* vendor() override;
  virtual const char* version() override;

 private:
  void reset();

  Backbuffer mBackbuffer;
  bool mBound;
  bool mNeedsResolve;
  Gles mApi;

  EGLContext mContext;
  EGLContext mSharedContext;
  EGLSurface mSurface;
  EGLDisplay mDisplay;
};

GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared)
    : mBound(false),
      mNeedsResolve(true),
      mContext(EGL_NO_CONTEXT),
      mSharedContext(shared == nullptr ? EGL_NO_CONTEXT : shared->mContext),
      mSurface(EGL_NO_SURFACE) {
  mDisplay = eglGetDisplay(EGL_DEFAULT_DISPLAY);
  EGLint error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to get EGL display: %d", error);
  }

  eglInitialize(mDisplay, nullptr, nullptr);
  error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to initialize EGL: %d", error);
  }

  eglBindAPI(EGL_OPENGL_ES_API);
  error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to bind EGL API: %d", error);
  }

  // Initialize with a default target.
  setBackbuffer(Backbuffer(8, 8, core::gl::GL_RGBA8,
                           core::gl::GL_DEPTH24_STENCIL8,
                           core::gl::GL_DEPTH24_STENCIL8));
}

GlesRendererImpl::~GlesRendererImpl() {
  reset();

  eglTerminate(mDisplay);
  EGLint error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_WARNING("Failed to terminate EGL: %d", error);
  }

  eglReleaseThread();
  error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_WARNING("Failed to release EGL thread: %d", error);
  }
}

Api* GlesRendererImpl::api() { return &mApi; }

void GlesRendererImpl::reset() {
  unbind();

  if (mSurface != EGL_NO_SURFACE) {
    eglDestroySurface(mDisplay, mSurface);
    EGLint error = eglGetError();
    if (error != EGL_SUCCESS) {
      GAPID_WARNING("Failed to destroy EGL surface: %d", error);
    }
    mSurface = EGL_NO_SURFACE;
  }

  if (mContext != EGL_NO_CONTEXT) {
    eglDestroyContext(mDisplay, mContext);
    EGLint error = eglGetError();
    if (error != EGL_SUCCESS) {
      GAPID_WARNING("Failed to destroy EGL context: %d", error);
    }
    mContext = EGL_NO_CONTEXT;
  }

  mBackbuffer = Backbuffer();
}

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
  if (mContext != EGL_NO_CONTEXT && mBackbuffer == backbuffer) {
    // No change
    return;
  }

  // TODO: Check for and handle resizing path.

  const bool wasBound = mBound;

  reset();

  int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
  core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
  core::gl::getDepthBits(backbuffer.format.depth, d);
  core::gl::getStencilBits(backbuffer.format.stencil, s);

  // Find a supported EGL context config.
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
  EGLConfig eglConfig;
  eglChooseConfig(mDisplay, configAttribList, &eglConfig, 1, &one);
  EGLint error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to choose EGL config: %d", error);
  }

  // Create an EGL context.
  const int contextAttribList[] = {
      // clang-format off
      EGL_CONTEXT_CLIENT_VERSION, 2,
      EGL_NONE
      // clang-format on
  };
  mContext =
      eglCreateContext(mDisplay, eglConfig, mSharedContext, contextAttribList);
  error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to create EGL context: %d", error);
  }

  // Create an EGL surface for the read/draw framebuffer.
  const int surfaceAttribList[] = {
      // clang-format off
      EGL_WIDTH, backbuffer.width,
      EGL_HEIGHT, backbuffer.height,
      EGL_NONE
      // clang-format on
  };
  mSurface = eglCreatePbufferSurface(mDisplay, eglConfig, surfaceAttribList);
  error = eglGetError();
  if (error != EGL_SUCCESS) {
    GAPID_FATAL("Failed to create EGL pbuffer surface: %d", error);
  }

  mBackbuffer = backbuffer;
  mNeedsResolve = true;

  if (wasBound) {
    bind();
  }
}

void GlesRendererImpl::bind() {
  if (!mBound) {
    eglMakeCurrent(mDisplay, mSurface, mSurface, mContext);
    EGLint error = eglGetError();
    if (error != EGL_SUCCESS) {
      GAPID_FATAL("Failed to make EGL current: %d", error);
    }

    mBound = true;

    if (mNeedsResolve) {
      mNeedsResolve = false;
      mApi.resolve();
    }
  }
}

void GlesRendererImpl::unbind() {
  if (mBound) {
    eglMakeCurrent(mDisplay, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
    EGLint error = eglGetError();
    if (error != EGL_SUCCESS) {
      GAPID_WARNING("Failed to release EGL context: %d", error);
    }
    mBound = false;
  }
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
