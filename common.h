/*
 * Copyright (C) 2022 Google Inc.
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

#pragma once
#include <array>
#include <cassert>
#include <cstdio>
#include <iostream>
#include <memory>
#include <stdexcept>
#include <string>

#define GAPID2_ERROR(x) assert(0 == x)
#define GAPID2_ASSERT(x, y) \
  if (!(x)) {               \
    GAPID2_ERROR(y);        \
  }
#define GAPID2_WARNING(y)          \
  OutputDebugStringA("Warning: "); \
  OutputDebugStringA(y);           \
  OutputDebugStringA("\n")

struct block {
  size_t size = 0;
  char* data = 0;
  size_t left = 0;
};

inline int run_system(const char* cmd, std::string& out) {
  std::array<char, 128> buffer;
  FILE* pipe = _popen(cmd, "r");
  if (!pipe) {
    throw std::runtime_error("popen() failed!");
  }
  while (fgets(buffer.data(), buffer.size(), pipe) != nullptr) {
    out += buffer.data();
  }

  return _pclose(pipe);
}

enum class message_type {
  debug = 0,
  info = 1,
  error = 2,
  critical = 3,
  object = 4
};

void output_message(message_type type, const std::string& str, uint32_t layer_index = static_cast<uint32_t>(-1));
void send_layer_data(const char* str, size_t length, uint64_t layer_index);
void send_layer_log(message_type type, const char* str, size_t length, uint64_t layer_index);