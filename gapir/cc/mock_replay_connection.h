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

#ifndef GAPIR_MOCK_REPLAY_CONNECTION_H
#define GAPIR_MOCK_REPLAY_CONNECTION_H

#include "replay_connection.h"

#include <memory>
#include <string>

#include <gmock/gmock.h>

namespace gapir {
namespace test {
class MockReplayConnection : public ReplayConnection {
 public:
  MockReplayConnection() : ReplayConnection(nullptr) {}
  MOCK_METHOD0(getPayload, std::unique_ptr<ReplayConnection::Payload>());
  MOCK_METHOD1(mockedGetResources, std::unique_ptr<ReplayConnection::Resources>(
                                       ReplayConnection::ResourceRequest*));
  std::unique_ptr<ReplayConnection::Resources> getResources(
      std::unique_ptr<ReplayConnection::ResourceRequest> req) override {
    return mockedGetResources(req.get());
  }
  MOCK_METHOD1(mockedSendPostData, bool(ReplayConnection::Posts*));
  bool sendPostData(std::unique_ptr<ReplayConnection::Posts> posts) override {
    return mockedSendPostData(posts.get());
  }
  // TODO: mock sendNotification and test it once it is used.
  MOCK_METHOD7(sendNotification,
               bool(uint64_t, int, uint32_t, uint64_t, const std::string&,
                    const void*, uint32_t));
};
}  // namespace test
}  // namespace gapir

#endif  // GAPIR_MOCK_REPLAY_CONNECTION_H
