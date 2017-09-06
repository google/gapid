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

#include "core/cc/target.h"

#ifndef GAPII_MEMORY_TRACKER_H
#define GAPII_MEMORY_TRACKER_H

#define COHERENT_TRACKING_ENABLED 1
#if COHERENT_TRACKING_ENABLED
#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID)
#define IS_POSIX 1
#include "core/memory_tracker/cc/posix/memory_tracker.h"
#elif (TARGET_OS == GAPID_OS_WINDOWS)
#define IS_POSIX 0
#include "core/memory_tracker/cc/windows/memory_tracker.h"
#else
#undef COHERENT_TRACKING_ENABLED
#define COHERENT_TRACKING_ENABLED 0
#endif
#endif // COHERENT_TRACKING_ENABLED

#if COHERENT_TRACKING_ENABLED

#include <algorithm>
#include <atomic>
#include <iterator>
#include <list>
#include <map>
#include <utility>
#include <vector>

#include "core/memory_tracker/cc/memory_protections.h"

namespace gapii {
namespace track_memory {

// Returns the lower bound aligned address for a given pointer |addr| and a
// given |alignment|. |alignment| must be a power of two and cannot be zero.
// If the given |alignment| is zero or not a power of two, the return is
// undefined.
inline void* GetAlignedAddress(void* addr, size_t alignment) {
  return reinterpret_cast<void*>(reinterpret_cast<uintptr_t>(addr) &
                                 ~(alignment - 1u));
}

// For a given memory range specified by staring address: |addr| and size:
// |size_unaligned|, returns the size of memory space occupied by the range in
// term of |alignment| aligned memory space.
// e.g.: staring address: 0x5, size_unaligned: 0x7, alignment: 0x8
//       => occupied range in term of aligned memory: 0x0 ~ 0x10
//       => size in aligned memory: 0x10
inline size_t GetAlignedSize(void* addr, size_t size_unaligned,
                             size_t alignment) {
  if (alignment == 0 || size_unaligned == 0) return 0;
  void* start_addr_aligned = GetAlignedAddress(addr, alignment);
  uintptr_t end_across_boundry =
      reinterpret_cast<uintptr_t>(addr) + size_unaligned + alignment - 1;
  if (end_across_boundry < reinterpret_cast<uintptr_t>(addr) ||
      end_across_boundry < size_unaligned ||
      end_across_boundry < alignment - 1) {
    // Overflow
    return 0;
  }
  void* end_addr_aligned =
      GetAlignedAddress(reinterpret_cast<void*>(end_across_boundry), alignment);
  return reinterpret_cast<uintptr_t>(end_addr_aligned) -
         reinterpret_cast<uintptr_t>(start_addr_aligned);
}

// A helper function to tell whether a given address is covered in a bunch of
// memory ranges. If |page_aligned_ranges| is set to true, the ranges' address
// will be aligned to page boundary, so if the |addr| is not in a range, but
// shares a common memory page with the range, it will be considered as in the
// range. By default |page_aligned_ranges| is set to false. Returns true if
// the address is considered in the range, otherwise returns false.
inline bool IsInRanges(uintptr_t addr, std::map<uintptr_t, size_t>& ranges,
                       bool page_aligned_ranges = false) {
  auto get_aligned_addr = [](uintptr_t addr) {
    return reinterpret_cast<uintptr_t>(
        GetAlignedAddress(reinterpret_cast<void*>(addr), GetPageSize()));
  };
  auto get_aligned_size = [](uintptr_t addr, size_t size) {
    return reinterpret_cast<uintptr_t>(
        GetAlignedSize(reinterpret_cast<void*>(addr), size, GetPageSize()));
  };
  // It is not safe to call std::prev() if the container is empty, so the empty
  // case is handled first.
  if (ranges.size() == 0) {
    return false;
  }
  auto it = ranges.lower_bound(addr);
  // Check if the lower bound range already covers the addr.
  if (it != ranges.end()) {
    if (it->first == addr) {
      return true;
    }
    if (page_aligned_ranges) {
      uintptr_t aligned_range_start = get_aligned_addr(it->first);
      if (aligned_range_start <= addr) {
        return true;
      }
    }
  }
  // Check the previous range
  auto pit = std::prev(it, 1);
  if (pit == ranges.end()) {
    return false;
  }
  uintptr_t range_start =
      page_aligned_ranges ? get_aligned_addr(pit->first) : pit->first;
  uintptr_t range_size = page_aligned_ranges
                             ? get_aligned_size(pit->first, pit->second)
                             : pit->second;
  if (addr < range_start || addr >= range_start + range_size) {
    return false;
  }
  return true;
}

// SpinLock is a spin lock implemented with atomic variable and opertions.
// Mutiple calls to Lock in a single thread will result into a deadlock.
class SpinLock {
 public:
  SpinLock() : var_(0) {}
  // Lock acquires the lock.
  void Lock() {
    uint32_t l = kUnlocked;
    while (!var_.compare_exchange_strong(l, kLocked)) {
      l = kUnlocked;
    }
  }
  // Unlock releases the lock.
  void Unlock() { var_.exchange(kUnlocked); }

