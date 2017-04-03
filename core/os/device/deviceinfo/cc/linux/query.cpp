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

#include <X11/Xresource.h>
#include <GL/glx.h>
#include <cstring>
#include <string.h>

#include <sys/utsname.h>
#include <unistd.h>

#define STR_OR_UNKNOWN(x) ((x != nullptr) ? x : "<unknown>")

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

void destroyContext() {
    if (gContext.mPbuffer) {
        glXDestroyPbuffer(gContext.mDisplay, gContext.mPbuffer);
        gContext.mPbuffer = 0;
    }
    if (gContext.mContext) {
        glXDestroyContext(gContext.mDisplay, gContext.mContext);
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

bool createContext(void*) {
    memset(&gContext, 0, sizeof(gContext));

    if (uname(&gContext.mUbuf) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "gethostname returned error: %d", errno);
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
    gContext.mFBConfigs = glXChooseFBConfig(gContext.mDisplay,
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
    gContext.mContext = glXCreateNewContext(gContext.mDisplay,
                                            fbConfig,
                                            GLX_RGBA_TYPE,
                                            0,
                                            True);
    if (!gContext.mContext) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "glXCreateNewContext failed");
        destroyContext();
        return false;
    }
    const int pbufferAttribs[] = {
        GLX_PBUFFER_WIDTH, 32, GLX_PBUFFER_HEIGHT, 32, None
    };

    gContext.mPbuffer = glXCreatePbuffer(gContext.mDisplay, fbConfig, pbufferAttribs);
    if (!gContext.mPbuffer) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "glXCreatePbuffer failed");
        destroyContext();
        return false;
    }

    glXMakeContextCurrent(gContext.mDisplay, gContext.mPbuffer, gContext.mPbuffer, gContext.mContext);
    return true;
}

const char* contextError() {
	return gContext.mError;
}

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_pointeralignment(alignof(void*));
    memory_layout->set_pointersize(sizeof(void*));
    memory_layout->set_integersize(sizeof(int));
    memory_layout->set_sizesize(sizeof(size_t));
    memory_layout->set_u64alignment(alignof(uint64_t));
    memory_layout->set_endian(device::LittleEndian);

    abi->set_name("X86_64");
    abi->set_os(device::Linux);
    abi->set_architecture(device::X86_64);
    abi->set_allocated_memorylayout(memory_layout);
}

int cpuNumCores() { return gContext.mNumCores; }

const char* gpuName() { return "<unknown>"; }

const char* gpuVendor() { return "<unknown>"; }

const char* instanceName() { return gContext.mHostName; }

const char* instanceSerial()  { return gContext.mHostName; }

const char* hardwareName() { return STR_OR_UNKNOWN(gContext.mUbuf.machine); }

device::OSKind osKind() { return device::Linux; }

const char* osName() { return STR_OR_UNKNOWN(gContext.mUbuf.release); }

const char* osBuild() { return STR_OR_UNKNOWN(gContext.mUbuf.version); }

int osMajor() { return 0; }

int osMinor() { return 0; }

int osPoint() { return 0; }

}  // namespace query
