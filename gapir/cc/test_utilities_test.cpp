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

#include "base_type.h"
#include "interpreter.h"
#include "server_connection.h"
#include "test_utilities.h"

#include "core/cc/mock_connection.h"

#include <gmock/gmock.h>

#include <memory>
#include <string>
#include <vector>

using ::testing::_;
using ::testing::DoAll;
using ::testing::NotNull;
using ::testing::Return;
using ::testing::ReturnArg;
using ::testing::StrictMock;
using ::testing::WithArg;

namespace gapir {
namespace test {

uint32_t instruction(Interpreter::InstructionCode code) {
    return static_cast<uint32_t>(code) << 26;
}

uint32_t instruction(Interpreter::InstructionCode code, uint32_t data) {
    return (static_cast<uint32_t>(code) << 26) | (data & 0x03ffffffU);
}

uint32_t instruction(Interpreter::InstructionCode code, BaseType type, uint32_t data) {
    return (static_cast<uint32_t>(code) << 26) | (static_cast<uint32_t>(type) << 20) |
           (data & 0x000fffff);
}

void pushBytes(std::vector<uint8_t>* buf, const std::vector<uint8_t>& v) {
  buf->insert(buf->end(), v.begin(), v.end());
}
void pushUint8(std::vector<uint8_t>* buf, uint8_t v) {
  buf->push_back(v);
}
void pushUint32(std::vector<uint8_t>* buf, uint32_t v) {
    for (uint8_t i = 0; i < 32; i += 8) {
      buf->push_back((v >> i) & 0xff);
  }
}
void pushString(std::vector<uint8_t>* buf, const std::string& str) {
  for(char c : str) {
      buf->push_back(c);
  }
  buf->push_back(0);
}
void pushString(std::vector<uint8_t>* buf, const char* str) {
  for(char c = *str; c != 0; str++, c = *str) {
      buf->push_back(c);
  }
  buf->push_back(0);
}

std::vector<uint8_t> createReplayData(uint32_t stackSize, uint32_t volatileMemorySize,
                                      const std::vector<uint8_t>& constantMemory,
                                      const std::vector<Resource>& resources,
                                      const std::vector<uint32_t>& instructions) {
    std::vector<uint8_t> replayData;
    pushUint32(&replayData, stackSize);
    pushUint32(&replayData, volatileMemorySize);
    pushUint32(&replayData, constantMemory.size());
    pushBytes(&replayData, constantMemory);
    pushUint32(&replayData, resources.size());
    for (auto& it : resources) {
        pushString(&replayData, it.id);
        pushUint32(&replayData, it.size);
    }
    pushUint32(&replayData, instructions.size() * sizeof(uint32_t));
    for (auto it : instructions) {
        pushUint32(&replayData, it);
    }
    return replayData;
}

std::unique_ptr<ServerConnection> createServerConnection(core::test::MockConnection* connection,
                                                         const std::string& replayId,
                                                         uint32_t replayLength) {
    pushString(&connection->in, replayId);
    pushUint32(&connection->in, replayLength);

    std::unique_ptr<ServerConnection> server =
        ServerConnection::create(std::unique_ptr<core::Connection>(connection));

    EXPECT_THAT(server, NotNull());

    return std::move(server);
}

std::unique_ptr<ServerConnection> createServerConnection(const std::string& replayId,
                                                         uint32_t replayLength) {
    return createServerConnection(new core::test::MockConnection(), replayId, replayLength);
}

}  // namespace test
}  // namespace gapir
