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

#include "common.h"

#include <chrono>
#include <format>
#include <sstream>

#include "base64.h"
#include "json.hpp"

namespace {
struct foo {
  foo() {
    std::cout << "[";
  }

  ~foo() {
    std::cout << "]" << std::endl
              << std::flush;
  }

  void message(const std::string& s) {
    if (needs_comma) {
      std::cout << ",";
    }
    needs_comma = true;
    std::cout << s << std::endl;
    std::cout << std::flush;
  }
  bool needs_comma = false;
};
}  // namespace

void send(const std::string& str) {
  static foo f;
  f.message(str);
}

static auto begin = std::chrono::high_resolution_clock::now();

float get_time() {
  return std::chrono::duration<float>(std::chrono::high_resolution_clock::now() - begin).count();
}

void output_message(message_type type, const std::string& str, uint32_t layer_index) {
  std::string t;
  switch (type) {
    case message_type::error:
      t = "Error";
      break;
    case message_type::info:
      t = "Info";
      break;
    case message_type::critical:
      t = "Critical";
      break;
    case message_type::debug:
      t = "Debug";
      break;
  }

  std::stringstream ss;
  ss << "{ \"";
  ss << "Message\":\"";
  ss << t;
  ss << "\"";
  ss << ",\"Time\":";
  ss << get_time();
  if (layer_index != static_cast<uint32_t>(-1)) {
    ss << ", \"LayerIndex\" : " << layer_index;
  }
  ss << ", \"Content\": ";
  auto j = nlohmann::json::basic_json(str);
  ss << j.dump();
  ss << " }";
  send(ss.str());
}

void send_layer_data(const char* str, size_t length, uint64_t layer_index) {
  auto nobj = nlohmann::json::object();
  nobj["Message"] = "Object";
  nobj["LayerIndex"] = layer_index;
  nobj["Time"] = get_time();
  nobj["Content"] = nlohmann::json::parse(str, str + length);
  send(nobj.dump());
}

void send_layer_log(message_type type, const char* str, size_t length, uint64_t layer_index) {
  output_message(type, std::string(str, length), layer_index);
}