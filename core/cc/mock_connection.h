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

#ifndef CORE_MOCK_CONNECTION_H
#define CORE_MOCK_CONNECTION_H

#include "core/cc/connection.h"

#include <gmock/gmock.h>

#include <memory>
#include <queue>
#include <string>
#include <vector>

#include <string.h>

namespace core {
namespace test {

class MockConnection : public Connection {
 public:
  MockConnection() : read_pos(0u), out_limit(-1) {}

  virtual size_t send(const void* data, size_t size) override {
    if (out_limit >= 0) {
      const auto limit = static_cast<size_t>(out_limit);
      if (out.size() + size > limit) {
        size = limit > out.size() ? limit - out.size() : 0u;
      }
    }
    out.insert(out.end(), (char*)data, (char*)data + size);
    return size;
  }

  virtual size_t recv(void* data, size_t size) override {
    if (read_pos + size > in.size()) {
      size = in.size() > read_pos ? in.size() - read_pos : 0u;
    }
    memcpy(data, &in[read_pos], size);
    read_pos += size;
    return size;
  }

  const char* error() override { return ""; }
  std::unique_ptr<Connection> accept(int timeoutMs) override {
    if (connections.size() == 0u) {
      return nullptr;
    }
    auto conn = connections.front();
    connections.pop();
    return std::unique_ptr<Connection>(conn);
  }
  void close() override {}

  std::queue<Connection*> connections;
  std::vector<uint8_t> in;
  size_t read_pos;
  std::vector<uint8_t> out;
  int out_limit;
};

}  // namespace test
}  // namespace core

#endif  // CORE_MOCK_CONNECTION_H
