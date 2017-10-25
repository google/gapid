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
#include "gapis/api/gles/gles_pb/api.pb.h"
#include "gapis/api/gles/gles_pb/extras.pb.h"

#include <cstdlib>
#include <vector>
#include <memory>

// CurrentCaptureVersion is incremented on breaking changes to the capture format.
// NB: Also update equally named field in capture.go
static const int CurrentCaptureVersion = 0;

#if TARGET_OS == GAPID_OS_WINDOWS
#include "windows/wgl.h"
#endif //  TARGET_OS == GAPID_OS_WINDOWS

#if TARGET_OS == GAPID_OS_ANDROID
#include "gapii/cc/android/installer.h"
#include "gapii/cc/android/gvr_install.h"

#include <sys/prctl.h>
#include <jni.h>

static std::unique_ptr<gapii::Installer> gInstaller;
static JavaVM* gJavaVM = nullptr;

extern "C"
jint JNI_OnLoad(JavaVM *vm, void *reserved) {
    GAPID_INFO("JNI_OnLoad() was called. vm = %p", vm);
    gJavaVM = vm;
    gapii::Spy::get(); // Construct the spy.
    return JNI_VERSION_1_6;
}

void* queryPlatformData() { return gJavaVM; }
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

std::recursive_mutex gMutex;  // Guards gSpy.
std::unique_ptr<gapii::Spy> gSpy;
thread_local gapii::CallObserver* gContext = nullptr;

} // anonymous namespace

