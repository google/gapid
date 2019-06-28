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

#include "server.h"

#include <grpc++/grpc++.h>
#include <string.h>

#include <functional>
#include <limits>
#include <vector>

#include "core/cc/log.h"
#include "gapir/replay_service/service.grpc.pb.h"
#include "grpc_replay_service.h"

namespace gapir {

using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerReaderWriter;
using grpc::Status;
using ReplayStream = grpc::ServerReaderWriter<replay_service::ReplayResponse,
                                              replay_service::ReplayRequest>;

// The key of the metadata value that contains authentication token. This is
// common knowledge shared between GAPIR client (which is GAPIS) and GAPIR
// server (which is GAPIR device)
const char GapirServiceImpl::kAuthTokenMetaDataName[] = "gapir-auth-token";

namespace {
bool CheckAuthToken(ServerContext* context, const std::string& expected) {
  if (expected.length() > 0) {
    auto auth_md = context->client_metadata().find(
        GapirServiceImpl::kAuthTokenMetaDataName);
    if (auth_md == context->client_metadata().end()) {
      return false;
    }
    if (strncmp(auth_md->second.data(), expected.data(), expected.size())) {
      return false;
    }
  }
  return true;
}
}  // namespace

Status GapirServiceImpl::Replay(ServerContext* context, ReplayStream* stream) {
  // Check the metadata for the authentication token
  if (!CheckAuthToken(context, mAuthToken)) {
    return Status(grpc::StatusCode::UNAUTHENTICATED,
                  grpc::string("Invalid auth token"));
  }

  std::unique_ptr<GrpcReplayService> replay_conn =
      GrpcReplayService::create(stream);
  if (replay_conn != nullptr) {
    mHandleReplay(replay_conn.get());
  }

  return Status::OK;
}

Status GapirServiceImpl::Ping(ServerContext* context,
                              const replay_service::PingRequest*,
                              replay_service::PingResponse* res) {
  if (!CheckAuthToken(context, mAuthToken)) {
    return Status(grpc::StatusCode::UNAUTHENTICATED,
                  grpc::string("Invalid auth token"));
  }
  mFeedWatchDog();
  return Status::OK;
}

Status GapirServiceImpl::Shutdown(ServerContext* context,
                                  const replay_service::ShutdownRequest*,
                                  replay_service::ShutdownResponse*) {
  if (!CheckAuthToken(context, mAuthToken)) {
    return Status(grpc::StatusCode::UNAUTHENTICATED,
                  grpc::string("Invalid auth token"));
  }
  mServer->shutdown();
  return Status::OK;
}

Server::Server(const char* authToken, int idleTimeoutSec,
               ReplayHandler handle_replay)
    : mSecCounter(0),
      mShuttingDown(false),
      mGrpcServer(nullptr),
      mServiceImpl(new GapirServiceImpl(
          authToken, handle_replay, [this]() { this->mSecCounter.store(0); })),
      mIdleTimeoutCloser(
          idleTimeoutSec > 0 ? new std::thread([idleTimeoutSec, this] {
            while (idleTimeoutSec > this->mSecCounter.fetch_add(1) &&
                   !this->mShuttingDown.load()) {
              std::this_thread::sleep_for(std::chrono::seconds(1));
            }
            this->shutdown();
          })
                             : nullptr) {}

std::unique_ptr<Server> Server::createAndStart(const char* uri,
                                               const char* authToken,
                                               int idleTimeoutSec,
                                               ReplayHandler handle_replay) {
  std::unique_ptr<Server> server(
      new Server(authToken, idleTimeoutSec, handle_replay));
  ServerBuilder builder;
  builder.SetMaxSendMessageSize(std::numeric_limits<int>::max());
  builder.SetMaxReceiveMessageSize(std::numeric_limits<int>::max());
  builder.AddListeningPort(std::string(uri), grpc::InsecureServerCredentials());
  builder.RegisterService(server->mServiceImpl.get());
  auto grpcServer = builder.BuildAndStart();
  if (grpcServer == nullptr) {
    GAPID_ERROR("grpcServer is nullptr");
    return nullptr;
  }
  server->mGrpcServer = std::move(grpcServer);
  server->mServiceImpl->mServer = server.get();
  return server;
}

}  // namespace gapir
