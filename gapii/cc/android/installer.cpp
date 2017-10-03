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

#include "installer.h"
#include "../gles_exports.h"

#include "core/cc/assert.h"
#include "core/cc/get_gles_proc_address.h"
#include "core/cc/log.h"

#include <dlfcn.h>

#include <unordered_map>

#if defined(__LP64__)
#define SYSTEM_LIB_PATH "/system/lib64/"
#else
#define SYSTEM_LIB_PATH "/system/lib/"
#endif

#define NELEM(x) (sizeof(x) / sizeof(x[0]))


extern "C" {
// For this to function on Android the entry-point names for GetDeviceProcAddr
// and GetInstanceProcAddr must be ${layer_name}/Get*ProcAddr.
// This is a bit surprising given that we *MUST* also export
// vkEnumerate*Layers without any prefix.

extern void gapid_vkGetDeviceProcAddr();
extern void gapid_vkGetInstanceProcAddr();
extern void gapid_vkEnumerateInstanceLayerProperties();
extern void gapid_vkEnumerateInstanceExtensionProperties();
extern void gapid_vkEnumerateDeviceLayerProperties();
extern void gapid_vkEnumerateDeviceExtensionProperties();
}


namespace {

//
// Run the installer automatically when the library is loaded.
//
// This is done so the only modification needed to a Java app is a call to
// load library in the main activity:
//
//   static {
//     System.loadLibrary("libgapii.so");
//   }
//
// As this means that the code runs before main, care needs to be taken to
// avoid using any other load time initialized globals, since they may not
// have been initialized yet.
//

typedef void* (InitializeInterceptorFunc)();
typedef void (TerminateInterceptorFunc)(void *interceptor);
typedef bool (InterceptFunctionFunc)(void* interceptor,
                                     void* old_function,
                                     const void* new_function,
                                     void** callback_function,
                                     void (*error_callback)(void*, const char*),
                                     void* error_callback_baton);

InitializeInterceptorFunc*             gInitializeInterceptor = nullptr;
TerminateInterceptorFunc*              gTerminateInterceptor = nullptr;
void*                                  gInterceptor = nullptr;
InterceptFunctionFunc*                 gInterceptFunction = nullptr;
std::unordered_map<std::string, void*> gCallbacks;
const char*                            gDriverPaths[] = {
  SYSTEM_LIB_PATH "libhwgl.so", // Huawei specific, must be first.
  SYSTEM_LIB_PATH "libGLES.so",
  SYSTEM_LIB_PATH "libEGL.so",
  SYSTEM_LIB_PATH "libGLESv1_CM.so",
  SYSTEM_LIB_PATH "libGLESv2.so",
  SYSTEM_LIB_PATH "libGLESv3.so",
};

void* resolveCallback(const char* name, bool bypassLocal) {
    if (void* ptr = gCallbacks[name]) {
        return ptr;
    }
    if(strcmp(name, "gapid_vkGetDeviceProcAddr") == 0) {
        return reinterpret_cast<void*>(&gapid_vkGetDeviceProcAddr);
    }
    else if(strcmp(name, "gapid_vkGetInstanceProcAddr") == 0) {
        return reinterpret_cast<void*>(&gapid_vkGetInstanceProcAddr);
    }
    else if(strcmp(name, "gapid_vkEnumerateInstanceLayerProperties") == 0) {
        return reinterpret_cast<void*>(&gapid_vkEnumerateInstanceLayerProperties);
    }
    else if(strcmp(name, "gapid_vkEnumerateInstanceExtensionProperties") == 0) {
        return reinterpret_cast<void*>(&gapid_vkEnumerateInstanceExtensionProperties);
    }
    else if(strcmp(name, "gapid_vkEnumerateDeviceLayerProperties") == 0) {
        return reinterpret_cast<void*>(&gapid_vkEnumerateDeviceLayerProperties);
    }
    else if(strcmp(name, "gapid_vkEnumerateDeviceExtensionProperties") == 0) {
        return reinterpret_cast<void*>(&gapid_vkEnumerateDeviceExtensionProperties);
    }
    GAPID_WARNING("%s was requested, but cannot be traced.", name);
    return nullptr;
}

void recordInterceptorError(void*, const char* message) {
    GAPID_WARNING("Interceptor error: %s", message);
}

}  // anonymous namespace

