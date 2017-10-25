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

#include "egl_lite.h"
#include "jni_helpers.h"

#include "../query.h"

#include "core/cc/assert.h"
#include "core/cc/log.h"
#include "core/cc/get_gles_proc_address.h"

#include <cstring>

#include <android/log.h>
#include <jni.h>

#define LOG_ERR(...) \
    __android_log_print(ANDROID_LOG_ERROR, "GAPID", __VA_ARGS__);

#define LOG_WARN(...) \
    __android_log_print(ANDROID_LOG_WARN, "GAPID", __VA_ARGS__);

typedef int GLint;
typedef unsigned int GLuint;
typedef uint8_t GLubyte;

namespace {

device::DataTypeLayout* new_dt_layout(int size, int alignment) {
    auto out = new device::DataTypeLayout();
    out->set_size(size);
    out->set_alignment(alignment);
    return out;
}

void abiByName(const std::string name, device::ABI* abi) {
    abi->set_name(name);
    abi->set_os(device::Android);

    if (name == "armeabi" || name == "armeabi-v7a") {
        // http://infocenter.arm.com/help/topic/com.arm.doc.ihi0042f/IHI0042F_aapcs.pdf
        // 4 DATA TYPES AND ALIGNMENT
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_allocated_pointer(new_dt_layout(4, 4));
        memory_layout->set_allocated_integer(new_dt_layout(4, 4));
        memory_layout->set_allocated_size(new_dt_layout(4, 4));
        memory_layout->set_allocated_char_(new_dt_layout(1, 1));
        memory_layout->set_allocated_i64(new_dt_layout(8, 8));
        memory_layout->set_allocated_i32(new_dt_layout(4, 4));
        memory_layout->set_allocated_i16(new_dt_layout(2, 2));
        memory_layout->set_allocated_i8(new_dt_layout(1, 1));
        memory_layout->set_allocated_f64(new_dt_layout(8, 8));
        memory_layout->set_allocated_f32(new_dt_layout(4, 4));
        memory_layout->set_allocated_f16(new_dt_layout(2, 2));
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::ARMv7a);
    } else if (name == "arm64-v8a") {
        // http://infocenter.arm.com/help/topic/com.arm.doc.ihi0055b/IHI0055B_aapcs64.pdf
        // 4 DATA TYPES AND ALIGNMENT
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_allocated_pointer(new_dt_layout(8, 8));
        memory_layout->set_allocated_integer(new_dt_layout(8, 8));
        memory_layout->set_allocated_size(new_dt_layout(8, 8));
        memory_layout->set_allocated_char_(new_dt_layout(1, 1));
        memory_layout->set_allocated_i64(new_dt_layout(8, 8));
        memory_layout->set_allocated_i32(new_dt_layout(4, 4));
        memory_layout->set_allocated_i16(new_dt_layout(2, 2));
        memory_layout->set_allocated_i8(new_dt_layout(1, 1));
        memory_layout->set_allocated_f64(new_dt_layout(8, 8));
        memory_layout->set_allocated_f32(new_dt_layout(4, 4));
        memory_layout->set_allocated_f16(new_dt_layout(2, 2));
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::ARMv8a);
    } else if (name == "x86") {
        // https://en.wikipedia.org/wiki/Data_structure_alignment#Typical_alignment_of_C_structs_on_x86
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_allocated_pointer(new_dt_layout(4, 4));
        memory_layout->set_allocated_integer(new_dt_layout(4, 4));
        memory_layout->set_allocated_size(new_dt_layout(4, 4));
        memory_layout->set_allocated_char_(new_dt_layout(1, 1));
        memory_layout->set_allocated_i64(new_dt_layout(8, 4));
        memory_layout->set_allocated_i32(new_dt_layout(4, 4));
        memory_layout->set_allocated_i16(new_dt_layout(2, 2));
        memory_layout->set_allocated_i8(new_dt_layout(1, 1));
        memory_layout->set_allocated_f64(new_dt_layout(8, 4));
        memory_layout->set_allocated_f32(new_dt_layout(4, 4));
        memory_layout->set_allocated_f16(new_dt_layout(2, 2));
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::X86);
    } else if (name == "x86_64") {
        auto memory_layout = new device::MemoryLayout();
        memory_layout->set_allocated_pointer(new_dt_layout(8, 8));
        memory_layout->set_allocated_integer(new_dt_layout(4, 4));
        memory_layout->set_allocated_size(new_dt_layout(8, 8));
        memory_layout->set_allocated_char_(new_dt_layout(1, 1));
        memory_layout->set_allocated_i64(new_dt_layout(8, 4));
        memory_layout->set_allocated_i32(new_dt_layout(4, 4));
        memory_layout->set_allocated_i16(new_dt_layout(2, 2));
        memory_layout->set_allocated_i8(new_dt_layout(1, 1));
        memory_layout->set_allocated_f64(new_dt_layout(8, 4));
        memory_layout->set_allocated_f32(new_dt_layout(4, 4));
        memory_layout->set_allocated_f16(new_dt_layout(2, 2));
        memory_layout->set_endian(device::LittleEndian);
        abi->set_allocated_memorylayout(memory_layout);
        abi->set_architecture(device::X86_64);
    } else {
        LOG_WARN("Unrecognised ABI: %s", name.c_str());
    }
}

}  // anonymous namespace

