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

#include "memory_tracker.h"

#if COHERENT_TRACKING_ENABLED
#include <map>

namespace gapii {
namespace track_memory {

MemoryTracker::derived_tracker_type* unique_tracker = nullptr;

template <>
MemoryTracker::tracking_range_list_type::iterator
MemoryTracker::FirstOverlappedRange(uintptr_t addr, size_t size) {
  uintptr_t end = addr + size;
  auto it = tracking_ranges_.upper_bound(addr);
  if (it == tracking_ranges_.end()) {
    return it;
  }
  if (it->second->Overlaps(addr, end)) {
    return it;
  }
  return tracking_ranges_.end();
}

template <>
bool MemoryTracker::TrackRangeImpl(void* start, size_t size) {
  if (!EnableMemoryTrackerImpl()) {
    return false;
  }

  if (size == 0) return false;
  uintptr_t addr = reinterpret_cast<uintptr_t>(start);
  auto overlap = FirstOverlappedRange(addr, size);
  if (overlap != tracking_ranges_.end()) {
    return false;
  }
  auto new_rng = std::unique_ptr<MemoryTracker::tracking_range_type>(
      new MemoryTracker::tracking_range_type(reinterpret_cast<uintptr_t>(start),
                                             size));
  bool result = set_protection(
      reinterpret_cast<void*>(new_rng->aligned_start()),
      new_rng->aligned_size(),
      track_read_ ? PageProtections::kNone : PageProtections::kRead);
  tracking_ranges_[addr + size] = std::move(new_rng);
  return result;
}

template <>
bool MemoryTracker::UntrackRangeImpl(void* start, size_t size) {
  if (size == 0) return false;
  uintptr_t addr = reinterpret_cast<uintptr_t>(start);
  uintptr_t end = addr + size;
  auto it = tracking_ranges_.find(end);
  if (it == tracking_ranges_.end()) return false;

  bool result = true;
  uintptr_t first_page = it->second->aligned_start();
  size_t aligned_size = it->second->aligned_size();
  uintptr_t last_page = first_page + aligned_size - GetPageSize();

  tracking_ranges_.erase(it);

  // Handle the non-boundary pages first. Boundary pages might also be tracked
  // in other ranges, and their flag should not be reset to ReadWrite if other
  // ranges are tracking them.
  if (first_page < last_page) {
    result &= set_protection(
        reinterpret_cast<void*>(first_page + GetPageSize()),
        aligned_size - 2 * GetPageSize(), PageProtections::kReadWrite);
  }

  if (FirstOverlappedRange(first_page, GetPageSize()) ==
      tracking_ranges_.end()) {
    result &= set_protection(reinterpret_cast<void*>(first_page), GetPageSize(),
                             PageProtections::kReadWrite);
  }
  if (FirstOverlappedRange(last_page, GetPageSize()) ==
      tracking_ranges_.end()) {
    result &= set_protection(reinterpret_cast<void*>(last_page), GetPageSize(),
                             PageProtections::kReadWrite);
  }

  return result;
}

template <>
bool MemoryTracker::HandleAndClearDirtyIntersectsImpl(
    void* start, size_t size,
    std::function<void(void* dirty_addr, size_t dirty_size)> handle_dirty) {
  if (size == 0) return true;
  uintptr_t addr = reinterpret_cast<uintptr_t>(start);
  uintptr_t end = RoundUpAlignedAddress(addr + size, GetPageSize());
  addr = RoundDownAlignedAddress(addr, GetPageSize());
  size = end - addr;

  bool set_protection_result = true;

  auto clear_dirty_intersects = [&handle_dirty, &set_protection_result, this](
                                    uintptr_t i_addr, size_t i_size) -> bool {
    handle_dirty(reinterpret_cast<void*>(i_addr), i_size);
    set_protection_result = set_protection(
        reinterpret_cast<void*>(i_addr), i_size,
        track_read_ ? PageProtections::kNone : PageProtections::kRead);
    // Return true so that in ForDirtyIntersects, marked dirty pages will
    // be cleared.
    return true;
  };

  auto first_rng = FirstOverlappedRange(addr, size);
  for (auto it = first_rng; it != tracking_ranges_.end(); it++) {
    if (it->second->Overlaps(addr, end)) {
      it->second->ForDirtyIntersects(addr, size, clear_dirty_intersects);
    } else {
      break;
    }
  }
  return set_protection_result;
}

template <>
bool MemoryTracker::HandleSegfaultImpl(void* fault_addr) {
  uintptr_t addr = reinterpret_cast<uintptr_t>(fault_addr);
  uintptr_t end = RoundUpAlignedAddress(addr + 1u, GetPageSize());
  addr = RoundDownAlignedAddress(addr, GetPageSize());
  auto first_rng_it = FirstOverlappedRange(addr, end - addr);
  if (first_rng_it == tracking_ranges_.end()) {
    return false;
  }
  bool result = true;
  for (auto& it = first_rng_it; it != tracking_ranges_.end(); it++) {
    if (it->second->Overlaps(addr, end)) {
      result &= it->second->SetDirty(
          addr, end - addr,
          [](uintptr_t dirty_addr, size_t dirty_size) -> bool {
            return set_protection(reinterpret_cast<void*>(dirty_addr),
                                  dirty_size, PageProtections::kReadWrite);
          });
    } else {
      break;
    }
  }
  return result;
}

}  // namespace track_memory
}  // namespace gapii

#endif  // COHERENT_TRACKING_ENABLED
