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

#include "replay_connection.h"

#include <memory>

#include "gapir/service/service.grpc.pb.h"

namespace gapir {

std::unique_ptr<ReplayConnection::Payload>
ReplayConnection::getPayload() {
  // Send a replay response with payload request
  service::ReplayResponse res;
  res.set_allocated_payload_request(new service::PayloadRequest());
  mGrpcStream.Write(res);
  return ReplayConnection::Payload::get(&mGrpcStream);
}

std::unique_ptr<ReplayConnection::Resources> ReplayConnection::getResources(
    std::unique_ptr<ReplayConnection::ResourceRequest> req) {
  // Send a replay response with resources request
  service::ReplayResponse res;
  res.set_allocated_resource_request(req->release_to_proto());
  mGrpcStream.Write(res);
  return ReplayConnection::Resources::get(&mGrpcStream);
}

bool ReplayConnection::sendReplayFinished() {
  service::ReplayResponse res;
  res.set_allocated_finished(new service::Finished());
  return mGrpcStream.Write(res);
}

bool ReplayConnection::sendCrashDump(const std::string& filepath,
                                     const void* crash_data,
                                     uint32_t crash_size) {
  service::ReplayResponse res;
  res.mutable_crash_dump()->set_filepath(filepath);
  res.mutable_crash_dump()->set_crash_data(crash_data, crash_size);
  return mGrpcStream.Write(res);
}

bool ReplayConnection::sendPostData(std::unique_ptr<Posts> posts) {
  service::ReplayResponse res;
  res.set_allocated_post_data(posts->release_to_proto());
  return mGrpcStream.Write(res);
}

bool ReplayConnection::sendNotification(uint64_t id, uint32_t api_index,
                                        uint64_t label, const std::string& msg,
                                        const void* data, uint32_t data_size) {
  service::ReplayResponse res;
  res.mutable_notification()->set_id(id);
  res.mutable_notification()->set_api_index(api_index);
  res.mutable_notification()->set_label(label);
  res.mutable_notification()->set_msg(msg);
  res.mutable_notification()->set_data(data, data_size);
  return mGrpcStream.Write(res);
}

}  // namespace gapir
