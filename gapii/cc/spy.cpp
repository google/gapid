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
#include "protocol.h"

#include "gapil/runtime/cc/runtime.h"

#include "gapii/cc/spy.h"

#include "core/cc/debugger.h"
#include "core/cc/lock.h"
#include "core/cc/log.h"
#include "core/cc/null_writer.h"
#include "core/cc/process_name.h"
#include "core/cc/target.h"
#include "core/cc/timer.h"
#include "core/os/device/deviceinfo/cc/query.h"

#include "gapis/capture/capture.pb.h"
#include "gapis/memory/memory_pb/memory.pb.h"

#include <cstdlib>
#include <memory>
#include <sstream>
#include <thread>
#include <vector>

#if TARGET_OS == GAPID_OS_ANDROID

#include <jni.h>
#include <sys/prctl.h>

extern "C" jint JNI_OnLoad(JavaVM* vm, void* reserved) {
  GAPID_INFO("JNI_OnLoad() was called. vm = %p", vm);
  gapii::Spy::get();  // Construct the spy.
  return JNI_VERSION_1_6;
}
#endif  // TARGET_OS == GAPID_OS_ANDROID

namespace {

const uint32_t kMaxFramebufferObservationWidth = 3840;
const uint32_t kMaxFramebufferObservationHeight = 2560;

const int32_t kSuspendIndefinitely = -1;

thread_local gapii::CallObserver* gContext = nullptr;

}  // anonymous namespace

