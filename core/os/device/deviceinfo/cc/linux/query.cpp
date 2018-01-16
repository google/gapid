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

#include "core/cc/gl/versions.h"
#include "core/cc/get_gles_proc_address.h"

#include <X11/Xresource.h>
#include <GL/glx.h>
#include <cstring>
#include <string.h>

#include <sys/utsname.h>
#include <unistd.h>

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace query {

struct Context {
    char mError[512];
    Display* mDisplay;
    GLXFBConfig* mFBConfigs;
    GLXContext mContext;
    GLXPbuffer mPbuffer;
    int mNumCores;
    utsname mUbuf;
    char mHostName[512];
};

static Context gContext;
static int gContextRefCount = 0;

typedef GLXFBConfig* (*pfn_glXChooseFBConfig)(Display* dpy, int screen, const int* attrib_list, int* nelements);
typedef GLXContext (*pfn_glXCreateNewContext)(Display* dpy, GLXFBConfig config, int render_type, GLXContext shader_list, bool direct);
typedef GLXPbuffer (*pfn_glXCreatePbuffer)(Display* dpy, GLXFBConfig config, const int* attrib_list);
typedef void (*pfn_glXDestroyPbuffer)(Display*  dpy, GLXPbuffer  pbuf);
typedef void (*pfn_glXDestroyContext)(Display *  dpy, GLXContext  ctx);


void destroyContext() {
    if (--gContextRefCount > 0) {
        return;
    }

    pfn_glXDestroyPbuffer fn_glXDestroyPbuffer = (pfn_glXDestroyPbuffer)core::GetGlesProcAddress("glXDestroyPbuffer", true);
    pfn_glXDestroyContext fn_glXDestroyContext = (pfn_glXDestroyContext)core::GetGlesProcAddress("glXDestroyContext", true);


    if (gContext.mPbuffer && fn_glXDestroyPbuffer) {
        (*fn_glXDestroyPbuffer)(gContext.mDisplay, gContext.mPbuffer);
        gContext.mPbuffer = 0;
    }
    if (gContext.mContext && fn_glXDestroyContext) {
        (*fn_glXDestroyContext)(gContext.mDisplay, gContext.mContext);
        gContext.mContext = nullptr;
    }
    if (gContext.mFBConfigs) {
        XFree(gContext.mFBConfigs);
        gContext.mFBConfigs = nullptr;
    }
    if (gContext.mDisplay) {
        XCloseDisplay(gContext.mDisplay);
        gContext.mDisplay = nullptr;
    }
}

bool createContext(void* platform_data) {
    if (gContextRefCount++ > 0) {
        return true;
    }
    pfn_glXChooseFBConfig fn_glXChooseFBConfig= (pfn_glXChooseFBConfig)core::GetGlesProcAddress("glXChooseFBConfig", true);
    pfn_glXCreateNewContext fn_glXCreateNewContext = (pfn_glXCreateNewContext)core::GetGlesProcAddress("glXCreateNewContext", true);
    pfn_glXCreatePbuffer fn_glXCreatePbuffer = (pfn_glXCreatePbuffer)core::GetGlesProcAddress("glXCreatePbuffer", true);
    if (!fn_glXChooseFBConfig || !fn_glXCreateNewContext || !fn_glXCreatePbuffer) {
        return false;
    }

    memset(&gContext, 0, sizeof(gContext));

    if (uname(&gContext.mUbuf) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "uname returned error: %d", errno);
        destroyContext();
        return false;
    }

    gContext.mNumCores = sysconf(_SC_NPROCESSORS_CONF);

    if (gethostname(gContext.mHostName, sizeof(gContext.mHostName)) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "gethostname returned error: %d", errno);
        destroyContext();
        return false;
    }

    gContext.mDisplay = XOpenDisplay(0);
    if (!gContext.mDisplay) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "XOpenDisplay returned nullptr");
        destroyContext();
        return false;
    }

    const int visualAttribs[] = {
        GLX_RED_SIZE, 8,
        GLX_GREEN_SIZE, 8,
        GLX_BLUE_SIZE, 8,
        GLX_ALPHA_SIZE, 8,
        GLX_DEPTH_SIZE, 24,
        GLX_STENCIL_SIZE, 8,
        GLX_RENDER_TYPE, GLX_RGBA_BIT,
        GLX_DRAWABLE_TYPE, GLX_PBUFFER_BIT,
        None
    };
    int fbConfigsCount = 0;
    gContext.mFBConfigs = (*fn_glXChooseFBConfig)(gContext.mDisplay,
                                            DefaultScreen(gContext.mDisplay),
                                            visualAttribs,
                                            &fbConfigsCount);
    if (!gContext.mFBConfigs) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "glXChooseFBConfig failed");
        destroyContext();
        return false;
    }

    GLXFBConfig fbConfig = gContext.mFBConfigs[0];

    typedef GLXContext (*glXCreateContextAttribsARBProc)(
            Display *dpy, GLXFBConfig config, GLXContext share_context,
            Bool direct, const int *attrib_list);

    glXCreateContextAttribsARBProc glXCreateContextAttribsARB =
        (glXCreateContextAttribsARBProc)core::GetGlesProcAddress("glXCreateContextAttribsARB", true);
    if (glXCreateContextAttribsARB == nullptr) {
        gContext.mContext = (*fn_glXCreateNewContext)(gContext.mDisplay,
                                                fbConfig,
                                                GLX_RGBA_TYPE,
                                                nullptr,
                                                True);
    } else {
        // Prevent X from taking down the process if the GL version is not supported.
        auto oldHandler = XSetErrorHandler([](Display*, XErrorEvent*)->int{ return 0; });
        for (auto gl_version : core::gl::sVersionSearchOrder) {
            // List of name-value pairs.
            const int contextAttribs[] = {
                GLX_RENDER_TYPE, GLX_RGBA_TYPE,
                GLX_CONTEXT_MAJOR_VERSION_ARB, gl_version.major,
                GLX_CONTEXT_MINOR_VERSION_ARB, gl_version.minor,
                GLX_CONTEXT_FLAGS_ARB, GLX_CONTEXT_DEBUG_BIT_ARB,
                GLX_CONTEXT_PROFILE_MASK_ARB, GLX_CONTEXT_CORE_PROFILE_BIT_ARB,
                None,
            };
            gContext.mContext = glXCreateContextAttribsARB(
                gContext.mDisplay, fbConfig, nullptr, /* direct */ True, contextAttribs);
            if (gContext.mContext != nullptr) {
                break;
            }
        }
        XSetErrorHandler(oldHandler);
    }

    if (!gContext.mContext) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "glXCreateNewContext failed");
        destroyContext();
        return false;
    }
    const int pbufferAttribs[] = {
        GLX_PBUFFER_WIDTH, 32, GLX_PBUFFER_HEIGHT, 32, None
    };

    gContext.mPbuffer = (*fn_glXCreatePbuffer)(gContext.mDisplay, fbConfig, pbufferAttribs);
    if (!gContext.mPbuffer) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "glXCreatePbuffer failed");
        destroyContext();
        return false;
    }
    typedef Bool
        (*glXMakeContextCurrentProc)(Display *dpy, GLXDrawable draw, GLXDrawable read, GLXContext ctx);

    glXMakeContextCurrentProc glXMakeContextCurrent =
        (glXMakeContextCurrentProc)core::GetGlesProcAddress("glXMakeContextCurrent", true);
    glXMakeContextCurrent(gContext.mDisplay, gContext.mPbuffer, gContext.mPbuffer, gContext.mContext);
    return true;
}

const char* contextError() {
	return gContext.mError;
}

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
    abi->set_name("x86_64");
    abi->set_os(device::Linux);
    abi->set_architecture(device::X86_64);
    abi->set_allocated_memorylayout(currentMemoryLayout());
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

}  // namespace query
