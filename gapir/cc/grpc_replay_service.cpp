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

#include "grpc_replay_service.h"

#include <grpc++/grpc++.h>
#include <memory>

#include "core/cc/log.h"
#include "gapir/replay_service/service.grpc.pb.h"
#include "gapir/replay_service/service.pb.h"
#include "gapis/service/severity/severity.pb.h"

namespace gapir {

std::unique_ptr<ReplayService::Payload> GrpcReplayService::getPayload() {
  // Send a replay response with payload request
  replay_service::ReplayResponse res;
  res.set_allocated_payload_request(new replay_service::PayloadRequest());
  mGrpcStream->Write(res);
  std::unique_ptr<replay_service::ReplayRequest> req(
      new replay_service::ReplayRequest());
  if (!mGrpcStream->Read(req.get())) {
    return nullptr;
  }
  if (req->req_case() != replay_service::ReplayRequest::kPayload) {
    return nullptr;
  }
  return std::unique_ptr<ReplayService::Payload>(new ReplayService::Payload(
      std::unique_ptr<replay_service::Payload>(req->release_payload())));
}

std::unique_ptr<ReplayService::Resources> GrpcReplayService::getResources(
    const Resource* resources, size_t resCount) {
  if (!mGrpcStream) {
    return nullptr;
  }
  if (resCount == 0) {
    return nullptr;
  }
  replay_service::ReplayResponse res;
  res.set_allocated_resource_request(new replay_service::ResourceRequest());
  size_t totalSize = 0;
  for (size_t i = 0; i < resCount; i++) {
    res.mutable_resource_request()->add_ids(resources[i].id);
    totalSize += resources[i].size;
  }
  res.mutable_resource_request()->set_expected_total_size(totalSize);
  mGrpcStream->Write(res);
  std::unique_ptr<replay_service::ReplayRequest> req(
      new replay_service::ReplayRequest());
  if (!mGrpcStream->Read(req.get())) {
    return nullptr;
  }
  if (req->req_case() != replay_service::ReplayRequest::kResources) {
    return nullptr;
  }
  return std::unique_ptr<ReplayService::Resources>(new ReplayService::Resources(
      std::unique_ptr<replay_service::Resources>(req->release_resources())));
}

bool GrpcReplayService::sendReplayFinished() {
  replay_service::ReplayResponse res;
  res.set_allocated_finished(new replay_service::Finished());
  return mGrpcStream->Write(res);
}

bool GrpcReplayService::sendCrashDump(const std::string& filepath,
                                      const void* crash_data,
                                      uint32_t crash_size) {
  replay_service::ReplayResponse res;
  res.mutable_crash_dump()->set_filepath(filepath);
  res.mutable_crash_dump()->set_crash_data(crash_data, crash_size);
  return mGrpcStream->Write(res);
}

bool GrpcReplayService::sendPosts(std::unique_ptr<ReplayService::Posts> posts) {
  replay_service::ReplayResponse res;
  res.set_allocated_post_data(posts->release_to_proto());
  return mGrpcStream->Write(res);
}

bool GrpcReplayService::sendNotification(uint64_t id, uint32_t severity,
                                         uint32_t api_index, uint64_t label,
                                         const std::string& msg,
                                         const void* data, uint32_t data_size) {
  using severity::Severity;
  const Severity log_levels[] = {
      Severity::FatalLevel, Severity::ErrorLevel, Severity::WarningLevel,
      Severity::InfoLevel,  Severity::DebugLevel, Severity::VerboseLevel,
  };
  Severity sev = Severity::DebugLevel;
  if (severity <= LOG_LEVEL_DEBUG) {
    sev = log_levels[severity];
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
