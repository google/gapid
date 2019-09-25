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

namespace {
// Notification ID 0 is reserved for issues report. The value needs to be kept
// in sync with |IssuesNotificationID| in `gapis/replay/builder/builder.go`
const uint64_t kIssuesNotificationId = 0;
// Notification ID 1 is reserved for replay status information transfer. The
// value needs to be kept in sync with |ReplayProgressNotificationID| in
// `gapis/replay/builder/builder.go`
const uint64_t kReplayProgressNotificationID = 1;
}  // namespace

void GrpcReplayService::handleCommunication(GrpcReplayService* _service) {
  while (true) {
    std::unique_ptr<replay_service::ReplayRequest> req =
        std::unique_ptr<replay_service::ReplayRequest>(
            new replay_service::ReplayRequest());
    if (!_service->mGrpcStream->Read(req.get())) {
      _service->mRequestSem.release();
      _service->mDataSem.release();
      return;
    }
    _service->mCommunicationLock.lock();
    if (req->req_case() == replay_service::ReplayRequest::kReplay ||
        req->req_case() == replay_service::ReplayRequest::kPrewarm) {
      _service->mDeferredRequests.push_back(std::move(req));
      _service->mRequestSem.release();
    } else {
      _service->mDeferredData.push_back(std::move(req));
      _service->mDataSem.release();
    }
    _service->mCommunicationLock.unlock();
  }
}

void GrpcReplayService::primeState(std::string prerun_id,
                                   std::string cleanup_id) {
  std::unique_ptr<replay_service::ReplayRequest> req =
      std::unique_ptr<replay_service::ReplayRequest>(
          new replay_service::ReplayRequest());
  auto r = new replay_service::PrewarmRequest();
  r->set_prerun_id(std::move(prerun_id));
  r->set_cleanup_id(std::move(cleanup_id));
  req->set_allocated_prewarm(r);
  mCommunicationLock.lock();
  mDeferredRequests.push_back(std::move(req));
  mRequestSem.release();
  mCommunicationLock.unlock();
}

std::unique_ptr<ReplayService::Payload> GrpcReplayService::getPayload(
    const std::string& id) {
  // Send a replay response with payload request
  replay_service::ReplayResponse res;
  auto plc = new replay_service::PayloadRequest();
  plc->set_payload_id(id);
  res.set_allocated_payload_request(plc);
  mGrpcStream->Write(res);

  std::unique_ptr<replay_service::ReplayRequest> req = getNonReplayRequest();
  if (!req) {
    return nullptr;
  }

  if (req->req_case() != replay_service::ReplayRequest::kPayload) {
    return nullptr;
  }
  return std::unique_ptr<ReplayService::Payload>(new ReplayService::Payload(
      std::unique_ptr<replay_service::Payload>(req->release_payload())));
}

std::unique_ptr<ReplayService::FenceReady> GrpcReplayService::getFenceReady(
    const uint32_t& id) {
  // Send a replay response with payload request
  replay_service::ReplayResponse res;
  auto frr = new replay_service::FenceReadyRequest();
  frr->set_id(id);
  res.set_allocated_fence_ready_request(frr);
  mGrpcStream->Write(res);
  std::unique_ptr<replay_service::ReplayRequest> req = getNonReplayRequest();
  if (!req) {
    return nullptr;
  }
  if (req->req_case() != replay_service::ReplayRequest::kFenceReady) {
    return nullptr;
  }
  return std::unique_ptr<ReplayService::FenceReady>(
      new ReplayService::FenceReady(std::unique_ptr<replay_service::FenceReady>(
          req->release_fence_ready())));
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
    res.mutable_resource_request()->add_ids(resources[i].getID());
    totalSize += resources[i].getSize();
  }
  res.mutable_resource_request()->set_expected_total_size(totalSize);
  mGrpcStream->Write(res);
  std::unique_ptr<replay_service::ReplayRequest> req = getNonReplayRequest();
  if (!req) {
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

bool GrpcReplayService::sendErrorMsg(uint64_t seq_num, uint32_t severity,
                                     uint32_t api_index, uint64_t label,
                                     const std::string& msg, const void* data,
                                     uint32_t data_size) {
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
  notification->set_id(kIssuesNotificationId);
  auto* error_msg = notification->mutable_error_msg();
  error_msg->set_seq_num(seq_num);
  error_msg->set_severity(sev);
  error_msg->set_api_index(api_index);
  error_msg->set_label(label);
  error_msg->set_msg(msg);
  error_msg->set_data(data, data_size);
  return mGrpcStream->Write(res);
}

bool GrpcReplayService::sendReplayStatus(uint64_t label, uint32_t total_instrs,
                                         uint32_t finished_instrs) {
  replay_service::ReplayResponse res;
  auto* notification = res.mutable_notification();
  notification->set_id(kReplayProgressNotificationID);
  auto* replay_status = notification->mutable_replay_status();
  replay_status->set_label(label);
  replay_status->set_total_instrs(total_instrs);
  replay_status->set_finished_instrs(finished_instrs);
  return mGrpcStream->Write(res);
}

bool GrpcReplayService::sendNotificationData(uint64_t id, uint64_t label,
                                             const void* data,
                                             uint32_t data_size) {
  replay_service::ReplayResponse res;
  auto* notification = res.mutable_notification();
  notification->set_id(id);
  auto* notification_data = notification->mutable_data();
  notification_data->set_label(label);
  notification_data->set_data(data, data_size);
  return mGrpcStream->Write(res);
}

std::unique_ptr<replay_service::ReplayRequest>
GrpcReplayService::getNonReplayRequest() {
  mDataSem.acquire();
  mCommunicationLock.lock();
  if (mDeferredData.empty()) {
    mDataSem.release();
    mCommunicationLock.unlock();
    return nullptr;
  }
  auto req = std::move(mDeferredData.front());
  mDeferredData.pop_front();
  mCommunicationLock.unlock();
  return req;
}

std::unique_ptr<replay_service::ReplayRequest>
GrpcReplayService::getReplayRequest() {
  mRequestSem.acquire();
  mCommunicationLock.lock();
  if (mDeferredRequests.empty()) {
    mRequestSem.release();
    mCommunicationLock.unlock();
    return nullptr;
  }
  auto req = std::move(mDeferredRequests.front());
  mDeferredRequests.pop_front();
  mCommunicationLock.unlock();
  return req;
}

}  // namespace gapir
