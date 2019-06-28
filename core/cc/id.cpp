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

#include "id.h"

#include <city.h>
#include <string.h>

#include <iomanip>
#include <sstream>
#include <string>

namespace {

void hash(const void* ptr, uint64_t size, core::Id& out) {
  auto buf = reinterpret_cast<const char*>(ptr);
  auto hash = CityHash128(buf, static_cast<size_t>(size));
  memcpy(&out.data[0], &hash, 16);
  auto len = static_cast<uint32_t>(size);
  memcpy(&out.data[16], &len, 4);
}

}  // anonymous namespace

namespace core {

Id Id::Hash(const void* ptr, uint64_t size) {
  Id id;
  hash(ptr, size, id);
  return id;
}

bool Id::operator==(const Id& rhs) const {
  return memcmp(data, rhs.data, sizeof(data)) == 0;
}

std::string Id::string() const {
  std::stringstream ss;
  ss << "0x";
  ss << std::setfill('0') << std::hex;
  for (int i = 0; i < 20; ++i) {
    ss << std::setw(2) << (unsigned int)data[i];
  }
  return ss.str();
}

}  // namespace core
