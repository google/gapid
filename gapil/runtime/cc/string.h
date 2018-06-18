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

#ifndef __GAPIL_RUNTIME_STRING_H__
#define __GAPIL_RUNTIME_STRING_H__

#include "runtime.h"

#include <functional>

namespace core {
class Arena;
}  // namespace core

namespace gapil {

// String is a string container that is compatible with the strings produced by
// the gapil compiler. Strings hold references to their data, and several
// strings may share the same underlying data.
class String {
 public:
  // Constructs a zero length string.
  String();

  // Constructs a string which shares the data with other.
  String(const String& other);

  // Constructs a new string with the given data.
  String(core::Arena* arena, const char*);
  String(core::Arena* arena, std::initializer_list<char>);
  String(core::Arena* arena, const char* start, const char* end);
  String(core::Arena* arena, const char* start, size_t len);

  String(String&&);
  ~String();

  // Makes this string refer to the RHS string.
  String& operator=(const String&);

  // Appends data to this string.
  String& operator+=(const String&);

  // Comparison operators. Strings are compared using their underlying data.
  bool operator==(const String& other) const;
  bool operator!=(const String& other) const;
  bool operator<(const String& other) const;
  bool operator<=(const String& other) const;
  bool operator>(const String& other) const;
  bool operator>=(const String& other) const;

  // Returns the length of the string in bytes.
  size_t length() const;

  // Returns a c-style string.
  const char* c_str() const;

  // Sets this string to a zero length string.
  void clear();

  // Returns the arena that owns this string's underlying data.
  inline core::Arena* arena() const;

 private:
  static string_t EMPTY;

  String(string_t*);

  void reference();
  void release();

  string_t* ptr;
};

inline core::Arena* String::arena() const {
  return reinterpret_cast<core::Arena*>(ptr->arena);
}

}  // namespace gapil

namespace std {
template <>
struct hash<gapil::String> {
  typedef gapil::String argument_type;
  typedef std::size_t result_type;
  result_type operator()(const argument_type& s) const noexcept {
    auto len = s.length();
    result_type hash = 0x32980321;
    if (auto p = s.c_str()) {
      for (size_t i = 0; i < len; i++) {
        hash = hash * 33 ^ p[i];
      }
    }
    return hash;
  }
};

}  // namespace std

#endif  // __GAPIL_RUNTIME_STRING_H__