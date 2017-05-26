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

#include "spy.h"

#include "connection_header.h"
#include "connection_stream.h"

#include "gapii/cc/gles_exports.h"
#include "gapii/cc/spy.h"
#include "gapii/cc/to_proto.h"

#include "core/cc/encoder.h"
#include "core/cc/gl/formats.h"
#include "core/cc/lock.h"
#include "core/cc/log.h"
#include "core/cc/target.h"
#include "core/os/device/deviceinfo/cc/query.h"

#include "gapis/capture/capture.pb.h"
#include "gapis/gfxapi/gles/gles_pb/api.pb.h"
#include "gapis/gfxapi/gles/gles_pb/extras.pb.h"

#include <cstdlib>
#include <vector>
#include <memory>

#if TARGET_OS == GAPID_OS_WINDOWS
#include "windows/wgl.h"
#endif //  TARGET_OS == GAPID_OS_WINDOWS

#if TARGET_OS == GAPID_OS_ANDROID
#include <jni.h>
static JavaVM* gJavaVM = nullptr;
extern "C"
jint JNI_OnLoad(JavaVM *vm, void *reserved) {
    GAPID_INFO("JNI_OnLoad() was called. vm = %p", vm);
    gJavaVM = vm;
    return JNI_VERSION_1_6;
}
void* queryPlatformData() {
    JNIEnv* env = nullptr;

    auto res = gJavaVM->GetEnv(reinterpret_cast<void**>(&env), JNI_VERSION_1_6);
    switch (res) {
    case JNI_OK:
        break;
    case JNI_EDETACHED:
        res = gJavaVM->AttachCurrentThread(&env, nullptr);
        if (res != 0) {
            GAPID_FATAL("Failed to attach thread to JavaVM. (%d)", res);
        }
        break;
    default:
        GAPID_FATAL("Failed to get Java env. (%d)", res);
    }
    GAPID_INFO("queryPlatformData() env = %p", env);
    return env;
}
#else  // TARGET_OS == GAPID_OS_ANDROID
void* queryPlatformData() { return nullptr; }
#endif  // TARGET_OS == GAPID_OS_ANDROID

using namespace gapii::GLenum;

namespace {

typedef uint32_t EGLint;

const EGLint EGL_TRUE                      = 0x0001;
const EGLint EGL_ALPHA_SIZE                = 0x3021;
const EGLint EGL_BLUE_SIZE                 = 0x3022;
const EGLint EGL_GREEN_SIZE                = 0x3023;
const EGLint EGL_RED_SIZE                  = 0x3024;
const EGLint EGL_DEPTH_SIZE                = 0x3025;
const EGLint EGL_STENCIL_SIZE              = 0x3026;
const EGLint EGL_CONFIG_ID                 = 0x3028;
const EGLint EGL_NONE                      = 0x3038;
const EGLint EGL_HEIGHT                    = 0x3056;
const EGLint EGL_WIDTH                     = 0x3057;
const EGLint EGL_SWAP_BEHAVIOR             = 0x3093;
const EGLint EGL_BUFFER_PRESERVED          = 0x3094;
const EGLint EGL_CONTEXT_MAJOR_VERSION_KHR = 0x3098;

const uint32_t GLX_WIDTH  = 0x801D;
const uint32_t GLX_HEIGHT = 0x801E;

const uint32_t kMaxFramebufferObservationWidth = 1920 / 2;
const uint32_t kMaxFramebufferObservationHeight = 1280 / 2;

const uint32_t kCGLCPSurfaceBackingSize = 304;

const uint32_t kStartMidExecutionCapture =  0xdeadbeef;

const int32_t  kSuspendIndefinitely = -1;

const uint8_t kCoreAPI = 0;
const uint8_t kGLESAPI = 1;
const uint8_t kVulkanAPI = 2;

core::Mutex gMutex;  // Guards gSpy.
std::unique_ptr<gapii::Spy> gSpy;

} // anonymous namespace

