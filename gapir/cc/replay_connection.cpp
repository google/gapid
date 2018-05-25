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

#include <grpc++/grpc++.h>
#include <memory>

#include "gapis/service/severity/severity.pb.h"
#include "gapir/replay_service/service.grpc.pb.h"
#include "core/cc/log.h"

namespace gapir {

// ResourceRequest member methods

ReplayConnection::ResourceRequest::~ResourceRequest() = default;

bool ReplayConnection::ResourceRequest::append(const std::string& id,
                                               size_t size) {
  if (mProtoResourceRequest == nullptr) {
    return false;
  }
  mProtoResourceRequest->add_ids(id);
  mProtoResourceRequest->set_expected_total_size(
      mProtoResourceRequest->expected_total_size() + size);
  return true;
}

replay_service::ResourceRequest*
ReplayConnection::ResourceRequest::release_to_proto() {
  auto ptr = mProtoResourceRequest.release();
  mProtoResourceRequest = nullptr;
  return ptr;
}

ReplayConnection::ResourceRequest::ResourceRequest()
    : mProtoResourceRequest(new replay_service::ResourceRequest()) {}

// Posts member methods

ReplayConnection::Posts::~Posts() = default;

bool ReplayConnection::Posts::append(uint64_t id, const void* data,
                                     size_t size) {
  if (mProtoPostData == nullptr) {
    return false;
  }
  auto* piece = mProtoPostData->add_post_data_pieces();
  piece->set_id(id);
  piece->set_data(data, size);
  return true;
}

replay_service::PostData* ReplayConnection::Posts::release_to_proto() {
  auto ptr = mProtoPostData.release();
  mProtoPostData = nullptr;
  return ptr;
}

size_t ReplayConnection::Posts::piece_count() const {
  return mProtoPostData->post_data_pieces_size();
}

size_t ReplayConnection::Posts::piece_size(int index) const {
  return mProtoPostData->post_data_pieces(index).data().size();
}

const void* ReplayConnection::Posts::piece_data(int index) const {
  return mProtoPostData->post_data_pieces(index).data().data();
}

uint64_t ReplayConnection::Posts::piece_id(int index) const {
  return mProtoPostData->post_data_pieces(index).id();
}

ReplayConnection::Posts::Posts() : mProtoPostData(new replay_service::PostData()) {}

// Payload member methods

std::unique_ptr<ReplayConnection::Payload> ReplayConnection::Payload::get(
    ReplayGrpcStream* stream) {
  std::unique_ptr<replay_service::ReplayRequest> req(new replay_service::ReplayRequest());
  if (!stream->Read(req.get())) {
    return nullptr;
  }
  if (req->req_case() != replay_service::ReplayRequest::kPayload) {
    return nullptr;
  }
  return std::unique_ptr<Payload>(new Payload(std::move(req)));
}

ReplayConnection::Payload::Payload(
    std::unique_ptr<replay_service::Payload> protoPayload)
    : mProtoReplayRequest(new replay_service::ReplayRequest()) {
  mProtoReplayRequest->set_allocated_payload(protoPayload.release());
}

ReplayConnection::Payload::~Payload() = default;

uint32_t ReplayConnection::Payload::stack_size() const {
  return mProtoReplayRequest->payload().stack_size();
}

uint32_t ReplayConnection::Payload::volatile_memory_size() const {
  return mProtoReplayRequest->payload().volatile_memory_size();
}

size_t ReplayConnection::Payload::constants_size() const {
  return mProtoReplayRequest->payload().constants().size();
}

const void* ReplayConnection::Payload::constants_data() const {
  return mProtoReplayRequest->payload().constants().data();
}

size_t ReplayConnection::Payload::resource_info_count() const {
  return mProtoReplayRequest->payload().resources_size();
}

const std::string ReplayConnection::Payload::resource_id(int index) const {
  return mProtoReplayRequest->payload().resources(index).id();
}

uint32_t ReplayConnection::Payload::resource_size(int index) const {
  return mProtoReplayRequest->payload().resources(index).size();
}

size_t ReplayConnection::Payload::opcodes_size() const {
  return mProtoReplayRequest->payload().opcodes().size();
}

const void* ReplayConnection::Payload::opcodes_data() const {
  return mProtoReplayRequest->payload().opcodes().data();
}

ReplayConnection::Payload::Payload(std::unique_ptr<replay_service::ReplayRequest> req)
    : mProtoReplayRequest(std::move(req)) {}

// Resources member methods

std::unique_ptr<ReplayConnection::Resources> ReplayConnection::Resources::get(
    ReplayGrpcStream* stream) {
  std::unique_ptr<replay_service::ReplayRequest> req(new replay_service::ReplayRequest());
  if (!stream->Read(req.get())) {
    return nullptr;
  }
  if (req->req_case() != replay_service::ReplayRequest::kResources) {
    return nullptr;
  }
  return std::unique_ptr<Resources>(new Resources(std::move(req)));
}

ReplayConnection::Resources::Resources(
    std::unique_ptr<replay_service::Resources> protoResources)
    : mProtoReplayRequest(new replay_service::ReplayRequest()) {
  mProtoReplayRequest->set_allocated_resources(protoResources.release());
}

ReplayConnection::Resources::~Resources() = default;

size_t ReplayConnection::Resources::size() const {
  return mProtoReplayRequest->resources().data().size();
}

const void* ReplayConnection::Resources::data() const {
  return mProtoReplayRequest->resources().data().data();
}

ReplayConnection::Resources::Resources(
    std::unique_ptr<replay_service::ReplayRequest> req)
    : mProtoReplayRequest(std::move(req)) {}

// ReplayConnection member methods

ReplayConnection::~ReplayConnection() {
  if (mGrpcStream != nullptr) {
    this->sendReplayFinished();
  }
}

std::unique_ptr<ReplayConnection::Payload> ReplayConnection::getPayload() {
  // Send a replay response with payload request
  replay_service::ReplayResponse res;
  res.set_allocated_payload_request(new replay_service::PayloadRequest());
  mGrpcStream->Write(res);
  return ReplayConnection::ReplayConnection::Payload::get(mGrpcStream);
}

std::unique_ptr<ReplayConnection::Resources> ReplayConnection::getResources(
    std::unique_ptr<ReplayConnection::ResourceRequest> req) {
  // Send a replay response with resources request
  replay_service::ReplayResponse res;
  res.set_allocated_resource_request(req->release_to_proto());
  mGrpcStream->Write(res);
  return ReplayConnection::ReplayConnection::Resources::get(mGrpcStream);
}

bool ReplayConnection::sendReplayFinished() {
  replay_service::ReplayResponse res;
  res.set_allocated_finished(new replay_service::Finished());
  return mGrpcStream->Write(res);
}

bool ReplayConnection::sendCrashDump(const std::string& filepath,
                                     const void* crash_data,
                                     uint32_t crash_size) {
  replay_service::ReplayResponse res;
  res.mutable_crash_dump()->set_filepath(filepath);
  res.mutable_crash_dump()->set_crash_data(crash_data, crash_size);
  return mGrpcStream->Write(res);
}

bool ReplayConnection::sendPostData(std::unique_ptr<Posts> posts) {
  replay_service::ReplayResponse res;
  res.set_allocated_post_data(posts->release_to_proto());
  return mGrpcStream->Write(res);
}

bool ReplayConnection::sendNotification(uint64_t id, int severity,
                                        uint32_t api_index, uint64_t label,
                                        const std::string& msg,
                                        const void* data, uint32_t data_size) {
  using severity::Severity;
  Severity sev = Severity::DebugLevel;
  switch (severity) {
    case LOG_LEVEL_FATAL:
      sev = Severity::FatalLevel;
      break;
    case LOG_LEVEL_ERROR:
      sev = Severity::ErrorLevel;
      break;
    case LOG_LEVEL_WARNING:
      sev = Severity::WarningLevel;
      break;
    case LOG_LEVEL_INFO:
      sev = Severity::InfoLevel;
      break;
    case LOG_LEVEL_DEBUG:
      sev = Severity::DebugLevel;
      break;
    case LOG_LEVEL_VERBOSE:
      sev = Severity::VerboseLevel;
      break;
    default:
      sev = Severity::DebugLevel;
      break;
  }

  replay_service::ReplayResponse res;
  auto* notification = res.mutable_notification();
  notification->set_id(id);
  notification->set_severity(sev);
  notification->set_api_index(api_index);
  notification->set_label(label);
  notification->set_msg(msg);
  notification->set_data(data, data_size);
  return mGrpcStream->Write(res);
}

}  // namespace gapir
