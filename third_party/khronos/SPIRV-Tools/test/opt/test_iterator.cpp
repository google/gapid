// Copyright (c) 2016 Google Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

#include <memory>
#include <vector>

#include "gmock/gmock.h"

#include "opt/iterator.h"
#include "opt/make_unique.h"

namespace {

using namespace spvtools;
using ::testing::ContainerEq;

TEST(Iterator, IncrementDeref) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> it(&data, data.begin());
  ir::UptrVectorIterator<int> end(&data, data.end());

  EXPECT_EQ(*data[0], *it);
  for (int i = 1; i < count; ++i) {
    EXPECT_NE(end, it);
    EXPECT_EQ(*data[i], *(++it));
  }
  EXPECT_EQ(end, ++it);
}

TEST(Iterator, DecrementDeref) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> begin(&data, data.begin());
  ir::UptrVectorIterator<int> it(&data, data.end());

  for (int i = count - 1; i >= 0; --i) {
    EXPECT_NE(begin, it);
    EXPECT_EQ(*data[i], *(--it));
  }
  EXPECT_EQ(begin, it);
}

TEST(Iterator, PostIncrementDeref) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> it(&data, data.begin());
  ir::UptrVectorIterator<int> end(&data, data.end());

  for (int i = 0; i < count; ++i) {
    EXPECT_NE(end, it);
    EXPECT_EQ(*data[i], *(it++));
  }
  EXPECT_EQ(end, it);
}

TEST(Iterator, PostDecrementDeref) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> begin(&data, data.begin());
  ir::UptrVectorIterator<int> end(&data, data.end());
  ir::UptrVectorIterator<int> it(&data, data.end());

  EXPECT_EQ(end, it--);
  for (int i = count - 1; i >= 1; --i) {
    EXPECT_EQ(*data[i], *(it--));
  }
  // Decrementing .begin() is undefined behavior.
  EXPECT_EQ(*data[0], *it);
}

TEST(Iterator, Access) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> it(&data, data.begin());

  for (int i = 0; i < count; ++i) EXPECT_EQ(*data[i], it[i]);
}

TEST(Iterator, Comparison) {
  const int count = 100;
  std::vector<std::unique_ptr<int>> data;
  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
  }

  ir::UptrVectorIterator<int> it(&data, data.begin());
  ir::UptrVectorIterator<int> end(&data, data.end());

  for (int i = 0; i < count; ++i, ++it) EXPECT_TRUE(it < end);
  EXPECT_EQ(end, it);
}

TEST(Iterator, InsertBeginEnd) {
  const int count = 100;

  std::vector<std::unique_ptr<int>> data;
  std::vector<int> expected;
  std::vector<int> actual;

  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
    expected.push_back(i);
  }

  // Insert at the beginning
  expected.insert(expected.begin(), -100);
  ir::UptrVectorIterator<int> begin(&data, data.begin());
  auto insert_point = begin.InsertBefore(MakeUnique<int>(-100));
  for (int i = 0; i < count + 1; ++i) {
    actual.push_back(*(insert_point++));
  }
  EXPECT_THAT(actual, ContainerEq(expected));

  // Insert at the end
  expected.push_back(-42);
  expected.push_back(-36);
  expected.push_back(-77);
  ir::UptrVectorIterator<int> end(&data, data.end());
  end = end.InsertBefore(MakeUnique<int>(-77));
  end = end.InsertBefore(MakeUnique<int>(-36));
  end = end.InsertBefore(MakeUnique<int>(-42));

  actual.clear();
  begin = ir::UptrVectorIterator<int>(&data, data.begin());
  for (int i = 0; i < count + 4; ++i) {
    actual.push_back(*(begin++));
  }
  EXPECT_THAT(actual, ContainerEq(expected));
}

TEST(Iterator, InsertMiddle) {
  const int count = 100;

  std::vector<std::unique_ptr<int>> data;
  std::vector<int> expected;
  std::vector<int> actual;

  for (int i = 0; i < count; ++i) {
    data.emplace_back(new int(i));
    expected.push_back(i);
  }

  const int insert_pos = 42;
  expected.insert(expected.begin() + insert_pos, -100);
  expected.insert(expected.begin() + insert_pos, -42);

  ir::UptrVectorIterator<int> it(&data, data.begin());
  for (int i = 0; i < insert_pos; ++i) ++it;
  it = it.InsertBefore(MakeUnique<int>(-100));
  it = it.InsertBefore(MakeUnique<int>(-42));
  auto begin = ir::UptrVectorIterator<int>(&data, data.begin());
  for (int i = 0; i < count + 2; ++i) {
    actual.push_back(*(begin++));
  }
  EXPECT_THAT(actual, ContainerEq(expected));
}

TEST(IteratorRange, Interface) {
  const uint32_t count = 100;

  std::vector<std::unique_ptr<uint32_t>> data;

  for (uint32_t i = 0; i < count; ++i) {
    data.emplace_back(new uint32_t(i));
  }

  auto b = ir::UptrVectorIterator<uint32_t>(&data, data.begin());
  auto e = ir::UptrVectorIterator<uint32_t>(&data, data.end());
  auto range = ir::IteratorRange<decltype(b)>(b, e);

  EXPECT_EQ(b, range.begin());
  EXPECT_EQ(e, range.end());
  EXPECT_FALSE(range.empty());
  EXPECT_EQ(count, range.size());
  EXPECT_EQ(0u, *range.begin());
  EXPECT_EQ(99u, *(--range.end()));

  // IteratorRange itself is immutable.
  ++b, --e;
  EXPECT_EQ(count, range.size());
  ++range.begin(), --range.end();
  EXPECT_EQ(count, range.size());
}

}  // anonymous namespace
