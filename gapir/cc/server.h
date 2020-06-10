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

#ifndef GAPIR_SERVER_H
#define GAPIR_SERVER_H

#include "gapir/replay_service/service.grpc.pb.h"
#include "gapir/replay_service/service.pb.h"
#include "grpc_replay_service.h"

#include "core/cc/log.h"

#include <grpc++/grpc++.h>

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <string>
#include <thread>

namespace {
// Duration for which pending RPC can cleanly terminate upon a shutdown.
const std::chrono::seconds kShutdownTimeout(1);
}  // namespace

namespace gapir {

using ReplayHandler = std::function<void(GrpcReplayService*)>;
using WatchDogFeeder = std::function<void()>;

class Server;

// Implements the protobuf+grpc generated GAPIR replay service.
class GapirServiceImpl final : public replay_service::Gapir::Service {
  friend Server;

 public:
  static const char kAuthTokenMetaDataName[];

  ~GapirServiceImpl() = default;

  GapirServiceImpl(const GapirServiceImpl&) = delete;
  GapirServiceImpl(GapirServiceImpl&&) = delete;
  GapirServiceImpl& operator=(const GapirServiceImpl&) = delete;
  GapirServiceImpl& operator=(GapirServiceImpl&&) = delete;

  grpc::Status Replay(
      grpc::ServerContext* context,
      grpc::ServerReaderWriter<replay_service::ReplayResponse,
                               replay_service::ReplayRequest>* stream) override;
  grpc::Status Ping(grpc::ServerContext* context,
                    const replay_service::PingRequest*,
                    replay_service::PingResponse* res) override;
  grpc::Status Shutdown(grpc::ServerContext* context,
                        const replay_service::ShutdownRequest*,
                        replay_service::ShutdownResponse*) override;

 private:
  GapirServiceImpl(const char* authToken, ReplayHandler handle_replay,
                   WatchDogFeeder feed_watchdog)
      : mHandleReplay(handle_replay),
        mFeedWatchDog(feed_watchdog),
        mServer(nullptr),
        mAuthToken(authToken == nullptr ? "" : authToken) {}

  // The thread-safe callback to process replay requests.
  ReplayHandler mHandleReplay;
  // The callback to feed idle time watch dog, it is to be called for every
  // valid Ping request.
  WatchDogFeeder mFeedWatchDog;
  // The server which is running this service implementation.
  Server* mServer;
  // The authentication token to be used for checking every request.
  std::string mAuthToken;
};

// Server setups a listening port and processes the replay request sent from
// GAPIS with a given replay handler. It also takes care of closing itself when
// server is in idle state for a speicifc length of time, and setting
// authentication token to check for all the received packets.
class Server {
 public:
  // Creates and starts a GAPIR replay server, returns the created server.
  // In case of any failure, returns nullptr. The server will be listening
  // the port specified by the given |uri|. If an non-null |authToken| is given,
  // it will be used in checking the metadata of the communication package
  // between GAPIS and GAPIR. If the given |idleTimeoutSec| is larger than 0,
  // the server will only be alive for |idelTimeSec| seconds since the last
  // Ping request. If |idleTimeoutSec| is 0 or minus 0, the server will be kept
  // alive. The callback |handleReplay| will be called whenever a replay request
  // package with replay ID is received.
  static std::unique_ptr<Server> createAndStart(const char* uri,
                                                const char* authToken,
                                                int idleTimeoutSec,
                                                ReplayHandler handleReplay);

  ~Server() {
    if (mIdleTimeoutCloser != nullptr) {
      mIdleTimeoutCloser->join();
    }
  };

  Server(const Server&) = delete;
  Server(Server&&) = delete;
  Server& operator=(const Server&) = delete;
  Server& operator=(Server&&) = delete;

  // Wait blocks until the server shuts down.
  void wait() { mGrpcServer->Wait(); }

  // Shuts down the server, give it a little time to finish RPC processing.
  void shutdown() {
    if (!mShuttingDown.exchange(true)) {
      std::thread([this] {
        auto deadline = std::chrono::system_clock::now() + kShutdownTimeout;
        this->mGrpcServer->Shutdown(deadline);
      })
          .detach();
    }
  }

 private:
  Server(const char* authToken, int idleTimeoutSec,
         ReplayHandler handle_replay);

  // Seconds since the last ping request.
  std::atomic<int> mSecCounter;
  // A flag to specify the server is to be shut down.
  std::atomic<bool> mShuttingDown;
  // The gRPC server.
  std::unique_ptr<grpc::Server> mGrpcServer;
  // The GAPIR service implementation.
  std::unique_ptr<GapirServiceImpl> mServiceImpl;
  // A separated thread to close server for idle time out.
  std::unique_ptr<std::thread> mIdleTimeoutCloser;
};

}  // namespace gapir

#endif  // GAPIR_SERVER_H
