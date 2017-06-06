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

#ifndef _MSC_VER // Doesn't compile under MSVC. TODO

#include "deqp.h"

#include <cstdlib>
#include <functional>
#include <memory>
#include <iostream>

#include "gapii/cc/abort_exception.h"
#include "gapii/cc/call_observer.h"
#include "gapii/cc/gles_types.h"

namespace null_driver {

void doWrapGetIntegerv(uint32_t param, gapii::GLint* values);

}  // namespace null_driver

namespace gapii {

void GlesNull::onThreadSwitched(CallObserver* observer, uint64_t threadID) {
    CoreSpy::switchThread(observer, threadID);
}

GlesNull::GlesNull() :
        mImportedGetIntegerv(nullptr),
        mReturnHandler(new ReturnHandler()) {
    CallObserver observer(this);

    CoreSpy::init();
    GlesSpy::init();
    SpyBase::init(&observer, std::shared_ptr<core::Encoder>(new core::NullEncoder()));

    setObserveApplicationPool(false);
    SpyBase::setHandler(abortHandler);

    GlesSpy::setReturnHandler(mReturnHandler);
    CoreSpy::architecture(&observer, alignof(void*), sizeof(void*), sizeof(int), true);
}

std::shared_ptr<ReturnHandler> GlesNull::getReturnHandler() {
    return mReturnHandler;
}

void GlesNull::wrapGetIntegerv(uint32_t param, GLint* values) {
    // Use the imported version of the function.
    if (mImportedGetIntegerv == nullptr) {
        GAPID_WARNING("GlesNull::wrapGetIntegerv called before GlesNull::Import");
        std::abort();
    }

    gapii::CallObserver observer(this);
    // Wrap the imported version of glGetIntegerv to give the GAPII values for
    // certain constants, because of the dEQP null driver does not give them
    // non-zero values.
    switch(param) {
        case gapii::GLenum::GL_NUM_COMPRESSED_TEXTURE_FORMATS: // fallthrough
        case gapii::GLenum::GL_NUM_PROGRAM_BINARY_FORMATS: // fallthrough
        case gapii::GLenum::GL_NUM_SHADER_BINARY_FORMATS:
            GlesSpy::mImports.glGetIntegerv = mImportedGetIntegerv;
            GlesSpy::glGetIntegerv(&observer, param, values);
            GlesSpy::mImports.glGetIntegerv = null_driver::doWrapGetIntegerv;
            return;
        default:
            mImportedGetIntegerv(param, values);
    }
}

uint32_t GlesNull::glGetError(CallObserver* observer) {
    auto call = [] {};
    std::shared_ptr<Context> ctx = findOrZero(this->Contexts, this->CurrentThread);
    if (ctx == nullptr) {
        return GLenum::GL_INVALID_OPERATION;
    }
    return observer->getError();
}

}  // namespace gapii

namespace {

// Provide a singleton instance of the null driver.

std::unique_ptr<gapii::GlesNull> gNull;    // Must be accessed via null() below, gets destroyed with the library

gapii::GlesNull* null() {
    if (!gNull) {
        GAPID_INFO("Constructing GLES null driver...");
        gNull.reset(new gapii::GlesNull());
    }
    return gNull.get();
}

}  // namespace <anonymous>

namespace null_driver {

class Gate {
  public:
    Gate(gapii::GlesNull* n) : mNull(n) {
        mEntered = n->try_to_enter();
        if (mEntered) {
            gapii::CallObserver observer(n);
            n->lock(&observer, "<deqp>");
        }
    }
    ~Gate() { if (mEntered) { mNull->unlock(); mNull->exit(); } }

