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
    std::vector<uint32_t> memorySizes = {MEMORY_SIZE};
    mMemoryManager.reset(new MemoryManager(memorySizes));
    mMemoryManager->setVolatileMemory(MEMORY_SIZE - CACHE_SIZE);

    mCache = InMemoryResourceCache::create(mMemoryManager->getBaseAddress());
    mMemoryCachedResourceLoader = CachedResourceLoader::create(
        mCache.get(),
        std::unique_ptr<ResourceLoader>(new StrictMock<MockResourceLoader>()));

    mFallbackLoader = static_cast<StrictMock<MockResourceLoader>*>(
        mMemoryCachedResourceLoader->getFallbackResourceLoader());

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
                                                    resource.size));
      offset += resource.size;
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
      size += resource.size;
    }
    std::vector<uint8_t> got(size);
    auto res_data = createResourcesData(resources);
    EXPECT_CALL(*mFallbackLoader, fetch(_, _))
        .With(Args<0, 1>(ElementsAreArray(resources)))
        .WillOnce(Return(ByMove(std::move(createResources(res_data)))))
        .RetiresOnSaturation();
    EXPECT_TRUE(mMemoryCachedResourceLoader->load(
        resources.data(), resources.size(), got.data(), size));

    EXPECT_EQ(got, res_data);
  }

  static const size_t TEMP_SIZE = 2048;

  StrictMock<MockResourceLoader>* mFallbackLoader;
  std::unique_ptr<InMemoryResourceCache> mCache;
  std::unique_ptr<MemoryManager> mMemoryManager;
  std::unique_ptr<CachedResourceLoader> mMemoryCachedResourceLoader;
  uint8_t mTemp[TEMP_SIZE];
};

}  // anonymous namespace

// Test that get() calls with uncached data propagages to the fallback provider.
TEST_F(ResourceInMemoryCacheTest, PopulateNoFill) {
  InSequence x;

  expectCacheMiss({A});
  expectCacheMiss({B});
  expectCacheMiss({C, D});
}

// Test that get() calls with cached data propagages to the fallback provider.
TEST_F(ResourceInMemoryCacheTest, CacheHit) {
  InSequence x;

  expectCacheMiss({A});
  expectCacheMiss({B});
  expectCacheHit({A, B});
}

TEST_F(ResourceInMemoryCacheTest, Prefetch) {
  InSequence x;

  EXPECT_CALL(*mFallbackLoader, fetch(_, _))
      .With(Args<0, 1>(ElementsAre(A, B, C, D)))
      .WillOnce(Return(ByMove(
          std::move(createResources(createResourcesData({A, B, C, D}))))));

  Resource resources[] = {A, B, C, D, E};
  mCache->resize(TEMP_SIZE);
  mCache->prefetch(resources, 5, mFallbackLoader);

  // These should be cached.
  expectCacheHit({C, B});

  // This shouldn't.
  expectCacheMiss({Z});
}

TEST_F(ResourceInMemoryCacheTest, PrefetchCacheHit) {
  mCache->resize(A.size + B.size + C.size + D.size);
  expectCacheMiss({A, B, C, D});
  // ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
  // ┃ offset:      0 ┃ offset:     64 ┃ offset:    320 ┃ offset:    832 ┃
  // ┃ size:       64 ┃ size:      256 ┃ size:      512 ┃ size:     1024 ┃
  // ┃ id:          A ┃ id:          B ┃ id:          C ┃ id:          D ┃
  // ┃ head           ┃                ┃                ┃                ┃
  // ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┛
  std::vector<Resource> resources{A, B, C, D};
  mCache->prefetch(resources.data(), 4, mFallbackLoader);
  // ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
  // ┃ offset:      0 ┃ offset:     64 ┃ offset:    320 ┃ offset:    832 ┃
  // ┃ size:       64 ┃ size:      256 ┃ size:      512 ┃ size:     1024 ┃
  // ┃ id:          A ┃ id:          B ┃ id:          C ┃ id:          D ┃
  // ┃ head           ┃                ┃                ┃                ┃
  // ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┛
  expectCacheHit({A, B, C, D});
}

