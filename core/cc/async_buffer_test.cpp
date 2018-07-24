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

#include "async_buffer.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <chrono>
#include <string>

using ::testing::_;
using ::testing::DoAll;
using ::testing::ElementsAre;
using ::testing::ElementsAreArray;
using ::testing::StrictMock;
using ::testing::WithArg;

namespace core {
namespace test {
namespace {

class Sink : public StreamWriter {
 public:
  Sink(std::vector<uint8_t>* buf) : buf_(buf) {}

  virtual uint64_t write(const void* data, uint64_t size) {
    std::this_thread::sleep_for(std::chrono::milliseconds(1));
    auto bytes = reinterpret_cast<const uint8_t*>(data);
    buf_->insert(buf_->end(), bytes, bytes + size);
    return size;
  }

  std::vector<uint8_t>* buf_;
};

class AsyncBufferTest : public ::testing::TestWithParam<size_t> {
 protected:
  void init(size_t buffer_size) {
    std::unique_ptr<Sink> sink(new Sink(&output_));
    buffer_.reset(new AsyncBuffer(std::move(sink), buffer_size));
  }

  virtual void SetUp() override {
    size_t size = 1 << 10;
    test_data_.reserve(size);
    uint32_t h = 0x3e20b923;
    for (size_t i = 0; i < size; i++) {
      test_data_.push_back(static_cast<uint8_t>(h));
      h = (h * 31) ^ h;
    }
  }

  std::vector<uint8_t> test_data_;
  std::vector<uint8_t> output_;
  std::unique_ptr<AsyncBuffer> buffer_;
};

}  // anonymous namespace

INSTANTIATE_TEST_CASE_P(buffer_sizes, AsyncBufferTest,
                        ::testing::Values(1, 10, 100, 1000, 10000));

TEST_P(AsyncBufferTest, Write) {
  init(GetParam());
  auto written = buffer_->write(&test_data_[0], test_data_.size());
  EXPECT_EQ(test_data_.size(), written);
  buffer_->flush();
  EXPECT_THAT(output_, ElementsAreArray(test_data_.begin(), test_data_.end()));
}

}  // namespace test
}  // namespace core
