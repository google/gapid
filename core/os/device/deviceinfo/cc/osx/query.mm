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

#import <OpenGL/gl.h>
#import <AppKit/AppKit.h>

#include <cstring>

#include <unistd.h>

#include <sys/types.h>
#include <sys/sysctl.h>

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace query {

struct Context {
    char mError[512];
    NSOpenGLPixelFormat* mGlFmt;
    NSOpenGLContext* mGlCtx;
    NSOperatingSystemVersion mOsVersion;
    int mNumCores;
    char* mHwModel;
    char mHostName[512];
};

static Context gContext;
static int gContextRefCount = 0;

void destroyContext() {
    if (--gContextRefCount > 0) {
        return;
    }

    if (gContext.mHwModel) {
        delete [] gContext.mHwModel;
    }
    if (gContext.mGlFmt) {
        [gContext.mGlFmt release];
        gContext.mGlFmt = nullptr;
    }
    if (gContext.mGlCtx) {
        [gContext.mGlCtx release];
        gContext.mGlCtx = nullptr;
    }
}

bool createContext(void* platform_data) {
    if (gContextRefCount++ > 0) {
        return true;
    }

    memset(&gContext, 0, sizeof(gContext));

    NSOpenGLPixelFormatAttribute attributes[] = {
        NSOpenGLPFANoRecovery,
        NSOpenGLPFAColorSize, (NSOpenGLPixelFormatAttribute)32,
        NSOpenGLPFADepthSize, (NSOpenGLPixelFormatAttribute)24,
        NSOpenGLPFAStencilSize, (NSOpenGLPixelFormatAttribute)8,
        NSOpenGLPFAAccelerated,
        NSOpenGLPFABackingStore,
        NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core,
        (NSOpenGLPixelFormatAttribute)0
    };

    size_t len = 0;
    int mib[2] = {CTL_HW, HW_MODEL};
    sysctl(mib, 2, nullptr, &len, nullptr, 0);
    gContext.mHwModel = new char[len];
    if (sysctl(mib, 2, gContext.mHwModel, &len, nullptr, 0) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "sysctl {CTL_HW, HW_MODEL} returned error: %d", errno);
        destroyContext();
        return false;
    }

    len = sizeof(gContext.mNumCores);
    if (sysctlbyname("hw.logicalcpu_max", &gContext.mNumCores, &len, nullptr, 0) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "sysctlbyname 'hw.logicalcpu_max' returned error: %d", errno);
        destroyContext();
        return false;
    }

    if (gethostname(gContext.mHostName, sizeof(gContext.mHostName)) != 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "gethostname returned error: %d", errno);
        destroyContext();
        return false;
    }

    gContext.mGlFmt = [[NSOpenGLPixelFormat alloc] initWithAttributes:attributes];
    if (gContext.mGlFmt == 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "NSOpenGLPixelFormat alloc failed");
        destroyContext();
        return false;
    }

    gContext.mGlCtx = [[NSOpenGLContext alloc] initWithFormat:gContext.mGlFmt shareContext:nil];
    if (gContext.mGlCtx == 0) {
		snprintf(gContext.mError, sizeof(gContext.mError),
				 "NSOpenGLContext alloc failed");
        destroyContext();
        return false;
    }

    [gContext.mGlCtx makeCurrentContext];

    gContext.mOsVersion = [[NSProcessInfo processInfo] operatingSystemVersion];

    return true;
}

const char* contextError() {
	return gContext.mError;
}

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
    abi->set_name("X86_64");
    abi->set_os(device::OSX);
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

const char* hardwareName() { return STR_OR_EMPTY(gContext.mHwModel); }

device::OSKind osKind() { return device::OSX; }

const char* osName() { return "OSX"; }

const char* osBuild() { return ""; }

int osMajor() { return gContext.mOsVersion.majorVersion; }

int osMinor() { return gContext.mOsVersion.minorVersion; }

int osPoint() { return gContext.mOsVersion.patchVersion; }

}  // namespace query
