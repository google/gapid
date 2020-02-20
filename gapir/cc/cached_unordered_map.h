/*
 * Copyright (C) 2020 Google Inc.
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

// This is a wrapper around std::unordered_map that provides a fast return for
// repeated queries with the same key. Our code has many places that maps are
// successively queried with the same key, or even where maps contain only a
// single key/value pair. In these cases this map provides a significant speed
// up.

#ifndef GAPIR_CACHED_UNORDERED_MAP_H
#define GAPIR_CACHED_UNORDERED_MAP_H

namespace gapir {

template <typename T1, typename T2>
class cached_unordered_map {
 public:
  T2& operator[](const T1& key) {
    if (mCachedValueInvalid == false && mLastKey == key) {
      return mLastValue->second;
    }

    mCachedValueInvalid = false;
    mLastKey = key;
    mLastValue = mMap.find(key);

    if (mLastValue == mMap.end()) {
      mMap[key];
      mLastValue = mMap.find(key);
    }

    return mLastValue->second;
  }

  size_t count(const T1& key) const { return mMap.count(key); }
  size_t erase(const T1& key) {
    mCachedValueInvalid = true;
    return mMap.erase(key);
  }

  typename std::unordered_map<T1, T2>::iterator erase(
      const typename std::unordered_map<T1, T2>::iterator& iter) {
    mCachedValueInvalid = true;
    return mMap.erase(iter);
  }

  typename std::unordered_map<T1, T2>::iterator find(const T1& key) {
    return mMap.find(key);
  }
  typename std::unordered_map<T1, T2>::const_iterator find(
      const T1& key) const {
    return mMap.find(key);
  }

  typename std::unordered_map<T1, T2>::const_iterator begin() const {
    return mMap.begin();
  }
  typename std::unordered_map<T1, T2>::iterator begin() { return mMap.begin(); }

  typename std::unordered_map<T1, T2>::const_iterator end() const {
    return mMap.end();
  }
  typename std::unordered_map<T1, T2>::iterator end() { return mMap.end(); }

 private:
  mutable bool mCachedValueInvalid = true;

  mutable T1 mLastKey;
  mutable typename std::unordered_map<T1, T2>::iterator mLastValue;

  std::unordered_map<T1, T2> mMap;
};

}  // namespace gapir

#endif