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

#ifndef GAPIR_REPLAY_ARCHIVE_H
#define GAPIR_REPLAY_ARCHIVE_H

#include "replay_service.h"
#include "resource.h"

#include "core/cc/archive.h"
#include "core/cc/log.h"

#include <memory>

namespace gapir {

// ArchiveReplayService implements ReplayService interface for exported replays.
// It represents an local on-disk source of replay payload data.
class ArchiveReplayService : public ReplayService {
 public:
  ArchiveReplayService(const std::string& fileprefix,
                       const std::string& postbackDir)
      : mFilePrefix(fileprefix), mPostbackDir(postbackDir) {}

  // Read payload from disk.
  std::unique_ptr<Payload> getPayload(const std::string& _payload) override;

  // Send post data to local on disk file.
  bool sendPosts(std::unique_ptr<Posts> posts) override;

  // We are reading from disk, so the following methods are not implemented.
  std::unique_ptr<Resources> getResources(const Resource* resources,
                                          size_t resCount) override {
    return nullptr;
  }

  std::unique_ptr<FenceReady> getFenceReady(const uint32_t& id) override {
    return nullptr;
  }

  std::unique_ptr<replay_service::ReplayRequest> getReplayRequest() override {
    return std::unique_ptr<replay_service::ReplayRequest>(
        new replay_service::ReplayRequest());
  }

  bool sendReplayFinished() override { return true; }

  bool sendCrashDump(const std::string& filepath, const void* crash_data,
                     uint32_t crash_size) override {
    GAPID_INFO("Crash dump saved at: %s", filepath.c_str());
    return true;
  }

  bool sendErrorMsg(uint64_t seq_num, uint32_t severity, uint32_t api_index,
                    uint64_t label, const std::string& msg, const void* data,
                    uint32_t data_size) override {
    return true;
  }

  bool sendReplayStatus(uint64_t label, uint32_t total_instrs,
                        uint32_t finished_instrs) override {
    return true;
  }

  bool sendNotificationData(uint64_t, uint64_t, const void*,
                            uint32_t) override {
    return true;
  }

 private:
  std::string mFilePrefix;
  std::string mPostbackDir;
};

}  // namespace gapir

#endif  // GAPIR_REPLAY_ARCHIVE_H
