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

#include "mru_cache.h"

#include <string>

#include <gmock/gmock.h>
#include <gtest/gtest.h>

using ::testing::ElementsAre;
using ::testing::ElementsAreArray;

namespace core {
namespace test {

class MRUCacheTest : public ::testing::Test {
public:
};

TEST_F(MRUCacheTest, Empty) {
    std::string tmp;
    MRUCache<std::string, std::string> cache(16);
    EXPECT_THAT(cache.size(), 0);
    EXPECT_THAT(cache.capacity(), 16);
    EXPECT_THAT(cache.get("foo", tmp), false);
}

TEST_F(MRUCacheTest, Single) {
    MRUCache<std::string, std::string> cache(16);

    cache.add("key", "value");
    EXPECT_THAT(cache.size(), 1);
    EXPECT_THAT(cache.capacity(), 16);

    std::string got;
    EXPECT_THAT(cache.get("key", got), true);
    EXPECT_THAT(got, "value");
}

TEST_F(MRUCacheTest, Filled) {
    MRUCache<std::string, std::string> cache(4);

    cache.add("keyA", "valueA");
    cache.add("keyB", "valueB");
    cache.add("keyC", "valueC");
    cache.add("keyD", "valueD");
    EXPECT_THAT(cache.size(), 4);
    EXPECT_THAT(cache.capacity(), 4);

    std::string got;
    EXPECT_THAT(cache.get("keyA", got), true);
    EXPECT_THAT(got, "valueA");
    EXPECT_THAT(cache.get("keyB", got), true);
    EXPECT_THAT(got, "valueB");
    EXPECT_THAT(cache.get("keyC", got), true);
    EXPECT_THAT(got, "valueC");
    EXPECT_THAT(cache.get("keyD", got), true);
    EXPECT_THAT(got, "valueD");
}

TEST_F(MRUCacheTest, Spill) {
    MRUCache<std::string, std::string> cache(4);

    cache.add("keyA", "valueA");
    cache.add("keyB", "valueB");
    cache.add("keyC", "valueC");
    cache.add("keyD", "valueD");
    cache.add("keyE", "valueE");
    cache.add("keyF", "valueF");
    EXPECT_THAT(cache.size(), 4);
    EXPECT_THAT(cache.capacity(), 4);

    std::string got;
    EXPECT_THAT(cache.get("keyA", got), false);
    EXPECT_THAT(got, "");
    EXPECT_THAT(cache.get("keyB", got), false);
    EXPECT_THAT(got, "");
    EXPECT_THAT(cache.get("keyC", got), true);
    EXPECT_THAT(got, "valueC");
    EXPECT_THAT(cache.get("keyD", got), true);
    EXPECT_THAT(got, "valueD");
    EXPECT_THAT(cache.get("keyE", got), true);
    EXPECT_THAT(got, "valueE");
    EXPECT_THAT(cache.get("keyF", got), true);
    EXPECT_THAT(got, "valueF");
}

TEST_F(MRUCacheTest, Replace) {
    MRUCache<std::string, std::string> cache(4);

    cache.add("keyA", "valueA");
    cache.add("keyB", "valueB");
    cache.add("keyC", "valueC");
    cache.add("keyB", "valueB2");
    cache.add("keyD", "valueD");
    cache.add("keyE", "valueE");
    cache.add("keyB", "valueB3");
    cache.add("keyF", "valueF");
    EXPECT_THAT(cache.size(), 4);
    EXPECT_THAT(cache.capacity(), 4);

    std::string got;
    EXPECT_THAT(cache.get("keyA", got), false);
    EXPECT_THAT(got, "");

    got = "";
    EXPECT_THAT(cache.get("keyB", got), true);
    EXPECT_THAT(got, "valueB3");

    got = "";
    EXPECT_THAT(cache.get("keyC", got), false);
    EXPECT_THAT(got, "");

    got = "";
    EXPECT_THAT(cache.get("keyD", got), true);
    EXPECT_THAT(got, "valueD");

    got = "";
    EXPECT_THAT(cache.get("keyE", got), true);
    EXPECT_THAT(got, "valueE");

    got = "";
    EXPECT_THAT(cache.get("keyF", got), true);
    EXPECT_THAT(got, "valueF");
}

TEST_F(MRUCacheTest, Clear) {
    MRUCache<std::string, std::string> cache(16);

    cache.add("keyA", "valueA");
    cache.add("keyB", "valueB");
    cache.add("keyC", "valueC");
    cache.clear();
    EXPECT_THAT(cache.size(), 0);
    EXPECT_THAT(cache.capacity(), 16);

    std::string got;
    EXPECT_THAT(cache.get("keyB", got), false);
}


} // namespace test
}  // namespace core
