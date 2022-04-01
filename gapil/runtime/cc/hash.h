// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef __GAPIL_RUNTIME_HASH_H__
#define __GAPIL_RUNTIME_HASH_H__

#include <cstdint>
#include <cstring>

#include "core/cc/assert.h"
#include "string.h"

// Note these implementations should match map.go
namespace gapil {
struct fixedValue {
  uint64_t fixed64;
};

template <typename T, typename Enable = void>
struct hash {
  uint64_t operator()(const T& t) {
    GAPID_ASSERT_MSG(false, "unknown hash type");
    return 0;
  }
};

inline uint64_t rotate_right(const uint64_t& v, const uint64_t bits) {
  return (v >> bits) | (v << (64 - bits));
}

inline uint64_t shift_left(const uint64_t& v, const uint64_t bits) {
  return v << bits;
}

template <>
struct hash<fixedValue, void> {
  uint64_t operator()(const fixedValue& val) {
    uint64_t v = val.fixed64;
    v += shift_left(v, 21);
    v ^= rotate_right(v, 24);
    v += shift_left(v, 3) + shift_left(v, 8);
    v ^= rotate_right(v, 14);
    v += shift_left(v, 2) + shift_left(v, 4);
    v ^= rotate_right(v, 28);
    v += shift_left(v, 31);
    return v;
  }
};

template <>
struct hash<float, void> {
  uint64_t operator()(const float& f) {
    uint32_t val;
    memcpy(&val, &f, sizeof(uint32_t));
    return hash<fixedValue>()(fixedValue{static_cast<uint64_t>(val)});
  }
};

template <>
struct hash<double, void> {
  uint64_t operator()(const double& f) {
    uint64_t val;
    memcpy(&val, &f, sizeof(uint64_t));
    return hash<fixedValue>()(fixedValue{val});
  }
};

template <typename T>
struct hash<T, typename std::enable_if<std::is_integral<T>::value>::type> {
  uint64_t operator()(const T& v) { return static_cast<uint64_t>(v); }
};

template <typename T>
struct hash<T*, void> {
  uint64_t operator()(const T* ptr) {
    return static_cast<uint64_t>(reinterpret_cast<uintptr_t>(ptr) >> 2);
  }
};

template <>
struct hash<gapil::String, void> {
  uint64_t operator()(const gapil::String& str) {
    uint64_t h = 0;
    const char* c_str = str.c_str();
    for (size_t i = 0; i < str.length(); ++i) {
      h += (h << 6) + (h << 16) + c_str[i];
    }
    return h;
  }
};

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_HASH_H__
