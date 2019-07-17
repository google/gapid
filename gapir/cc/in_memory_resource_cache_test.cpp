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

#include "in_memory_resource_cache.h"
#include "cached_resource_loader.h"
#include "memory_manager.h"
#include "mock_resource_loader.h"
#include "replay_service.h"
#include "test_utilities.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>
#include <vector>

using namespace ::testing;

namespace gapir {
namespace test {
namespace {

const uint32_t MEMORY_SIZE = 4096;
const uint32_t CACHE_SIZE = 2048;

const Resource A("A", 64);
const Resource B("B", 256);
const Resource C("C", 512);
const Resource D("D", 1024);
const Resource E("E", 2048);
const Resource Z("Z", 1);

class ResourceInMemoryCacheTest : public Test {
 protected:
  virtual void SetUp() {
    mMemoryAllocator =
        std::shared_ptr<MemoryAllocator>(new MemoryAllocator(MEMORY_SIZE));

    mMemoryManager.reset(new MemoryManager(mMemoryAllocator));
    mMemoryManager->setVolatileMemory(MEMORY_SIZE - CACHE_SIZE);

    mCache = InMemoryResourceCache::create(mMemoryAllocator, CACHE_SIZE);
    mMemoryCachedResourceLoader = CachedResourceLoader::create(
        mCache.get(),
        std::unique_ptr<ResourceLoader>(new StrictMock<MockResourceLoader>()));

    mFallbackLoader = new StrictMock<MockResourceLoader>();

    mCache->resize(CACHE_SIZE);
  }

  inline void expectCacheHit(std::vector<Resource> resources) {
    SCOPED_TRACE("expectCacheHit");

    auto res_data = createResourcesData(resources);
    size_t size = res_data.size();

    std::vector<uint8_t> got(size);

    // Test as a single request.
    EXPECT_TRUE(mMemoryCachedResourceLoader->load(
        resources.data(), resources.size(), got.data(), size));

    EXPECT_EQ(got, res_data);

    // Test individually
    size_t offset = 0;
    for (auto resource : resources) {
      EXPECT_TRUE(mMemoryCachedResourceLoader->load(&resource, 1, &got[offset],
                                                    resource.getSize()));
      offset += resource.getSize();
    }

    EXPECT_EQ(got, res_data);

    if (HasFailure()) {
      mMemoryCachedResourceLoader->getCache()->dump(stdout);
    }
  }

  inline void expectCacheMiss(std::vector<Resource> resources) {
    SCOPED_TRACE("expectCacheMiss");

    size_t size = 0;
    for (auto resource : resources) {
      size += resource.getSize();
    }
    std::vector<uint8_t> got(size);
    auto res_data = createResourcesData(resources);
    EXPECT_CALL(*mFallbackLoader, fetch(_, _))
        .With(Args<0, 1>(ElementsAreArray(resources)))
        .WillOnce(Return(ByMove(createResources(res_data))))
        .RetiresOnSaturation();
    EXPECT_TRUE(mMemoryCachedResourceLoader->load(
        resources.data(), resources.size(), got.data(), size));

    EXPECT_EQ(got, res_data);
  }

  static const size_t TEMP_SIZE = 2048;

  StrictMock<MockResourceLoader>* mFallbackLoader;
  std::unique_ptr<InMemoryResourceCache> mCache;
  std::shared_ptr<MemoryAllocator> mMemoryAllocator;
  std::unique_ptr<MemoryManager> mMemoryManager;
  std::unique_ptr<CachedResourceLoader> mMemoryCachedResourceLoader;
  uint8_t mTemp[TEMP_SIZE];
};

}  // anonymous namespace
}  // namespace test
}  // namespace gapir
