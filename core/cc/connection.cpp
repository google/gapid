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

#include "connection.h"

#include <cstring>
#include <string>

namespace core {

bool Connection::sendString(const std::string& s) {
  const auto length = static_cast<size_t>(s.size() + 1u);
  return this->send(s.c_str(), length) == length;
}

// This handles nullptr, since the overload above would crash on the implicit
// conversion.
bool Connection::sendString(const char* s) {
  if (s == nullptr) {
    s = "";
  }
  const auto length = strlen(s) + 1u;
  return this->send(s, length) == length;
}

bool Connection::readString(std::string* s) {
  char c;
  s->clear();
  while (true) {
    if (this->recv(&c, 1) != 1) {
      return false;
    }
    if (c == 0) {
      return true;
    }
    s->push_back(c);
  }
}

}  // namespace core
