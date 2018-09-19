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

#include "core/cc/dl_loader.h"
#include "core/cc/get_gles_proc_address.h"
#include "core/cc/gl/formats.h"
#include "core/cc/gl/versions.h"
#include "core/cc/log.h"

#include <X11/Xresource.h>
#include <cstring>

namespace gapir {
namespace {

typedef XID GLXPbuffer;
typedef XID GLXDrawable;
typedef /*struct __GLXcontextRec*/ void* GLXContext;
typedef /*struct __GLXFBConfigRec*/ void* GLXFBConfig;

enum {
  // Used by glXChooseFBConfig.
  GLX_RED_SIZE = 8,
  GLX_GREEN_SIZE = 9,
  GLX_BLUE_SIZE = 10,
  GLX_ALPHA_SIZE = 11,
  GLX_DEPTH_SIZE = 12,
  GLX_STENCIL_SIZE = 13,
  GLX_DRAWABLE_TYPE = 0x8010,
  GLX_RENDER_TYPE = 0x8011,
  GLX_RGBA_BIT = 0x00000001,
  GLX_PBUFFER_BIT = 0x00000004,

  // Used by glXCreateNewContext.
  GLX_RGBA_TYPE = 0x8014,

  // Used by glXCreatePbuffer.
  GLX_PBUFFER_HEIGHT = 0x8040,
  GLX_PBUFFER_WIDTH = 0x8041,

  // Attribute name for glXCreateContextAttribsARB.
  GLX_CONTEXT_MAJOR_VERSION_ARB = 0x2091,
  GLX_CONTEXT_MINOR_VERSION_ARB = 0x2092,
  GLX_CONTEXT_FLAGS_ARB = 0x2094,
  GLX_CONTEXT_PROFILE_MASK_ARB = 0x9126,

  // Attribute value for glXCreateContextAttribsARB.
  GLX_CONTEXT_DEBUG_BIT_ARB = 0x0001,
  GLX_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB = 0x0002,
  GLX_CONTEXT_CORE_PROFILE_BIT_ARB = 0x0001,
  GLX_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB = 0x0002,
};

extern "C" {

typedef GLXFBConfig* (*pfn_glXChooseFBConfig)(Display* dpy, int screen,
                                              const int* attrib_list,
                                              int* nelements);
typedef GLXContext (*pfn_glXCreateNewContext)(Display* dpy, GLXFBConfig config,
                                              int render_type,
                                              GLXContext share_list,
                                              Bool direct);
typedef GLXPbuffer (*pfn_glXCreatePbuffer)(Display* dpy, GLXFBConfig config,
                                           const int* attrib_list);
typedef void (*pfn_glXDestroyPbuffer)(Display* dpy, GLXPbuffer pbuf);
typedef Bool (*pfn_glXMakeContextCurrent)(Display* dpy, GLXDrawable draw,
                                          GLXDrawable read, GLXContext ctx);
typedef Bool (*pfn_glXQueryVersion)(Display* dpy, int* maj, int* min);
typedef void (*pfn_glXDestroyContext)(Display* dpy, GLXContext ctx);
typedef void* (*pfn_glXGetProcAddress)(const char* procName);
typedef GLXContext (*glXCreateContextAttribsARBProc)(Display* dpy,
                                                     GLXFBConfig config,
                                                     GLXContext share_context,
                                                     Bool direct,
                                                     const int* attrib_list);

typedef int (*pfn_XFree)(void*);
typedef int (*pfn_XCloseDisplay)(Display*);
typedef Display* (*pfn_XOpenDisplay)(_Xconst char*);
typedef XErrorHandler (*pfn_XSetErrorHandler)(XErrorHandler);
typedef int (*pfn_XSync)(Display*, Bool);

}  // extern "C"

class GlesRendererImpl : public GlesRenderer {
 public:
  GlesRendererImpl(GlesRendererImpl* shared_context);
  virtual ~GlesRendererImpl() override;

