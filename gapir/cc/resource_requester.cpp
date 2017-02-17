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

#include "resource_requester.h"
#include "server_connection.h"

#include <vector>

namespace gapir {

std::unique_ptr<ResourceRequester> ResourceRequester::create() {
    return std::unique_ptr<ResourceRequester>(new ResourceRequester());
}

bool ResourceRequester::get(const Resource*         resources,
                            size_t                  count,
                            const ServerConnection& server,
                            void*                   target,
                            size_t                  size) {
    if (count == 0) {
        return true;
    }
    size_t requestSize = 0;
    auto ids = std::vector<ResourceId>(count);
    for (size_t i = 0; i < count; i++) {
        ids[i] = resources[i].id;
        requestSize += resources[i].size;
    }
    if (requestSize > size) {
        return false; // not enough space.
    }
    return server.getResources(ids.data(), count, target, requestSize);
}

void ResourceRequester::prefetch(const Resource*         resources,
                                 size_t                  count,
                                 const ServerConnection& server,
                                 void*                   temp,
                                 size_t                  tempSize) {}

}  // namespace gapir
