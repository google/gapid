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

#ifndef GAPID_SERVER_CONNECTION_H
#define GAPID_SERVER_CONNECTION_H

#include <grpc++/grpc++.h>
#include <functional>
#include <memory>
#include <string>

#include "core/cc/log.h"
#include "gapir/service/service.grpc.pb.h"
#include "replay_connection.h"

namespace gapir {

using ReplayHandler =
    std::function<void(ReplayConnection*, const std::string& replay_id)>;

class Server;

class GapirServiceImpl final : public service::Gapir::Service {
  friend Server;
 public:
  ~GapirServiceImpl() = default;

  GapirServiceImpl(const GapirServiceImpl&) = delete;
  GapirServiceImpl(GapirServiceImpl&&) = delete;
  GapirServiceImpl& operator=(const GapirServiceImpl&) = delete;
  GapirServiceImpl& operator=(GapirServiceImpl&&) = delete;

  grpc::Status Replay(
      grpc::ServerContext* context,
      grpc::ServerReaderWriter<service::ReplayResponse, service::ReplayRequest>*
          stream) override;
  grpc::Status Ping(grpc::ServerContext* context, const service::PingRequest*,
                    service::PingResponse* res) override;
  grpc::Status Shutdown(grpc::ServerContext* context, const service::ShutdownRequest*,
                        service::ShutdownResponse*) override;

 private:
  GapirServiceImpl(const char* authToken, ReplayHandler handle_replay)
      : mHandleReplay(handle_replay), mGrpcServer(nullptr), mAuthToken(authToken == nullptr ? "" : authToken) {}

  ReplayHandler mHandleReplay;
  grpc::Server* mGrpcServer;
  std::string mAuthToken;

  static const char kAuthTokenMetaDataName[];
};

class Server {
 public:
  static std::unique_ptr<Server> createAndStart(const char* uri, const char* authToken, ReplayHandler handleReplay);

  ~Server() = default;

  Server(const Server&) = delete;
  Server(Server&&) = delete;
  Server& operator=(const Server&) = delete;
  Server& operator=(Server&&) = delete;

  void wait() { GAPID_INFO("calling grpc server wait"); mGrpcServer->Wait(); }

  void shutdown() { mGrpcServer->Shutdown(); }

 private:
  Server(const char* authToken, ReplayHandler handle_replay);

  std::unique_ptr<grpc::Server> mGrpcServer;
  std::unique_ptr<GapirServiceImpl> mServiceImpl;
};

}  // namespace gapir

#endif  // GAPID_SERVER_CONNECTION_H