namespace gapii {

Installer::Installer(const char* libInterceptorPath) {
    GAPID_INFO("Installing GAPII hooks...")

    auto lib = dlopen(libInterceptorPath, RTLD_NOW);
    if (lib == nullptr) {
        GAPID_FATAL("Couldn't load interceptor library from: %s", libInterceptorPath);
    }

    gInitializeInterceptor = reinterpret_cast<InitializeInterceptorFunc*>(dlsym(lib, "InitializeInterceptor"));
    gTerminateInterceptor = reinterpret_cast<TerminateInterceptorFunc*>(dlsym(lib, "TerminateInterceptor"));
    gInterceptFunction = reinterpret_cast<InterceptFunctionFunc*>(dlsym(lib, "InterceptFunction"));

    if (gInitializeInterceptor == nullptr ||
        gTerminateInterceptor == nullptr ||
        gInterceptFunction == nullptr) {
        GAPID_FATAL("Couldn't resolve the interceptor methods. "
                "Did you forget to load libinterceptor.so before libgapii.so?\n"
                "gInitializeInterceptor = %p\n"
                "gTerminateInterceptor  = %p\n"
                "gInterceptFunction     = %p\n",
                gInitializeInterceptor, gTerminateInterceptor, gInterceptFunction);
    }

    GAPID_INFO("Interceptor functions resolved")

    GAPID_INFO("Calling gInitializeInterceptor at %p...", gInitializeInterceptor);
    gInterceptor = gInitializeInterceptor();
    GAPID_ASSERT(gInterceptor != nullptr);

    // Patch the driver to trampoline to the spy for all OpenGL ES functions.
    GAPID_INFO("Installing OpenGL ES hooks...");
    install_gles();

    // Switch to using the callbacks instead of the patched driver functions.
    core::GetGlesProcAddress = resolveCallback;

    GAPID_INFO("OpenGL ES hooks successfully installed");
}

Installer::~Installer() {
    gTerminateInterceptor(gInterceptor);
}

void* Installer::install(void* func_import, const void* func_export) {
    void* callback = nullptr;
    if (!gInterceptFunction(gInterceptor,
                            func_import,
                            func_export,
                            &callback,
                            &recordInterceptorError,
                            nullptr)) {
        return nullptr;
    }
    return callback;
}

void Installer::install_gles() {
    // Start by loading all the drivers.
    void* drivers[NELEM(gDriverPaths)];
    for (int i = 0; i < NELEM(gDriverPaths); ++i) {
        drivers[i] = dlopen(gDriverPaths[i], RTLD_NOW | RTLD_LOCAL);
    }

    struct func {
        const char* name;
        void* func_export;
    };

    // Now resolve all the imported functions. We do this early so that
    // the function resolver doesn't end up using patched functions.
    std::unordered_map<void*, func> functions;
    for (int i = 0; gapii::kGLESExports[i].mName != nullptr; ++i) {
        const char* name = gapii::kGLESExports[i].mName;
        void* func_export = gapii::kGLESExports[i].mFunc;
        bool import_found = false;
        if (drivers[0] != nullptr) { // libhwgl.so
            // Huawei implements all functions in this library with prefix,
            // all GL functions in libGLES*.so are just trampolines to his.
            // However, we do not support trampoline interception for now,
            // so try to intercept the internal implementation instead.
            std::string hwName = "hw_" + std::string(name);
            if (void* func_import = dlsym(drivers[0], hwName.c_str())) {
                import_found = true;
                functions[func_import] = func{name, func_export};
                continue; // Do not do any other lookups.
            }
        }
        for (int i = 1; i < NELEM(gDriverPaths); ++i) {
            if (void* func_import = dlsym(drivers[i], name)) {
                import_found = true;
                functions[func_import] = func{name, func_export};
            }
        }
        if (void* func_import = core::GetGlesProcAddress(name, true)) {
            import_found = true;
            functions[func_import] = func{name, func_export};
        }
        if (!import_found) {
            // Don't export this function if the driver didn't export it.
            gapii::kGLESExports[i].mFunc = nullptr;
        }
    }

    // Now patch each of the functions.
    for (auto it : functions) {
        void* func_import = it.first;
        void* func_export = it.second.func_export;
        const char* name = it.second.name;
        GAPID_DEBUG("Patching '%s' at %p with %p...", name, func_import, func_export);
        if (auto callback = install(func_import, func_export)) {
            GAPID_DEBUG("Replaced function %s at %p with %p (callback %p)",
                    name, func_import, func_export, callback);
            gCallbacks[name] = callback;
        } else {
            GAPID_ERROR("Couldn't intercept function %s at %p", name, func_import);
        }
    }
}

} // namespace gapii
