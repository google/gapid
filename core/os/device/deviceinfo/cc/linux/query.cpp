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

#include "../query.h"

#include "core/cc/dl_loader.h"
#include "core/cc/get_gles_proc_address.h"
#include "core/cc/gl/versions.h"

#include <GL/glx.h>
#include <X11/Xresource.h>
#include <string.h>
#include <cstring>

#include <sys/utsname.h>
#include <unistd.h>

#if defined(__LP64__)
#define SYSTEM_LIB_PATH "/system/lib64/"
#else
#define SYSTEM_LIB_PATH "/system/lib/"
#endif

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace query {

struct Context {
  Display* mDisplay;
  GLXFBConfig* mFBConfigs;
  GLXContext mGlCtx;
  GLXPbuffer mPbuffer;
  int mNumCores;
  utsname mUbuf;
  char mHostName[512];
};

static Context gContext;
static int gContextRefCount = 0;

typedef GLXFBConfig* (*pfn_glXChooseFBConfig)(Display* dpy, int screen,
                                              const int* attrib_list,
                                              int* nelements);
typedef GLXContext (*pfn_glXCreateNewContext)(Display* dpy, GLXFBConfig config,
                                              int render_type,
                                              GLXContext shader_list,
                                              bool direct);
typedef GLXPbuffer (*pfn_glXCreatePbuffer)(Display* dpy, GLXFBConfig config,
                                           const int* attrib_list);
typedef void (*pfn_glXDestroyPbuffer)(Display* dpy, GLXPbuffer pbuf);
typedef void (*pfn_glXDestroyContext)(Display* dpy, GLXContext ctx);
typedef Bool (*pfn_glXMakeContextCurrent)(Display* dpy, GLXDrawable draw,
                                          GLXDrawable read, GLXContext ctx);
typedef GLXContext (*pfn_glXCreateContextAttribsARB)(Display* dpy,
                                                     GLXFBConfig config,
                                                     GLXContext share_context,
                                                     Bool direct,
                                                     const int* attrib_list);

typedef int (*pfn_XFree)(void*);
typedef int (*pfn_XCloseDisplay)(Display*);
typedef Display* (*pfn_XOpenDisplay)(_Xconst char*);
typedef XErrorHandler (*pfn_XSetErrorHandler)(XErrorHandler);

void destroyContext() {
  if (--gContextRefCount > 0) {
    return;
  }

  if (!core::DlLoader::can_load("libX11.so")) {
    return;
  }

  if (!core::hasGLorGLES()) {
    return;
  }

  core::DlLoader libX("libX11.so");
  pfn_XFree fn_XFree = (pfn_XFree)libX.lookup("XFree");
  pfn_XCloseDisplay fn_XCloseDisplay =
      (pfn_XCloseDisplay)libX.lookup("XCloseDisplay");

  pfn_glXDestroyPbuffer fn_glXDestroyPbuffer =
      (pfn_glXDestroyPbuffer)core::GetGlesProcAddress("glXDestroyPbuffer");
  pfn_glXDestroyContext fn_glXDestroyContext =
      (pfn_glXDestroyContext)core::GetGlesProcAddress("glXDestroyContext");

  if (gContext.mPbuffer && fn_glXDestroyPbuffer) {
    (*fn_glXDestroyPbuffer)(gContext.mDisplay, gContext.mPbuffer);
    gContext.mPbuffer = 0;
  }
  if (gContext.mGlCtx && fn_glXDestroyContext) {
    (*fn_glXDestroyContext)(gContext.mDisplay, gContext.mGlCtx);
    gContext.mGlCtx = nullptr;
  }
  if (gContext.mFBConfigs) {
    fn_XFree(gContext.mFBConfigs);
    gContext.mFBConfigs = nullptr;
  }
  if (gContext.mDisplay) {
    fn_XCloseDisplay(gContext.mDisplay);
    gContext.mDisplay = nullptr;
  }
}

void createGlContext() {
  if (!core::hasGLorGLES()) {
    return;
  }
  auto fn_glXChooseFBConfig =
      (pfn_glXChooseFBConfig)core::GetGlesProcAddress("glXChooseFBConfig");
  auto fn_glXCreateNewContext =
      (pfn_glXCreateNewContext)core::GetGlesProcAddress("glXCreateNewContext");
  auto fn_glXCreatePbuffer =
      (pfn_glXCreatePbuffer)core::GetGlesProcAddress("glXCreatePbuffer");
  auto fn_glXMakeContextCurrent =
      (pfn_glXMakeContextCurrent)core::GetGlesProcAddress(
          "glXMakeContextCurrent");
  auto fn_glXCreateContextAttribsARB =
      (pfn_glXCreateContextAttribsARB)core::GetGlesProcAddress(
          "glXCreateContextAttribsARB");

  if (!fn_glXChooseFBConfig || !fn_glXCreateNewContext ||
      !fn_glXCreatePbuffer || !fn_glXMakeContextCurrent) {
    return;
  }

  if (!core::DlLoader::can_load("libX11.so")) {
    return;
  }

  core::DlLoader libX("libX11.so");

  pfn_XOpenDisplay fn_XOpenDisplay =
      (pfn_XOpenDisplay)libX.lookup("XOpenDisplay");
  pfn_XSetErrorHandler fn_XSetErrorHandler =
      (pfn_XSetErrorHandler)libX.lookup("XSetErrorHandler");

  gContext.mDisplay = fn_XOpenDisplay(nullptr);
  if (gContext.mDisplay == nullptr) {
    return;
  }

  const int visualAttribs[] = {
      // clang-format off
      GLX_RED_SIZE, 8,
      GLX_GREEN_SIZE, 8,
      GLX_BLUE_SIZE, 8,
      GLX_ALPHA_SIZE, 8,
      GLX_DEPTH_SIZE, 24,
      GLX_STENCIL_SIZE, 8,
      GLX_RENDER_TYPE, GLX_RGBA_BIT,
      GLX_DRAWABLE_TYPE, GLX_PBUFFER_BIT,
      None
      // clang-format on
  };
  int fbConfigsCount = 0;
  gContext.mFBConfigs = (*fn_glXChooseFBConfig)(
      gContext.mDisplay, DefaultScreen(gContext.mDisplay), visualAttribs,
      &fbConfigsCount);
  if (!gContext.mFBConfigs) {
    return;
  }

  GLXFBConfig fbConfig = gContext.mFBConfigs[0];

  if (fn_glXCreateContextAttribsARB == nullptr) {
    gContext.mGlCtx = (*fn_glXCreateNewContext)(gContext.mDisplay, fbConfig,
                                                GLX_RGBA_TYPE, nullptr, True);
  } else {
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
      gContext.mGlCtx =
          fn_glXCreateContextAttribsARB(gContext.mDisplay, fbConfig, nullptr,
                                        /* direct */ True, contextAttribs);
      if (gContext.mGlCtx != nullptr) {
        break;
      }
    }
    fn_XSetErrorHandler(oldHandler);
  }

  if (!gContext.mGlCtx) {
    return;
  }

  const int pbufferAttribs[] = {GLX_PBUFFER_WIDTH, 32, GLX_PBUFFER_HEIGHT, 32,
                                None};

  gContext.mPbuffer =
      fn_glXCreatePbuffer(gContext.mDisplay, fbConfig, pbufferAttribs);
  if (!gContext.mPbuffer) {
    return;
  }

  fn_glXMakeContextCurrent(gContext.mDisplay, gContext.mPbuffer,
                           gContext.mPbuffer, gContext.mGlCtx);
}