namespace gapii {

Spy* Spy::get() {
    bool init;
    {
        core::Lock<core::Mutex> lock(&gMutex);
        init = !gSpy;
        if (init) {
            GAPID_INFO("Constructing spy...");
            gSpy.reset(new Spy());
            GAPID_INFO("Registering spy symbols...");
            for (int i = 0; kGLESExports[i].mName != NULL; ++i) {
                gSpy->RegisterSymbol(kGLESExports[i].mName, kGLESExports[i].mFunc);
            }
        }
    }
    if (init) {
        auto s = gSpy.get();
        if (!s->try_to_enter()) {
            GAPID_FATAL("Couldn't enter on init?!")
        }
        CallObserver observer(s, kCoreAPI);
        s->lock(&observer, "writeHeader");
        s->writeHeader();
        s->unlock();
        s->exit();
    }
    return gSpy.get();
}

Spy::Spy()
  : mNumFrames(0)
  , mSuspendCaptureFrames(0)
  , mCaptureFrames(0)
  , mNumDraws(0)
  , mNumDrawsPerFrame(0)
  , mObserveFrameFrequency(0)
  , mObserveDrawFrequency(0)
  , mDisablePrecompiledShaders(false)
  , mRecordGLErrorState(false) {


#if TARGET_OS == GAPID_OS_ANDROID
    // Use a "localabstract" pipe on Android to prevent depending on the traced application
    // having the INTERNET permission set, required for opening and listening on a TCP socket.
    mConnection = ConnectionStream::listenPipe("gapii", true);
#else // TARGET_OS
    mConnection = ConnectionStream::listenSocket("127.0.0.1", "9286");
#endif // TARGET_OS

    GAPID_INFO("Connection made");

    ConnectionHeader header;
    if (header.read(mConnection.get())) {
        mObserveFrameFrequency = header.mObserveFrameFrequency;
        mObserveDrawFrequency = header.mObserveDrawFrequency;
        mDisablePrecompiledShaders =
                (header.mFlags & ConnectionHeader::FLAG_DISABLE_PRECOMPILED_SHADERS) != 0;
        mRecordGLErrorState =
                (header.mFlags & ConnectionHeader::FLAG_RECORD_ERROR_STATE) != 0;
        // This will be over-written if we also set the header flags
        mSuspendCaptureFrames = header.mStartFrame;
        mCaptureFrames = header.mNumFrames;
        mSuspendCaptureFrames.store((header.mFlags & ConnectionHeader::FLAG_DEFER_START)?
            kSuspendIndefinitely: mSuspendCaptureFrames.load());
    } else {
        GAPID_WARNING("Failed to read connection header");
    }
    set_valid_apis(header.mAPIs);
    GAPID_ERROR("APIS %08x", header.mAPIs);
    GAPID_INFO("GAPII connection established. Settings:");
    GAPID_INFO("Observe framebuffer every %d frames", mObserveFrameFrequency);
    GAPID_INFO("Observe framebuffer every %d draws", mObserveDrawFrequency);
    GAPID_INFO("Disable precompiled shaders: %s", mDisablePrecompiledShaders ? "true" : "false");

    CallObserver observer(this, kCoreAPI);

    mEncoder = gapii::PackEncoder::create(mConnection);

    GlesSpy::init();
    CoreSpy::init();
    VulkanSpy::init();
    SpyBase::init(&observer, mEncoder);

    if (mSuspendCaptureFrames.load() == kSuspendIndefinitely) {
        mDeferStartJob = std::unique_ptr<core::AsyncJob>(
        new core::AsyncJob([this]() {
            uint32_t buffer;
            if (4 == mConnection->read(&buffer, 4)) {
                if (buffer == kStartMidExecutionCapture) {
                    mSuspendCaptureFrames.store(1);
                }
            }
        }));
    }
    set_suspended(mSuspendCaptureFrames.load() != 0);
    set_observing(mObserveFrameFrequency != 0 || mObserveDrawFrequency != 0);
}

void Spy::writeHeader() {
    if (!query::createContext(queryPlatformData())) {
        GAPID_ERROR("query::createContext() errored: %s", query::contextError());
    }
    capture::Header file_header;
    file_header.set_allocated_device(query::getDeviceInstance(queryPlatformData()));
    file_header.set_allocated_abi(query::currentABI());
    mEncoder->message(&file_header);
    query::destroyContext();
}

void Spy::resolveImports() {
    GlesSpy::mImports.resolve();
}

EGLBoolean Spy::eglInitialize(CallObserver* observer, EGLDisplay dpy, EGLint* major, EGLint* minor) {
    EGLBoolean res = GlesSpy::eglInitialize(observer, dpy, major, minor);
    if (res != 0) {
        resolveImports(); // Imports may have changed. Re-resolve.
    }
    return res;
}

EGLContext Spy::eglCreateContext(CallObserver* observer, EGLDisplay display, EGLConfig config,
                                 EGLContext share_context, EGLint* attrib_list) {
    EGLContext res = GlesSpy::eglCreateContext(observer, display, config, share_context, attrib_list);
    for (int i = 0; attrib_list != nullptr && attrib_list[i] != EGL_NONE; i += 2) {
        if (attrib_list[i] == EGL_CONTEXT_MAJOR_VERSION_KHR) {
            GAPID_INFO("eglCreateContext requested GL version %i", attrib_list[i+1]);
        }
    }
    return res;
}

#define EGL_QUERY_SURFACE(name, var) \
    if (GlesSpy::mImports.eglQuerySurface(display, draw, name, var) != EGL_TRUE) { \
        GAPID_WARNING("eglQuerySurface(0x%p, 0x%p, " #name ", " #var ") failed", display, draw); \
    }
#define EGL_GET_CONFIG_ATTRIB(name, var) \
    if (GlesSpy::mImports.eglGetConfigAttrib(display, config, name, var) != EGL_TRUE) { \
        GAPID_WARNING("eglGetConfigAttrib(0x%p, 0x%p, " #name ", " #var ") failed", display, config); \
    }

std::shared_ptr<StaticContextState> GlesSpy::GetEGLStaticContextState(CallObserver* observer, EGLDisplay display, EGLSurface draw, EGLContext context) {
    if (draw == nullptr) {
        return nullptr;
    }

    Constants constants;
    getContextConstants(constants);
    std::shared_ptr<StaticContextState> out(new StaticContextState(constants));

    // Store the StaticContextState as an extra.
    observer->addExtra(out->toProto());

    return out;
}

std::shared_ptr<DynamicContextState> GlesSpy::GetEGLDynamicContextState(CallObserver* observer, EGLDisplay display, EGLSurface draw, EGLContext context) {
    if (draw == nullptr) {
        return nullptr;
    }

    EGLint width = 0;
    EGLint height = 0;
    EGLint swapBehavior = 0;
    EGL_QUERY_SURFACE(EGL_WIDTH, &width);
    EGL_QUERY_SURFACE(EGL_HEIGHT, &height);
    EGL_QUERY_SURFACE(EGL_SWAP_BEHAVIOR, &swapBehavior);

    // Get the backbuffer formats.
    uint32_t backbufferColorFmt = GL_RGBA8;
    uint32_t backbufferDepthFmt = GL_DEPTH24_STENCIL8;
    uint32_t backbufferStencilFmt = GL_DEPTH24_STENCIL8;
    bool usingDefaults = true;

    EGLint configId = 0;
    EGLint r = 0, g = 0, b = 0, a = 0, d = 0, s = 0;
    if (GlesSpy::mImports.eglQueryContext(display, context, EGL_CONFIG_ID, &configId) == EGL_TRUE) {
        GAPID_INFO("Active context ID: %d", configId);
        EGLint attribs[] = { EGL_CONFIG_ID, configId, EGL_NONE };
        EGLConfig config;
        EGLint count = 0;
        if (GlesSpy::mImports.eglChooseConfig(display, attribs, &config, 1, &count) == EGL_TRUE) {
            EGL_GET_CONFIG_ATTRIB(EGL_RED_SIZE, &r);
            EGL_GET_CONFIG_ATTRIB(EGL_GREEN_SIZE, &g);
            EGL_GET_CONFIG_ATTRIB(EGL_BLUE_SIZE, &b);
            EGL_GET_CONFIG_ATTRIB(EGL_ALPHA_SIZE, &a);
            EGL_GET_CONFIG_ATTRIB(EGL_DEPTH_SIZE, &d);
            EGL_GET_CONFIG_ATTRIB(EGL_STENCIL_SIZE, &s);
            GAPID_INFO("Framebuffer config: R%d G%d B%d A%d D%d S%d", r, g, b, a, d, s);

            // Get the formats from the bit depths.
            if (!core::gl::getColorFormat(r, g, b, a, backbufferColorFmt)) {
                GAPID_WARNING("getColorFormat(%d, %d, %d, %d) failed", r, g, b, a);
            }
            if (!core::gl::getDepthStencilFormat(d, s, backbufferDepthFmt, backbufferStencilFmt)) {
                GAPID_WARNING("getDepthStencilFormat(%d, %d) failed", d, s);
            }
            usingDefaults = false;
        } else {
            GAPID_WARNING("eglChooseConfig() failed for config ID %d. Assuming defaults.", configId);
        }
    } else {
        GAPID_WARNING("eglQueryContext(0x%p, 0x%p, EGL_CONFIG_ID, &configId) failed. "
                "Assuming defaults.", display, context);
    }

    bool resetViewportScissor = true;
    bool preserveBuffersOnSwap = swapBehavior == EGL_BUFFER_PRESERVED;

    std::shared_ptr<DynamicContextState> out(new DynamicContextState(
        width, height,
        backbufferColorFmt, backbufferDepthFmt, backbufferStencilFmt,
        resetViewportScissor,
        preserveBuffersOnSwap,
        r, g, b, a, d, s
    ));

    // Store the DynamicContextState as an extra.
    observer->addExtra(out->toProto());

    return out;
}

#undef EGL_QUERY_SURFACE
#undef EGL_GET_CONFIG_ATTRIB


void Spy::onPostDrawCall(uint8_t api) {
    if (is_suspended()) {
         return;
    }
    if (mObserveDrawFrequency != 0 && (mNumDraws % mObserveDrawFrequency == 0)) {
        GAPID_DEBUG("Observe framebuffer after draw call %d", mNumDraws);
        observeFramebuffer(api);
    }
    mNumDraws++;
    mNumDrawsPerFrame++;
}

void Spy::onPreEndOfFrame(uint8_t api) {
    if (is_suspended()) {
        return;
    }
    if (mObserveFrameFrequency != 0 && (mNumFrames % mObserveFrameFrequency == 0)) {
        GAPID_DEBUG("Observe framebuffer after frame %d", mNumFrames);
        observeFramebuffer(api);
    }
    GAPID_DEBUG("NumFrames:%d NumDraws:%d NumDrawsPerFrame:%d",
               mNumFrames, mNumDraws, mNumDrawsPerFrame);
    mNumFrames++;
    mNumDrawsPerFrame = 0;
}

void Spy::onPostEndOfFrame(CallObserver* observer) {
    if (!is_suspended() && mCaptureFrames >= 1) {
        mCaptureFrames -= 1;
        if (mCaptureFrames == 0) {
            mConnection->close();
            set_suspended(true);
        }
    }
    if (mSuspendCaptureFrames.load() > 0) {
        if (is_suspended() && mSuspendCaptureFrames.fetch_sub(1) == 1) {
            set_suspended(false);
            EnumerateVulkanResources(observer);
        }
    }
}

static bool downsamplePixels(const std::vector<uint8_t>& srcData, uint32_t srcW, uint32_t srcH,
                             std::vector<uint8_t>* outData, uint32_t* outW, uint32_t* outH,
                             uint32_t maxW, uint32_t maxH) {
    // Calculate the minimal scaling factor as integer fraction.
    uint32_t mul = 1;
    uint32_t div = 1;
    if (mul*srcW > maxW*div) { // if mul/div > maxW/srcW
        mul = maxW;
        div = srcW;
    }
    if (mul*srcH > maxH*div) { // if mul/div > maxH/srcH
        mul = maxH;
        div = srcH;
    }

    // Calculate the final dimensions (round up) and allocate new buffer.
    uint32_t dstW = (srcW*mul + div - 1) / div;
    uint32_t dstH = (srcH*mul + div - 1) / div;
    outData->reserve(dstW*dstH*4);

    // Downsample the image by averaging the colours of neighbouring pixels.
    for (uint32_t srcY = 0, y = 0, dstY = 0; dstY < dstH; srcY = y, dstY++) {
        for (uint32_t srcX = 0, x = 0, dstX = 0; dstX < dstW; srcX = x, dstX++) {
            uint32_t r = 0, g = 0, b = 0, a = 0, n = 0;
            // We need to loop over srcX/srcY ranges several times, so we keep them in x/y,
            // and we update srcX/srcY to the last x/y only once we are done with the pixel.
            for (y = srcY; y*dstH < (dstY+1)*srcH; y++) { // while y*yScale < dstY+1
                const uint8_t* src = &srcData[(srcX + y*srcW) * 4];
                for (x = srcX; x*dstW < (dstX+1)*srcW; x++) { // while x*xScale < dstX+1
                    r += *(src++);
                    g += *(src++);
                    b += *(src++);
                    a += *(src++);
                    n += 1;
                }
            }
            outData->push_back(r/n);
            outData->push_back(g/n);
            outData->push_back(b/n);
            outData->push_back(a/n);
        }
    }

    *outW = dstW;
    *outH = dstH;
    return true;
}

// observeFramebuffer captures the currently bound framebuffer, and writes
// it to a FramebufferObservation atom.
void Spy::observeFramebuffer(uint8_t api) {
    uint32_t w = 0;
    uint32_t h = 0;
    std::vector<uint8_t> data;
    switch(api) {
        case kCoreAPI:
            if (!CoreSpy::observeFramebuffer(&w, &h, &data)) {
                return;
            }
            break;
        case kGLESAPI:
            if (!GlesSpy::observeFramebuffer(&w, &h, &data)) {
                return;
            }
            break;
        case kVulkanAPI:
            if (!VulkanSpy::observeFramebuffer(&w, &h, &data)) {
                return;
            }
            break;
    }

    uint32_t downsampledW, downsampledH;
    std::vector<uint8_t> downsampledData;
    if (downsamplePixels(data, w, h,
                         &downsampledData, &downsampledW, &downsampledH,
                         kMaxFramebufferObservationWidth, kMaxFramebufferObservationHeight)) {
        atom_pb::FramebufferObservation observation;
        observation.set_originalwidth(w);
        observation.set_originalheight(h);
        observation.set_datawidth(downsampledW);
        observation.set_dataheight(downsampledH);
        observation.set_data(downsampledData.data(), downsampledData.size());
        mEncoder->message(&observation);
    }
}

void Spy::onPostFence(CallObserver* observer) {
    if (mRecordGLErrorState) {
        auto traceErr = GlesSpy::mImports.glGetError();

        // glGetError() cleared the error in the driver.
        // Fake it the next time the user calls glGetError().
        if (traceErr != 0) {
            setFakeGlError(traceErr);
        }

        auto es = new gles_pb::ErrorState();
        es->set_tracedriversglerror(traceErr);
        es->set_interceptorsglerror(observer->getError());
        observer->addExtra(es);
    }
}

void Spy::setFakeGlError(GLenum_Error error) {
    std::shared_ptr<Context> ctx = this->Contexts[this->CurrentThread];
    if (ctx) {
        GLenum_Error& fakeGlError = this->mFakeGlError[ctx->mIdentifier];
        if (fakeGlError == 0) {
            fakeGlError = error;
        }
    }
}

uint32_t Spy::glGetError(CallObserver* observer) {
    std::shared_ptr<Context> ctx = this->Contexts[this->CurrentThread];
    if (ctx) {
        GLenum_Error& fakeGlError = this->mFakeGlError[ctx->mIdentifier];
        if (fakeGlError != 0) {
            observer->encodeAndDeleteCommand(new gles_pb::glGetError());
            GLenum_Error err = fakeGlError;
            fakeGlError = 0;
            return err;
        }
    }
    return GlesSpy::glGetError(observer);
}

void Spy::onThreadSwitched(CallObserver* observer, uint64_t threadID) {
    CoreSpy::switchThread(observer, threadID);
}

#if 0 // NON-EGL CONTEXTS ARE CURRENTLY NOT SUPPORTED
std::shared_ptr<ContextState> Spy::getWGLContextState(CallObserver*, HDC hdc, HGLRC hglrc) {
    if (hglrc == nullptr) {
        return nullptr;
    }

#if TARGET_OS == GAPID_OS_WINDOWS
    wgl::FramebufferInfo info;
    wgl::getFramebufferInfo(hdc, info);
    return getContextState(info.width, info.height,
            info.colorFormat, info.depthFormat, info.stencilFormat,
            /* resetViewportScissor */ true,
            /* preserveBuffersOnSwap */ false);
#else // TARGET_OS
    return nullptr;
#endif // TARGET_OS
}

std::shared_ptr<ContextState> Spy::getCGLContextState(CallObserver* observer, CGLContextObj ctx) {
    if (ctx == nullptr) {
        return nullptr;
    }

    CGSConnectionID cid;
    CGSWindowID wid;
    CGSSurfaceID sid;
    double bounds[4] = {0, 0, 0, 0};

    if (GlesSpy::mImports.CGLGetSurface(ctx, &cid, &wid, &sid) == 0) {
        GlesSpy::mImports.CGSGetSurfaceBounds(cid, wid, sid, bounds);
    } else {
        GAPID_WARNING("Could not get CGL surface");
    }
    int width = bounds[2] - bounds[0];  // size.x - origin.x
    int height = bounds[3] - bounds[1]; // size.y - origin.y

    // TODO: Probe formats
    return getContextState(width, height,
            GL_RGBA8, GL_DEPTH_COMPONENT16, GL_STENCIL_INDEX8,
            /* resetViewportScissor */ true,
            /* preserveBuffersOnSwap */ false);
}

std::shared_ptr<ContextState> Spy::getGLXContextState(CallObserver* observer, void* display, GLXDrawable draw, GLXDrawable read, GLXContext ctx) {
    if (display == nullptr) {
        return nullptr;
    }
    int width = 0;
    int height = 0;
    GlesSpy::mImports.glXQueryDrawable(display, draw, GLX_WIDTH, &width);
    GlesSpy::mImports.glXQueryDrawable(display, draw, GLX_HEIGHT, &height);

    // TODO: Probe formats
    return getContextState(width, height,
            GL_RGBA8, GL_DEPTH_COMPONENT16, GL_STENCIL_INDEX8,
            /* resetViewportScissor */ true,
            /* preserveBuffersOnSwap */ false);
}
#endif // #if 0

} // namespace gapii