namespace gapii {

struct spy_creator {
  spy_creator() {
    GAPID_LOGGER_INIT(LOG_LEVEL_INFO, "gapii", nullptr);
    GAPID_INFO("Constructing spy...");
    m_spy.reset(new Spy());
  }
  std::unique_ptr<gapii::Spy> m_spy;
};

Spy* Spy::get() {
  static spy_creator creator;
  return creator.m_spy.get();
}

Spy::Spy()
    : mNumFrames(0),
      mSuspendCaptureFrames(0),
      mCaptureFrames(0),
      mNumDraws(0),
      mNumDrawsPerFrame(0),
      mObserveFrameFrequency(0),
      mObserveDrawFrequency(0),
      mNestedFrameStart(0),
      mNestedFrameEnd(0),
      mFrameNumber(0) {
  bool this_executable = true;
  char* pn = getenv("GAPID_CAPTURE_PROCESS_NAME");
  if (pn) {
    auto proc_name = core::get_process_name();
    this_executable = (proc_name == pn);
  }
  if (this_executable) {
#if TARGET_OS == GAPID_OS_ANDROID
    // Use a "localabstract" pipe on Android to prevent depending on the traced
    // application having the INTERNET permission set, required for opening and
    // listening on a TCP socket.
    std::string pipe = "gapii";
    char* envPipe = getenv("GAPII_PIPE_NAME");
    if (envPipe != nullptr) {
      pipe = envPipe;
    }
    mConnection = ConnectionStream::listenPipe(pipe.c_str(), true);
#else                                           // TARGET_OS
    mConnection = ConnectionStream::listenSocket("127.0.0.1", "9286");
#endif                                          // TARGET_OS
    if (mConnection->write("gapii", 5) != 5) {  // handshake magic
      GAPID_FATAL("Couldn't send handshake magic");
    }

    GAPID_INFO("Connection made");
  }
  ConnectionHeader header;
  if (this_executable) {
    if (!header.read(mConnection.get())) {
      GAPID_FATAL("Failed to read connection header");
    }
  } else {
    header.read_fake();
  }

  GAPID_INFO("Connection header read");

  mObserveFrameFrequency = header.mObserveFrameFrequency;
  mObserveDrawFrequency = header.mObserveDrawFrequency;
  SpyBase::mHideUnknownExtensions =
      (header.mFlags & ConnectionHeader::FLAG_HIDE_UNKNOWN_EXTENSIONS) != 0;
  SpyBase::mDisableCoherentMemoryTracker =
      (header.mFlags &
       ConnectionHeader::FLAG_DISABLE_COHERENT_MEMORY_TRACKER) != 0;
  set_record_timestamps(
      0 != (header.mFlags & ConnectionHeader::FLAG_STORE_TIMESTAMPS));

  mSuspendCaptureFrames = (header.mFlags & ConnectionHeader::FLAG_DEFER_START)
                              ? kSuspendIndefinitely
                              : header.mStartFrame;
  mCaptureFrames = header.mNumFrames;

  set_valid_apis(header.mAPIs);
  GAPID_ERROR("APIS %08x", header.mAPIs);
  GAPID_INFO("GAPII connection established. Settings:");
  GAPID_INFO("Observe framebuffer every %d frames", mObserveFrameFrequency);
  GAPID_INFO("Observe framebuffer every %d draws", mObserveDrawFrequency);
  GAPID_INFO("Hide unknown extensions: %s",
             mHideUnknownExtensions ? "true" : "false");

  if (this_executable) {
    mEncoder = gapii::PackEncoder::create(
        mConnection, header.mFlags & ConnectionHeader::FLAG_NO_BUFFER);
  } else {
    auto nw = std::make_shared<core::NullWriter>();
    mEncoder = gapii::PackEncoder::create(nw, false);
  }

  // writeHeader needs to come before the installer is created as the
  // deviceinfo queries want to call into EGL / GL commands which will be
  // patched.
  query::Option query_opt;
  std::string error;
  SpyBase::set_device_instance(query::getDeviceInstance(query_opt, &error));
  if (!error.empty()) {
    GAPID_ERROR("Failed to get device info: %s", error.c_str());
  }

  SpyBase::set_current_abi(query::currentABI());
  if (!SpyBase::writeHeader()) {
    GAPID_ERROR("Failed at writing trace header.");
  }

  // Waiting for debugger must come after we sent back the trace header,
  // otherwise GAPIS thinks GAPII had an issue at init time.
  if (header.mFlags & ConnectionHeader::FLAG_WAIT_FOR_DEBUGGER) {
    GAPID_INFO("Wait for debugger");
    core::Debugger::waitForAttach();
  }

  auto context = enter("init", 0);
  VulkanSpy::init();
  SpyBase::init(context);
  exit();

  if (this_executable) {
    mMessageReceiverJob =
        std::unique_ptr<core::AsyncJob>(new core::AsyncJob([this]() {
          uint8_t buffer[protocol::kHeaderSize] = {};
          uint64_t count;
          do {
            count = mConnection->read(&buffer[0], protocol::kHeaderSize);
            if (count == protocol::kHeaderSize) {
              switch (static_cast<protocol::MessageType>(buffer[0])) {
                case protocol::MessageType::kStartTrace:
                  GAPID_DEBUG("Received start trace message");
                  if (is_suspended()) {
                    GAPID_DEBUG("Starting capture");
                    mSuspendCaptureFrames = 1;
                  }
                  break;
                case protocol::MessageType::kEndTrace:
                  GAPID_DEBUG("Received end trace message");
                  if (!is_suspended()) {
                    GAPID_DEBUG("Ending capture");
                    // If app uses frame boundaries, end capture at next one
                    // otherwise at next traced graphics API call
                    const bool usesFrameBounds = mFrameNumber > 0u;
                    mCaptureFrames = usesFrameBounds ? 1 : -1;
                  }
                  break;
                default:
                  GAPID_WARNING("Invalid message type: %u", buffer[0]);
                  break;
              }
            } else if (count > 0u) {
              GAPID_WARNING("Received unexpected data");
            }
          } while (count == protocol::kHeaderSize);
        }));
  }
  set_suspended(mSuspendCaptureFrames != 0);
  set_observing(mObserveFrameFrequency != 0 || mObserveDrawFrequency != 0);
}

Spy::~Spy() {
  mCaptureFrames = -1;
  endTraceIfRequested();
}

CallObserver* Spy::enter(const char* name, uint32_t api) {
  lock();
  auto ctx = new CallObserver(this, gContext, api);
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

void Spy::endTraceIfRequested() {
  if (!is_suspended() && mCaptureFrames < 0) {
    GAPID_DEBUG("Ended capture");
    mEncoder->flush();
    // Error messages can be transferred any time during the trace, e.g.:
    // auto err = protocol::createError("end of the world");
    // mConnection->write(err.data(), err.size());
    auto msg = protocol::createHeader(protocol::MessageType::kEndTrace);
    mConnection->write(msg.data(), msg.size());
    // allow some time for the message to arrive
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    mConnection->close();
    set_suspended(true);
  }
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
  GAPID_ASSERT(mNestedFrameEnd < 2048);
  if (++mNestedFrameStart > 1) {
    return;
  }
  if (is_suspended()) {
    return;
  }
  if (mObserveFrameFrequency != 0 &&
      (mNumFrames % mObserveFrameFrequency == 0)) {
    GAPID_DEBUG("Observe framebuffer after frame %d", mNumFrames);
    observeFramebuffer(observer, api);
  }
  GAPID_DEBUG("NumFrames:%d NumDraws:%d NumDrawsPerFrame:%d", mNumFrames,
              mNumDraws, mNumDrawsPerFrame);
  mNumFrames++;
  mNumDrawsPerFrame = 0;
}

void Spy::saveInitialState() {
  GAPID_INFO("Saving initial state");

  set_recording_state(true);
  if (should_record_timestamps()) {
    capture::TraceMessage timestamp;
    timestamp.set_timestamp(core::GetNanoseconds());
    timestamp.set_message("State serialization started");
    mEncoder->object(&timestamp);
  }

  saveInitialStateForApi<VulkanSpy>("vulkan-initial-state");
  if (should_record_timestamps()) {
    capture::TraceMessage timestamp;
    timestamp.set_timestamp(core::GetNanoseconds());
    timestamp.set_message("State serialization finished");
    mEncoder->object(&timestamp);
  }
  set_recording_state(false);
}

template <typename T>
void Spy::saveInitialStateForApi(const char* name) {
  if (should_trace(T::kApiIndex)) {
    auto observer = enter(name, T::kApiIndex);
    StateSerializer serializer(this, T::kApiIndex, observer);

    serializer.encodeState(
        T::mState, [this](StateSerializer* s) { T::serializeGPUBuffers(s); });
    exit();
  }
}

void Spy::onPostFrameBoundary(bool isStartOfFrame) {
  mFrameNumber++;
  if (should_record_timestamps()) {
    std::stringstream fn;
    fn << "Frame Number: " << mFrameNumber;
    capture::TraceMessage timestamp;
    timestamp.set_timestamp(core::GetNanoseconds());
    timestamp.set_message(fn.str());
    mEncoder->object(&timestamp);
  }

  if (is_suspended()) {
    if (mSuspendCaptureFrames > 0) {
      if (--mSuspendCaptureFrames == 0) {
        GAPID_DEBUG("Started capture");
        // We must change suspended state BEFORE releasing the Spy lock with
        // exit(), because the suspended state affects concurrent CallObservers.
        set_suspended(false);
        exit();
        saveInitialState();
        enter("RecreateState", 2);
      }
    }
  } else {
    if (mCaptureFrames > 0) {
      if (--mCaptureFrames == 0) {
        mCaptureFrames = -1;
        endTraceIfRequested();
      }
    }
  }
}

void Spy::onPostStartOfFrame() {
  GAPID_ASSERT(mNestedFrameStart > 0);
  if (--mNestedFrameStart == 0) {
    onPostFrameBoundary(true);
  }
}

void Spy::onPreEndOfFrame(CallObserver* observer, uint8_t api) {
  GAPID_ASSERT(mNestedFrameEnd < 2048);
  if (++mNestedFrameEnd > 1) {
    return;
  }
  if (is_suspended()) {
    return;
  }
  if (mObserveFrameFrequency != 0 &&
      (mNumFrames % mObserveFrameFrequency == 0)) {
    GAPID_DEBUG("Observe framebuffer after frame %d", mNumFrames);
    observeFramebuffer(observer, api);
  }
  GAPID_DEBUG("NumFrames:%d NumDraws:%d NumDrawsPerFrame:%d", mNumFrames,
              mNumDraws, mNumDrawsPerFrame);
  mNumFrames++;
  mNumDrawsPerFrame = 0;
}

void Spy::onPostEndOfFrame() {
  GAPID_ASSERT(mNestedFrameEnd > 0);
  if (--mNestedFrameEnd == 0) {
    onPostFrameBoundary(false);
  }
}

static bool downsamplePixels(const std::vector<uint8_t>& srcData, uint32_t srcW,
                             uint32_t srcH, std::vector<uint8_t>* outData,
                             uint32_t* outW, uint32_t* outH, uint32_t maxW,
                             uint32_t maxH) {
  // Calculate the minimal scaling factor as integer fraction.
  uint32_t mul = 1;
  uint32_t div = 1;
  if (mul * srcW > maxW * div) {  // if mul/div > maxW/srcW
    mul = maxW;
    div = srcW;
  }
  if (mul * srcH > maxH * div) {  // if mul/div > maxH/srcH
    mul = maxH;
    div = srcH;
  }

  // Calculate the final dimensions (round up) and allocate new buffer.
  uint32_t dstW = (srcW * mul + div - 1) / div;
  uint32_t dstH = (srcH * mul + div - 1) / div;
  outData->reserve(dstW * dstH * 4);

  // Downsample the image by averaging the colours of neighbouring pixels.
  for (uint32_t srcY = 0, y = 0, dstY = 0; dstY < dstH; srcY = y, dstY++) {
    for (uint32_t srcX = 0, x = 0, dstX = 0; dstX < dstW; srcX = x, dstX++) {
      uint32_t r = 0, g = 0, b = 0, a = 0, n = 0;
      // We need to loop over srcX/srcY ranges several times, so we keep them in
      // x/y, and we update srcX/srcY to the last x/y only once we are done with
      // the pixel.
      for (y = srcY; y * dstH < (dstY + 1) * srcH;
           y++) {  // while y*yScale < dstY+1
        const uint8_t* src = &srcData[(srcX + y * srcW) * 4];
        for (x = srcX; x * dstW < (dstX + 1) * srcW;
             x++) {  // while x*xScale < dstX+1
          r += *(src++);
          g += *(src++);
          b += *(src++);
          a += *(src++);
          n += 1;
        }
      }
      outData->push_back(r / n);
      outData->push_back(g / n);
      outData->push_back(b / n);
      outData->push_back(a / n);
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
  switch (api) {
    case VulkanSpy::kApiIndex:
      if (!VulkanSpy::observeFramebuffer(observer, &w, &h, &data)) {
        return;
      }
      break;
  }

  uint32_t downsampledW, downsampledH;
  std::vector<uint8_t> downsampledData;
  if (downsamplePixels(data, w, h, &downsampledData, &downsampledW,
                       &downsampledH, kMaxFramebufferObservationWidth,
                       kMaxFramebufferObservationHeight)) {
    capture::FramebufferObservation observation;
    observation.set_original_width(w);
    observation.set_original_height(h);
    observation.set_data_width(downsampledW);
    observation.set_data_height(downsampledH);
    observation.set_data(downsampledData.data(), downsampledData.size());
    observer->encode_message(&observation);
  }
}

void Spy::onPostFence(CallObserver* observer) {
  // TODO: consider removing, this only did GLES error faking
}

}  // namespace gapii
