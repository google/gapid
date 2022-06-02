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

String::Allocation String::EMPTY = {1, nullptr, 0, {0}};

String::Allocation* String::make_allocation(core::Arena* arena, uint64_t length,
                                            const void* data) {
  String::Allocation* str = reinterpret_cast<String::Allocation*>(
      arena->allocate(sizeof(String::Allocation) + length + 1, 1));
  str->arena = arena;
  str->ref_count = 1;
  str->length = length;

  if (data != nullptr) {
    memcpy(str->data, data, length);
    str->data[length] = 0;
  } else {
    memset(str->data, 0, length + 1);
  }
  return str;
}

int32_t String::compare(String::Allocation* a, String::Allocation* b) {
  if (a == b) {
    return 0;
  }
  return strncmp(reinterpret_cast<const char*>(a->data),
                 reinterpret_cast<const char*>(b->data),
                 std::max(a->length, b->length));
}

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
  ptr = make_allocation(arena, len, s);
}

String::String(core::Arena* arena, std::initializer_list<char> s) {
  ptr = make_allocation(arena, s.size(), s.begin());
}

String::String(core::Arena* arena, const char* start, const char* end) {
  auto len = end - start;
  ptr = make_allocation(arena, len, start);
}

String::String(core::Arena* arena, const char* s, size_t len) {
  ptr = make_allocation(arena, len, s);
}

String::String(String::Allocation* p) : ptr(p) {}

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
  return compare(ptr, other.ptr) == 0;
}

bool String::operator!=(const String& other) const {
  return compare(ptr, other.ptr) != 0;
}

bool String::operator<(const String& other) const {
  return compare(ptr, other.ptr) < 0;
}

bool String::operator<=(const String& other) const {
  return compare(ptr, other.ptr) <= 0;
}

bool String::operator>(const String& other) const {
  return compare(ptr, other.ptr) > 0;
}

bool String::operator>=(const String& other) const {
  return compare(ptr, other.ptr) >= 0;
}

String::operator bool() const { return ptr->length; }

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
    ptr->arena->free(ptr);
  }
  ptr = nullptr;
}

void String::reference() {
  GAPID_ASSERT_MSG(ptr->ref_count > 0,
                   "attempting to reference freed string (%s)", ptr->data);
  ptr->ref_count++;
}

}  // namespace gapil
