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

#include <gmock/gmock.h>

#include <vector>

#include "replay_service.h"
#include "resource_loader.h"

namespace gapir {
namespace test {

class MockResourceLoader : public ResourceLoader {
 public:
  MOCK_METHOD4(load, bool(const Resource* resources, size_t count, void* target,
                          size_t targetSize));
  MOCK_METHOD2(fetch, std::unique_ptr<ReplayService::Resources>(
                          const Resource* resources, size_t count));
};

}  // namespace test
}  // namespace gapir

#endif  // GAPIR_MOCK_RESOURCE_PROVIDER_H
