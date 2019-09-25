/*
 * Copyright (C) 2018 Google Inc.
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

#ifndef GAPIR_REPLAY_CONNECTION_H
#define GAPIR_REPLAY_CONNECTION_H

#include "core/cc/semaphore.h"
#include "replay_service.h"
#include "resource.h"

#include <deque>
#include <memory>
#include <string>
#include <thread>

namespace grpc {
template <typename RES, typename REQ>
class ServerReaderWriter;
}

namespace replay_service {
class ReplayRequest;
class ReplayResponse;
}  // namespace replay_service

namespace gapir {

using ReplayGrpcStream =
    grpc::ServerReaderWriter<replay_service::ReplayResponse,
                             replay_service::ReplayRequest>;

// GrpcReplayService implements ReplayService interface for GRPC connection.
// It represents a source of all replay data which is based on grpc stream.
class GrpcReplayService : public ReplayService {
 public:
  // Creates a GrpcReplayService from the gRPC stream. If the gRPC stream is
  // nullptr, returns nullptr
  static std::unique_ptr<GrpcReplayService> create(ReplayGrpcStream* stream) {
    if (stream == nullptr) {
      return nullptr;
    }
    return std::unique_ptr<GrpcReplayService>(new GrpcReplayService(stream));
  }

  virtual ~GrpcReplayService() override {
    if (mGrpcStream != nullptr) {
      this->sendReplayFinished();
      mCommunicationThread.join();
    }
  }

  GrpcReplayService(const GrpcReplayService&) = delete;
  GrpcReplayService(GrpcReplayService&&) = delete;
  GrpcReplayService& operator=(const GrpcReplayService&) = delete;
  GrpcReplayService& operator=(GrpcReplayService&&) = delete;

  // Sends PayloadRequest and returns the received Payload. Returns nullptr in
  // case of error.
  std::unique_ptr<ReplayService::Payload> getPayload(
      const std::string& payload) override;
  // Sends ResourceRequest and returns the received Resources. Returns nullptr
  // in case of error.
  std::unique_ptr<ReplayService::Resources> getResources(
      const Resource* resource, size_t resCount) override;

  std::unique_ptr<ReplayService::FenceReady> getFenceReady(
      const uint32_t& id) override;

  // Sends ReplayFinished signal. Returns true if succeeded, otherwise returns
  // false.
  bool sendReplayFinished() override;
  // Sends crash dump. Returns true if succeeded, otherwise returns false.
  bool sendCrashDump(const std::string& filepath, const void* crash_data,
                     uint32_t crash_size) override;
  // Sends post data. Returns true if succeeded, otherwise returns false.
  bool sendPosts(std::unique_ptr<ReplayService::Posts> posts) override;
  // Sends error message notification. Returns true if succeeded, otherwise
  // returns false.
  bool sendErrorMsg(uint64_t seq_num, uint32_t severity, uint32_t api_index,
                    uint64_t label, const std::string& msg, const void* data,
                    uint32_t data_size) override;
  // Sends replay status notification. Returns true if succeeded, otherwise
  // returns false.
  bool sendReplayStatus(uint64_t label, uint32_t total_instrs,
                        uint32_t finished_instrs) override;
  // Sends notification. Returns true if succeeded, otherwise returns false.
  bool sendNotificationData(uint64_t, uint64_t, const void*, uint32_t) override;

  std::unique_ptr<replay_service::ReplayRequest> getReplayRequest() override;

  void primeState(std::string prerun_id, std::string cleanup_id);

 protected:
  GrpcReplayService(ReplayGrpcStream* stream) : mGrpcStream(stream) {
    if (mGrpcStream != nullptr) {
      mCommunicationThread = std::thread(&handleCommunication, this);
    }
  }

  std::unique_ptr<replay_service::ReplayRequest> getNonReplayRequest();

  static void handleCommunication(GrpcReplayService* _service);

 private:
  // The gRPC stream connection.
  ReplayGrpcStream* mGrpcStream;

  std::mutex mCommunicationLock;
  core::Semaphore mRequestSem;
  core::Semaphore mDataSem;
  std::deque<std::unique_ptr<replay_service::ReplayRequest>> mDeferredRequests;
  std::deque<std::unique_ptr<replay_service::ReplayRequest>> mDeferredData;
  std::thread mCommunicationThread;
};
}  // namespace gapir

#endif  // GAPIR_REPLAY_CONNECTION_H
