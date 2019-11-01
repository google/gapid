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

#include "archive_replay_service.h"
#include "core/cc/log.h"

#include <fstream>
#include <memory>

namespace gapir {

std::unique_ptr<ReplayService::Payload> ArchiveReplayService::getPayload(
    const std::string&) {
  std::fstream input(mFilePrefix, std::ios::in | std::ios::binary);
  if (!input) {
    GAPID_ERROR("Replay archive does not exist at path %s.",
                mFilePrefix.c_str());
    return NULL;
  }

  std::unique_ptr<replay_service::Payload> payload(new replay_service::Payload);
  payload->ParseFromIstream(&input);

  return std::unique_ptr<Payload>(new Payload(std::move(payload)));
}

bool ArchiveReplayService::sendPosts(
    std::unique_ptr<ReplayService::Posts> posts) {
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
}  // namespace gapir
