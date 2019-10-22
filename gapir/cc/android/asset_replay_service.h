/*
 * Copyright (C) 2019 Google Inc.
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

#ifndef GAPIR_ASSET_REPLAY_SERVICE_H
#define GAPIR_ASSET_REPLAY_SERVICE_H

#include "asset_replay_service.h"
#include "gapir/cc/replay_service.h"
#include "gapir/cc/resource.h"

#include "core/cc/archive.h"
#include "core/cc/log.h"

#include <android/asset_manager.h>
#include <android/asset_manager_jni.h>

#include <memory>

namespace gapir {

// AssetReplayService implements ReplayService interface for exported replays on
// Android. It accesses replay payload data via Android assets.
class AssetReplayService : public ReplayService {
 public:
  AssetReplayService(AAssetManager* assetManager)
      : mAssetManager(assetManager) {}

  // Read payload from Android assets.
  std::unique_ptr<Payload> getPayload(const std::string& _payload) override;

  // We are reading from Android assets, so the following methods are not
  // implemented.
  std::unique_ptr<Resources> getResources(const Resource* resources,
                                          size_t resCount) override {
    return nullptr;
  }

  std::unique_ptr<FenceReady> getFenceReady(const uint32_t& id) override {
    return nullptr;
  }

  bool sendReplayFinished() override { return true; }

  bool sendCrashDump(const std::string& filepath, const void* crash_data,
                     uint32_t crash_size) override {
    return true;
  }

  bool sendPosts(std::unique_ptr<Posts> posts) override { return true; }

  bool sendErrorMsg(uint64_t seq_num, uint32_t severity, uint32_t api_index,
                    uint64_t label, const std::string& msg, const void* data,
                    uint32_t data_size) override {
    return true;
  }

  bool sendReplayStatus(uint64_t label, uint32_t total_instrs,
                        uint32_t finished_instrs) override {
    return true;
  }

  bool sendNotificationData(uint64_t id, uint64_t label, const void* data,
                            uint32_t data_size) override {
    return true;
  }

  std::unique_ptr<replay_service::ReplayRequest> getReplayRequest() override {
    return std::unique_ptr<replay_service::ReplayRequest>(
        new replay_service::ReplayRequest());
  }

 private:
  AAssetManager* mAssetManager;
};

}  // namespace gapir

#endif  // GAPIR_ASSET_REPLAY_SERVICE_H
