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

#ifndef CORE_INTERVAL_LIST_H
#define CORE_INTERVAL_LIST_H

#include <algorithm>
#include <cstdint>
#include <vector>

#include "core/cc/target.h"
#include "range.h"

namespace core {

// Interval represents a single interval range of type T.
// This is the default interval type to be used by IntervalList<T>.
template <typename T>
struct Interval {
  typedef T interval_unit_type;

  inline T start() const { return mStart; }
  inline T end() const { return mEnd; }
  inline void adjust(T start, T end) {
    mStart = start;
    mEnd = end;
  }

  T mStart;  // The index of the first item in the interval range.
  T mEnd;    // The index of the one-past the last item in the interval range.
};

// Interval equality operator.
template <typename T>
inline bool operator==(const Interval<T>& lhs, const Interval<T>& rhs) {
  return lhs.start() == rhs.start() && lhs.end() == rhs.end();
}

// CustomIntervalList holds a ascendingly-sorted list of custom interval types.
// Intervals can be added to the list using merge(), where they may be merged
// with existing intervals if the spans are within the specified
// merge-threshold. Intervals can also be added to the list using replace(),
// where any completely overlapping intervals are removed and partially
// overlapping intervals are trimmed, before inserting the new interval.
// CustomIntervalList supports for-range looping.
template <typename T>
class CustomIntervalList {
 public:
  typedef T value_type;
  typedef const T* const_iterator;
  typedef typename T::interval_unit_type interval_unit_type;

  // Constructs an CustomIntervalList with a default merge threshold of 1.
  inline CustomIntervalList();

  // intersect returns a iterable range covering all the intervals that
  // intersect the span between start and end.
  inline Range<T> intersect(interval_unit_type start,
                            interval_unit_type end) const;

  // index_of returns the index of the interval that contains v, or -1 if
  // there is no interval containing v.
  inline ssize_t index_of(interval_unit_type v) const;

  // replace removes and/or trims any intervals overlapping i and then adds i
  // to this list. No merging is performed.
  inline void replace(const T& i);

  // merge adds the interval i to this list, merging any overlapping intervals.
  inline void merge(const T& i);

  // setMergeThreshold sets the edge-distance threshold for merging intervals
  // when calling merge(). Intervals will merge if: edge-distance < threshold.
  // Examples:
  // • A threshold of 0 will require intervals to overlap before they are
  //   merged.
  // • A threshold of 1 will merge intervals if they overlap or touch
  //   edges.
  // • A threshold of 2 will merge intervals as described above, and those with
  //   a single unit gap.
  // Changing the merge thresholds will not affect any existing intervals in the
  // list.
  inline void setMergeThreshold(interval_unit_type threshold);

  // clear removes all intervals from the list.
  inline void clear();

  // count returns the number of intervals in the list.
  inline uint32_t count() const;

  // begin() returns the pointer to the first interval in the list.
  inline const T* begin() const;

  // end() returns the pointer to one-past the last interval in the list.
  inline const T* end() const;

  // operator[] returns the const reference to the element at the specified
  // location pos.
  inline const T& operator[](size_t pos) const;

 protected:
  // rangeFirst returns the index of the first interval + bias that touches or
  // exceeds start.
  inline ssize_t rangeFirst(interval_unit_type start,
                            interval_unit_type bias) const;

  // rangeLast returns the index of the last interval that touches or
  // is less than end + bias.
  inline ssize_t rangeLast(interval_unit_type end,
                           interval_unit_type bias) const;

