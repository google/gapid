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

#ifndef GAPIR_MOCK_RESOURCE_PROVIDER_H
#define GAPIR_MOCK_RESOURCE_PROVIDER_H

#include "resource_provider.h"

#include <gmock/gmock.h>
#include <vector>

#include "replay_connection.h"

namespace gapir {
namespace test {

class MockResourceProvider : public ResourceProvider {
 public:
  MOCK_METHOD5(get,
               bool(const Resource* resources, size_t count,
                    ReplayConnection* conn, void* target, size_t targetSize));

  MOCK_METHOD5(prefetch,
               void(const Resource* resources, size_t count,
                    ReplayConnection* conn, void* temp, size_t tempSize));
};

// PatternedResourceProvider is a ResourceProvider that writes a pattern to the
// pointer handed to get, simulating the loading of a resource before calling
// the inner ResourceProvider. The PatternedResourceProvider is expected to be
// placed between the ResourceProvider being tested and the
// MockResourceProvider.
class PatternedResourceProvider : public ResourceProvider {
 public:
  inline PatternedResourceProvider(std::unique_ptr<ResourceProvider> inner)
      : mInner(std::move(inner)) {}

  inline bool get(const Resource* resources, size_t count,
                  ReplayConnection* conn, void* target,
                  size_t targetSize) override {
    auto pattern =
        patternFor(std::vector<Resource>(resources, resources + count));
    memcpy(target, &pattern[0], pattern.size());
    return mInner->get(resources, count, conn, target, targetSize);
  }

  inline void prefetch(const Resource* resources, size_t count,
                       ReplayConnection* conn, void* temp,
                       size_t tempSize) override {
    mInner->prefetch(resources, count, conn, temp, tempSize);
  }

  // patternFor returns the memory pattern that will be written to the target
  // pointer when calling get.
  static std::vector<uint8_t> patternFor(
      const std::vector<Resource>& resources) {
    std::vector<uint8_t> v;
    for (auto resource : resources) {
      for (size_t i = 0; i < resource.size; i++) {
        v.push_back(resource.id[i % resource.id.size()]);
      }
    }
    return v;
  }

 private:
  std::unique_ptr<ResourceProvider> mInner;
};

}  // namespace test
}  // namespace gapir

#endif  // GAPIR_MOCK_RESOURCE_PROVIDER_H