TEST_F(ResourceInMemoryCacheTest, PrefetchPartialCacheHit) {
  InSequence x;

  mCache->resize(A.size + B.size + C.size + D.size);
  expectCacheMiss({A, C});
  // ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
  // ┃ offset:      0 ┃ offset:     64 ┃ offset:    320 ┃ offset:    832 ┃
  // ┃ size:       64 ┃ size:      256 ┃ size:      512 ┃ size:     1024 ┃
  // ┃ id:          A ┃ id:          B ┃ id:          C ┃ id:          D ┃
  // ┃ head           ┃                ┃                ┃                ┃
  // ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┛
  std::vector<Resource> resources{A, B, C, D};
  EXPECT_CALL(*mFallbackLoader, fetch(_, _))
      .With(Args<0, 1>(ElementsAre(B, D)))
      .WillOnce(Return(
          ByMove(std::move(createResources(createResourcesData({B, D}))))));
  EXPECT_TRUE(
      mMemoryCachedResourceLoader->load(resources.data(), 4, mTemp, TEMP_SIZE));
  // ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
  // ┃ offset:      0 ┃ offset:     64 ┃ offset:    320 ┃ offset:    832 ┃
  // ┃ size:       64 ┃ size:      256 ┃ size:      512 ┃ size:     1024 ┃
  // ┃ id:          A ┃ id:          B ┃ id:          C ┃ id:          D ┃
  // ┃ head           ┃                ┃                ┃                ┃
  // ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┛
  expectCacheHit({A, B, C, D});
}

TEST_F(ResourceInMemoryCacheTest, PrefetchPartialCacheHitWithWrapped) {
  InSequence x;

  mCache->resize(B.size + C.size + D.size);
  expectCacheMiss({C, B, A, D});
  // ┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
  // ┃ offset:     64 ┃ offset:    512 ┃ offset:    768 ┃ offset:    832 ┃
  // ┃ size:      448 ┃ size:      256 ┃ size:       64 ┃ size:     1024 ┃
  // ┃ free           ┃ id:          B ┃ id:          A ┃ id:          D ┃
  // ┃ head           ┃                ┃                ┃                ┃
  // ┗━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━┛
  Resource resources[] = {B, C, D};
  EXPECT_CALL(*mFallbackLoader, fetch(_, _))
      .With(Args<0, 1>(ElementsAre(C)))
      .WillOnce(
          Return(ByMove(std::move(createResources(createResourcesData({C}))))));
  mMemoryCachedResourceLoader->load(resources, 3, mTemp, TEMP_SIZE);
  // ┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━┳━━━━━━━━━━━━━┳━━━━━━━━━━━━━━┓
  // ┃ offset:  64 ┃ offset: 576 ┃ offset: 768 ┃ offset:  832 ┃
  // ┃ size:   512 ┃ size:   192 ┃ size:    64 ┃ size:   1024 ┃
  // ┃ id:       C ┃ free        ┃ id:       A ┃ id:        D ┃
  // ┃             ┃ head        ┃             ┃              ┃
  // ┗━━━━━━━━━━━━━┻━━━━━━━━━━━━━┻━━━━━━━━━━━━━┻━━━━━━━━━━━━━━┛
  expectCacheHit({C, A, D});
}

TEST_F(ResourceInMemoryCacheTest, Resize) {
  InSequence x;

  mMemoryCachedResourceLoader->getCache()->resize(D.size / 2);

  // D is too big to fit in the cache.
  expectCacheMiss({D});
  expectCacheMiss({D});

  mCache->resize(D.size);
  expectCacheMiss({D});
  expectCacheHit({D});  // Now should be big enough to hold D.

  // Same size. Should be an effective no-op.
  mCache->resize(D.size);
  expectCacheHit({D});

  // Expand the buffer to also include space for C.
  mCache->resize(D.size + C.size);
  expectCacheHit({D});
  expectCacheMiss({C});
  expectCacheHit({D, C});

  // Reduce the buffer so that it can no longer fit C.
  mCache->resize(D.size + B.size);
  expectCacheHit({D});
  expectCacheMiss({C});

  // Reduce the buffer so that it is empty.
  mCache->resize(0);
  expectCacheMiss({D, C});

  // Grow the buffer to hold A, B, C, D, E and a bit of space.
  mCache->resize(A.size + B.size + C.size + D.size + E.size + 10);
  expectCacheMiss({A, B, C, D, E});
  expectCacheHit({A, B, C, D, E});

  // Reduce the buffer so that it can fit A, B and a bit of space.
  mCache->resize(A.size + B.size + 10);
  expectCacheHit({A, B});
  expectCacheMiss({C, D, E});
}

