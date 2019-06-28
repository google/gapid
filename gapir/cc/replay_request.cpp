/*
 * Copyright (C) 2017 Google Inc.
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

#include "replay_request.h"

#include "core/cc/log.h"
#include "memory_manager.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>
#include <string.h>

#include <string>
#include <utility>
#include <vector>

namespace gapir {

std::unique_ptr<ReplayRequest> ReplayRequest::create(
    ReplayService* srv, const std::string& id, MemoryManager* memoryManager) {
  // Request the replay data from the server.
  if (srv == nullptr) {
    GAPID_ERROR("Failed to create ReplayRequest: null ReplayService");
    return nullptr;  // no replay service.
  }
  std::unique_ptr<ReplayService::Payload> payload = srv->getPayload(id);
  if (payload == nullptr) {
    GAPID_ERROR("Failed to create ReplayRequest %s: null Payload", id.c_str())
    return nullptr;  // failed at getting payload.
  }

  // initialize this replay request.
  std::unique_ptr<ReplayRequest> req(new ReplayRequest());
  req->mStackSize = payload->stack_size();
  GAPID_DEBUG("Stack size: %d", req->mStackSize);
  req->mVolatileMemorySize = payload->volatile_memory_size();
  GAPID_DEBUG("Volatile memory size: %d", req->mVolatileMemorySize);
  req->mConstantMemory = {payload->constants_data(), payload->constants_size()};
  GAPID_DEBUG("Constant memory size: %zu", payload->constants_size());
  req->mResources.reserve(payload->resource_info_count());
  for (size_t i = 0; i < payload->resource_info_count(); i++) {
    req->mResources.emplace_back(payload->resource_id(i),
                                 payload->resource_size(i));
  }
  GAPID_DEBUG("Resources: %zu", req->mResources.size());
  const uint32_t instCount = payload->opcodes_size() / sizeof(uint32_t);
  req->mInstructionList = {
      static_cast<const uint32_t*>(payload->opcodes_data()), instCount};
  GAPID_DEBUG("Instruction count: %" PRIu32, instCount);
  memoryManager->setReplayData(
      (const uint8_t*)payload->constants_data(), payload->constants_size(),
      (const uint8_t*)payload->opcodes_data(), payload->opcodes_size());
  req->mPayload = std::move(payload);
  return req;
}

uint32_t ReplayRequest::getStackSize() const { return mStackSize; }

uint32_t ReplayRequest::getVolatileMemorySize() const {
  return mVolatileMemorySize;
}

const std::vector<Resource>& ReplayRequest::getResources() const {
  return mResources;
}

const std::pair<const void*, uint32_t>& ReplayRequest::getConstantMemory()
    const {
  return mConstantMemory;
}

const std::pair<const uint32_t*, uint32_t>& ReplayRequest::getInstructionList()
    const {
  return mInstructionList;
}

}  // namespace gapir