namespace gapii {

Spy* Spy::get() {
    std::lock_guard<std::recursive_mutex> lock(gMutex);
    if (!gSpy) {
        GAPID_INFO("Constructing spy...");
        gSpy.reset(new Spy());
        GAPID_INFO("Registering spy symbols...");
        for (int i = 0; kGLESExports[i].mName != NULL; ++i) {
            gSpy->RegisterSymbol(kGLESExports[i].mName, kGLESExports[i].mFunc);
        }
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

    if (!mConnection->write("gapii", 5)) { // handshake magic
        GAPID_FATAL("Couldn't send handshake magic");
    }

    GAPID_INFO("Connection made");

    ConnectionHeader header;
    if (!header.read(mConnection.get())) {
        GAPID_FATAL("Failed to read connection header");
    }

    GAPID_INFO("Connection header read");

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

    set_valid_apis(header.mAPIs);
    GAPID_ERROR("APIS %08x", header.mAPIs);
    GAPID_INFO("GAPII connection established. Settings:");
    GAPID_INFO("Observe framebuffer every %d frames", mObserveFrameFrequency);
    GAPID_INFO("Observe framebuffer every %d draws", mObserveDrawFrequency);
    GAPID_INFO("Disable precompiled shaders: %s", mDisablePrecompiledShaders ? "true" : "false");

    mEncoder = gapii::PackEncoder::create(mConnection);

    // writeHeader needs to come before the installer is created as the
    // deviceinfo queries want to call into EGL / GL commands which will be
    // patched.
    writeHeader();

#if TARGET_OS == GAPID_OS_ANDROID
    if (strlen(header.mLibInterceptorPath) > 0) {
        gInstaller = std::unique_ptr<Installer>(new Installer(header.mLibInterceptorPath));
    }
    if (header.mGvrHandle != 0) {
        auto gvr_lib = reinterpret_cast<void*>(header.mGvrHandle);
        install_gvr(gInstaller.get(), gvr_lib, &this->GvrSpy::mImports);
    }
#endif // TARGET_OS == GAPID_OS_ANDROID

    auto context = enter("init", 0);
    GlesSpy::init();
    VulkanSpy::init();
    SpyBase::init(context);
    exit();

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
    capture::Header file_header;
    file_header.set_version(CurrentCaptureVersion);
    file_header.set_allocated_device(query::getDeviceInstance(queryPlatformData()));
    file_header.set_allocated_abi(query::currentABI());
    mEncoder->object(&file_header);
}

void Spy::resolveImports() {
    GlesSpy::mImports.resolve();
}

CallObserver* Spy::enter(const char* name, uint32_t api) {
    auto ctx = new CallObserver(this, gContext, api);
    lock(ctx);
    ctx->setCurrentCommandName(name);
    gContext = ctx;
    return ctx;
}

void Spy::exit() {
    auto context = gContext;
    gContext = context->getParent();
    delete context;
    unlock();
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
    GAPID_INFO("eglCreateContext requested: GL %i.%i, profile 0x%x, flags 0x%x -> %p",
               attribs[EGL_CONTEXT_MAJOR_VERSION_KHR], attribs[EGL_CONTEXT_MINOR_VERSION_KHR],
               attribs[EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR], attribs[EGL_CONTEXT_FLAGS_KHR],
               res);
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

std::shared_ptr<StaticContextState> GlesSpy::GetEGLStaticContextState(CallObserver* observer, EGLDisplay display, EGLContext context) {
    Constants constants;
    getContextConstants(constants);

    std::string threadName;
#if TARGET_OS == GAPID_OS_ANDROID
    char buffer[256] = { 0 };
    prctl(PR_GET_NAME, (unsigned long)buffer, 0, 0, 0);
    threadName = std::string(buffer);
#endif

    std::shared_ptr<StaticContextState> out(new StaticContextState(constants, threadName));

    observer->encodeAndDelete(out->toProto());

    return out;
}

#define EGL_QUERY_SURFACE(name, draw, var) \
if (GlesSpy::mImports.eglQuerySurface(display, draw, name, var) != EGL_TRUE) { \
    GAPID_WARNING("eglQuerySurface(0x%p, 0x%p, " #name ", " #var ") failed", display, draw); \
}
#define EGL_GET_CONFIG_ATTRIB(name, var) \
if (GlesSpy::mImports.eglGetConfigAttrib(display, config, name, var) != EGL_TRUE) { \
    GAPID_WARNING("eglGetConfigAttrib(0x%p, 0x%p, " #name ", " #var ") failed", display, config); \
}

std::shared_ptr<DynamicContextState> GlesSpy::GetEGLDynamicContextState(CallObserver* observer, EGLDisplay display, EGLSurface draw, EGLContext context) {
    EGLint width = 0;
    EGLint height = 0;
    EGLint swapBehavior = 0;
    if (draw != nullptr) {
        EGL_QUERY_SURFACE(EGL_WIDTH, draw, &width);
        EGL_QUERY_SURFACE(EGL_HEIGHT, draw, &height);
        EGL_QUERY_SURFACE(EGL_SWAP_BEHAVIOR, draw, &swapBehavior);
    }

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
    observer->encodeAndDelete(out->toProto());

    return out;
}

#undef EGL_QUERY_SURFACE
#undef EGL_GET_CONFIG_ATTRIB

void Spy::gvr_frame_submit(CallObserver* observer, gvr_frame** frame, gvr_buffer_viewport_list* list, gvr_mat4_abi head_space_from_start_space) {
    GvrSpy::mLastSubmittedFrame = (frame != nullptr) ? (*frame) : nullptr;
    GvrSpy::gvr_frame_submit(observer, frame, list, head_space_from_start_space);
}

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

void Spy::onPostStartOfFrame() {
    if (!is_suspended() && mCaptureFrames >= 1) {
        mCaptureFrames -= 1;
        if (mCaptureFrames == 0) {
            mEncoder->flush();
            mConnection->close();
            set_suspended(true);
        }
    }
    if (mSuspendCaptureFrames.load() > 0) {
        if (is_suspended() && mSuspendCaptureFrames.fetch_sub(1) == 1) {
            exit();
            set_suspended(false);
            set_recording_state(true);
            auto spy_ctx = enter("RecreateState", 2);
            spy_ctx->enter(cmd::RecreateState{});
            EnumerateVulkanResources(spy_ctx);
            spy_ctx->exit();
            set_recording_state(false);
            // The outer call will handle the spy->exit() for us.
        }
    }
}

void Spy::onPreEndOfFrame(CallObserver* observer, uint8_t api) {
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

void Spy::onPostEndOfFrame() {
    if (!is_suspended() && mCaptureFrames >= 1) {
        mCaptureFrames -= 1;
        if (mCaptureFrames == 0) {
            mEncoder->flush();
            mConnection->close();
            set_suspended(true);
        }
    }
    if (mSuspendCaptureFrames.load() > 0) {
        if (is_suspended() && mSuspendCaptureFrames.fetch_sub(1) == 1) {
            exit();
            set_suspended(false);
            set_recording_state(true);
            auto spy_ctx = enter("RecreateState", 2);
            spy_ctx->enter(cmd::RecreateState{});
            EnumerateVulkanResources(spy_ctx);
            spy_ctx->exit();
            set_recording_state(false);
            // The outer call to VkQueuePresent will handle the spy->exit() for us.
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
// it to a FramebufferObservation extra.
void Spy::observeFramebuffer(CallObserver* observer, uint8_t api) {
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
        case GvrSpy::kApiIndex:
            if (!GvrSpy::observeFramebuffer(observer, &w, &h, &data)) {
                return;
            }
            break;
    }

    uint32_t downsampledW, downsampledH;
    std::vector<uint8_t> downsampledData;
    if (downsamplePixels(data, w, h,
                         &downsampledData, &downsampledW, &downsampledH,
                         kMaxFramebufferObservationWidth, kMaxFramebufferObservationHeight)) {
        auto observation = new capture::FramebufferObservation();
        observation->set_original_width(w);
        observation->set_original_height(h);
        observation->set_data_width(downsampledW);
        observation->set_data_height(downsampledH);
        observation->set_data(downsampledData.data(), downsampledData.size());
        observer->encodeAndDelete(observation);
    }
}

void Spy::onPostFence(CallObserver* observer) {
    if (mRecordGLErrorState) {
        auto traceErr = GlesSpy::mImports.glGetError();

        // glGetError() cleared the error in the driver.
        // Fake it the next time the user calls glGetError().
        if (traceErr != 0) {
            setFakeGlError(observer, traceErr);
        }

        auto es = new gles_pb::ErrorState();
        es->set_tracedriversglerror(traceErr);
        es->set_interceptorsglerror(observer->getError());
        observer->encodeAndDelete(es);
    }
}

void Spy::setFakeGlError(CallObserver* observer, GLenum_Error error) {
    std::shared_ptr<Context> ctx = this->Contexts[observer->getCurrentThread()];
    if (ctx) {
        GLenum_Error& fakeGlError = this->mFakeGlError[ctx->mIdentifier];
        if (fakeGlError == 0) {
            fakeGlError = error;
        }
    }
}

uint32_t Spy::glGetError(CallObserver* observer) {
    std::shared_ptr<Context> ctx = this->Contexts[observer->getCurrentThread()];
    if (ctx) {
        GLenum_Error& fakeGlError = this->mFakeGlError[ctx->mIdentifier];
        if (fakeGlError != 0) {
            observer->encodeAndDelete(new gles_pb::glGetError());
            GLenum_Error err = fakeGlError;
            fakeGlError = 0;
            return err;
        }
    }
    return GlesSpy::glGetError(observer);
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
