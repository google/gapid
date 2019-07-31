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

#ifndef GAPIR_RESOURCE_LOADER_H
#define GAPIR_RESOURCE_LOADER_H

#include "replay_service.h"
#include "resource.h"

#include <stdint.h>
#include <memory>

namespace gapir {

// RessourceLoader is an interface which can load a list of resources in-orderly
// to the specified location.
// TODO(qining): Change the load() or fetch() interface to accept a callback
// function to process the fetched data, then we won't need two methods anymore.
class ResourceLoader {
 public:
  virtual ~ResourceLoader() {}

  // Loads count resources from the provider and writes them, in-order, to
  // target. If the net size of all the resources exceeds size, then false is
  // returned.
  virtual bool load(const Resource* resources, size_t count, void* target,
                    size_t targetSize) = 0;

  // Fetch queries the specified resources and returns a
  // ReplayService::Resources instance which contains the resources data.
  virtual std::unique_ptr<ReplayService::Resources> fetch(
      const Resource* resources, size_t count) = 0;
};

// PassThroughResourceLoader implements the ResourceLoader interface. It pull
// resources from a ReplayService instance for every resource loading request.
class PassThroughResourceLoader : public ResourceLoader {
 public:
  static std::unique_ptr<PassThroughResourceLoader> create(ReplayService* srv) {
    return std::unique_ptr<PassThroughResourceLoader>(
        new PassThroughResourceLoader(srv));
  }

  // fetch returns the resources instance fetched from
  // PassThroughResourceLoader's ReplayService, does not load it to anywhere.
  std::unique_ptr<ReplayService::Resources> fetch(const Resource* resources,
                                                  size_t count) override {
    if (resources == nullptr || count == 0) {
      return nullptr;
    }
    if (mSrv == nullptr) {
      return nullptr;
    }
    return mSrv->getResources(resources, count);
  }

  // Request all of the requested resources from the ServerConnection with a
  // single GET request then loads the data to the target location.
  bool load(const Resource* resources, size_t count, void* target,
            size_t size) override {
    if (count == 0) {
      return true;
    }
    size_t requestSize = 0;
    for (size_t i = 0; i < count; i++) {
      requestSize += resources[i].getSize();
    }
    if (requestSize > size) {
      return false;  // not enough space.
    }
    auto res = fetch(resources, count);
    if (res == nullptr) {
      return false;
    }
    if (res->size() != requestSize) {
      return false;  // unexpected resource size.
    }
    memcpy(target, res->data(), res->size());
    return true;
  }

 private:
  PassThroughResourceLoader(ReplayService* srv) : mSrv(srv) {}
  ReplayService* mSrv;
};

}  // namespace gapir

#endif  // GAPIR_RESOURCE_LOADER_H
