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

#include "asset_replay_service.h"

#include <unistd.h>

#include <memory>

#include "core/cc/log.h"

namespace {

const char* kAssetPathPayloadBin = "replay_export/payload.bin";

}  // namespace

namespace gapir {

std::unique_ptr<ReplayService::Payload> AssetReplayService::getPayload(
    const std::string& id) {
  AAsset* asset_payload = AAssetManager_open(
      mAssetManager, kAssetPathPayloadBin, AASSET_MODE_STREAMING);

  off64_t offset;
  off64_t length;
  int payload_fd = AAsset_openFileDescriptor64(asset_payload, &offset, &length);
  if (payload_fd < 0) {
    GAPID_FATAL(
        "AssetReplayService::getPayload() cannot open payload asset as a "
        "file descriptor (because the asset was stored compressed?)");
  }
  AAsset_close(asset_payload);

  off64_t ret = lseek64(payload_fd, offset, SEEK_SET);
  if (ret == (off64_t)-1) {
    GAPID_FATAL("AssetReplayService::getPayload() lseek64 failed");
  }

  std::unique_ptr<replay_service::Payload> payload(new replay_service::Payload);
  payload->ParseFromFileDescriptor(payload_fd);
  close(payload_fd);
  return std::unique_ptr<Payload>(new Payload(std::move(payload)));
}

}  // namespace gapir