    bool Entered() { return mEntered; }
  private:
    gapii::GlesNull *mNull;
    bool mEntered;
};

// Build a function pointer by calling the member function on the singleton.
template <typename MemberFun, MemberFun Fun> class fun_type;

template <typename C, typename... Args, void (C::*Fun)(Args...)>
class fun_type<void (C::*)(Args...), Fun> {
  public:
    static void fun(Args... args) {
        auto n = null();
        Gate gate(n);
        if (gate.Entered()) {
            (n->*Fun)(args...);
        } else {
            GAPID_WARNING("Null interceptor re-entered");
            std::abort();
        }
    }
};

template <typename Ret, typename C, typename... Args, Ret (C::*Fun)(Args...)>
class fun_type<Ret (C::*)(Args...), Fun> {
  public:
    static Ret fun(Args... args) {
        auto n = null();
        Gate gate(n);
        if (gate.Entered()) {
            auto returnHandler = n->getReturnHandler();
            auto ret = (n->*Fun)(args...);
            if (returnHandler->hasReturnValue<Ret>()) {
                return returnHandler->getAndClearReturnValue<Ret>();
            } else {
                return ret;
            }
        } else {
            GAPID_WARNING("Null interceptor re-entered");
            std::abort();
            return Ret();
        }
    }
};

// glw::Functions has both GLES and GL function APIs.
// We use FUN for functions which we are providing (GLES) and NO_FUN for
// functions we are not providing (if called the program will std::abort).

// Setup the macros to export the GlesNull API:
#define FUN(f) *gl = reinterpret_cast<void*>(fun_type<decltype(&gapii::GlesNull::f), &gapii::GlesNull::f>::fun); gl++;

#define NO_FUN(f) *gl = reinterpret_cast<void*>(&std::abort); gl++;

void Export(void** gl) {
#include "deqp.inl"
}

// Setup the macros to import the functions from dEQP:
#undef FUN
#undef NO_FUN

#define FUN(f) (GlesSpy::mImports.f) = reinterpret_cast<decltype(GlesSpy::mImports.f)>(*gl); gl++;
#define NO_FUN(f) gl++;

void Import(void** gl) {
    null()->Import(gl);
}

void doWrapGetIntegerv(uint32_t param, gapii::GLint* values) {
    null()->wrapGetIntegerv(param, values);
}

}  // namespace null_driver

namespace gapii {

void GlesNull::Import(void** gl) {
#include "deqp.inl"
    // wrap the imported version of glGetIntegerv
    mImportedGetIntegerv = GlesSpy::mImports.glGetIntegerv;
    GlesSpy::mImports.glGetIntegerv = null_driver::doWrapGetIntegerv;
}

}  // namespace gapii

// Forward declaration so dEQP caller does not have to do any casting.
namespace glw {
class Functions;
}

// Make the symbol visible in the shared object.
__attribute__ ((visibility ("default")))
extern void InstallGapiiInterceptor(glw::Functions* gl) {
    void** handle = reinterpret_cast<void**>(gl);
    // Import the DeQP null driver to form a trivial basis for Gapi null driver
    null_driver::Import(handle);
    // Export the Gapi null driver back to DeQP.
    null_driver::Export(handle);

    // Match the values in deqp: gluRenderConfig
    const int width = 256;
    const int height = 256;
    auto staticState = std::shared_ptr<gapii::StaticContextState>(new gapii::StaticContextState(
        gapii::Constants()
    ));

    auto dynamicState = std::shared_ptr<gapii::DynamicContextState>(new gapii::DynamicContextState(
        width, height,
        gapii::GLenum::GL_RGBA8,
        gapii::GLenum::GL_DEPTH24_STENCIL8,
        gapii::GLenum::GL_DEPTH24_STENCIL8,
        true,  // ResetViewportScissor
        false, // PreserveBuffersOnSwap
        8, 8, 8, 8, 24, 8
    ));

    // The follow simulates: eglInitialize
    auto n = null();
    if (!n->try_to_enter()) {
      std::abort();
    }
    gapii::CallObserver observer(n);
    n->lock(&observer, "<deqp>");
    auto call = [] {};
    std::shared_ptr<gapii::Context> ctx = n->subCreateContext(&observer, call, nullptr);
    n->subSetContext(&observer, call, ctx);
    n->subApplyStaticContextState(&observer, call, ctx, staticState);
    n->subApplyDynamicContextState(&observer, call, ctx, dynamicState);

    n->unlock();
    n->exit();
}

#endif // _MSC_VER // MSVC