namespace query {

struct Context {
    char mError[512];
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
static int gContextRefCount = 0;

void destroyContext() {
    if (--gContextRefCount > 0) {
        return;
    }

    auto eglDestroyContext = reinterpret_cast<PFNEGLDESTROYCONTEXT>(core::GetGlesProcAddress("eglDestroyContext", true));
    auto eglDestroySurface = reinterpret_cast<PFNEGLDESTROYSURFACE>(core::GetGlesProcAddress("eglDestroySurface", true));
    auto eglTerminate = reinterpret_cast<PFNEGLTERMINATE>(core::GetGlesProcAddress("eglTerminate", true));

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
    if (gContextRefCount++ > 0) {
        return true;
    }

    gContext.mDisplay = nullptr;
    gContext.mSurface = nullptr;
    gContext.mContext = nullptr;
    gContext.mNumCores = 0;

    if (platform_data == nullptr) {
        snprintf(gContext.mError, sizeof(gContext.mError),
                "platform_data was nullptr");
        return false;
    }

#define RESOLVE(name, pfun) \
    auto name = reinterpret_cast<pfun>(core::GetGlesProcAddress(#name, true)); \
    GAPID_ASSERT(name != nullptr)

    RESOLVE(eglGetError,             PFNEGLGETERROR);
    RESOLVE(eglInitialize,           PFNEGLINITIALIZE);
    RESOLVE(eglBindAPI,              PFNEGLBINDAPI);
    RESOLVE(eglChooseConfig,         PFNEGLCHOOSECONFIG);
    RESOLVE(eglCreateContext,        PFNEGLCREATECONTEXT);
    RESOLVE(eglCreatePbufferSurface, PFNEGLCREATEPBUFFERSURFACE);
    RESOLVE(eglMakeCurrent,          PFNEGLMAKECURRENT);
    RESOLVE(eglGetDisplay,           PFNEGLGETDISPLAY);

#undef RESOLVE

#define CHECK(x) \
    x; \
    { \
        EGLint error = eglGetError(); \
        if (error != EGL_SUCCESS) { \
            snprintf(gContext.mError, sizeof(gContext.mError), \
                     "EGL error: 0x%x when executing:\n   " #x, error); \
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
        snprintf(gContext.mError, sizeof(gContext.mError), \
                 "JNI error:\n   " #x); \
        destroyContext(); \
        return false; \
    }

    JavaVM* vm = reinterpret_cast<JavaVM*>(platform_data);
    JNIEnv* env = nullptr;
    auto res = vm->GetEnv(reinterpret_cast<void**>(&env), JNI_VERSION_1_6);
    bool shouldDetach = false;
    switch (res) {
    case JNI_OK:
        break;
    case JNI_EDETACHED:
        res = vm->AttachCurrentThread(&env, nullptr);
        if (res != 0) {
            snprintf(gContext.mError, sizeof(gContext.mError),
                    "Failed to attach thread to JavaVM. (%d)", res);
            destroyContext();
            return false;
        }
        shouldDetach = true;
        break;
    default:
        snprintf(gContext.mError, sizeof(gContext.mError),
                "Failed to get Java env. (%d)", res);
        destroyContext();
        return false;
    }

    Class build(env, "android/os/Build");
    CHECK(build.get_field("SUPPORTED_ABIS", gContext.mSupportedABIs));
    CHECK(build.get_field("HOST", gContext.mHost));
    CHECK(build.get_field("SERIAL", gContext.mSerial));
    CHECK(build.get_field("HARDWARE", gContext.mHardware));
    CHECK(build.get_field("DISPLAY", gContext.mOSBuild));

    Class version(env, "android/os/Build$VERSION");
    CHECK(version.get_field("RELEASE", gContext.mOSName));
    CHECK(version.get_field("SDK_INT", gContext.mOSVersion));

    if (shouldDetach) {
        vm->DetachCurrentThread();
    }

#undef CHECK

    if (gContext.mSupportedABIs.size() > 0) {
        auto primaryABI = gContext.mSupportedABIs[0];
        if (primaryABI == "armeabi" || primaryABI == "armeabi-v7a") {
            gContext.mCpuArchitecture = device::ARMv7a;
        } else if (primaryABI == "arm64-v8a") {
            gContext.mCpuArchitecture = device::ARMv8a;
        } else if (primaryABI == "x86") {
            gContext.mCpuArchitecture = device::X86;
        } else if (primaryABI == "x86_64") {
            gContext.mCpuArchitecture = device::X86_64;
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

const char* contextError() {
    return gContext.mError;
}

int numABIs() { return gContext.mSupportedABIs.size(); }

device::ABI* currentABI() {
    device::ABI* out = new device::ABI();
#if defined(__arm__)
    abiByName("armeabi", out);
#elif defined(__aarch64__)
    abiByName("arm64-v8a", out);
#elif defined(__i686__)
    abiByName("x86", out);
#elif defined(__x86_64__)
    abiByName("x86_64", out);
#else
#   error "Unknown ABI"
#endif
    return out;
}

void abi(int idx, device::ABI* abi) {
    return abiByName(gContext.mSupportedABIs[idx], abi);
}

int cpuNumCores() { return gContext.mNumCores; }

const char* cpuName() { return ""; }

const char* cpuVendor() { return ""; }

device::Architecture cpuArchitecture() { return gContext.mCpuArchitecture; }

const char* gpuName() { return ""; }

const char* gpuVendor() { return ""; }

const char* instanceName()  { return gContext.mSerial.c_str(); }

const char* hardwareName() { return gContext.mHardware.c_str(); }

device::OSKind osKind() { return device::Android; }

const char* osName() { return gContext.mOSName.c_str(); }

const char* osBuild() { return gContext.mOSBuild.c_str(); }

int osMajor() { return gContext.mOSVersionMajor; }

int osMinor() { return gContext.mOSVersionMinor; }

int osPoint() { return 0; }

}  // namespace query
