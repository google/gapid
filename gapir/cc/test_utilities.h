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

#ifndef GAPIR_TEST_UTILITIES_H
#define GAPIR_TEST_UTILITIES_H

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "base_type.h"
#include "interpreter.h"
#include "replay_service.h"
#include "resource.h"

namespace gapir {

class ServerConnection;

namespace test {

// Action for setting the values pointed by a void pointer (arg0). Data should
// be an iterable collection of uint8_t types and arg should be a void pointer.
ACTION_P(SetVoidPointee, data) {
  uint32_t i = 0;
  uint8_t* typedArg = static_cast<uint8_t*>(arg0);
  for (uint8_t it : data) {
    typedArg[i] = it;
    ++i;
  }
}

// Create an instruction code from the given details that can be interpreted by
// the interpreter
uint32_t instruction(Interpreter::InstructionCode code);
uint32_t instruction(Interpreter::InstructionCode code, uint32_t data);
uint32_t instruction(Interpreter::InstructionCode code, BaseType type,
                     uint32_t data);

void pushBytes(std::vector<uint8_t>* buf, const std::vector<uint8_t>& v);
void pushUint8(std::vector<uint8_t>* buf, uint8_t v);
void pushUint32(std::vector<uint8_t>* buf, uint32_t v);
void pushString(std::vector<uint8_t>* buf, const std::string& str);
void pushString(std::vector<uint8_t>* buf, const char* str);

std::unique_ptr<ReplayService::Payload> createPayload(
    uint32_t stackSize, uint32_t volatileMemorySize,
    const std::vector<uint8_t>& constantMemory,
    const std::vector<Resource>& resources,
    const std::vector<uint32_t>& instructions);

std::unique_ptr<ReplayService::Resources> createResources(
    const std::vector<uint8_t>& data);

std::vector<uint8_t> createResourcesData(
    const std::vector<Resource>& resources);

}  // namespace test
}  // namespace gapir

#endif  // GAPIR_MOCK_UTILITIES_H
