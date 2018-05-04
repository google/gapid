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

#include "gapir/cc/server.h"

#include <grpc++/grpc++.h>
#include <string.h>
#include <functional>
#include <limits>
#include <vector>

#include "core/cc/log.h"
#include "gapir/service/service.grpc.pb.h"
#include "replay_connection.h"

namespace gapir {

using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerReaderWriter;
using grpc::Status;
using ReplayStream =
    grpc::ServerReaderWriter<service::ReplayResponse, service::ReplayRequest>;

// This is common knowledge shared with GAPIS.
const char GapirServiceImpl::kAuthTokenMetaDataName[] = "gapir-auth-token";

Status GapirServiceImpl::Replay(ServerContext* context, ReplayStream* stream) {
  // Check the metadata for the authentication token
  if (mAuthToken.length() > 0) {
    // need to check the metadata
    auto auth_md = context->client_metadata().find(kAuthTokenMetaDataName);
    if (auth_md == context->client_metadata().end()) {
      return Status(grpc::StatusCode::UNAUTHENTICATED,
                    grpc::string("No auth token"));
    }
    if (strncmp(auth_md->second.data(), mAuthToken.data(), mAuthToken.size())) {
      return Status(
          grpc::StatusCode::UNAUTHENTICATED,
          grpc::string("Invalid auth token: ") +
              grpc::string(auth_md->second.data(), auth_md->second.length()));
    }
  }
  service::ReplayRequest req;
  while (stream->Read(&req)) {
    if (req.req_case() == service::ReplayRequest::kReplayId) {
      std::unique_ptr<ReplayConnection> replay_conn =
          ReplayConnection::create(stream);
      if (replay_conn != nullptr) {
        mHandleReplay(replay_conn.get(), req.replay_id());
      }
    }
  }
  return Status::OK;
}

Status GapirServiceImpl::Ping(ServerContext* context,
                              const service::PingRequest*,
                              service::PingResponse* res) {
  res->set_pong("PONG");
  return Status::OK;
}

Status GapirServiceImpl::Shutdown(ServerContext* context,
                                  const service::ShutdownRequest*,
                                  service::ShutdownResponse*) {
  if (mGrpcServer != nullptr) {
    mGrpcServer->Shutdown();
  }
  return Status::OK;
}

Server::Server(const char* authToken, ReplayHandler handle_replay)
    : mGrpcServer(nullptr),
      mServiceImpl(std::unique_ptr<GapirServiceImpl>(
          new GapirServiceImpl(authToken, handle_replay))) {}

std::unique_ptr<Server> Server::createAndStart(const char* uri,
                                               const char* authToken,
                                               ReplayHandler handle_replay) {
  std::unique_ptr<Server> server(new Server(authToken, handle_replay));
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
  server->mServiceImpl->mGrpcServer = server->mGrpcServer.get();
  return server;
}

}  // namespace gapir
