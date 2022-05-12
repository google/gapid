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

#include "string.h"

#include "core/cc/assert.h"
#include "core/memory/arena/cc/arena.h"

#include <cstring>

namespace gapil {

string_t String::EMPTY = {1, nullptr, 0, {0}};

String::String() {
  ptr = &EMPTY;
  reference();
}

String::String(const String& s) {
  ptr = s.ptr;
  reference();
}

String::String(String&& s) {
  ptr = s.ptr;
  s.ptr = nullptr;
}

String::String(core::Arena* arena, const char* s) {
  auto len = strlen(s);
  ptr = gapil_make_string(reinterpret_cast<arena_t*>(arena), len,
                          const_cast<char*>(s));
}

String::String(core::Arena* arena, std::initializer_list<char> s) {
  ptr = gapil_make_string(reinterpret_cast<arena_t*>(arena), s.size(),
                          const_cast<char*>(s.begin()));
}

String::String(core::Arena* arena, const char* start, const char* end) {
  auto len = end - start;
  ptr = gapil_make_string(reinterpret_cast<arena_t*>(arena), len,
                          const_cast<char*>(start));
}

String::String(core::Arena* arena, const char* s, size_t len) {
  ptr = gapil_make_string(reinterpret_cast<arena_t*>(arena), len,
                          const_cast<char*>(s));
}

String::String(string_t* p) : ptr(p) {}

String::~String() {
  if (ptr != nullptr) {  // note: nullptr is only valid in the case of a move
    release();
  }
}

String& String::operator=(const String& other) {
  GAPID_ASSERT_MSG(other.ptr->ref_count > 0,
                   "attempting to reference freed string (%s)",
                   other.ptr->data);
  if (ptr != other.ptr) {
    release();
    ptr = other.ptr;
    reference();
  }
  return *this;
}

bool String::operator==(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) == 0;
}

bool String::operator!=(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) != 0;
}

bool String::operator<(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) < 0;
}

bool String::operator<=(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) <= 0;
}

bool String::operator>(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) > 0;
}

bool String::operator>=(const String& other) const {
  return gapil_string_compare(ptr, other.ptr) >= 0;
}

size_t String::length() const { return ptr->length; }

const char* String::c_str() const { return reinterpret_cast<char*>(ptr->data); }

void String::clear() {
  release();
  ptr = &EMPTY;
  reference();
}

void String::release() {
  GAPID_ASSERT_MSG(ptr->ref_count > 0,
                   "attempting to release freed string (%s)", ptr->data);
  ptr->ref_count--;
  if (ptr->ref_count == 0) {
    gapil_free_string(ptr);
  }
  ptr = nullptr;
}

void String::reference() {
  GAPID_ASSERT_MSG(ptr->ref_count > 0,
                   "attempting to reference freed string (%s)", ptr->data);
  ptr->ref_count++;
}

}  // namespace gapil
