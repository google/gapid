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

namespace gapir {
namespace {

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

    Backbuffer mBackbuffer;
    bool mBound;
    std::string mExtensions;
    bool mQueriedExtensions;
    NSWindow* mWindow;
    NSOpenGLContext* mContext;
    bool mNeedsResolve;
    Gles mApi;
};

GlesRendererImpl::GlesRendererImpl()
        : mBound(false)
        , mQueriedExtensions(false)
        , mWindow(nullptr)
        , mContext(nullptr)
        , mNeedsResolve(true) {

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

    if (mBackbuffer.format == backbuffer.format) {
        // Only a resize is necessary
        GAPID_INFO("Resizing renderer: %dx%d -> %dx%d",
                mBackbuffer.width, mBackbuffer.height, backbuffer.width, backbuffer.height);
        [mWindow setContentSize: NSMakeSize(backbuffer.width, backbuffer.height)];
        [mContext update];
        mBackbuffer = backbuffer;
        return;
    }

    const bool wasBound = mBound;

    [NSApplication sharedApplication];

    reset();

    NSRect rect = NSMakeRect(0, 0, backbuffer.width, backbuffer.height);
    mWindow = [[NSWindow alloc]
        initWithContentRect:rect
        styleMask:NSBorderlessWindowMask
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

    mContext = [[NSOpenGLContext alloc] initWithFormat:format shareContext:nil];
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
    if (!mBound) {
        [mContext makeCurrentContext];
        mBound = true;

        if (mNeedsResolve) {
            mNeedsResolve = false;
            mApi.resolve();
        }

        int major = 0;
        int minor = 0;
        mApi.mFunctionStubs.glGetIntegerv(Gles::GLenum::GL_MAJOR_VERSION, &major);
        mApi.mFunctionStubs.glGetIntegerv(Gles::GLenum::GL_MINOR_VERSION, &minor);
        GAPID_WARNING("Bound OpenGL %d.%d renderer", major, minor);
    }
}

void GlesRendererImpl::unbind() {
    if (mBound) {
        [NSOpenGLContext clearCurrentContext];
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

GlesRenderer* GlesRenderer::create(GlesRenderer* sharedContext) {
    return new GlesRendererImpl();
}

}  // namespace gapir
