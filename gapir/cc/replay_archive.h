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

#include "core/cc/archive.h"
#include "replay_connection.h"
namespace gapir {

// ReplayArchive implements ReplayConnection for exported replays.
class ReplayArchive : public ReplayConnection {
 public:
  ReplayArchive(const std::string& fileprefix, const std::string& postbackDir)
      : ReplayConnection(nullptr),
        mFileprefix(fileprefix),
        mPostbackDir(postbackDir) {}
  // Read payload from disk.
  virtual std::unique_ptr<Payload> getPayload() override;

  // We are reading from disk, so the following methods are not implemented.
  virtual std::unique_ptr<Resources> getResources(
      std::unique_ptr<ResourceRequest> req) override;
  virtual bool sendReplayFinished() override;
  virtual bool sendCrashDump(const std::string& filepath,
                             const void* crash_data,
                             uint32_t crash_size) override;
  virtual bool sendPostData(std::unique_ptr<Posts> posts) override;
  virtual bool sendNotification(uint64_t id, uint32_t severity,
                                uint32_t api_index, uint64_t label,
                                const std::string& msg, const void* data,
                                uint32_t data_size) override;

 private:
  std::string mFileprefix;
  std::string mPostbackDir;
};

}  // namespace gapir

#endif  // GAPIR_REPLAY_ARCHIVE_H
