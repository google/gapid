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

#include "gapir/cc/gles_gfx_api.h"
#include "gapir/cc/gles_renderer.h"

#include "core/cc/gl/formats.h"
#include "core/cc/gl/versions.h"
#include "core/cc/log.h"

#include <memory>
#include <windows.h>
#include <winuser.h>
#include <string>

namespace gapir {
namespace {

DECLARE_HANDLE(HPBUFFERARB);

typedef BOOL(*PFNWGLCHOOSEPIXELFORMATARB)(HDC hdc, const int *piAttribIList, const FLOAT *pfAttribFList, UINT nMaxFormats, int *piFormats, UINT *nNumFormats);
typedef HGLRC(*PFNWGLCREATECONTEXTATTRIBSARBPROC)(HDC hDC, HGLRC hShareContext, const int *attribList);
typedef HPBUFFERARB(*PFNWGLCREATEPBUFFERARB)(HDC hDC, int iPixelFormat, int iWidth, int iHeight, const int *piAttribList);
typedef HDC(*PFNWGLGETPBUFFERDCARB)(HPBUFFERARB hPbuffer);
typedef int(*PFNWGLRELEASEPBUFFERDCARB)(HPBUFFERARB hPbuffer, HDC hDC);
typedef BOOL(*PFNWGLDESTROYPBUFFERARB)(HPBUFFERARB hPbuffer);
typedef BOOL(*PFNWGLQUERYPBUFFERARB)(HPBUFFERARB hPbuffer, int iAttribute, int *piValue);

const int WGL_CONTEXT_RELEASE_BEHAVIOR_ARB = 0x2097;
const int WGL_CONTEXT_MAJOR_VERSION_ARB = 0x2091;
const int WGL_CONTEXT_MINOR_VERSION_ARB = 0x2092;
const int WGL_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB = 0x0000;
const int WGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB = 0x2098;

const int WGL_DRAW_TO_PBUFFER_ARB = 0x202D;
const int WGL_SUPPORT_OPENGL_ARB = 0x2010;
const int WGL_RED_BITS_ARB = 0x2015;
const int WGL_GREEN_BITS_ARB = 0x2017;
const int WGL_BLUE_BITS_ARB = 0x2019;
const int WGL_ALPHA_BITS_ARB = 0x201B;
const int WGL_DEPTH_BITS_ARB = 0x2022;
const int WGL_STENCIL_BITS_ARB = 0x2023;

const TCHAR* wndClassName = TEXT("gapir");

void registerWindowClass() {
    WNDCLASS wc;
    memset(&wc, 0, sizeof(wc));

    auto hInstance = GetModuleHandle(nullptr);
    if (hInstance == nullptr) {
        GAPID_FATAL("Failed to get module handle. Error: %d", GetLastError());
    }

    wc.style         = 0;
    wc.lpfnWndProc   = DefWindowProc;
    wc.hInstance     = hInstance;
    wc.hCursor       = LoadCursor(0, IDC_ARROW); // TODO: Needed?
    wc.hbrBackground = HBRUSH(COLOR_WINDOW + 1);
    wc.lpszMenuName  = TEXT("");
    wc.lpszClassName = wndClassName;

    if (RegisterClass(&wc) == 0) {
        GAPID_FATAL("Failed to register window class. Error: %d", GetLastError());
    }
}

class WGL {
public:
    class PBuffer {
    public:
        PBuffer(HPBUFFERARB pbuf, HGLRC ctx, HDC hdc);
        ~PBuffer();

        void bind();
        void unbind();

    private:
        friend class WGL;
        HPBUFFERARB mPBuf;
        HGLRC mCtx;
        HDC  mHDC;
    };

    WGL();

    static const WGL& get();

    std::shared_ptr<PBuffer> create_pbuffer(GlesRenderer::Backbuffer backbuffer, PBuffer* shared_ctx) const;

private:
    HWND mWindow;
    HDC mHDC;

