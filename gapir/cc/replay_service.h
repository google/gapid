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

#ifndef GAPIR_REPLAY_SERVICE_H
#define GAPIR_REPLAY_SERVICE_H

#include "resource.h"

#include "gapir/replay_service/service.pb.h"

#include <functional>
#include <memory>
#include <string>
#include <tuple>
#include <vector>

namespace replay_service {
class Payload;
class Resources;
class FenceReady;
class ReplayRequest;
class PostData;
class ReplayResponse;
}  // namespace replay_service

namespace gapir {

// ReplayService is an interface that wraps all the server-client data
// communication methods needed for a replay.
class ReplayService {
 public:
  // Posts is a wrapper class of replay_service::PostData, it hides the
  // new/delete operations of the proto object from the outer code.
  class Posts {
   public:
    // Returns a new created empty Posts
    static std::unique_ptr<Posts> create() {
      return std::unique_ptr<Posts>(new Posts());
    }

    ~Posts();

    Posts(const Posts&) = delete;
    Posts(Posts&&) = delete;
    Posts& operator=(const Posts&) = delete;
    Posts& operator=(Posts&&) = delete;

    // Appends a new piece of post data to the posts.
    bool append(uint64_t id, const void* data, size_t size);
    // Gets the raw pointer of the internal proto object, and gives away the
    // ownership of the proto object.
    replay_service::PostData* release_to_proto();
    // Returns the number of pieces of post data.
    size_t piece_count() const;
    // Returns size in bytes of the 'index'th (starts from 0) piece of post
    // data.
    size_t piece_size(int index) const;
    // Returns a pointer to the data of the 'index'th (starts from 0) piece of
    // post data.
    const void* piece_data(int index) const;
    // Returns the ID of the 'index'th (starts from 0) piece of post data.
    uint64_t piece_id(int index) const;

   private:
    Posts();

    // The internal proto object.
    std::unique_ptr<replay_service::PostData> mProtoPostData;
  };

  // Payload is a wrapper class of replay_service::Payload, it hides the
  // new/delete operations of the proto object from outer code.
  class Payload {
   public:
    // Creates a new Payload from a protobuf payload object.
    Payload(std::unique_ptr<replay_service::Payload> protoPayload);

    ~Payload();
    Payload() = delete;
    Payload(const Payload&) = delete;
    Payload(Payload&&) = delete;
    Payload& operator=(const Payload&) = delete;
    Payload& operator=(Payload&&) = delete;

    // Returns the stack size in bytes specified by this replay payload.
    uint32_t stack_size() const;
    // Returns the volatile memory size in bytes specified by this replay
    // payload.
    uint32_t volatile_memory_size() const;
    // Returns the constant memory size in bytes specified by this replay
    // payload.
    size_t constants_size() const;
    // Gets a pointer to the payload constant data.
    const void* constants_data() const;
    // Returns the count of resource info.
    size_t resource_info_count() const;
    // Returns the ID of the 'index'th (starts from 0) resource info.
    const std::string resource_id(int index) const;
    // Returns the expected size of the 'index'th (starts from 0) resource info.
    uint32_t resource_size(int index) const;
    // Returns the size in bytes of the opcodes in this replay payload.
    size_t opcodes_size() const;
    // Gets a pointer to the opcodes in this replay payload.
    const void* opcodes_data() const;

   private:
    // The internal proto object.
    std::unique_ptr<replay_service::Payload> mProtoPayload;
    // std::unique_ptr<replay_service::ReplayRequest> mProtoReplayRequest;
  };

  // FenceReady is a wrapper class of replay_service::FenceReady, it hides
  // the new/delete operations of the proto object from outer code.
  class FenceReady {
   public:
    FenceReady(std::unique_ptr<replay_service::FenceReady> protoFenceReady);

    ~FenceReady();
    FenceReady(const FenceReady&) = delete;
    FenceReady(FenceReady&&) = delete;
    FenceReady& operator=(const FenceReady&) = delete;
    FenceReady& operator=(FenceReady&&) = delete;
    uint32_t id() const;

   private:
    std::unique_ptr<replay_service::FenceReady> mProtoFenceReady;
  };

  // Resources is a wrapper class of replay_service::Resources, it hides the
  // new/delete operations of the proto object from outer code.
  class Resources {
   public:
    // Creates a new Resources from a protobuf resources object.
    Resources(std::unique_ptr<replay_service::Resources> protoResources);

    ~Resources();
    Resources(const Resources&) = delete;
    Resources(Resources&&) = delete;
    Resources& operator=(const Resources&) = delete;
    Resources& operator=(Resources&&) = delete;

    // Returns the size in bytes of the data contained by this Resources.
    size_t size() const;
    // Gets a pointer to the data contained by this Resources.
    const void* data() const;

   private:
    Resources(std::unique_ptr<replay_service::ReplayRequest> req);

    // The internal proto object.
    std::unique_ptr<replay_service::Resources> mProtoResources;
  };

  ReplayService() = default;
  virtual ~ReplayService() {}

  ReplayService(const ReplayService&) = delete;
  ReplayService(ReplayService&&) = delete;
  ReplayService& operator=(const ReplayService&) = delete;
  ReplayService& operator=(ReplayService&&) = delete;

  // Gets a Payload. Returns nullptr in case of error.
  virtual std::unique_ptr<Payload> getPayload(const std::string& id) = 0;
  // Get Resources. Returns nullptr in case of error.
  virtual std::unique_ptr<Resources> getResources(const Resource* resources,
                                                  size_t resCount) = 0;

  virtual std::unique_ptr<FenceReady> getFenceReady(const uint32_t& id) = 0;

  // Sends ReplayFinished signal. Returns true if succeeded, otherwise returns
  // false.
  virtual bool sendReplayFinished() = 0;
  // Sends crash dump. Returns true if succeeded, otherwise returns false.
  virtual bool sendCrashDump(const std::string& filepath,
                             const void* crash_data, uint32_t crash_size) = 0;
  // Sends post data. Returns true if succeeded, otherwise returns false.
  virtual bool sendPosts(std::unique_ptr<Posts> posts) = 0;
  // Sends error message notification. Returns true if succeeded,
  // otherwise returns false.
  virtual bool sendErrorMsg(uint64_t seq_num, uint32_t severity,
                            uint32_t api_index, uint64_t label,
                            const std::string& msg, const void* data,
                            uint32_t data_size) = 0;
  // Sends replay status notification. Returns true if succeeded, otherwise
  // returns false.
  virtual bool sendReplayStatus(uint64_t label, uint32_t total_instrs,
                                uint32_t finished_instrs) = 0;
  // Sends data notification. Returns true if succeeded, otherwise returns
  // false.
  virtual bool sendNotificationData(uint64_t id, uint64_t label,
                                    const void* data, uint32_t data_size) = 0;

  // Returns the next replay request from the server.
  virtual std::unique_ptr<replay_service::ReplayRequest> getReplayRequest() = 0;
};
}  // namespace gapir

#endif  // GAPIR_REPLAY_SERVICE_H
