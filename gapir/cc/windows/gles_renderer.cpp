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

#include "core/cc/thread.h"
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

class WGL {
public:
    class PBuffer {
    public:
        ~PBuffer();

        static std::shared_ptr<PBuffer> create(const GlesRenderer::Backbuffer& backbuffer, PBuffer* shared_ctx);

        void bind();
        void unbind();
        void set_backbuffer(const GlesRenderer::Backbuffer& backbuffer);

    private:
        PBuffer(const GlesRenderer::Backbuffer& backbuffer, PBuffer* shared_ctx);

        void create_buffer(const GlesRenderer::Backbuffer& backbuffer);
        void release_buffer();

        HPBUFFERARB mPBuf;
        HGLRC mCtx;
        HDC  mHDC;
    };

    WGL();

    static const WGL& get();

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

WGL::PBuffer::PBuffer(const GlesRenderer::Backbuffer& backbuffer, PBuffer* shared_ctx)
        : mPBuf(nullptr)
        , mCtx(nullptr)
        , mHDC(nullptr) {

    create_buffer(backbuffer);

    auto wgl = WGL::get();
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
        auto ctx = wgl.CreateContextAttribsARB(mHDC, (shared_ctx != nullptr) ? shared_ctx->mCtx : nullptr, attribs.data());
        if (ctx != nullptr) {
            mCtx = ctx;
            return;
        }
    }
    GAPID_FATAL("Failed to create GL context using wglCreateContextAttribsARB. Error: 0x%x", GetLastError());
}

WGL::PBuffer::~PBuffer() {
    release_buffer();
    if (!wglDeleteContext(mCtx)) {
        GAPID_ERROR("Failed to delete GL context. Error: 0x%x", GetLastError());
    }
}

void WGL::PBuffer::create_buffer(const GlesRenderer::Backbuffer& backbuffer) {
    release_buffer();

    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    // Some exotic extensions let you create contexts without a backbuffer.
    // In these cases the backbuffer is zero size - just create a small one.
    int safe_width = (backbuffer.width > 0) ? backbuffer.width : 8;
    int safe_height = (backbuffer.height > 0) ? backbuffer.height : 8;

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

    auto wgl = WGL::get();

    if (!wgl.ChoosePixelFormatARB(wgl.mHDC, fmt_attribs, nullptr, MAX_FORMATS, formats, &num_formats)) {
        GAPID_FATAL("wglChoosePixelFormatARB failed. Error: 0x%x", GetLastError());
    }
    if (num_formats == 0) {
        GAPID_FATAL("wglChoosePixelFormatARB returned no compatibile formats");
    }
    auto format = formats[0]; // TODO: Examine returned formats?
    const int create_attribs[] = { 0 };
    mPBuf = wgl.CreatePbufferARB(wgl.mHDC, format, safe_width, safe_height, create_attribs);
    if (mPBuf == nullptr) {
        GAPID_FATAL("wglCreatePbufferARB(%p, %d, %d, %d, %p) failed. Error: 0x%x",
            wgl.mHDC, format, safe_width, safe_height, create_attribs, GetLastError());
    }

    mHDC = wgl.GetPbufferDCARB(mPBuf);
    if (mHDC == nullptr) {
        GAPID_FATAL("wglGetPbufferDCARB(%p) failed. Error: 0x%x", mPBuf, GetLastError());
    }
}

void WGL::PBuffer::release_buffer() {
    auto wgl = WGL::get();
    if (mHDC != nullptr) {
        if (!wgl.ReleasePbufferDCARB(mPBuf, mHDC)) {
            GAPID_ERROR("Failed to release HDC. Error: 0x%x", GetLastError());
        }
        mHDC = nullptr;
    }
    if (mPBuf != nullptr) {
        if (!wgl.DestroyPbufferARB(mPBuf)) {
            GAPID_ERROR("Failed to destroy pbuffer. Error: 0x%x", GetLastError());
        }
        mPBuf = nullptr;
    }
}

void WGL::PBuffer::bind() {
    if (!wglMakeCurrent(mHDC, mCtx)) {
        GAPID_FATAL("Failed to bind GL context. Error: 0x%x", GetLastError());
    }
}

void WGL::PBuffer::unbind() {
    if (!wglMakeCurrent(mHDC, nullptr)) {
        GAPID_FATAL("Failed to unbind GL context. Error: 0x%x", GetLastError());
    }
}

void WGL::PBuffer::set_backbuffer(const GlesRenderer::Backbuffer& backbuffer) {
    // Kill the pbuffer, and create a new one with the new backbuffer settings.
    //
    // Note - according to the MSDN documentation of wglMakeCurrent:
    // "It need not be the same hdc that was passed to wglCreateContext when
    // hglrc was created, but it must be on the same device and have the same
    // pixel format."
    //
    // This means pixel format changes should error. If this happens, we're
    // going to have to come up with a different approach.
    create_buffer(backbuffer);
}

