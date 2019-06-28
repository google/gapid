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

#include <gmock/gmock.h>
#include <string.h>

#include <memory>
#include <queue>
#include <string>
#include <vector>

#include "core/cc/connection.h"

namespace core {
namespace test {

class MockConnection : public Connection {
 public:
  MockConnection() : read_pos(0), out_limit(-1) {}
  virtual size_t send(const void* data, size_t size) override {
    if ((out_limit >= 0) && (size > out_limit - out.size())) {
      size = out_limit - out.size();
    }
    out.insert(out.end(), (char*)data, (char*)data + size);
    return size;
  }
  virtual size_t recv(void* data, size_t size) override {
    if (size > in.size() - read_pos) {
      size = in.size() - read_pos;
    }
    memcpy(data, &in[read_pos], size);
    read_pos += size;
    return size;
  }

  const char* error() override { return ""; }
  std::unique_ptr<Connection> accept(int timeoutMs) override {
    if (connections.size() == 0) {
      return nullptr;
    }
    auto conn = connections.front();
    connections.pop();
    return std::unique_ptr<Connection>(conn);
  }
  void close() override {}

  std::queue<Connection*> connections;
  std::vector<uint8_t> in;
  int read_pos;
  std::vector<uint8_t> out;
  int out_limit;
};

}  // namespace test
}  // namespace core

#endif  // CORE_MOCK_CONNECTION_H