  virtual Api* api() override;
  virtual void setBackbuffer(Backbuffer backbuffer) override;
  virtual void bind(bool resetViewportScissor) override;
  virtual void unbind() override;
  virtual const char* name() override;
  virtual const char* extensions() override;
  virtual const char* vendor() override;
  virtual const char* version() override;

 private:
  void createPbuffer(int width, int height);

  Backbuffer mBackbuffer;
  bool mNeedsResolve;
  Gles mApi;
  std::string mExtensions;
  bool mQueriedExtensions;

  Display* mDisplay;
  bool mOwnsDisplay;  // True if we created mDisplay
  GLXContext mContext;
  GLXContext mSharedContext;
  GLXPbuffer mPbuffer;
  GLXFBConfig mFBConfig;

  pfn_XFree fn_XFree;
  pfn_XCloseDisplay fn_XCloseDisplay;
  pfn_XOpenDisplay fn_XOpenDisplay;
  pfn_XSetErrorHandler fn_XSetErrorHandler;
  pfn_XSync fn_XSync;
  core::DlLoader libX;

  pfn_glXChooseFBConfig fn_glXChooseFBConfig;
  pfn_glXCreateNewContext fn_glXCreateNewContext;
  pfn_glXCreatePbuffer fn_glXCreatePbuffer;
  pfn_glXDestroyPbuffer fn_glXDestroyPbuffer;
  pfn_glXMakeContextCurrent fn_glXMakeContextCurrent;
  pfn_glXQueryVersion fn_glXQueryVersion;
  pfn_glXDestroyContext fn_glXDestroyContext;
  pfn_glXGetProcAddress fn_glXGetProcAddress;
};

// NB: We keep a reference the shared GL context, so "parent" context
//     must stay alive at least for the duration of this context.
//     We create "root" context for this purpose so it is satisfied.
//     TODO: Add assert/refcounting to enforce this.
GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared_context)
    : mNeedsResolve(false),
      mDisplay(nullptr),
      mContext(nullptr),
      mSharedContext(shared_context != nullptr ? shared_context->mContext : 0),
      mPbuffer(0),
      libX("libX11.so") {
  fn_XFree = (pfn_XFree)libX.lookup("XFree");
  fn_XCloseDisplay = (pfn_XCloseDisplay)libX.lookup("XCloseDisplay");
  fn_XOpenDisplay = (pfn_XOpenDisplay)libX.lookup("XOpenDisplay");
  fn_XSetErrorHandler = (pfn_XSetErrorHandler)libX.lookup("XSetErrorHandler");
  fn_XSync = (pfn_XSync)libX.lookup("XSync");

  fn_glXChooseFBConfig =
      (pfn_glXChooseFBConfig)core::GetGlesProcAddress("glXChooseFBConfig");
  fn_glXCreateNewContext =
      (pfn_glXCreateNewContext)core::GetGlesProcAddress("glXCreateNewContext");
  fn_glXCreatePbuffer =
      (pfn_glXCreatePbuffer)core::GetGlesProcAddress("glXCreatePbuffer");
  fn_glXDestroyPbuffer =
      (pfn_glXDestroyPbuffer)core::GetGlesProcAddress("glXDestroyPbuffer");
  fn_glXMakeContextCurrent =
      (pfn_glXMakeContextCurrent)core::GetGlesProcAddress(
          "glXMakeContextCurrent");
  fn_glXQueryVersion =
      (pfn_glXQueryVersion)core::GetGlesProcAddress("glXQueryVersion");
  fn_glXDestroyContext =
      (pfn_glXDestroyContext)core::GetGlesProcAddress("glXDestroyContext");
  fn_glXGetProcAddress =
      (pfn_glXGetProcAddress)core::GetGlesProcAddress("glXGetProcAddress");

  if (shared_context != nullptr) {
    // Ensure that shared contexts also share X-display.
    // Drivers are know to misbehave/crash without this.
    // NB: This relies on the shared_context to stay alive.
    mDisplay = shared_context->mDisplay;
    mOwnsDisplay = false;
  } else {
    mDisplay = fn_XOpenDisplay(nullptr);
    if (mDisplay == nullptr) {
      // Default display was not found. This may be because we're executing in
      // the bazel sandbox. Attempt to connect to the 0'th display instead.
      mDisplay = fn_XOpenDisplay(":0");
    }
    if (mDisplay == nullptr) {
      GAPID_FATAL("Unable to to open X display");
    }
    mOwnsDisplay = true;
  }

  int major;
  int minor;
  if (!fn_glXQueryVersion(mDisplay, &major, &minor) ||
      (major == 1 && minor < 3)) {
    GAPID_FATAL("GLX 1.3+ unsupported by X server (was %d.%d)", major, minor);
  }
}

GlesRendererImpl::~GlesRendererImpl() {
  unbind();

  if (mContext != nullptr) {
    fn_glXDestroyContext(mDisplay, mContext);
    GAPID_DEBUG("Destroyed context %p", mContext);
  }

  if (mPbuffer != 0) {
    fn_glXDestroyPbuffer(mDisplay, mPbuffer);
  }

  if (mOwnsDisplay && mDisplay != nullptr) {
    fn_XCloseDisplay(mDisplay);
  }
}

Api* GlesRendererImpl::api() { return &mApi; }

void GlesRendererImpl::createPbuffer(int width, int height) {
  if (mContext != nullptr) {
    unbind();  // Flush before yanking the surface.
  }

  if (mPbuffer != 0) {
    fn_glXDestroyPbuffer(mDisplay, mPbuffer);
    mPbuffer = 0;
  }
  const int pbufferAttribs[] = {
      // clang-format off
      GLX_PBUFFER_WIDTH, width,
      GLX_PBUFFER_HEIGHT, height,
      None
      // clang-format on
  };
  mPbuffer = fn_glXCreatePbuffer(mDisplay, mFBConfig, pbufferAttribs);
}

static void DebugCallback(uint32_t source, uint32_t type, Gles::GLuint id,
                          uint32_t severity, Gles::GLsizei length,
                          const Gles::GLchar* message, const void* user_param) {
  auto renderer = reinterpret_cast<const GlesRendererImpl*>(user_param);
  auto listener = renderer->getListener();
  if (listener != nullptr) {
    if (type == Gles::GLenum::GL_DEBUG_TYPE_ERROR ||
        severity == Gles::GLenum::GL_DEBUG_SEVERITY_HIGH) {
      listener->onDebugMessage(LOG_LEVEL_ERROR, Gles::INDEX, message);
    } else {
      listener->onDebugMessage(LOG_LEVEL_DEBUG, Gles::INDEX, message);
    }
  }
}

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

