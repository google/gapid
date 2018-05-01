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

#include <grpc++/grpc++.h>
#include <functional>
#include <string>
#include <tuple>
#include <vector>

#include "gapir/service/service.grpc.pb.h"

namespace gapir {

using ReplayGrpcStream =
    grpc::ServerReaderWriter<service::ReplayResponse, service::ReplayRequest>;
using PayloadHandler = std::function<bool(const service::Payload&)>;
using ResourcesHandler = std::function<bool(const service::Resources&)>;

class ReplayConnection {
 public:
  // ResourceRequest is a wraper class of service::ResourceRequest
  class ResourceRequest {
   public:
    static std::unique_ptr<ResourceRequest> create() {
      return std::unique_ptr<ResourceRequest>(new ResourceRequest());
    }

    ~ResourceRequest() {
      if (mProtoResourceRequest != nullptr) {
        delete mProtoResourceRequest;
      }
    }

    ResourceRequest(const ResourceRequest&) = delete;
    ResourceRequest(ResourceRequest&&) = delete;
    ResourceRequest& operator=(const ResourceRequest&) = delete;
    ResourceRequest& operator=(ResourceRequest&&) = delete;

    inline bool append(const std::string& id, size_t size) {
      if (mProtoResourceRequest == nullptr) {
        return false;
      }
      mProtoResourceRequest->add_ids(id);
      mProtoResourceRequest->set_expected_total_size(
          mProtoResourceRequest->expected_total_size() + size);
      return true;
    }

    inline service::ResourceRequest* release_to_proto() {
      auto ptr = mProtoResourceRequest;
      mProtoResourceRequest = nullptr;
      return ptr;
    }

   private:
    ResourceRequest() : mProtoResourceRequest(new service::ResourceRequest()) {}

    service::ResourceRequest* mProtoResourceRequest;
  };

  // Posts is a wrapper class of service::PostData
  class Posts {
    public:
    static std::unique_ptr<Posts> create() {
      return std::unique_ptr<Posts>(new Posts());
    }

    ~Posts() {
      if (mProtoPostData != nullptr) {
        delete mProtoPostData;
      }
    }

    Posts(const Posts&) = delete;
    Posts(Posts&&) = delete;
    Posts& operator=(const Posts&) = delete;
    Posts& operator=(Posts&&) = delete;

    inline bool append(const void* data, size_t size) {
      if (mProtoPostData == nullptr) {
        return false;
      }
      mProtoPostData->add_posts(data, size);
      return true;
    }

    inline service::PostData* release_to_proto() {
      auto ptr = mProtoPostData;
      mProtoPostData = nullptr;
      return ptr;
    }

    private:
    Posts() : mProtoPostData(new service::PostData()) {}

    service::PostData* mProtoPostData;
  };

  // Payload is a wraper class of service::Payload
  class Payload {
   public:
    static std::unique_ptr<Payload> get(ReplayGrpcStream* stream) {
      std::unique_ptr<service::ReplayRequest> req(new service::ReplayRequest());
      if (!stream->Read(req.get())) {
        return nullptr;
      }
      if (req->req_case() != service::ReplayRequest::kPayload) {
        return nullptr;
      }
      return std::unique_ptr<Payload>(new Payload(std::move(req)));
    }

    ~Payload() = default;
    Payload(const Payload&) = delete;
    Payload(Payload&&) = delete;
    Payload& operator=(const Payload&) = delete;
    Payload& operator=(Payload&&) = delete;

    // TODO: accessors to help pulling content from the proto message.
    inline uint32_t stack_size() const {
      return mProtoReplayRequest->payload().stack_size();
    }

    inline uint32_t volatile_memory_size() const {
      return mProtoReplayRequest->payload().volatile_memory_size();
    }

    inline size_t constants_size() const {
      return mProtoReplayRequest->payload().constants().size();
    }
    inline const void* constants_data() const {
      return mProtoReplayRequest->payload().constants().data();
    }

    inline size_t resource_info_count() const {
      return mProtoReplayRequest->payload().resources_size();
    }
    inline const std::string resource_id(int index) const {
      return mProtoReplayRequest->payload().resources(index).id();
    }
    inline uint32_t resource_size(int index) const {
      return mProtoReplayRequest->payload().resources(index).size();
    }

    inline size_t opcodes_size() const {
      return mProtoReplayRequest->payload().opcodes().size();
    }
    inline const void* opcodes_data() const {
      return mProtoReplayRequest->payload().opcodes().data();
    }

   private:
    Payload(std::unique_ptr<service::ReplayRequest> req)
        : mProtoReplayRequest(std::move(req)) {}

    std::unique_ptr<service::ReplayRequest> mProtoReplayRequest;
  };

  // Resources is a wraper class of service::Resources
  class Resources {
   public:
    static std::unique_ptr<Resources> get(ReplayGrpcStream* stream) {
      std::unique_ptr<service::ReplayRequest> req(new service::ReplayRequest());
      if (!stream->Read(req.get())) {
        return nullptr;
      }
      if (req->req_case() != service::ReplayRequest::kResources) {
        return nullptr;
      }
      return std::unique_ptr<Resources>(new Resources(std::move(req)));
    }

    ~Resources() = default;
    Resources(const Resources&) = delete;
    Resources(Resources&&) = delete;
    Resources& operator=(const Resources&) = delete;
    Resources& operator=(Resources&&) = delete;

    inline size_t size() const {
      return mProtoReplayRequest->resources().data().size();
    }
    inline const void* data() const {
      return mProtoReplayRequest->resources().data().data();
    }

   private:
    Resources(std::unique_ptr<service::ReplayRequest> req)
        : mProtoReplayRequest(std::move(req)) {}

    std::unique_ptr<service::ReplayRequest> mProtoReplayRequest;
  };

  // Declaration of class ReplayConnection
  static std::unique_ptr<ReplayConnection> create(ReplayGrpcStream* stream) {
    if (stream == nullptr) {
      return nullptr;
    }
    return std::unique_ptr<ReplayConnection>(new ReplayConnection(stream));
  }

  ~ReplayConnection() = default;

  ReplayConnection(const ReplayConnection&) = delete;
  ReplayConnection(ReplayConnection&&) = delete;
  ReplayConnection& operator=(const ReplayConnection&) = delete;
  ReplayConnection& operator=(ReplayConnection&&) = delete;

  std::unique_ptr<Payload> getPayload();
  std::unique_ptr<Resources> getResources(std::unique_ptr<ResourceRequest> req);

  bool sendReplayFinished();
  bool sendCrashDump(const std::string& filepath, const void* crash_data,
                     uint32_t crash_size);
  bool sendPostData(std::unique_ptr<Posts> posts);
  bool sendNotification(uint64_t id, uint32_t api_index, uint64_t label,
                        const std::string& msg, const void* data,
                        uint32_t data_size);

 private:
  ReplayConnection(ReplayGrpcStream* stream) : mGrpcStream(*stream) {}

  ReplayGrpcStream& mGrpcStream;
};
}  // namespace gapir

#endif  // GAPIR_REPLAY_CONNECTION_H
