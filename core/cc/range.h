/*
 * Copyright (C) 2018 Google Inc.
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

#ifndef CORE_RANGE_H
#define CORE_RANGE_H

#include <cstring>  // size_t

namespace core {

// Range is a stl-compatibile iteratable that allows iteration over all
// memory-contiguous elements of type T between a start and end pointer.
template <typename T>
class Range {
 public:
  typedef T value_type;
  typedef const T* const_iterator;

  // Constructs the range with a pointer to the first element in the list
  // and a pointer to one-past the last element in the list.
  inline Range(const T* start, const T* end) : mStart(start), mEnd(end) {}

  // Constructs the range with a pointer to the first element in the list
  // and list count.
  inline Range(const T* start, size_t count)
      : mStart(start), mEnd(start + count) {}

  // begin() returns the pointer to the first element in the list.
  inline const T* begin() const { return mStart; }
  // end() returns the pointer to one-past the last element in the list.
  inline const T* end() const { return mEnd; }
  // size() returns the number of items in the list.
  inline const size_t size() const { return mEnd - mStart; }

 private:
  const T* mStart;
  const T* mEnd;
};

}  // namespace core

#endif  // CORE_RANGE_H
