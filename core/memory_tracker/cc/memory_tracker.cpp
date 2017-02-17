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

#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_ANDROID
#include "memory_tracker.h"

#include <map>

namespace gapii {
namespace TrackMemory {
MemoryTracker* MemoryTracker::unique_tracker = nullptr;

void DirtyPageTable::RecollectIfPossible(size_t num_stale_pages) {
  if (pages_.size() - stored_ > num_stale_pages) {
    // If we have more extra spaces for recollection than required, remove
    // required number of not used spaces.
    size_t new_size = pages_.size() - num_stale_pages;
    pages_.resize(new_size);
  } else {
    // Otherwise shrink the storage to hold all not-yet-dumped dirty pages.
    pages_.resize(stored_ + 1);
  }
}

std::vector<void*> DirtyPageTable::DumpAndClearInRange(void* start,
                                                       size_t size) {
  uintptr_t start_addr = reinterpret_cast<uintptr_t>(start);
  std::vector<void*> r;
  r.reserve(stored_);
  for (auto it = pages_.begin(); it != current_;) {
    auto nt = std::next(it, 1);
    if (*it >= start_addr && *it < start_addr + size) {
      r.push_back(reinterpret_cast<void*>(*it));
      pages_.splice(pages_.end(), pages_, it);
      stored_ -= 1;
    }
    it = nt;
  }
  return r;
}

std::vector<void*> DirtyPageTable::DumpAndClearAll() {
  std::vector<void*> r;
  r.reserve(stored_);
  std::for_each(pages_.begin(), current_, [&r](uintptr_t addr) {
    r.push_back(reinterpret_cast<void*>(addr));
  });
  // Set the space for the next page address to the beginning of the storage,
  // and reset the counter.
  current_ = pages_.begin();
  stored_ = 0u;
  return r;
}

bool MemoryTracker::AddTrackingRangeImpl(void* start, size_t size) {
  if (size == 0) return false;
  if (IsInRanges(reinterpret_cast<uintptr_t>(start), ranges_)) return false;

  void* start_page_addr = GetAlignedAddress(start, page_size_);
  size_t size_aligned = GetAlignedSize(start, size, page_size_);
  dirty_pages_.Reserve(size_aligned / page_size_);
  ranges_[reinterpret_cast<uintptr_t>(start)] = size;
  // TODO(qining): Add Windows support
  return mprotect(start_page_addr, size_aligned,
                  track_read_ ? PROT_NONE : PROT_READ) == 0;
}

bool MemoryTracker::RemoveTrackingRangeImpl(void* start, size_t size) {
  if (size == 0) return false;
  auto it = ranges_.find(reinterpret_cast<uintptr_t>(start));
  if (it == ranges_.end()) return false;

  ranges_.erase(it);
  void* start_page_addr = GetAlignedAddress(start, page_size_);
  size_t size_aligned = GetAlignedSize(start, size, page_size_);
  dirty_pages_.RecollectIfPossible(size_aligned / page_size_);
  // TODO(qining): Add Windows support
  return mprotect(start_page_addr, size_aligned, PROT_WRITE | PROT_READ) == 0;
}

bool MemoryTracker::ClearTrackingRangesImpl() {
  if (std::any_of(ranges_.begin(), ranges_.end(),
                  [this](std::pair<uintptr_t, size_t> r) {
                    void* start = reinterpret_cast<void*>(r.first);
                    size_t size = r.second;
                    void* start_page_addr =
                        GetAlignedAddress(start, page_size_);
                    size_t size_aligned =
                        GetAlignedSize(start, size, page_size_);
                    dirty_pages_.RecollectIfPossible(size_aligned / page_size_);
                    // TODO(qining): Add Windows support
                    return mprotect(start_page_addr, size_aligned,
                                    PROT_WRITE | PROT_READ) != 0;
                  })) {
    return false;
  }
  ranges_.clear();
  return true;
}

void MemoryTracker::HandleSegfaultImpl(int sig, siginfo_t* info, void* unused) {
  void* fault_addr = info->si_addr;
  if (!IsInRanges(reinterpret_cast<uintptr_t>(fault_addr), ranges_)) {
    return (*orig_action_.sa_sigaction)(sig, info, unused);
  }

  // The fault address is within a tracking range
  void* page_addr = GetAlignedAddress(fault_addr, page_size_);
  if (dirty_pages_.Has(page_addr)) {
    return;
  }
  if (!dirty_pages_.Record(page_addr)) {
    // The dirty page table does not have enough space pre-allocated,
    // fallback to the original handler.
    return (*orig_action_.sa_sigaction)(sig, info, unused);
  }
  mprotect(page_addr, page_size_, PROT_READ | PROT_WRITE);
}

}  // namespace TrackMemory
}  // namespace gapii
#endif  // TARGET_OS
