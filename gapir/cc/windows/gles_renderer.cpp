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

#include "../gles_gfx_api.h"
#include "../gles_renderer.h"

#include "core/cc/gl/formats.h"
#include "core/cc/log.h"

#include <windows.h>
#include <winuser.h>
#include <string>

namespace gapir {
namespace {

const TCHAR* wndClassName = TEXT("gapir");

void registerWindowClass() {
    WNDCLASS wc;
    memset(&wc, 0, sizeof(wc));

    auto hInstance = GetModuleHandle(nullptr);
    if (hInstance == nullptr) {
        GAPID_FATAL("Failed to get module handle. Error: %d", GetLastError());
    }

    //static volatile bool waiting_for_debugger = true;
    //while (waiting_for_debugger) {}

    wc.style         = 0;
    wc.lpfnWndProc   = DefWindowProc;
    wc.hInstance     = hInstance;
    wc.hCursor       = LoadCursor(0, IDC_ARROW); // TODO: Needed?
    wc.hbrBackground = HBRUSH(COLOR_WINDOW + 1);
    wc.lpszMenuName  = TEXT("");
    wc.lpszClassName = wndClassName;

    GAPID_WARNING("style=%d", wc.style);
    GAPID_WARNING("lpfnWndProc=0x%llx", (uintptr_t)wc.lpfnWndProc);
    GAPID_WARNING("hInstance=0x%llx", (uintptr_t)wc.hInstance);
    GAPID_WARNING("hCursor=%d", wc.hCursor);
    GAPID_WARNING("hbrBackground=%d", wc.hbrBackground);
    GAPID_WARNING("lpszMenuName=%s", wc.lpszMenuName);
    GAPID_WARNING("lpszClassName=%s", wc.lpszClassName);

    if (RegisterClass(&wc) == 0) {
        GAPID_FATAL("Failed to register window class. Error: %d", GetLastError());
    }
}

class GlesRendererImpl : public GlesRenderer {
public:
    GlesRendererImpl();
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

    Gles mApi;
    Backbuffer mBackbuffer;

    bool mBound;
    bool mNeedsResolve;
    bool mQueriedExtensions;
    std::string mExtensions;
    HGLRC mRenderingContext;
    HDC mDeviceContext;
    HWND mWindow;
};

GlesRendererImpl::GlesRendererImpl()
        : mBound(false)
        , mNeedsResolve(true)
        , mQueriedExtensions(false)
        , mRenderingContext(nullptr)
        , mDeviceContext(0)
        , mWindow(0) {

    // Initialize with a default target.
    setBackbuffer(Backbuffer(
          8, 8,
          core::gl::GL_RGBA8,
          core::gl::GL_DEPTH24_STENCIL8,
          core::gl::GL_DEPTH24_STENCIL8));
}

GlesRendererImpl::~GlesRendererImpl() {
    reset();
}

Api* GlesRendererImpl::api() {
  return &mApi;
}

void GlesRendererImpl::reset() {
    unbind();

    if (mRenderingContext != nullptr) {
        if (!wglDeleteContext(mRenderingContext)) {
            GAPID_FATAL("Failed to delete GL context. Error: %d", GetLastError());
        }
        mRenderingContext = nullptr;
    }

    if (mDeviceContext != nullptr) {
        // TODO: Does this need to be released?
        mDeviceContext = nullptr;
    }

    if (mWindow != nullptr) {
        if (!DestroyWindow(mWindow)) {
            GAPID_FATAL("Failed to destroy window. Error: %d", GetLastError());
        }
        mWindow = nullptr;
    }

    mBackbuffer = Backbuffer();
}

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
    if (mBackbuffer == backbuffer) {
        return; // No change
    }

    if (mBackbuffer.format == backbuffer.format) {
        // Only a resize is necessary
        GAPID_INFO("Resizing renderer: %dx%d -> %dx%d",
                mBackbuffer.width, mBackbuffer.height, backbuffer.width, backbuffer.height);
        SetWindowPos(mWindow, nullptr, 0, 0, backbuffer.width, backbuffer.height, SWP_NOMOVE);
        mBackbuffer = backbuffer;
        return;
    }

    const bool wasBound = mBound;

    reset();

    static bool inited = false;
    if (!inited) {
        inited = true;
        registerWindowClass(); // Only needs to be done once per app life-time.
    }

    mWindow = CreateWindow(wndClassName, TEXT(""), WS_POPUP, 0, 0,
            backbuffer.width, backbuffer.height, 0, 0, GetModuleHandle(0), 0);
    if (mWindow == 0) {
        GAPID_FATAL("Failed to create window. Error: %d", GetLastError());
    }


    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    PIXELFORMATDESCRIPTOR pfd;
    memset(&pfd, 0, sizeof(pfd));

    pfd.nSize = sizeof(PIXELFORMATDESCRIPTOR);
    pfd.nVersion = 1;
    pfd.dwFlags = PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL;
    pfd.iPixelType = PFD_TYPE_RGBA;
    pfd.cRedBits = r;
    pfd.cGreenBits = g;
    pfd.cBlueBits = b;
    pfd.cAlphaBits = a;
    pfd.cDepthBits = d;
    pfd.cStencilBits = s;
    pfd.cColorBits = r+g+b+a;
    pfd.iLayerType = PFD_MAIN_PLANE;

    mDeviceContext = GetDC(mWindow);

    int pixelFormat = ChoosePixelFormat(mDeviceContext, &pfd);
    SetPixelFormat(mDeviceContext, pixelFormat, &pfd);

    mRenderingContext = wglCreateContext(mDeviceContext);
    if (mRenderingContext == nullptr) {
        GAPID_FATAL("Failed to create GL context. Error: %d", GetLastError());
    }

    mBackbuffer = backbuffer;
    mNeedsResolve = true;

    if (wasBound) {
        bind();
    }
}

void GlesRendererImpl::bind() {
    if (!mBound) {
        if (!wglMakeCurrent(mDeviceContext, mRenderingContext)) {
            GAPID_FATAL("Failed to attach GL context. Error: %d", GetLastError());
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
        if (!wglMakeCurrent(mDeviceContext, nullptr)) {
            GAPID_FATAL("Failed to detach GL context. Error: %d", GetLastError());
        }

        mBound = false;
    }
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
        bool first = true;
        for (i = 0; i < n; i++) {
            const char* extension = reinterpret_cast<const char*>(
                mApi.mFunctionStubs.glGetStringi(Gles::GLenum::GL_EXTENSIONS, i));
            if (extension == nullptr) {
                GAPID_WARNING("glGetStringi(GL_EXTENSIONS, %d) return nullptr", i);
                continue;
            }
            if (!first) {
              mExtensions += " ";
            }
            mExtensions += extension;
            first = false;
        }
    }
    return (mExtensions.size() > 0) ? &mExtensions[0] : nullptr;
}

const char* GlesRendererImpl::vendor() {
    return reinterpret_cast<const char*>(
        mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_VENDOR));
}

const char* GlesRendererImpl::version() {
    return reinterpret_cast<const char*>(
        mApi.mFunctionStubs.glGetString(Gles::GLenum::GL_VERSION));
}

} // anonymous namespace

GlesRenderer* GlesRenderer::create() {
    return new GlesRendererImpl();
}

}  // namespace gapir