 private:
  std::atomic<uint32_t> var_;
  const uint32_t kLocked = 1u;
  const uint32_t kUnlocked = 0u;
};

// SpinLockGuard acquires the specified spin lock when it is constructed and
// release the lock when it is destroyed.
class SpinLockGuard {
 public:
  SpinLockGuard(SpinLock* l) : l_(l) {
    if (l_) {
      l_->Lock();
    }
  }
  ~SpinLockGuard() {
    if (l_) {
      l_->Unlock();
    }
  }
  // Not copyable, not movable.
  SpinLockGuard(const SpinLockGuard&) = delete;
  SpinLockGuard(SpinLockGuard&&) = delete;
  SpinLockGuard& operator=(const SpinLockGuard&) = delete;
  SpinLockGuard& operator=(SpinLockGuard&&) = delete;

 private:
  SpinLock* l_;
};

// SpinLockGuarded wraps a given non-static class member function with the
// creation of a SpinLockGuard object. Suppose a class: |Task|, object: |task|,
// a non-static member function: |Do(int)|, and a spin lock: |spin_lock|, the
// wrapper should be used in the following way:
//   auto wrapped = SpinLockGuarded<Task, decltype(Task::Do)>(
//      &task, &Task::Do, &spin_lock);
template <typename OwnerTy, typename MemberFuncPtrTy>
class SpinLockGuarded {
 public:
  SpinLockGuarded(OwnerTy* o, const MemberFuncPtrTy& f, SpinLock* l)
      : owner_(o), f_(f), l_(l) {}

  template <typename... Args>
  auto operator()(Args&&... args) ->
      typename std::result_of<MemberFuncPtrTy(OwnerTy*, Args&&...)>::type {
    SpinLockGuard g(l_);
    return ((*owner_).*f_)(std::forward<Args>(args)...);
  }

 protected:
  OwnerTy* owner_;
  MemberFuncPtrTy f_;
  SpinLock* l_;
};

// SignalSafe wraps a given non-static class member function with the creation
// of a SignalBlocker object then a SpinLockGuard object. Suppose a class:
// |Task|, object: |task|, a non-static member function: |Do(int)|, a signal to
// block: |signal_value|, and a spin lock: |spin_lock|, the wrapper should be
// used in the following way:
//   auto wrapped = SignalSafe<Task, decltype(Task::Do)>(
//      &task, &Task::Do, &spin_lock, signal_value);
template <typename OwnerTy, typename MemberFuncPtrTy>
class SignalSafe : public SpinLockGuarded<OwnerTy, MemberFuncPtrTy> {
  using SpinLockGuardedFunctor = SpinLockGuarded<OwnerTy, MemberFuncPtrTy>;

