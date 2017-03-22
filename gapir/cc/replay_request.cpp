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

#include "memory_manager.h"
#include "replay_request.h"
#include "resource_provider.h"
#include "server_connection.h"

#include "core/cc/log.h"

#include <string.h>

#include <string>
#include <utility>
#include <vector>

namespace gapir {

std::unique_ptr<ReplayRequest> ReplayRequest::create(const ServerConnection& server,
                                                     ResourceProvider* resourceProvider,
                                                     MemoryManager* memoryManager) {
    memoryManager->setReplayDataSize(server.replayLength());

    // Request the replay data from the server
    void* address = memoryManager->getReplayAddress();
    auto resource = Resource(server.replayId(), server.replayLength());
    if (!resourceProvider->get(&resource, 1, server, address, resource.size)) {
        GAPID_WARNING("Can't load replay request: %s", resource.id.c_str());
        return nullptr;
    }

    std::unique_ptr<ReplayRequest> req(new ReplayRequest());
    if (req->load(address, resource.size)) {
        return req;
    } else {
        return nullptr;
    }
}

uint32_t ReplayRequest::getStackSize() const {
    return mStackSize;
}

uint32_t ReplayRequest::getVolatileMemorySize() const {
    return mVolatileMemorySize;
}

const std::vector<Resource>& ReplayRequest::getResources() const {
    return mResources;
}

const std::pair<const void*, uint32_t>& ReplayRequest::getConstantMemory() const {
    return mConstantMemory;
}

const std::pair<const uint32_t*, uint32_t>& ReplayRequest::getInstructionList() const {
    return mInstructionList;
}

bool ReplayRequest::load(void* data, uint32_t size) {
    // Parse the data fetched from the gazer connection
    const uint8_t* ptr = static_cast<uint8_t*>(data);
    ptr = loadStackSize(ptr);
    ptr = loadVolatileMemorySize(ptr);
    ptr = loadConstantMemory(ptr);
    ptr = loadResourceIds(ptr);
    ptr = loadInstructionList(ptr);

    GAPID_INFO("Replay request loaded");

    return ptr - size == data;
}

const uint8_t* ReplayRequest::loadVolatileMemorySize(const uint8_t* ptr) {
    mVolatileMemorySize = *reinterpret_cast<const uint32_t*>(ptr);
    ptr += sizeof(uint32_t);
    GAPID_DEBUG("Volatile memory size: %d", mVolatileMemorySize);
    return ptr;
}

const uint8_t* ReplayRequest::loadStackSize(const uint8_t* ptr) {
    mStackSize = *reinterpret_cast<const uint32_t*>(ptr);
    ptr += sizeof(uint32_t);
    GAPID_DEBUG("Stack size: %d", mStackSize);
    return ptr;
}

const uint8_t* ReplayRequest::loadConstantMemory(const uint8_t* ptr) {
    uint32_t constantMemorySize = *reinterpret_cast<const uint32_t*>(ptr);
    ptr += sizeof(uint32_t);

    mConstantMemory = {ptr, constantMemorySize};
    GAPID_DEBUG("Constant memory size: %d", constantMemorySize);
    ptr += constantMemorySize;

    return ptr;
}

const uint8_t* ReplayRequest::loadResourceIds(const uint8_t* ptr) {
    uint32_t resourceCount = *reinterpret_cast<const uint32_t*>(ptr);
    ptr += sizeof(uint32_t);

    mResources.reserve(resourceCount);
    for (uint32_t i = 0; i < resourceCount; ++i) {
        std::string resourceName(reinterpret_cast<const char*>(ptr));
        ptr += resourceName.length() + 1;

        uint32_t resourceSize = *reinterpret_cast<const uint32_t*>(ptr);
        ptr += sizeof(uint32_t);

        mResources.emplace_back(std::move(resourceName), resourceSize);
    }
    GAPID_DEBUG("Resources: %d", resourceCount);

    return ptr;
}

const uint8_t* ReplayRequest::loadInstructionList(const uint8_t* ptr) {
    uint32_t instructionListSize = *reinterpret_cast<const uint32_t*>(ptr);
    ptr += sizeof(uint32_t);

    const uint32_t instructionCount = instructionListSize / sizeof(uint32_t);
    mInstructionList = {reinterpret_cast<const uint32_t*>(ptr), instructionCount};
    GAPID_DEBUG("Instruction count: %d", instructionCount);
    ptr += instructionListSize;

    return ptr;
}

}  // namespace gapir