    PFNWGLCHOOSEPIXELFORMATARB ChoosePixelFormatARB;
    PFNWGLCREATECONTEXTATTRIBSARBPROC CreateContextAttribsARB;
    PFNWGLCREATEPBUFFERARB CreatePbufferARB;
    PFNWGLGETPBUFFERDCARB GetPbufferDCARB;
    PFNWGLRELEASEPBUFFERDCARB ReleasePbufferDCARB;
    PFNWGLDESTROYPBUFFERARB DestroyPbufferARB;
    PFNWGLQUERYPBUFFERARB QueryPbufferARB;
};


WGL::PBuffer::PBuffer(HPBUFFERARB pbuf, HGLRC ctx, HDC hdc) : mPBuf(pbuf), mCtx(ctx), mHDC(hdc) {}

WGL::PBuffer::~PBuffer() {
    auto wgl = WGL::get();
    if (!wgl.ReleasePbufferDCARB(mPBuf, mHDC)) {
        GAPID_FATAL("Failed to release HDC. Error: %d", GetLastError());
    }
    if (!wgl.DestroyPbufferARB(mPBuf)) {
        GAPID_FATAL("Failed to destroy pbuffer. Error: %d", GetLastError());
    }
    if (!wglDeleteContext(mCtx)) {
        GAPID_FATAL("Failed to delete GL context. Error: %d", GetLastError());
    }
}

void WGL::PBuffer::bind() {
    if (!wglMakeCurrent(mHDC, mCtx)) {
        GAPID_FATAL("Failed to bind GL context. Error: %d", GetLastError());
    }
}

void WGL::PBuffer::unbind() {
    if (!wglMakeCurrent(mHDC, nullptr)) {
        GAPID_FATAL("Failed to unbind GL context. Error: %d", GetLastError());
    }
}

WGL::WGL() {
    registerWindowClass();

    mWindow = CreateWindow(wndClassName, TEXT(""), WS_POPUP, 0, 0, 8, 8, 0, 0, GetModuleHandle(0), 0);
    if (mWindow == 0) {
        GAPID_FATAL("Failed to create window. Error: %d", GetLastError());
    }

    mHDC = GetDC(mWindow);

    PIXELFORMATDESCRIPTOR pfd;
    memset(&pfd, 0, sizeof(pfd));
    pfd.nSize = sizeof(PIXELFORMATDESCRIPTOR);
    pfd.nVersion = 1;
    pfd.dwFlags = PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL;
    pfd.iPixelType = PFD_TYPE_RGBA;
    pfd.cRedBits = 8;
    pfd.cGreenBits = 8;
    pfd.cBlueBits = 8;
    pfd.cAlphaBits = 8;
    pfd.cDepthBits = 24;
    pfd.cStencilBits = 8;
    pfd.cColorBits = 32;
    pfd.iLayerType = PFD_MAIN_PLANE;
    int pixel_fmt = ChoosePixelFormat(mHDC, &pfd);
    SetPixelFormat(mHDC, pixel_fmt, &pfd);

    // Resolve extension functions.
    CreateContextAttribsARB = nullptr;
    auto temp_context = wglCreateContext(mHDC);
    if (temp_context == nullptr) {
        GAPID_FATAL("Couldn't create temporary WGL context. Error: %d", GetLastError());
    }

    wglMakeCurrent(mHDC, temp_context);

#define RESOLVE(name) \
    name = reinterpret_cast< decltype(name) >(wglGetProcAddress("wgl"#name)); \
    if (name == nullptr) { GAPID_FATAL("Couldn't resolve function 'wgl" #name "'"); }

    RESOLVE(CreateContextAttribsARB);
    RESOLVE(ChoosePixelFormatARB);
    RESOLVE(CreateContextAttribsARB);
    RESOLVE(CreatePbufferARB);
    RESOLVE(GetPbufferDCARB);
    RESOLVE(ReleasePbufferDCARB);
    RESOLVE(DestroyPbufferARB);
    RESOLVE(QueryPbufferARB);

#undef RESOLVE

    wglMakeCurrent(mHDC, nullptr);
    wglDeleteContext(temp_context);
}

const WGL& WGL::get() {
    static WGL instance;
    return instance;
}

std::shared_ptr<WGL::PBuffer> WGL::create_pbuffer(GlesRenderer::Backbuffer backbuffer, PBuffer* shared_ctx) const {
    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    const unsigned int MAX_FORMATS = 32;

    int formats[MAX_FORMATS];
    unsigned int num_formats;
    const int fmt_attribs[] = {
        WGL_DRAW_TO_PBUFFER_ARB, 1,
        WGL_SUPPORT_OPENGL_ARB, 1,
        WGL_DEPTH_BITS_ARB, d,
        WGL_STENCIL_BITS_ARB, s,
        WGL_RED_BITS_ARB, r,
        WGL_GREEN_BITS_ARB, g,
        WGL_BLUE_BITS_ARB, b,
        WGL_ALPHA_BITS_ARB, a,
        0, // terminator
    };
    if (!ChoosePixelFormatARB(mHDC, fmt_attribs, nullptr, MAX_FORMATS, formats, &num_formats)) {
        GAPID_FATAL("wglChoosePixelFormatARB failed. Error: %d", GetLastError());
    }
    if (num_formats == 0) {
        GAPID_FATAL("wglChoosePixelFormatARB returned no compatibile formats");
    }
    auto format = formats[0]; // TODO: Examine returned formats?
    const int create_attribs[] = { 0 };
    auto pbuffer = CreatePbufferARB(mHDC, format, backbuffer.width, backbuffer.height, create_attribs);
    if (pbuffer == nullptr) {
        GAPID_FATAL("wglCreatePbufferARB failed. Error: %d", GetLastError());
    }
    auto hdc = GetPbufferDCARB(pbuffer);
    if (hdc == nullptr) {
        GAPID_FATAL("wglGetPbufferDCARB failed. Error: %d", GetLastError());
    }
    for (auto gl_version : core::gl::sVersionSearchOrder) {
        std::vector<int> attribs;
        attribs.push_back(WGL_CONTEXT_MAJOR_VERSION_ARB);
        attribs.push_back(gl_version.major);
        attribs.push_back(WGL_CONTEXT_MINOR_VERSION_ARB);
        attribs.push_back(gl_version.minor);
        // https://www.khronos.org/registry/OpenGL/extensions/KHR/KHR_context_flush_control.txt
        // These are disabled as they don't seem to improve performance.
        // attribs.push_back(WGL_CONTEXT_RELEASE_BEHAVIOR_ARB);
        // attribs.push_back(WGL_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB);
        attribs.push_back(0);
        auto ctx = CreateContextAttribsARB(hdc, (shared_ctx != nullptr) ? shared_ctx->mCtx : nullptr, attribs.data());
        if (ctx != nullptr) {
            return std::shared_ptr<PBuffer>(new PBuffer(pbuffer, ctx, hdc));
        }
    }
    GAPID_FATAL("Failed to create GL context using wglCreateContextAttribsARB. Error: %d", GetLastError());
    return nullptr;
}


class GlesRendererImpl : public GlesRenderer {
public:
    GlesRendererImpl(GlesRendererImpl* shared_context);
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
    std::shared_ptr<WGL::PBuffer> mContext;
    std::shared_ptr<WGL::PBuffer> mSharedContext;
};

GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared_context)
        : mBound(false)
        , mNeedsResolve(true)
        , mQueriedExtensions(false)
        , mSharedContext(shared_context != nullptr ? shared_context->mContext : nullptr) {
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
    mContext = nullptr;
    mBackbuffer = Backbuffer();
}

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
    if (mBackbuffer == backbuffer) {
        return; // No change
    }

    const bool wasBound = mBound;

    reset();

    mContext = WGL::get().create_pbuffer(backbuffer, mSharedContext.get());
    mBackbuffer = backbuffer;
    mNeedsResolve = true;

    if (wasBound) {
        bind();
    }
}

void GlesRendererImpl::bind() {
    if (!mBound) {
        mContext->bind();
        mBound = true;

        if (mNeedsResolve) {
            mNeedsResolve = false;
            mApi.resolve();
        }
    }
}

void GlesRendererImpl::unbind() {
    if (mBound) {
        mContext->unbind();
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

GlesRenderer* GlesRenderer::create(GlesRenderer* shared_context) {
    return new GlesRendererImpl(reinterpret_cast<GlesRendererImpl*>(shared_context));
}

}  // namespace gapir
