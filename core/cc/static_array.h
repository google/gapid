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

#ifndef CORE_STATIC_ARRAY_H
#define CORE_STATIC_ARRAY_H

#include "assert.h"

namespace core {

template <typename T, int N>
struct CStaticArray {
  T mData[N];
};

// StaticArray represents a fixed size array with implicit conversions to and
// from T[N].
template <typename T, int N>
class StaticArray : protected CStaticArray<T, N> {
public:

  inline StaticArray();
  inline StaticArray(const CStaticArray<T, N>& other);
  inline StaticArray(T arr[N]);
  inline StaticArray(std::initializer_list<T> l);

  inline operator T*();
  inline operator const T*() const;
};

template <typename T, int N>
inline StaticArray<T, N>::StaticArray() {
    for (int i = 0; i < N; i++) {
        this->mData[i] = T();
    }
}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(const CStaticArray<T, N>& other) {
    for (int i = 0; i < N; i++) {
        this->mData[i] = other.mData[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(T arr[N]) {
    for (int i = 0; i < N; i++) {
        this->mData[i] = arr[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(std::initializer_list<T> l) {
    GAPID_ASSERT(l.size() == N);
    for (int i = 0; i < N; i++) {
        this->mData[i] = l.begin()[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::operator T*() {
    return &this->mData[0];
}

template <typename T, int N>
inline StaticArray<T, N>::operator const T*() const {
    return &this->mData[0];
}

}  // namespace core

#endif  // CORE_STATIC_ARRAY_H