TEST_F(ResourceInMemoryCacheTest, CachingLogic) {
  InSequence x;

  Resource A1("A1", 1), B1("B1", 1), C1("C1", 1), D1("D1", 1);
  Resource E1("E1", 1), F1("F1", 1), G1("G1", 1), H1("H1", 1);
  Resource A2("A2", 2), B2("B2", 2), C2("C2", 2), D2("D2", 2);
  mMemoryCachedResourceLoader->getCache()->resize(8);
  // ┏━━━━━━━━━━━━━━━━┓
  // ┃ offset:      0 ┃
  // ┃ size:        8 ┃
  // ┃ free           ┃
  // ┃ head           ┃
  // ┗━━━━━━━━━━━━━━━━┛
  {
    SCOPED_TRACE(".1");
    expectCacheMiss({A1, B1, C1, D1, E1, F1, G1, H1});
    expectCacheHit({A1, B1, C1, D1, E1, F1, G1, H1});
  }
  // clang-format off
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  0 ┃ offset:  1 ┃ offset:  2 ┃ offset:  3 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃
  // ┃ id:     A1 ┃ id:     B1 ┃ id:     C1 ┃ id:     D1 ┃ id:     E1 ┃ id:     F1 ┃ id:     G1 ┃ id:     H1 ┃
  // ┃ head       ┃            ┃            ┃            ┃            ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".2");
    expectCacheMiss({A2, B2});
    expectCacheHit({A2, B2, E1, F1, G1, H1});
  }
  // clang-format off
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  0 ┃ offset:  2 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:    2 ┃ size:    2 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃
  // ┃ id:     A2 ┃ id:     B2 ┃ id:     E1 ┃ id:     F1 ┃ id:     G1 ┃ id:     H1 ┃
  // ┃            ┃            ┃ head       ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".3");
    expectCacheMiss({A1, B1, C1});
    expectCacheHit({A1, B1, C1, A2, B2, H1});
  }
  // clang-format off
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  0 ┃ offset:  2 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:    2 ┃ size:    2 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃
  // ┃ id:     A2 ┃ id:     B2 ┃ id:     A1 ┃ id:     B1 ┃ id:     C1 ┃ id:     H1 ┃
  // ┃            ┃            ┃            ┃            ┃            ┃ head       ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".4");
    expectCacheMiss({C2});
    expectCacheHit({A1, B1, C1, B2, C2});
  }
  // clang-format off
  // ┏━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset: 1 ┃ offset:  2 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:   1 ┃ size:    2 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    2 ┃
  // ┃ free      ┃ id:     B2 ┃ id:     A1 ┃ id:     B1 ┃ id:     C1 ┃ id:     C2 ┃
  // ┃ head      ┃            ┃            ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".5");
    expectCacheMiss({D1});
    expectCacheHit({B2, A1, B1, C1, C2, D1});
  }
  // clang-format off
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  1 ┃ offset:  2 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:    1 ┃ size:    2 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    2 ┃
  // ┃ id:     D1 ┃ id:     B2 ┃ id:     A1 ┃ id:     B1 ┃ id:     C1 ┃ id:     C2 ┃
  // ┃            ┃ head       ┃            ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".6");
    Resource resources[] = {A1, B1, C1, D1, E1};
    EXPECT_CALL(*mFallbackLoader, fetch(_, _))
        .With(Args<0, 1>(ElementsAre(E1)))
        .WillOnce(Return(
            ByMove(std::move(createResources(createResourcesData({E1}))))));
    mMemoryCachedResourceLoader->load(resources, 5, mTemp, TEMP_SIZE);
    expectCacheHit({A1, B1, C1, D1, E1});
  }
  // clang-format off
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  1 ┃ offset:  2 ┃ offset: 3 ┃ offset:  4 ┃ offset:  5 ┃ offset:  6 ┃ offset:  7 ┃
  // ┃ size:    1 ┃ size:    1 ┃ size:   1 ┃ size:    1 ┃ size:    1 ┃ size:    1 ┃ size:    2 ┃
  // ┃ id:     D1 ┃ id:     E1 ┃ free      ┃ id:     A1 ┃ id:     B1 ┃ id:     C1 ┃ id:     C2 ┃
  // ┃            ┃            ┃ head      ┃            ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  // clang-format on
  {
    SCOPED_TRACE(".7");
    expectCacheMiss({A2, B2});
    expectCacheHit({D1, E1, A2, B2, C2});
  }
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  1 ┃ offset:  2 ┃ offset:  3 ┃ offset:  5 ┃ offset:  7 ┃
  // ┃ size:    1 ┃ size:    1 ┃ size:    2 ┃ size:    2 ┃ size:    2 ┃
  // ┃ id:     D1 ┃ id:     E1 ┃ id:     A2 ┃ id:     B2 ┃ id:     C2 ┃
  // ┃ head       ┃            ┃            ┃            ┃ head       ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  {
    Resource resources[] = {A1, C2, D2};
    EXPECT_CALL(*mFallbackLoader, fetch(_, _))
        .With(Args<0, 1>(ElementsAre(A1, D2)))
        .WillOnce(Return(
            ByMove(std::move(createResources(createResourcesData({A1, D2}))))));
    mMemoryCachedResourceLoader->load(resources, 3, mTemp, TEMP_SIZE);
  }
  // ┏━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━┓
  // ┃ offset:  0 ┃ offset:  2 ┃ offset:  3 ┃ offset:  5 ┃ offset:  7 ┃
  // ┃ size:    2 ┃ size:    1 ┃ size:    2 ┃ size:    2 ┃ size:    1 ┃
  // ┃ id:     D2 ┃ id:     E1 ┃ id:     A2 ┃ id:     B2 ┃ id:     A1 ┃
  // ┃            ┃ head       ┃            ┃            ┃            ┃
  // ┗━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━━┛
  {
    SCOPED_TRACE(".8");
    expectCacheHit({D2, E1, A2, B2, A1});
  }
}

