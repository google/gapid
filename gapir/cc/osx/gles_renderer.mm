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
#include "core/cc/log.h"

#include <string>

#import <OpenGL/OpenGL.h>
#import <AppKit/AppKit.h>

// Some versions of AppKit include these GL defines.
#undef GL_EXTENSIONS
#undef GL_RENDERER
#undef GL_VENDOR
#undef GL_VERSION

#if MAC_OS_X_VERSION_MAX_ALLOWED < 101200
#define NSWindowStyleMaskBorderless  NSBorderlessWindowMask
#endif

namespace gapir {
namespace {

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

    Backbuffer mBackbuffer;
    std::string mExtensions;
    bool mQueriedExtensions;
    NSWindow* mWindow;
    NSOpenGLContext* mContext;
    NSOpenGLContext* mSharedContext;
    bool mNeedsResolve;
    Gles mApi;

    static thread_local GlesRendererImpl* tlsBound;
};

thread_local GlesRendererImpl* GlesRendererImpl::tlsBound = nullptr;

GlesRendererImpl::GlesRendererImpl(GlesRendererImpl* shared_context)
        : mQueriedExtensions(false)
        , mWindow(nullptr)
        , mContext(nullptr)
        , mNeedsResolve(true)
        , mSharedContext(shared_context != nullptr ? shared_context->mContext : 0) {

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

    if (mWindow != nullptr) {
        [mWindow close];
        [mWindow release];
        mWindow = nullptr;
    }

    if (mContext != nullptr) {
        [mContext release];
        mContext = nullptr;
    }

    mBackbuffer = Backbuffer();
}

void GlesRendererImpl::setBackbuffer(Backbuffer backbuffer) {
    if (mBackbuffer == backbuffer) {
        return; // No change
    }

    // Some exotic extensions let you create contexts without a backbuffer.
    // In these cases the backbuffer is zero size - just create a small one.
    int safe_width = (backbuffer.width > 0) ? backbuffer.width : 8;
    int safe_height = (backbuffer.height > 0) ? backbuffer.height : 8;

    if (mBackbuffer.format == backbuffer.format) {
        // Only a resize is necessary
        GAPID_INFO("Resizing renderer: %dx%d -> %dx%d",
                mBackbuffer.width, mBackbuffer.height, backbuffer.width, backbuffer.height);
        [mWindow setContentSize: NSMakeSize(safe_width, safe_height)];
        [mContext update];
        mBackbuffer = backbuffer;
        return;
    }

    auto wasBound = tlsBound == this;

    [NSApplication sharedApplication];

    reset();

    NSRect rect = NSMakeRect(0, 0, safe_width, safe_height);
    mWindow = [[NSWindow alloc]
        initWithContentRect:rect
        styleMask:NSWindowStyleMaskBorderless
        backing:NSBackingStoreBuffered
        defer:NO
    ];
    if (mWindow == nullptr) {
        GAPID_FATAL("Unable to create NSWindow");
    }

    int r = 8, g = 8, b = 8, a = 8, d = 24, s = 8;
    core::gl::getColorBits(backbuffer.format.color, r, g, b, a);
    core::gl::getDepthBits(backbuffer.format.depth, d);
    core::gl::getStencilBits(backbuffer.format.stencil, s);

    NSOpenGLPixelFormatAttribute attributes[] = {
        NSOpenGLPFANoRecovery,
        NSOpenGLPFAColorSize, (NSOpenGLPixelFormatAttribute)(r+g+b),
        NSOpenGLPFAAlphaSize, (NSOpenGLPixelFormatAttribute)(a),
        NSOpenGLPFADepthSize, (NSOpenGLPixelFormatAttribute)d,
        NSOpenGLPFAStencilSize, (NSOpenGLPixelFormatAttribute)s,
        NSOpenGLPFAAccelerated,
        NSOpenGLPFABackingStore,
        NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core,
        (NSOpenGLPixelFormatAttribute)0
    };

    NSOpenGLPixelFormat* format = [[NSOpenGLPixelFormat alloc] initWithAttributes:attributes];
    if (format == nullptr) {
        GAPID_FATAL("Unable to create NSOpenGLPixelFormat");
    }

    mContext = [[NSOpenGLContext alloc]
            initWithFormat:format
            shareContext:mSharedContext];
    if (mContext == nullptr) {
        GAPID_FATAL("Unable to create NSOpenGLContext");
    }

    [mContext setView:[mWindow contentView]];

    mBackbuffer = backbuffer;
    mNeedsResolve = true;

    if (wasBound) {
        bind();
    }
}

void GlesRendererImpl::bind() {
    auto bound = tlsBound;
    if (bound == this) {
        return;
    }

    if (bound != nullptr) {
        bound->unbind();
    }

    [mContext makeCurrentContext];
    tlsBound = this;

    if (mNeedsResolve) {
        mNeedsResolve = false;
        mApi.resolve();
    }

    int major = 0;
    int minor = 0;
    mApi.mFunctionStubs.glGetIntegerv(Gles::GLenum::GL_MAJOR_VERSION, &major);
    mApi.mFunctionStubs.glGetIntegerv(Gles::GLenum::GL_MINOR_VERSION, &minor);
}

void GlesRendererImpl::unbind() {
    if (tlsBound == this) {
        [NSOpenGLContext clearCurrentContext];
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
        for (i = 0; i < n; i++) {
            if (i > 0) {
              mExtensions += " ";
            }
            mExtensions += reinterpret_cast<const char*>(
                mApi.mFunctionStubs.glGetStringi(Gles::GLenum::GL_EXTENSIONS, i));
        }
    }
    return &mExtensions[0];
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
