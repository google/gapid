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

#include "replay_archive.h"
#include "core/cc/log.h"

#include <grpc++/grpc++.h>
#include <fstream>
#include <memory>

#include "gapir/replay_service/service.grpc.pb.h"

namespace gapir {

std::unique_ptr<ReplayConnection::Payload> ReplayArchive::getPayload() {
  std::fstream input(mFileprefix, std::ios::in | std::ios::binary);
  std::unique_ptr<replay_service::Payload> payload(new replay_service::Payload);
  payload->ParseFromIstream(&input);
  return std::unique_ptr<Payload>(new Payload(std::move(payload)));
}

std::unique_ptr<ReplayConnection::Resources> ReplayArchive::getResources(
    std::unique_ptr<ResourceRequest> req) {
  return nullptr;
}
bool ReplayArchive::sendReplayFinished() { return true; }
bool ReplayArchive::sendCrashDump(const std::string& filepath,
                                  const void* crash_data, uint32_t crash_size) {
  GAPID_INFO("Crash dump saved at: %s", filepath.c_str());
  return true;
}
bool ReplayArchive::sendPostData(std::unique_ptr<Posts> posts) {
  if (mPostbackDir.empty()) {
    return true;
  }
  std::unique_ptr<replay_service::PostData> postdata(posts->release_to_proto());
  int nMessages = postdata->post_data_pieces_size();
  for (int i = 0; i < nMessages; ++i) {
    uint64_t id = postdata->post_data_pieces(i).id();
    std::string data = postdata->post_data_pieces(i).data();
    std::string path = mPostbackDir + "/" + std::to_string(id) + ".bin";
    std::fstream output(path, std::ios::out | std::ios::binary);
    output.write(data.data(), data.size());
  }
  return true;
}
bool ReplayArchive::sendNotification(uint64_t id, uint32_t severity,
                                     uint32_t api_index, uint64_t label,
                                     const std::string& msg, const void* data,
                                     uint32_t data_size) {
  return true;
}
}  // namespace gapir