  std::vector<T> mIntervals;
  interval_unit_type mMergeBias;
};

template <typename T>
CustomIntervalList<T>::CustomIntervalList() : mMergeBias(0) {}

template <typename T>
inline Range<T> CustomIntervalList<T>::intersect(interval_unit_type start,
                                                 interval_unit_type end) const {
  auto first = begin() + rangeFirst(start, -1);
  auto last = begin() + rangeLast(end, -1);
  return Range<T>(first, last + 1);
}

template <typename T>
inline ssize_t CustomIntervalList<T>::index_of(interval_unit_type v) const {
  auto i = rangeFirst(v, -1);
  if (i >= count()) {
    return -1;
  }
  if (mIntervals[i].mStart > v || mIntervals[i].mEnd < v) {
    return -1;
  }
  return i;
}

template <typename T>
inline void CustomIntervalList<T>::replace(const T& i) {
  auto first = rangeFirst(i.start(), mMergeBias);
  auto last = rangeLast(i.end(), mMergeBias);

  if (first <= last) {
    bool trimTail = mIntervals[first].start() < i.start();
    bool trimHead = mIntervals[last].end() > i.end();

    if (first == last && trimTail && trimHead) {
      // i sits within a single interval. Split it into two.
      //           ┏━━━━━━━━━━━━━━┓
      //           ┗━━━━━━━━━━━━━━┛
      //━━━━━━━━━━━┳═─═─═─═─═─═─═─┳━━━━━━━━━━━
      //━━━━━━━━━━━┻─═─═─═─═─═─═─═┻━━━━━━━━━━━
      auto interval = mIntervals.begin() + first;
      mIntervals.insert(interval, *interval);
      last++;
    }
    if (trimTail) {
      // Trim end of first interval.
      //           ┏━━━━━━━━━━━━━━━━
      //           ┗━━━━━━━━━━━━━━━━
      //━━━━━━━━━━━┳═─═─╗
      //━━━━━━━━━━━┻─═─═┘
      auto interval = mIntervals.begin() + first;
      interval->adjust(interval->start(), i.start());
      first++;  // Don't erase the first interval.
    }
    if (trimHead) {
      // Trim front of last interval.
      //━━━━━━━━━━━━━━━━┓
      //━━━━━━━━━━━━━━━━┛
      //           ┌═─═─┳━━━━━━━━━━━
      //           ╚─═─═┻━━━━━━━━━━━
      auto interval = mIntervals.begin() + last;
      interval->adjust(i.end(), interval->end());
      last--;  // Don't erase the last interval.
    }
    if (first <= last) {
      auto from = mIntervals.begin() + first;
      auto to = mIntervals.begin() + last + 1;
      mIntervals.erase(from, to);
    }
  }
  mIntervals.insert(mIntervals.begin() + first, i);
}

template <typename T>
inline void CustomIntervalList<T>::merge(const T& i) {
  auto first = rangeFirst(i.start(), mMergeBias);
  auto last = rangeLast(i.end(), mMergeBias);
  auto from = mIntervals.begin() + first;
  if (first <= last) {
    auto to = mIntervals.begin() + last;
    auto low = std::min(from->start(), i.start());
    auto high = std::max(to->end(), i.end());
    T* f = &(*from);
    mIntervals.erase(from, to);
    f->adjust(low, high);
  } else {
    mIntervals.insert(from, i);
  }
}

template <typename T>
inline void CustomIntervalList<T>::setMergeThreshold(
    interval_unit_type threshold) {
  mMergeBias = threshold - 1;
}

template <typename T>
inline void CustomIntervalList<T>::clear() {
  mIntervals.clear();
}

template <typename T>
inline uint32_t CustomIntervalList<T>::count() const {
  return mIntervals.size();
}

template <typename T>
inline const T* CustomIntervalList<T>::begin() const {
  if (count() > 0) {
    return &mIntervals[0];
  } else {
    return nullptr;
  }
}

template <typename T>
inline const T* CustomIntervalList<T>::end() const {
  size_t c = mIntervals.size();
  if (c > 0) {
    return &mIntervals[0] + c;
  } else {
    return nullptr;
  }
}

template <typename T>
inline const T& CustomIntervalList<T>::operator[](size_t pos) const {
  return mIntervals[pos];
}

template <typename T>
inline ssize_t CustomIntervalList<T>::rangeFirst(
    interval_unit_type start, interval_unit_type bias) const {
  ssize_t l = 0;
  ssize_t h = mIntervals.size();
  while (l != h) {
    ssize_t m = (l + h) / 2;
    if (mIntervals[m].end() + bias >= start) {
      h = m;
    } else {
      l = m + 1;
    }
  }
  return l;
}

template <typename T>
inline ssize_t CustomIntervalList<T>::rangeLast(interval_unit_type end,
                                                interval_unit_type bias) const {
  ssize_t l = -1;
  ssize_t h = mIntervals.size() - 1;
  while (l != h) {
    ssize_t m = (l + h + 1) / 2;
    if (mIntervals[m].start() <= end + bias) {
      l = m;
    } else {
      h = m - 1;
    }
  }
  return l;
}

// IntervalList holds a ascendingly-sorted list of Interval<T>s.
// See CustomIntervalList for more information.
template <typename T>
class IntervalList : public CustomIntervalList<Interval<T> > {};

}  // namespace core

#endif  // CORE_INTERVAL_LIST_H
