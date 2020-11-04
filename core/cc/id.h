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

#ifndef CORE_ID_H
#define CORE_ID_H

#include <stdint.h>
#include <cstring>
#include <functional>
#include <string>

namespace core {

// Id is a 20-byte unique identifier.
struct Id {
  // Construct an Id with the hash of the given memory address.
  static Id Hash(const void* ptr, uint64_t size);

  bool operator==(const Id& rhs) const;

  inline operator uint8_t*();
  inline operator const uint8_t*() const;

  std::string string() const;

  uint8_t data[20];
};

inline Id::operator uint8_t*() { return data; }

inline Id::operator const uint8_t*() const { return data; }

}  // namespace core

namespace std {

template <>
struct hash<core::Id> {
  inline size_t operator()(const core::Id& id) const {
    size_t hash;
    memcpy(&hash, id.data, sizeof(hash));
    return hash;
  }
};

}  // namespace std

#endif  // CORE_ID_H
