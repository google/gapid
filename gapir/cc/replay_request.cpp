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

#include <string.h>
#include <string>
#include <utility>
#include <vector>

#include "core/cc/log.h"
#include "memory_manager.h"
#include "resource_provider.h"

namespace gapir {

std::unique_ptr<ReplayRequest> ReplayRequest::create(
    ReplayConnection* conn, MemoryManager* memoryManager) {
  // Request the replay data from the server.
  if (conn == nullptr) {
    return nullptr;  // no replay connection.
  }
  GAPID_INFO("ReplayRequest::create conn not nullptr");
  std::unique_ptr<ReplayConnection::Payload> payload = conn->getPayload();
  GAPID_INFO("ReplayRequest::create Payload got");
  if (payload == nullptr) {
    GAPID_INFO("ReplayRequest::create Payload is nullptr");
    return nullptr;  // failed at getting payload.
  }
  GAPID_INFO("ReplayRequest::create Payload is not nullptr");
  // Reserve Replay data segments and load data into the memory manager.
  if (!memoryManager->setReplayDataSize(payload->constants_size(),
                                       payload->opcodes_size())) {
    GAPID_INFO("ReplayRequest::create setReplayDataSize failed");
    return nullptr;
  }
  memcpy(memoryManager->getConstantAddress(), payload->constants_data(),
         payload->constants_size());
  memcpy(memoryManager->getOpcodeAddress(), payload->opcodes_data(),
         payload->opcodes_size());
  GAPID_INFO("ReplayRequest::create memcpy done");

  // initialize this replay request.
  std::unique_ptr<ReplayRequest> req(new ReplayRequest());
  GAPID_INFO("ReplayRequest::create req pointer done");
  req->mStackSize = payload->stack_size();
  GAPID_DEBUG("Stack size: %d", req->mStackSize);
  req->mVolatileMemorySize = payload->volatile_memory_size();
  GAPID_DEBUG("Volatile memory size: %d", req->mVolatileMemorySize);
  req->mConstantMemory = {memoryManager->getConstantAddress(),
                          payload->constants_size()};
  GAPID_DEBUG("Constant memory size: %d", payload->constants_size());
  req->mResources.reserve(payload->resource_info_count());
  for (size_t i = 0; i < payload->resource_info_count(); i++) {
    req->mResources.emplace_back(payload->resource_id(i),
                                 payload->resource_size(i));
  }
  GAPID_DEBUG("Resources: %d", req->mResources.size());
  const uint32_t instCount = payload->opcodes_size() / sizeof(uint32_t);
  req->mInstructionList = {
      static_cast<uint32_t*>(memoryManager->getOpcodeAddress()), instCount};
  GAPID_DEBUG("Instruction count: %d", instCount);
  GAPID_INFO("Replay request loaded");
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
