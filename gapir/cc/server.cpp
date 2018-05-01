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

#include <functional>
#include <limits>
#include <vector>
#include <grpc++/grpc++.h>

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

Status GapirServiceImpl::Replay(ServerContext* context, ReplayStream* stream) {
  GAPID_INFO("In GapirServiceImpl::Replay");
  service::ReplayRequest req;
  while (stream->Read(&req)) {
    GAPID_INFO("In GapirServiceImpl::Replay while loop");
    if (req.req_case() == service::ReplayRequest::kReplayId) {
      GAPID_INFO("In GapirServiceImpl::Replay ReplayRequest is a replay ID: %s", req.replay_id().c_str());
      std::unique_ptr<ReplayConnection> replay_conn =
          ReplayConnection::create(stream);
      GAPID_INFO("In GapirServiceImpl::Replay ReplayRequest is a replay ID: replay_conn: %p", replay_conn.get());
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
  GAPID_INFO("In GapirServiceImpl::Ping");
  res->set_pong("PONG");
  return Status::OK;
}

Status GapirServiceImpl::Shutdown(ServerContext* context,
                                  const service::ShutdownRequest*,
                                  service::ShutdownResponse*) {
  GAPID_INFO("In GapirServiceImpl::Shutdown");
  if (mGrpcServer != nullptr) {
    mGrpcServer->Shutdown();
  }
  return Status::OK;
}

Server::Server(std::string uri, ReplayHandler handle_replay)
    : mGrpcServer(nullptr),
      mServiceImpl(std::unique_ptr<GapirServiceImpl>(
          new GapirServiceImpl(handle_replay))),
      mUri(uri) {}

std::unique_ptr<Server> Server::createAndStart(std::string uri,
                                               ReplayHandler handle_replay) {
  std::unique_ptr<Server> server(new Server(uri, handle_replay));
  GAPID_INFO("server pointer created");
  ServerBuilder builder;
  builder.SetMaxSendMessageSize(std::numeric_limits<int>::max());
  builder.SetMaxReceiveMessageSize(std::numeric_limits<int>::max());
  builder.AddListeningPort(server->mUri, grpc::InsecureServerCredentials());
  GAPID_INFO("listening port added on: %s", server->mUri.c_str());
  builder.RegisterService(server->mServiceImpl.get());
  GAPID_INFO("service registered");
  server->mGrpcServer = builder.BuildAndStart();
  GAPID_INFO("grpc server built and started: %p", server->mGrpcServer.get());
  server->mServiceImpl->mGrpcServer = server->mGrpcServer.get();
  GAPID_INFO("grpc server assigned to service impl");
  return server;
}

}  // namespace gapir
