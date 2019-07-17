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

#include "test_utilities.h"
#include "base_type.h"
#include "interpreter.h"
#include "replay_service.h"
#include "resource.h"

#include "gapir/replay_service/service.pb.h"

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

uint32_t instruction(Interpreter::InstructionCode code, BaseType type,
                     uint32_t data) {
  return (static_cast<uint32_t>(code) << 26) |
         (static_cast<uint32_t>(type) << 20) | (data & 0x000fffff);
}

void pushBytes(std::vector<uint8_t>* buf, const std::vector<uint8_t>& v) {
  buf->insert(buf->end(), v.begin(), v.end());
}
void pushUint8(std::vector<uint8_t>* buf, uint8_t v) { buf->push_back(v); }
void pushUint32(std::vector<uint8_t>* buf, uint32_t v) {
  for (uint8_t i = 0; i < 32; i += 8) {
    buf->push_back((v >> i) & 0xff);
  }
}
void pushString(std::vector<uint8_t>* buf, const std::string& str) {
  for (char c : str) {
    buf->push_back(c);
  }
  buf->push_back(0);
}
void pushString(std::vector<uint8_t>* buf, const char* str) {
  for (char c = *str; c != 0; str++, c = *str) {
    buf->push_back(c);
  }
  buf->push_back(0);
}

std::unique_ptr<ReplayService::Payload> createPayload(
    uint32_t stackSize, uint32_t volatileMemorySize,
    const std::vector<uint8_t>& constantMemory,
    const std::vector<Resource>& resources,
    const std::vector<uint32_t>& instructions) {
  auto p =
      std::unique_ptr<replay_service::Payload>(new replay_service::Payload);
  p->set_stack_size(stackSize);
  p->set_volatile_memory_size(volatileMemorySize);
  p->set_constants(constantMemory.data(), constantMemory.size());
  p->set_opcodes(instructions.data(), instructions.size() * sizeof(uint32_t));
  for (size_t i = 0; i < resources.size(); i++) {
    auto* r = p->add_resources();
    r->set_id(resources[i].getID());
    r->set_size(resources[i].getSize());
  }
  return std::unique_ptr<ReplayService::Payload>(
      new ReplayService::Payload(std::move(p)));
}

std::unique_ptr<ReplayService::Resources> createResources(
    const std::vector<uint8_t>& data) {
  auto p =
      std::unique_ptr<replay_service::Resources>(new replay_service::Resources);
  p->set_data(data.data(), data.size());
  return std::unique_ptr<ReplayService::Resources>(
      new ReplayService::Resources(std::move(p)));
}

std::vector<uint8_t> createResourcesData(
    const std::vector<Resource>& resources) {
  std::vector<uint8_t> v;
  for (auto resource : resources) {
    for (size_t i = 0; i < resource.getSize(); i++) {
      v.push_back(resource.getID()[i % resource.getID().size()]);
    }
  }
  return v;
}

}  // namespace test
}  // namespace gapir
