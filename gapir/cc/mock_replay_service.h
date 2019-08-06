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

#include "grpc_replay_service.h"
#include "resource.h"

#include <memory>
#include <string>

#include <gmock/gmock.h>

namespace gapir {
namespace test {
class MockReplayService : public GrpcReplayService {
 public:
  MockReplayService() : GrpcReplayService(nullptr) {}
  MOCK_METHOD1(getPayload,
               std::unique_ptr<ReplayService::Payload>(const std::string&));
  MOCK_METHOD2(getResources, std::unique_ptr<ReplayService::Resources>(
                                 const Resource* resources, size_t resSize));
  MOCK_METHOD1(mockedSendPosts, bool(ReplayService::Posts*));
  bool sendPosts(std::unique_ptr<ReplayService::Posts> posts) override {
    return mockedSendPosts(posts.get());
  }
  MOCK_METHOD7(sendErrorMsg, bool(uint64_t, uint32_t, uint32_t, uint64_t,
                                  const std::string&, const void*, uint32_t));
  MOCK_METHOD3(sendReplayStatus, bool(uint64_t, uint32_t, uint32_t));
};
}  // namespace test
}  // namespace gapir

#endif  // GAPIR_MOCK_REPLAY_CONNECTION_H