TEST_F(ResourceInMemoryCacheTest, PrefecthOverrun) {
  InSequence x;

  Resource A1("A1", 1), B1("B1", 1), C1("C1", 1), D1("D1", 1);
  Resource E1("E1", 1), F1("F1", 1), G1("G1", 1), H1("H1", 1);
  Resource A2("A2", 2), B2("B2", 2), C2("C2", 2), D2("D2", 2);
  mCache->resize(8);

  std::vector<Resource> resources1{A1, B1, C1, D1, E1};
  EXPECT_CALL(*mFallbackLoader, fetch(_, _))
      .With(Args<0, 1>(ElementsAre(A1, B1, C1, D1, E1)))
      .WillOnce(Return(
          ByMove(std::move(createResources(createResourcesData(resources1))))));
  mMemoryCachedResourceLoader->load(resources1.data(), 5, mTemp, TEMP_SIZE);

  // ┏━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┓
  // ┃  A1  ┃  B1  ┃  C1  ┃  D1  ┃  E1  ┃      ┃      ┃      ┃
  // ┗━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┛

  expectCacheHit({A1, B1, C1, D1, E1});
  expectCacheMiss({A2, B2});

  // ┏━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┳━━━━━━┓
  // ┃  B2  ┃  B1  ┃  C1  ┃  D1  ┃  E1  ┃     A2      ┃  B2  ┃
  // ┗━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┻━━━━━━┛

  expectCacheHit({B2, B1, C1, D1, E1, A2});
  expectCacheMiss({A1});
}

}  // namespace test
}  // namespace gapir