bool createContext(std::string* errorMsg) {
  if (gContextRefCount++ > 0) {
    return true;
  }

  memset(&gContext, 0, sizeof(gContext));

  if (uname(&gContext.mUbuf) != 0) {
    errorMsg->append("uname returned error: " + std::to_string(errno));
    destroyContext();
    return false;
  }

  gContext.mNumCores = sysconf(_SC_NPROCESSORS_CONF);

  if (gethostname(gContext.mHostName, sizeof(gContext.mHostName)) != 0) {
    errorMsg->append("gethostname returned error: " + std::to_string(errno));
    destroyContext();
    return false;
  }

  createGlContext();

  return true;
}

bool hasGLorGLES() { return gContext.mGlCtx != nullptr; }

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
  abi->set_name("x86_64");
  abi->set_os(device::Linux);
  abi->set_architecture(device::X86_64);
  abi->set_allocated_memory_layout(currentMemoryLayout());
}

device::ABI* currentABI() {
  auto out = new device::ABI();
  abi(0, out);
  return out;
}

int cpuNumCores() { return gContext.mNumCores; }

const char* gpuName() { return ""; }

const char* gpuVendor() { return ""; }

const char* instanceName() { return gContext.mHostName; }

const char* hardwareName() { return STR_OR_EMPTY(gContext.mUbuf.machine); }

device::OSKind osKind() { return device::Linux; }

const char* osName() { return STR_OR_EMPTY(gContext.mUbuf.release); }

const char* osBuild() { return STR_OR_EMPTY(gContext.mUbuf.version); }

int osMajor() { return 0; }

int osMinor() { return 0; }

int osPoint() { return 0; }

void glDriverPlatform(device::OpenGLDriver*) {}

device::VulkanProfilingLayers* get_vulkan_profiling_layers() {
  auto layers = new device::VulkanProfilingLayers();
  layers->set_cpu_timing(true);
  layers->set_memory_tracker(true);
  return layers;
}

bool hasAtrace() { return false; }

}  // namespace query