    // Find the FB config matching the requested format.
    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    const int visualAttribs[] = {
        // clang-format off
        GLX_RED_SIZE, r,
        GLX_GREEN_SIZE, g,
        GLX_BLUE_SIZE, b,
        GLX_ALPHA_SIZE, a,
        GLX_DEPTH_SIZE, d,
        GLX_STENCIL_SIZE, s,
        GLX_RENDER_TYPE, GLX_RGBA_BIT,
        GLX_DRAWABLE_TYPE, GLX_PBUFFER_BIT,
        None
        // clang-format on
    };
    int fbConfigsCount;
    GLXFBConfig* fbConfigs = fn_glXChooseFBConfig(
        mDisplay, DefaultScreen(mDisplay), visualAttribs, &fbConfigsCount);
    if (fbConfigs == nullptr) {
      GAPID_FATAL("Unable to find a suitable X framebuffer config");
    }
    mFBConfig = fbConfigs[0];
    fn_XFree(fbConfigs);
  }

  // Some exotic extensions let you create contexts without a backbuffer.
  // In these cases the backbuffer is zero size - just create a small one.
  int safe_width = (backbuffer.width > 0) ? backbuffer.width : 8;
  int safe_height = (backbuffer.height > 0) ? backbuffer.height : 8;
  createPbuffer(safe_width, safe_height);

  if (mContext == nullptr) {
    glXCreateContextAttribsARBProc glXCreateContextAttribsARB =
        (glXCreateContextAttribsARBProc)fn_glXGetProcAddress(
            "glXCreateContextAttribsARB");
    if (glXCreateContextAttribsARB == nullptr) {
      GAPID_FATAL("Unable to get address of glXCreateContextAttribsARB");
    }
    // Prevent X from taking down the process if the GL version is not
    // supported.
    auto oldHandler =
        fn_XSetErrorHandler([](Display*, XErrorEvent*) -> int { return 0; });
    for (auto gl_version : core::gl::sVersionSearchOrder) {
      // List of name-value pairs.
      const int contextAttribs[] = {
          // clang-format off
          GLX_RENDER_TYPE, GLX_RGBA_TYPE,
          GLX_CONTEXT_MAJOR_VERSION_ARB, gl_version.major,
          GLX_CONTEXT_MINOR_VERSION_ARB, gl_version.minor,
          GLX_CONTEXT_FLAGS_ARB, GLX_CONTEXT_DEBUG_BIT_ARB,
          GLX_CONTEXT_PROFILE_MASK_ARB, GLX_CONTEXT_CORE_PROFILE_BIT_ARB,
          None,
          // clang-format on
      };
      mContext = glXCreateContextAttribsARB(mDisplay, mFBConfig, mSharedContext,
                                            /* direct */ True, contextAttribs);
      if (mContext != nullptr) {
        GAPID_DEBUG("Created GL %i.%i context %p (shaded with context %p)",
                    gl_version.major, gl_version.minor, mContext,
                    mSharedContext);
        break;
      }
    }
    fn_XSetErrorHandler(oldHandler);
    if (mContext == nullptr) {
      GAPID_FATAL("Failed to create glX context");
    }
    fn_XSync(mDisplay, False);
    mNeedsResolve = true;
  }

  mBackbuffer = backbuffer;
}

