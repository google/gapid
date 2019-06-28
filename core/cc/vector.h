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

#ifndef CORE_VECTOR_H
#define CORE_VECTOR_H

#include <stdint.h>

#include "assert.h"

namespace core {

// Vector is a fixed-capacity container of plain-old-data elements of type T.
// Do not use elements that require destruction in this container.
template <typename T>
class Vector {
 public:
  // Default constructor.
  // Vector is unusable until it is assigned from a Vector constructed using one
  // of the other constructors.
  Vector();

  // Constructs a vector pre-sized to count and with the first element at the
  // specified address. The vector's capacity is fixed to count.
  Vector(T* first, size_t count);

  // Constructs a vector pre-sized to count and with the first element at the
  // specified address. The vector's capacity is fixed to capacity.
  Vector(T* first, size_t count, size_t capacity);

  // clear sets the vector count to 0.
  inline void clear();

  // append grows the vector by 1 by appending el to the end of the vector.
  // It is a fatal error if the vector has no more capacity.
  inline void append(const T& el);

  // append grows the vector and appends all elements from the other vector.
  // It is a fatal error if the vector has no more capacity.
  inline void append(const Vector<T>& other);

  // data returns the pointer to the first element in the vector.
  // If the vector is empty, then data returns nullptr.
  inline T* data() const;

  // count returns the number of elements in the vector.
  inline size_t count() const;

  // Returns a reference to a single element in the slice.
  inline T& operator[](size_t index) const;

  // Support for range-based for looping
  inline T* begin() const;
  inline T* end() const;

 private:
  T* mBase;          // Address of the first element in the vector.
  size_t mCapacity;  // Maximum number of elements this vector can hold.
  size_t mCount;     // Number of elements in the vector.
};

template <typename T>
Vector<T>::Vector() : mBase(nullptr), mCapacity(0), mCount(0) {}

template <typename T>
Vector<T>::Vector(T* first, size_t count)
    : mBase(first), mCapacity(count), mCount(count) {}

template <typename T>
Vector<T>::Vector(T* first, size_t count, size_t capacity)
    : mBase(first), mCapacity(capacity), mCount(count) {
  GAPID_ASSERT(count <= capacity);
}

template <typename T>
inline void Vector<T>::clear() {
  mCount = 0;
}

template <typename T>
inline void Vector<T>::append(const T& el) {
  GAPID_ASSERT(mCount < mCapacity);
  new (&mBase[mCount]) T(el);
  mCount++;
}

template <typename T>
inline void Vector<T>::append(const Vector<T>& other) {
  for (auto it : other) {
    append(it);
  }
}

template <typename T>
inline T* Vector<T>::data() const {
  return mCount > 0 ? mBase : nullptr;
}

template <typename T>
inline size_t Vector<T>::count() const {
  return mCount;
}

// Returns a reference to a single element in the slice.
template <typename T>
inline T& Vector<T>::operator[](size_t index) const {
  GAPID_ASSERT(index < mCount);
  return mBase[index];
}

// Support for range-based for looping
template <typename T>
inline T* Vector<T>::begin() const {
  return mBase;
}

template <typename T>
inline T* Vector<T>::end() const {
  return mBase + mCount;
}

}  // namespace core

#endif  // CORE_VECTOR_H
