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
#include "gapis/api/core/core_pb/api.pb.h"
#include "gapis/api/gles/gles_pb/api.pb.h"
#include "gapis/api/gles/gles_pb/extras.pb.h"

#include <cstdlib>
#include <vector>
#include <memory>

#if TARGET_OS == GAPID_OS_WINDOWS
#include "windows/wgl.h"
#endif //  TARGET_OS == GAPID_OS_WINDOWS

#if TARGET_OS == GAPID_OS_ANDROID
#include <sys/prctl.h>
#include <jni.h>
static JavaVM* gJavaVM = nullptr;
extern "C"
jint JNI_OnLoad(JavaVM *vm, void *reserved) {
    GAPID_INFO("JNI_OnLoad() was called. vm = %p", vm);
    gJavaVM = vm;
    gapii::Spy::get(); // Construct the spy.
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

const EGLint EGL_NO_SURFACE                = 0x0000;
const EGLint EGL_FALSE                     = 0x0000;
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
const EGLint EGL_DRAW                      = 0x3059;
const EGLint EGL_READ                      = 0x305A;
const EGLint EGL_SWAP_BEHAVIOR             = 0x3093;
const EGLint EGL_BUFFER_PRESERVED          = 0x3094;

const EGLint EGL_CONTEXT_MAJOR_VERSION_KHR       = 0x3098;
const EGLint EGL_CONTEXT_MINOR_VERSION_KHR       = 0x30FB;
const EGLint EGL_CONTEXT_FLAGS_KHR               = 0x30FC;
const EGLint EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR    = 0x0001;
const EGLint EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR = 0x30FD;

const uint32_t GLX_WIDTH  = 0x801D;
const uint32_t GLX_HEIGHT = 0x801E;

const uint32_t kMaxFramebufferObservationWidth = 1920 / 2;
const uint32_t kMaxFramebufferObservationHeight = 1280 / 2;

const uint32_t kCGLCPSurfaceBackingSize = 304;

const uint32_t kStartMidExecutionCapture =  0xdeadbeef;

const int32_t  kSuspendIndefinitely = -1;

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
        CallObserver observer(s, 0);
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
  , mRecordGLErrorState(false)
  , mPendingFramebufferObservation(nullptr) {


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

    CallObserver observer(this, 0);

    mEncoder = gapii::PackEncoder::create(mConnection);

    GlesSpy::init();
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
    // Read attrib list
    std::map<EGLint, EGLint> attribs;
    while(attrib_list != nullptr && *attrib_list != EGL_NONE) {
        EGLint key = *(attrib_list++);
        EGLint val = *(attrib_list++);
        attribs[key] = val;
    }

    // Modify attrib list
    if (mRecordGLErrorState) {
        attribs[EGL_CONTEXT_FLAGS_KHR] |= EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR;
    }

    // Write attrib list
    std::vector<EGLint> attrib_vector;
    for(auto it: attribs) {
        attrib_vector.push_back(it.first);
        attrib_vector.push_back(it.second);
    }
    attrib_vector.push_back(EGL_NONE);
    attrib_vector.push_back(EGL_NONE);

    auto res = GlesSpy::eglCreateContext(observer, display, config, share_context, attrib_vector.data());

    // NB: The getters modify the std::map, so this log must be last.
    GAPID_INFO("eglCreateContext requested: GL %i.%i, profile 0x%x, flags 0x%x",
               attribs[EGL_CONTEXT_MAJOR_VERSION_KHR], attribs[EGL_CONTEXT_MINOR_VERSION_KHR],
               attribs[EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR], attribs[EGL_CONTEXT_FLAGS_KHR]);
    return res;
}

static void STDCALL DebugCallback(uint32_t source, uint32_t type, uint32_t id, uint32_t severity,
                                  uint32_t length, const char* message, void* user_param) {
    Spy* spy = reinterpret_cast<Spy*>(user_param);
    if (type == GL_DEBUG_TYPE_PUSH_GROUP || type == GL_DEBUG_TYPE_POP_GROUP) {
        return; // Ignore
    } else if (type == GL_DEBUG_TYPE_ERROR || severity == GL_DEBUG_SEVERITY_HIGH) {
        GAPID_ERROR("KHR_debug: %s", message);
    } else {
        GAPID_INFO("KHR_debug: %s", message);
    }
    // TODO: We should store the message in the trace.
}

EGLBoolean Spy::eglMakeCurrent(CallObserver* observer, EGLDisplay display, EGLSurface draw, EGLSurface read, EGLContext context) {
    EGLBoolean res = GlesSpy::eglMakeCurrent(observer, display, draw, read, context);
    if (mRecordGLErrorState && Extension.mGL_KHR_debug) {
        void* old_callback = nullptr;
        void* new_callback = reinterpret_cast<void*>(&DebugCallback);
        GlesSpy::mImports.glGetPointerv(GL_DEBUG_CALLBACK_FUNCTION, &old_callback);
        if (old_callback != new_callback) {
            GlesSpy::mImports.glDebugMessageCallback(new_callback, this);
            GlesSpy::mImports.glEnable(GL_DEBUG_OUTPUT);
            GlesSpy::mImports.glEnable(GL_DEBUG_OUTPUT_SYNCHRONOUS);
            GAPID_INFO("KHR_debug extension enabled");
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

    std::string threadName;
#if TARGET_OS == GAPID_OS_ANDROID
    char buffer[256] = { 0 };
    prctl(PR_GET_NAME, (unsigned long)buffer, 0, 0, 0);
    threadName = std::string(buffer);
#endif

    std::shared_ptr<StaticContextState> out(new StaticContextState(constants, threadName));

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


void Spy::onPostDrawCall(CallObserver* observer, uint8_t api) {
    if (is_suspended()) {
         return;
    }
    if (mObserveDrawFrequency != 0 && (mNumDraws % mObserveDrawFrequency == 0)) {
        GAPID_DEBUG("Observe framebuffer after draw call %d", mNumDraws);
        observeFramebuffer(observer, api);
    }
    mNumDraws++;
    mNumDrawsPerFrame++;
}

void Spy::onPreStartOfFrame(CallObserver* observer, uint8_t api) {
    if (is_suspended()) {
        return;
    }
    if (mObserveFrameFrequency != 0 && (mNumFrames % mObserveFrameFrequency == 0)) {
        GAPID_DEBUG("Observe framebuffer after frame %d", mNumFrames);
        observeFramebuffer(observer, api);
    }
    GAPID_DEBUG("NumFrames:%d NumDraws:%d NumDrawsPerFrame:%d",
               mNumFrames, mNumDraws, mNumDrawsPerFrame);
    mNumFrames++;
    mNumDrawsPerFrame = 0;
}

void Spy::onPostStartOfFrame(CallObserver* observer) {
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

void Spy::onPreEndOfFrame(CallObserver* observer, uint8_t api) {
    if (is_suspended()) {
        return;
    }
    if (mObserveFrameFrequency != 0 && (mNumFrames % mObserveFrameFrequency == 0)) {
        GAPID_DEBUG("Observe framebuffer after frame %d", mNumFrames);
        // The EndOfFrame atom for Vulkan is vkQueuePresentKHR. Because once an
        // image is sent to device for presenting, we cannot access the image
        // again before acquiring it back. The data of the framebuffer image to
        // be presented must be gather before calling the underlying
        // vkQueuePresentKHR.  However, the data must be messaged after the
        // vkQueuePresentKHR so the server knows the observed framebuffer is
        // that specific vkQueuePresentKHR.
        observeFramebuffer(observer, api, api == VulkanSpy::kApiIndex);
    }
    GAPID_DEBUG("NumFrames:%d NumDraws:%d NumDrawsPerFrame:%d",
               mNumFrames, mNumDraws, mNumDrawsPerFrame);
    mNumFrames++;
    mNumDrawsPerFrame = 0;
}

void Spy::onPostEndOfFrame(CallObserver* observer) {
    if (mPendingFramebufferObservation) {
        mEncoder->message(mPendingFramebufferObservation.get());
        mPendingFramebufferObservation.reset(nullptr);
    }
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
void Spy::observeFramebuffer(CallObserver* observer, uint8_t api, bool pendMessaging) {
    uint32_t w = 0;
    uint32_t h = 0;
    std::vector<uint8_t> data;
    switch(api) {
        case GlesSpy::kApiIndex:
            if (!GlesSpy::observeFramebuffer(observer, &w, &h, &data)) {
                return;
            }
            break;
        case VulkanSpy::kApiIndex:
            if (!VulkanSpy::observeFramebuffer(observer, &w, &h, &data)) {
                return;
            }
            break;
    }

    uint32_t downsampledW, downsampledH;
    std::vector<uint8_t> downsampledData;
    if (downsamplePixels(data, w, h,
                         &downsampledData, &downsampledW, &downsampledH,
                         kMaxFramebufferObservationWidth, kMaxFramebufferObservationHeight)) {
        // atom_pb::FramebufferObservation observation;
        auto observation = std::unique_ptr<atom_pb::FramebufferObservation>(
            new atom_pb::FramebufferObservation());
        observation->set_originalwidth(w);
        observation->set_originalheight(h);
        observation->set_datawidth(downsampledW);
        observation->set_dataheight(downsampledH);
        observation->set_data(downsampledData.data(), downsampledData.size());
        if (pendMessaging) {
          mPendingFramebufferObservation = std::move(observation);
        } else {
          mEncoder->message(observation.get());
        }
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
    std::shared_ptr<Context> ctx = this->Contexts[mCurrentThread];
    if (ctx) {
        GLenum_Error& fakeGlError = this->mFakeGlError[ctx->mIdentifier];
        if (fakeGlError == 0) {
            fakeGlError = error;
        }
    }
}

uint32_t Spy::glGetError(CallObserver* observer) {
    std::shared_ptr<Context> ctx = this->Contexts[mCurrentThread];
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
    auto st = new core_pb::switchThread();
    st->set_threadid(threadID);
    observer->encodeAndDeleteCommand(st);
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
