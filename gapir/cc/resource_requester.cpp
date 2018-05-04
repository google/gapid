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
#include "replay_connection.h"

#include <cstring>
#include <memory>
#include <vector>

namespace gapir {

std::unique_ptr<ResourceRequester> ResourceRequester::create() {
    return std::unique_ptr<ResourceRequester>(new ResourceRequester());
}

bool ResourceRequester::get(const Resource*         resources,
                            size_t                  count,
                            ReplayConnection*       conn,
                            void*                   target,
                            size_t                  size) {
  if (count == 0) {
    return true;
  }
  if (conn == nullptr) {
    return false;  // no replay connection to get data.
  }
  size_t requestSize = 0;
  std::unique_ptr<ReplayConnection::ResourceRequest> req =
      ReplayConnection::ResourceRequest::create();
  for (size_t i = 0; i < count; i++) {
    req->append(resources[i].id, resources[i].size);
    requestSize += resources[i].size;
  }
  if (requestSize > size) {
    return false;  // not enough space.
  }

  std::unique_ptr<ReplayConnection::Resources> res =
      conn->getResources(std::move(req));
  if (res == nullptr) {
    return false;
  }
  if (res->size() != requestSize) {
    return false;  // unexpected resource size.
  }
  memcpy(target, res->data(), res->size());
  return true;
}

void ResourceRequester::prefetch(const Resource*         resources,
                                 size_t                  count,
                                 ReplayConnection*       conn,
                                 void*                   temp,
                                 size_t                  tempSize) {}

}  // namespace gapir
