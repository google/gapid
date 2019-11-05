/*
 * Copyright (C) 2019 Google Inc.
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

#include <gtest/gtest.h>

#include "memory_allocator.h"

using namespace gapir;

#define ALLOCATOR_SIZE 2048

void TEST_ALLOCATIONS(const std::vector<MemoryAllocator::Handle>& allocations,
                      const std::vector<size_t>& allocationSizes,
                      const std::vector<bool>& expectPass) {
  EXPECT_EQ(allocations.size(), allocationSizes.size());
  EXPECT_EQ(allocations.size(), expectPass.size());

  for (unsigned int i = 0; i < allocations.size(); ++i) {
    EXPECT_EQ(allocations[i] != nullptr, expectPass[i]);

    for (unsigned int j = 0; j < allocations.size(); ++j) {
      if (allocations[j] != nullptr) {
        for (unsigned int k = 0; k < allocationSizes[j]; ++k) {
          allocations[j][k] = 0;
        }
      }
    }

    if (allocations[i] != nullptr) {
      for (unsigned int k = 0; k < allocationSizes[i]; ++k) {
        allocations[i][k] = 255;
      }
    }

    bool passedContaminationCheck = true;

    for (unsigned int j = 0; j < allocations.size(); ++j) {
      if (allocations[j] != nullptr) {
        for (unsigned int k = 0; k < allocationSizes[j]; ++k) {
          passedContaminationCheck &=
              (int)(j == i ? 255 : 0) == (int)allocations[j][k];
        }
      }
    }

    EXPECT_TRUE(passedContaminationCheck);
  }
}

TEST(MemoryAllocator, SimpleStaticAllocate) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);
  EXPECT_TRUE(allocator->allocateStatic(1024) != nullptr);
}

TEST(MemoryAllocator, StaticAllocateTooMuch) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);
  EXPECT_TRUE(allocator->allocateStatic(8192) == nullptr);
}

TEST(MemoryAllocator, SimpleMultipleStaticAllocate) {
  std::vector<size_t> allocationSizes = {1024, 512, 256, 128, 64, 32, 16,
                                         8,    4,   2,   1,   1,  1};
  std::vector<bool> expectPass = {true, true, true, true, true, true, true,
                                  true, true, true, true, true, false};

  std::vector<MemoryAllocator::Handle> addresses;

  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  for (unsigned int i = 0; i < allocationSizes.size(); ++i) {
    addresses.push_back(allocator->allocateStatic(allocationSizes[i]));
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);
}

TEST(MemoryAllocator, SimpleGrowStaticAllocate) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  auto alloc = allocator->allocateStatic(1024);
  EXPECT_TRUE(alloc != nullptr);

  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, ALLOCATOR_SIZE));
  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, 512));
  EXPECT_FALSE(allocator->resizeStaticAllocation(alloc, 4096));
}

TEST(MemoryAllocator, ComplexGrowStaticAllocate) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  auto alloc = allocator->allocateStatic(1024);
  EXPECT_TRUE(alloc != nullptr);

  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, ALLOCATOR_SIZE));
  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, 512));
  EXPECT_FALSE(allocator->resizeStaticAllocation(alloc, 4096));

  auto alloc2 = allocator->allocateStatic(1024);
  EXPECT_TRUE(alloc2 != nullptr);

  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, 768));
  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc2, 1280));

  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc, 256));
  EXPECT_TRUE(allocator->resizeStaticAllocation(alloc2, 256));

  EXPECT_FALSE(allocator->resizeStaticAllocation(alloc, 1024));
  EXPECT_FALSE(allocator->resizeStaticAllocation(alloc2, ALLOCATOR_SIZE));

  TEST_ALLOCATIONS({alloc, alloc2}, {256, 256}, {true, true});
}

TEST(MemoryAllocator, SimplePurgableAllocate) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 257; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 256);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);
}

TEST(MemoryAllocator, SimplePurgableAllocateAroundStatic) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  allocator->allocateStatic(1024);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 129; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 128);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);
}

TEST(MemoryAllocator, SimplePurgableAllocateAroundMultipleStatic) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  allocator->allocateStatic(1024);
  allocator->allocateStatic(512);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 65; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 64);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);
}

TEST(MemoryAllocator, SimplePurgableAllocateRelocate) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 257; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 256);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);

  for (unsigned int i = 0; i < addresses.size(); ++i) {
    if (i % 5 == 0 || i % 7 == 0 || i % 11 == 0) {
      EXPECT_TRUE(allocator->releaseAllocation(addresses[i]));
      addresses[i] = MemoryAllocator::Handle();
      expectPass[i] = false;
    }
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);

  for (unsigned int i = 0; i < 98; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 97);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);

  for (unsigned int i = 257; i < addresses.size(); ++i) {
    if (i % 13 == 0 || i % 17 == 0 || i % 19 == 0) {
      EXPECT_TRUE(allocator->releaseAllocation(addresses[i]));
      addresses[i] = MemoryAllocator::Handle();
      expectPass[i] = false;
    }
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);

  for (unsigned int i = 0; i < 18; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 17);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);
}

TEST(MemoryAllocator, SimplePurgableAllocatePurge) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 256; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 256);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  TEST_ALLOCATIONS(addresses, allocationSizes, expectPass);

  EXPECT_NE(allocator->allocateStatic(1024), nullptr);

  unsigned int purgedAllocations = 0;
  for (auto&& alloc : addresses) {
    if (alloc == nullptr) {
      purgedAllocations++;
    }
  }

  EXPECT_EQ(128, purgedAllocations);

  allocator->releaseAllocation(addresses[0]);
  addresses.push_back(allocator->allocatePurgable(8));
  allocationSizes.push_back(8);
  expectPass.push_back(true);

  purgedAllocations = 0;
  for (auto&& alloc : addresses) {
    if (alloc == nullptr) {
      purgedAllocations++;
    }
  }

  EXPECT_EQ(129, purgedAllocations);
}

TEST(MemoryAllocator, SimplePurgableAllocatePurgeViaGrow) {
  std::unique_ptr<MemoryAllocator> allocator =
      MemoryAllocator::create(ALLOCATOR_SIZE);

  std::vector<size_t> allocationSizes;
  std::vector<bool> expectPass;

  std::vector<MemoryAllocator::Handle> addresses;

  for (unsigned int i = 0; i < 256; ++i) {
    allocationSizes.push_back(8);
    expectPass.push_back(i < 256);

    auto alloc = allocator->allocatePurgable(8);
    addresses.push_back(alloc);
  }

  auto alloc = allocator->allocateStatic(1024);
  EXPECT_NE(alloc, nullptr);

  allocator->resizeStaticAllocation(alloc, 1536);

  allocator->resizeStaticAllocation(alloc, 1024);
}