void GlesRendererImpl::bind(bool resetViewportScissor) {
  if (!fn_glXMakeContextCurrent(mDisplay, mPbuffer, mPbuffer, mContext)) {
    GAPID_FATAL("Unable to make GLX context current");
  }

  if (mNeedsResolve) {
    mNeedsResolve = false;
    mApi.resolve();
  }

  if (mApi.mFunctionStubs.glDebugMessageCallback != nullptr) {
    mApi.mFunctionStubs.glDebugMessageCallback(
        reinterpret_cast<void*>(&DebugCallback), this);
    mApi.mFunctionStubs.glEnable(Gles::GLenum::GL_DEBUG_OUTPUT);
    mApi.mFunctionStubs.glEnable(Gles::GLenum::GL_DEBUG_OUTPUT_SYNCHRONOUS);
    GAPID_DEBUG("Enabled KHR_debug extension");
  }

  if (resetViewportScissor) {
    mApi.mFunctionStubs.glViewport(0, 0, mBackbuffer.width, mBackbuffer.height);
    mApi.mFunctionStubs.glScissor(0, 0, mBackbuffer.width, mBackbuffer.height);
  }
}

void GlesRendererImpl::unbind() {
  fn_glXMakeContextCurrent(mDisplay, None, None, nullptr);
}

const char* GlesRendererImpl::name() {
  return reinterpret_cast<const char*>(
      mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_RENDERER));
}

const char* GlesRendererImpl::extensions() {
  if (!mQueriedExtensions) {
    mQueriedExtensions = true;
    int32_t n, i;
    mApi.mFunctionStubs.glGetIntegerv(Gles::GLenum::GL_NUM_EXTENSIONS, &n);
    for (i = 0; i < n; i++) {
      if (i > 0) {
        mExtensions += " ";
      }
      mExtensions += reinterpret_cast<const char*>(
          mApi.mFunctionStubs.glGetStringi(Gles::GLenum::GL_EXTENSIONS, i));
    }
  }
  return &mExtensions[0];
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

GlesRenderer* GlesRenderer::create(GlesRenderer* shared_context) {
  if (core::hasGLorGLES() && core::DlLoader::can_load("libX11.so")) {
    return new GlesRendererImpl(
        reinterpret_cast<GlesRendererImpl*>(shared_context));
  } else {
    return nullptr;
  }
}

}  // namespace gapir
