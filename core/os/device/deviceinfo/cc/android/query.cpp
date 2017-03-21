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

#include "platform_data.h"
#include "jni_helpers.h"

#include "../query.h"

#include <cstring>

#include <android/log.h>
#include <EGL/egl.h>
#include <GLES2/gl2.h>

#define LOG_ERR(...) \
    __android_log_print(ANDROID_LOG_ERROR, "GAPID", __VA_ARGS__);

#define LOG_WARN(...) \
    __android_log_print(ANDROID_LOG_WARN, "GAPID", __VA_ARGS__);

typedef int GLint;
typedef unsigned int GLuint;
typedef uint8_t GLubyte;

namespace query {

struct Context {
    EGLDisplay mDisplay;
    EGLSurface mSurface;
    EGLContext mContext;
    int mNumCores;
    std::string mHost;
    std::string mSerial;
    std::string mHardware;
    std::string mOSName;
    std::string mOSBuild;
    int mOSVersion;
    int mOSVersionMajor;
    int mOSVersionMinor;
    std::vector<std::string> mSupportedABIs;
    device::Architecture mCpuArchitecture;
};

static Context gContext;

void destroyContext() {
    if (gContext.mContext) {
        eglDestroyContext(gContext.mDisplay, gContext.mContext);
        gContext.mContext = 0;
    }
    if (gContext.mSurface) {
        eglDestroySurface(gContext.mDisplay, gContext.mSurface);
        gContext.mSurface = 0;
    }
    if (gContext.mDisplay) {
        eglTerminate(gContext.mDisplay);
        gContext.mDisplay = nullptr;
    }
}

bool createContext(void* platform_data) {
    gContext.mDisplay = nullptr;
    gContext.mSurface = nullptr;
    gContext.mContext = nullptr;
    gContext.mNumCores = 0;

#define CHECK(x) \
    x; \
    { \
        EGLint error = eglGetError(); \
        if (error != EGL_SUCCESS) { \
            LOG_ERR("EGL error: 0x%x when executing:\n   " #x, error); \
            destroyContext(); \
            return false; \
        } \
    }

    CHECK(auto display = eglGetDisplay(EGL_DEFAULT_DISPLAY));

    CHECK(eglInitialize(display, nullptr, nullptr));

    gContext.mDisplay = display;

    CHECK(eglBindAPI(EGL_OPENGL_ES_API));

    // Find a supported EGL context config.
    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    const int configAttribList[] = {
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
    };

    int one = 1;
    EGLConfig eglConfig;

    CHECK(eglChooseConfig(display, configAttribList, &eglConfig, 1, &one));

    // Create an EGL context.
    const int contextAttribList[] = {
        EGL_CONTEXT_CLIENT_VERSION, 2,
        EGL_NONE
    };

    CHECK(gContext.mContext = eglCreateContext(display, eglConfig, EGL_NO_CONTEXT, contextAttribList));

    const int surfaceAttribList[] = {
        EGL_WIDTH, 16,
        EGL_HEIGHT, 16,
        EGL_NONE
    };

    CHECK(gContext.mSurface = eglCreatePbufferSurface(display, eglConfig, surfaceAttribList));

    CHECK(eglMakeCurrent(display, gContext.mSurface, gContext.mSurface, gContext.mContext));

#undef CHECK

#define CHECK(x) \
    if (!x) { \
        LOG_ERR("JNI error:\n   " #x); \
        destroyContext(); \
        return false; \
    }

    auto data = reinterpret_cast<AndroidPlatformData*>(platform_data);
    Class build(data->env, "android/os/Build");
    CHECK(build.get_field("SUPPORTED_ABIS", gContext.mSupportedABIs));
    CHECK(build.get_field("HOST", gContext.mHost));
    CHECK(build.get_field("SERIAL", gContext.mSerial));
    CHECK(build.get_field("HARDWARE", gContext.mHardware));
    CHECK(build.get_field("DISPLAY", gContext.mOSBuild));

    Class version(data->env, "android/os/Build$VERSION");
    CHECK(version.get_field("RELEASE", gContext.mOSName));
    CHECK(version.get_field("SDK_INT", gContext.mOSVersion));

#undef CHECK

    if (gContext.mSupportedABIs.size() > 0) {
        auto primaryABI = gContext.mSupportedABIs[0];
        if (primaryABI == "armeabi" || primaryABI == "armeabi-v7a") {
            gContext.mCpuArchitecture = device::ARMv7a;
        } else if (primaryABI == "arm64-v8a") {
            gContext.mCpuArchitecture = device::ARMv8a;
        } else {
            LOG_WARN("Unrecognised ABI: %s", primaryABI.c_str());
        }
    }

    switch (gContext.mOSVersion) {
        case 25:  // Nougat
            gContext.mOSVersionMajor = 7;
            gContext.mOSVersionMinor = 1;
            break;
        case 24:  // Nougat
            gContext.mOSVersionMajor = 7;
            gContext.mOSVersionMinor = 0;
            break;
        case 23:  // Marshmallow
            gContext.mOSVersionMajor = 6;
            gContext.mOSVersionMinor = 0;
            break;
        case 22:  // Lollipop
            gContext.mOSVersionMajor = 5;
            gContext.mOSVersionMinor = 1;
            break;
        case 21:  // Lollipop
            gContext.mOSVersionMajor = 5;
            gContext.mOSVersionMinor = 0;
            break;
        case 19:  // KitKat
            gContext.mOSVersionMajor = 4;
            gContext.mOSVersionMinor = 4;
            break;
        case 18:  // Jelly Bean
            gContext.mOSVersionMajor = 4;
            gContext.mOSVersionMinor = 3;
            break;
        case 17:  // Jelly Bean
            gContext.mOSVersionMajor = 4;
            gContext.mOSVersionMinor = 2;
            break;
        case 16:  // Jelly Bean
            gContext.mOSVersionMajor = 4;
            gContext.mOSVersionMinor = 1;
            break;
        case 15:  // Ice Cream Sandwich
        case 14:  // Ice Cream Sandwich
            gContext.mOSVersionMajor = 4;
            gContext.mOSVersionMinor = 0;
            break;
        case 13:  // Honeycomb
            gContext.mOSVersionMajor = 3;
            gContext.mOSVersionMinor = 2;
            break;
        case 12:  // Honeycomb
            gContext.mOSVersionMajor = 3;
            gContext.mOSVersionMinor = 1;
            break;
        case 11:  // Honeycomb
            gContext.mOSVersionMajor = 3;
            gContext.mOSVersionMinor = 0;
            break;
        case 10:  // Gingerbread
        case 9:   // Gingerbread
            gContext.mOSVersionMajor = 2;
            gContext.mOSVersionMinor = 3;
            break;
        case 8:   // Froyo
            gContext.mOSVersionMajor = 2;
            gContext.mOSVersionMinor = 2;
            break;
        case 7:   // Eclair
            gContext.mOSVersionMajor = 2;
            gContext.mOSVersionMinor = 1;
            break;
        case 6:   // Eclair
        case 5:   // Eclair
            gContext.mOSVersionMajor = 2;
            gContext.mOSVersionMinor = 0;
            break;
        case 4:   // Donut
            gContext.mOSVersionMajor = 1;
            gContext.mOSVersionMinor = 6;
            break;
        case 3:   // Cupcake
            gContext.mOSVersionMajor = 1;
            gContext.mOSVersionMinor = 5;
            break;
        case 2:   // (no code name)
            gContext.mOSVersionMajor = 1;
            gContext.mOSVersionMinor = 1;
            break;
        case 1:   // (no code name)
            gContext.mOSVersionMajor = 1;
            gContext.mOSVersionMinor = 0;
            break;
    }

    return true;
}

int numABIs() { return gContext.mSupportedABIs.size(); }

void abi(int idx, device::ABI* abi) {
    auto name = gContext.mSupportedABIs[idx];
    abi->set_name(name);
    abi->set_os(device::Android);

    if (name == "armeabi" || name == "armeabi-v7a") {
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_pointeralignment(4);
        memory_layout->set_pointersize(4);
        memory_layout->set_integersize(4);
        memory_layout->set_sizesize(4);
        memory_layout->set_u64alignment(8);
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::ARMv7a);
    } else if (name == "arm64-v8a") {
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_pointeralignment(8);
        memory_layout->set_pointersize(8);
        memory_layout->set_integersize(8);
        memory_layout->set_sizesize(8);
        memory_layout->set_u64alignment(8);
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::ARMv8a);
    } else {
        LOG_WARN("Unrecognised ABI: %s", name.c_str());
    }
}

int cpuNumCores() { return gContext.mNumCores; }

const char* cpuName() { return "<unknown>"; }

const char* cpuVendor() { return "<unknown>"; }

device::Architecture cpuArchitecture() { return gContext.mCpuArchitecture; }

const char* gpuName() { return "<unknown>"; }

const char* gpuVendor() { return "<unknown>"; }

const char* instanceName()  { return gContext.mSerial.c_str(); }

const char* instanceSerial()  { return gContext.mSerial.c_str(); }

const char* hardwareName() { return gContext.mHardware.c_str(); }

device::OSKind osKind() { return device::Android; }

const char* osName() { return gContext.mOSName.c_str(); }

const char* osBuild() { return gContext.mOSBuild.c_str(); }

int osMajor() { return gContext.mOSVersionMajor; }

int osMinor() { return gContext.mOSVersionMinor; }

int osPoint() { return 0; }

}  // namespace query