std::shared_ptr<WGL::PBuffer> WGL::PBuffer::create(const GlesRenderer::Backbuffer& backbuffer, PBuffer* shared_ctx) {
    return std::shared_ptr<PBuffer>(new PBuffer(backbuffer, shared_ctx));
}

WGL::WGL() {
    class registerWindowClass {
    public:
        registerWindowClass() {
            WNDCLASS wc;
            memset(&wc, 0, sizeof(wc));

            auto hInstance = GetModuleHandle(nullptr);
            if (hInstance == nullptr) {
                GAPID_FATAL("Failed to get module handle. Error: 0x%x", GetLastError());
            }

            wc.style = 0;
            wc.lpfnWndProc = DefWindowProc;
            wc.hInstance = hInstance;
            wc.hCursor = LoadCursor(0, IDC_ARROW); // TODO: Needed?
            wc.hbrBackground = HBRUSH(COLOR_WINDOW + 1);
            wc.lpszMenuName = TEXT("");
            wc.lpszClassName = wndClassName;

            if (RegisterClass(&wc) == 0) {
                GAPID_FATAL("Failed to register window class. Error: 0x%x", GetLastError());
            }
        }
    };

    static registerWindowClass rwc;

    mWindow = CreateWindow(wndClassName, TEXT(""), WS_POPUP, 0, 0, 8, 8, 0, 0, GetModuleHandle(0), 0);
    if (mWindow == 0) {
        GAPID_FATAL("Failed to create window. Error: 0x%x", GetLastError());
    }

    mHDC = GetDC(mWindow);
    if (mHDC == nullptr) {
        GAPID_FATAL("GetDC failed. Error: 0x%x", GetLastError());
    }

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
        GAPID_FATAL("Couldn't create temporary WGL context. Error: 0x%x", GetLastError());
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
    // This is thread-local as anything touching a HDC is pretty much
    // non-thread safe.
    static thread_local WGL instance;
    return instance;
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

    bool mNeedsResolve;
    bool mQueriedExtensions;
    std::string mExtensions;
    std::shared_ptr<WGL::PBuffer> mContext;
    std::shared_ptr<WGL::PBuffer> mSharedContext;

    static thread_local GlesRendererImpl* tlsBound;
};

thread_local GlesRendererImpl* GlesRendererImpl::tlsBound = nullptr;

GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared_context)
        : mNeedsResolve(true)
        , mQueriedExtensions(false)
        , mSharedContext(shared_context != nullptr ? shared_context->mContext : nullptr) {
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

static void DebugCallback(Gles::GLenum source, Gles::GLenum type, Gles::GLuint id, Gles::GLenum severity,
                           Gles::GLsizei length, const Gles::GLchar* message, const void* user_param) {
    auto renderer = reinterpret_cast<const GlesRendererImpl*>(user_param);
    auto listener = renderer->getListener();
    if (listener != nullptr) {
        if (type == Gles::GLenum::GL_DEBUG_TYPE_ERROR || severity == Gles::GLenum::GL_DEBUG_SEVERITY_HIGH) {
            listener->onDebugMessage(LOG_LEVEL_ERROR, message);
        } else {
            listener->onDebugMessage(LOG_LEVEL_DEBUG, message);
        }
    }
}

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
    auto wasBound = tlsBound == this;
    GAPID_ASSERT(wasBound /* The renderer has to be bound when changing the backbuffer */);

    if (mBackbuffer == backbuffer) {
        return; // No change
    }

    if (mContext == nullptr) {
        mContext = WGL::PBuffer::create(backbuffer, mSharedContext.get());
        mContext->bind();
        mApi.resolve();
        mNeedsResolve = false;
        if (mApi.mFunctionStubs.glDebugMessageCallback != nullptr) {
            mApi.mFunctionStubs.glDebugMessageCallback(reinterpret_cast<void*>(&DebugCallback), this);
            mApi.mFunctionStubs.glEnable(Gles::GLenum::GL_DEBUG_OUTPUT);
            mApi.mFunctionStubs.glEnable(Gles::GLenum::GL_DEBUG_OUTPUT_SYNCHRONOUS);
            GAPID_DEBUG("Enabled KHR_debug extension");
        }
    } else {
        unbind();
        mContext->set_backbuffer(backbuffer);
        mNeedsResolve = true;
        bind();
    }

    mBackbuffer = backbuffer;
}

void GlesRendererImpl::bind() {
    auto bound = tlsBound;
    if (bound == this) {
        return;
    }

    if (bound != nullptr) {
        bound->unbind();
    }

    tlsBound = this;

    if (mContext == nullptr) {
        return;
    }

    mContext->bind();

    if (mNeedsResolve) {
        mNeedsResolve = false;
        mApi.resolve();
    }
}

void GlesRendererImpl::unbind() {
    if (tlsBound == this) {
        if (mContext != nullptr) {
            mContext->unbind();
        }
        tlsBound = nullptr;
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