 public:
  SignalSafe(OwnerTy* o, const MemberFuncPtrTy& f, SpinLock* l, int sig)
      : SpinLockGuarded<OwnerTy, MemberFuncPtrTy>(o, f, l), sig_(sig) {}

  template <typename... Args>
  auto operator()(Args&&... args) ->
      typename std::result_of<MemberFuncPtrTy(OwnerTy*, Args&&...)>::type {
    SignalBlocker g(sig_);
    return this->SpinLockGuardedFunctor::template operator()<Args...>(std::forward<Args>(args)...);
  }

 protected:
  int sig_;
};

// DirtyPageTable holds the addresses of dirty memory pages. It pre-allocates
// its storage space for recording. Recording the space for new dirty pages
// will not acquire new memory space.
class DirtyPageTable {
 public:
  DirtyPageTable() : stored_(0u), pages_(1u), current_(pages_.begin()) {}

  // Not copyable, not movable.
  DirtyPageTable(const DirtyPageTable&) = delete;
  DirtyPageTable(DirtyPageTable&&) = delete;
  DirtyPageTable& operator=(const DirtyPageTable&) = delete;
  DirtyPageTable& operator=(DirtyPageTable&&) = delete;

  ~DirtyPageTable() {}

  // Record records the given address to the next storage space if such a space
  // is available, increases the counter of stored addresses then returns true.
  // If such a space is not available, returns false without trying to record
  // the address. Record does not check whether the given page_addr has been
  // already recorded before.
  bool Record(void* page_addr) {
    if (std::next(current_, 1) == pages_.end()) {
      return false;
    }
    *current_ = reinterpret_cast<uintptr_t>(page_addr);
    stored_++;
    current_++;
    return true;
  }

  // Has returns true if the given page_addr has already been recorded and not
  // yet dumpped, otherwise returns false;
  bool Has(void* page_addr) const {
    for (auto i = pages_.begin(); i != current_; i++) {
      if (reinterpret_cast<uintptr_t>(page_addr) == *i) {
        return true;
      }
    }
    return false;
  }

  // Reserve reservers the spaces for recording |num_new_pages| number of
  // pages.
  void Reserve(size_t num_new_pages) {
    pages_.resize(pages_.size() + num_new_pages, 0);
  }

  // RecollectIfPossible tries to recollect the space used for recording
  // |num_stale_pages| number of pages. If there are fewer not-used spaces than
  // specified, it shrinks the storage to hold just enough space for recorded
  // dirty pages, or at least one page when no page has been recorded.
  void RecollectIfPossible(size_t num_stale_pages);

  // DumpAndClearInRange dumps the recorded page addresses within a specified
  // range, starting from |start| with |size| large, to a std::vector, clears
  // the internal records without releasing the spaces and returns the page
  // address vector.
  std::vector<void*> DumpAndClearInRange(void* start, size_t size);

  // DumpAndClearInRange dumps all the recorded page addresses to a
  // std::vector, clears the internal records without releasing the spaces and
  // returns the page address vector.
  std::vector<void*> DumpAndClearAll();

 protected:
  size_t stored_;  // A counter for the number of page addresses stored.
  std::list<uintptr_t> pages_;  // Internal storage of the page addresses.
  std::list<uintptr_t>::iterator
      current_;  // The space for the last page address stored.
};

// MemoryTrackerImpl utilizes Segfault signal on Linux to track accesses to
// memories.
template<typename SpecificMemoryTracker>
class MemoryTrackerImpl : public SpecificMemoryTracker {
 public:
  using derived_tracker_type = SpecificMemoryTracker;
  // Creates a memory tracker to track memory write operations. If
  // |track_read| is set to true, also tracks the memory read operations.
  // By default |track_read| is set to false.
  MemoryTrackerImpl(bool track_read = false)
      : SpecificMemoryTracker([this](void* v) { return DoHandleSegfault(v); }),
        track_read_(track_read),
        page_size_(GetPageSize()),
        l_(),
        ranges_(),
        dirty_pages_(),
#define CONSTRUCT_LOCKED(function) \
  function(this, &MemoryTrackerImpl::function##Impl, &l_)
        CONSTRUCT_LOCKED(HandleSegfault),
#undef CONSTRUCT_LOCKED
#define CONSTRUCT_SIGNAL_SAFE(function) \
  function(this, &MemoryTrackerImpl::function##Impl, &l_, SIGSEGV)
        CONSTRUCT_SIGNAL_SAFE(AddTrackingRange),
        CONSTRUCT_SIGNAL_SAFE(RemoveTrackingRange),
        CONSTRUCT_SIGNAL_SAFE(GetDirtyPagesInRange),
        CONSTRUCT_SIGNAL_SAFE(ResetPagesToTrack),
        CONSTRUCT_SIGNAL_SAFE(GetAndResetAllDirtyPages),
        CONSTRUCT_SIGNAL_SAFE(GetAndResetDirtyPagesInRange),
        CONSTRUCT_SIGNAL_SAFE(ClearTrackingRanges),
        CONSTRUCT_SIGNAL_SAFE(EnableMemoryTracker),
        CONSTRUCT_SIGNAL_SAFE(DisableMemoryTracker)
#undef CONSTRUCT_SIGNAL_SAFE
  {
  }
  // Not copyable, not movable.
  MemoryTrackerImpl(const MemoryTrackerImpl&) = delete;
  MemoryTrackerImpl(MemoryTrackerImpl&&) = delete;
  MemoryTrackerImpl& operator=(const MemoryTrackerImpl&) = delete;
  MemoryTrackerImpl& operator=(MemoryTrackerImpl&&) = delete;

  ~MemoryTrackerImpl() {
    DisableMemoryTracker();
    ClearTrackingRanges();
  }

 protected:
  // Adds an address range specified by starting address and size for tracking
  // and set accessing permissions of the corresponding pages to track write
  // (and read, if track_read is true) operations. Returns true if the range
  // is added successfully, returns false when the operaion is not done
  // successfully, e.g. the range overlaps with exisiting ranges or the size is
  // 0. The ranges must not contain any pages that store the data of this
  // memory tracker object, otherwise the segfault signal will be handled by
  // the original handler and the application may crash. Safe memory ranges can
  // be obtained from calls to vkMapMemory() or posix_memalign()/memalign()
  // with the page size used as the alignment parameter.
  bool AddTrackingRangeImpl(void* start, size_t size);

  // Removes an address range specified with starting address and size from
  // tracking. Also recovers the write and read permission of the corresponding
  // memory pages. Returns true if the range is removed successfully, otherwise
  // returns false. To have a successful removal, the given range must match
  // exactly with an existing range, i.e. both the starting address and the
  // size must match.
  bool RemoveTrackingRangeImpl(void* start, size_t size);

  // Removes all the tracking ranges, recover the write and read permission of
  // all the corresponding memory pages. Returns true if all the pages are
  // recovered and ranges are removed, otherwise returns false;
  bool ClearTrackingRangesImpl();

  // HandleSegfaultImpl is the core of memory tracker function. It
  // checks wheter the fault address is within a tracking range. If so, it
  // records the page address and recovers the read/write permission of that
  // page and returns true. Otherwise, it return false.
  // Besides, recording of the dirty page address must not allocates new memory
  // space. If new space is required, fallback to the original segfault
  // handler.
  bool HandleSegfaultImpl(void* fault_addr);

  // Returns a vector of dirty pages addresses and clear all the records of
  // dirty pages, also resets the pages access permission, if they overlaps
  // with any tracking ranges, to not-accessible or readonly depends on whether
  // this memory tracker should track read operations.
  std::vector<void*> GetAndResetAllDirtyPagesImpl() {
    auto r =
        dirty_pages_.DumpAndClearInRange(reinterpret_cast<void*>(0x0), ~0x0);
    ResetPagesToTrackImpl(r);
    return r;
  }

  // Returns a vector of dirty pages addresses that overlaps a memory range
  // specified with |start| and |size|.
  std::vector<void*> GetDirtyPagesInRangeImpl(void* start, size_t size) {
    void* start_page_aligned = GetAlignedAddress(start, page_size_);
    size_t size_page_aligned = GetAlignedSize(start, size, page_size_);
    return dirty_pages_.DumpAndClearInRange(start_page_aligned,
                                            size_page_aligned);
  }

  // Sets the access permission flag of a given vector of |pages| back to
  // readonly or not-accessible if the page overlaps with any tracking ranges.
  // Returns true if all the page flags are set successfully, otherwise returns
  // false.
  bool ResetPagesToTrackImpl(const std::vector<void*>& pages) {
    bool succeeded = true;
    std::for_each(pages.begin(), pages.end(), [this, &succeeded](void* p) {
      if (IsInRanges(reinterpret_cast<uintptr_t>(p), ranges_, true)) {
        succeeded &=
            set_protection(p, page_size_, track_read_ ? PageProtections::kNone : PageProtections::kRead) == 0;
      }
    });
    return succeeded;
  }

  // Dummy function that we can pass down to the specific memory tracker.
  bool DoHandleSegfault(void* v) {
    return HandleSegfault(v);
  }

  // Returns a vector of dirty pages addresses that overlaps a memory range
  // specified with |start| and |size|. Then clears all the records of dirty
  // pages in that range, and resets the pages access permission, if they
  // overlaps with any tracking ranges, to not-accessible or readonly depends
  // on whether this memory tracker should track read operations.
  std::vector<void*> GetAndResetDirtyPagesInRangeImpl(void* start,
                                                      size_t size) {
    auto r = GetDirtyPagesInRangeImpl(start, size);
    ResetPagesToTrackImpl(r);
    return r;
  }

  bool track_read_;  // A flag to indicate whether to track read operations on
                     // the tracking memory ranges.

  const size_t page_size_;          // Size of a memory page in byte
  SpinLock l_;  // Spin lock to guard the accesses of shared data
  std::map<uintptr_t, size_t> ranges_;  // Memory ranges registered for tracking
  DirtyPageTable dirty_pages_;          // Storage of dirty pages

 public:
  size_t page_size() const { return page_size_; }

// SignalSafe wrapped methods that access shared data and cannot be
// interrupted by SIGSEGV signal.
#define SIGNAL_SAFE(function) \
  SignalSafe<MemoryTrackerImpl, decltype(&MemoryTrackerImpl::function##Impl)> function;
  SIGNAL_SAFE(AddTrackingRange);
  SIGNAL_SAFE(RemoveTrackingRange);
  SIGNAL_SAFE(GetDirtyPagesInRange);
  SIGNAL_SAFE(ResetPagesToTrack);
  SIGNAL_SAFE(GetAndResetAllDirtyPages);
  SIGNAL_SAFE(GetAndResetDirtyPagesInRange);
  SIGNAL_SAFE(ClearTrackingRanges);
  SIGNAL_SAFE(EnableMemoryTracker);
  SIGNAL_SAFE(DisableMemoryTracker);
#undef SIGNAL_SAFE

// SpinLockGuarded wrapped methods that access critical region.
#define LOCKED(function)                                                   \
  SpinLockGuarded<MemoryTrackerImpl, decltype(&MemoryTrackerImpl::function##Impl)> \
      function;
  LOCKED(HandleSegfault);
#undef LOCKED
};
}  // namespace track_memory
}  // namespace gapii

#if IS_POSIX
#include  "core/memory_tracker/cc/posix/memory_tracker.inl"
#else
#include  "core/memory_tracker/cc/windows/memory_tracker.inl"
#endif

#endif // COHERENT_TRACKING_ENABLED
#endif  // GAPII_MEMORY_TRACKER_H