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
#include "core/memory/arena/cc/arena.h"
#include "core/memory/arena/cc/stl_compatible_allocator.h"

#ifndef GAPII_MEMORY_TRACKER_H
#define GAPII_MEMORY_TRACKER_H

#define COHERENT_TRACKING_ENABLED 1
#if COHERENT_TRACKING_ENABLED
#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID)
#define IS_POSIX 1
#include "posix/memory_tracker.h"
#elif (TARGET_OS == GAPID_OS_WINDOWS)
#define IS_POSIX 0
#include "windows/memory_tracker.h"
#else
#undef COHERENT_TRACKING_ENABLED
#define COHERENT_TRACKING_ENABLED 0
#endif
#endif  // COHERENT_TRACKING_ENABLED

#if COHERENT_TRACKING_ENABLED

#include <algorithm>
#include <atomic>
#include <functional>
#include <iterator>
#include <list>
#include <map>
#include <memory>
#include <utility>
#include <vector>

#include "core/memory_tracker/cc/memory_protections.h"

namespace gapii {
namespace track_memory {

// Returns the uppper bound aligned address for a given pointer |addr| and a
// given |alignment|. |alignment| must be a power of two and cannot be zero.
// If the given |alignment| is zero or not a power of two, the return is
// undefined.
inline uintptr_t RoundUpAlignedAddress(uintptr_t addr, size_t alignment) {
  return (addr + alignment - 1u) & ~(alignment - 1u);
}

// Returns the lower bound aligned address for a given pointer |addr| and a
// given |alignment|. |alignment| must be a power of two and cannot be zero.
// If the given |alignment| is zero or not a power of two, the return is
// undefined.
inline uintptr_t RoundDownAlignedAddress(uintptr_t addr, size_t alignment) {
  return addr & ~(alignment - 1u);
}

// SpinLock is a spin lock implemented with atomic variable and operations.
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
    return this->SpinLockGuardedFunctor::template operator()<Args...>(
        std::forward<Args>(args)...);
  }

 protected:
  int sig_;
};

// MarkList is a preallocated container of marked T, keeps track of the marked
// T. It allows the user to traverse through the marked Ts and specify if a
// marked T should be unmarked. However, MarkList does not guarantee any
// traversing order.
template <typename T>
class MarkList {
 public:
  MarkList(size_t size) : v_(size), marked_count_(0), marked_end_(0) {}

  // Not copyable, not movable
  MarkList(const MarkList&) = delete;
  MarkList(MarkList&&) = delete;
  MarkList& operator=(const MarkList&) = delete;
  MarkList& operator=(MarkList&&) = delete;

  // Add |val| to this mark list.
  bool Mark(T val) {
    if (marked_end_ == v_.size()) {
      return false;
    }
    v_[marked_end_] = val;
    marked_count_++;
    marked_end_++;
    return true;
  }

  // Traverse through all the marked items in this mark list with the given
  // callback function, unmark the item if the callback function returns true
  // for the feeding item.
  void ForEachMarked(std::function<bool(const T&)> unmark_if) {
    for (size_t i = 0; i < marked_end_;) {
      if (unmark_if(v_[i])) {
        v_[i] = v_[marked_end_ - 1];
        marked_end_ -= 1;
        continue;
      }
      i++;
    }
  }

 protected:
  std::vector<T> v_;
  size_t marked_count_;
  // One over the index of the last marked element.
  size_t marked_end_;
};

// PageCell records the 'dirtiness' state for multiple actual memory pages.
struct PageCell {
  // The maximum allowed page index of pages grouped in one PageCell.
  // The maximum allowed value for kMaxPageIndex is 255.
  static const uint8_t kMaxPageIndex = 0x7;
  // The maximum allowed number of pages can be represented by one PageCell.
  // This is the kMaxPageIndex plus one, a maximum value for it is 256, on a
  // 4K aligned machine, this represents 1MB memory span.
  static const size_t kMaxPageCount = size_t(kMaxPageIndex) + 1;

  enum class State : uint8_t { kClear = 0x00000000, kDirty = 0x00000001 };

  // The last index of the pages represented in this PageCell. This value
  // should never exceed kMaxPageIndex.
  uint8_t last_page_index;

  State state;
};

// TrackingRange represents a memory range to be tracked. It contains a list of
// PageCells and manage the state of those cells.
template <typename PageCellIndex>
class TrackingRange {
 public:
  using OnSetDirty = bool (*)(uintptr_t dirty_addr, size_t dirty_size);

  TrackingRange(uintptr_t start, size_t size)
      : start_(start),
        aligned_start_(RoundDownAlignedAddress(start, GetPageSize())),
        size_(size),
        aligned_size_(RoundUpAlignedAddress(start + size, GetPageSize()) -
                      aligned_start_),
        num_pages_(aligned_size_ / GetPageSize()),
        num_cells_((num_pages_ + PageCell::kMaxPageIndex) /
                   PageCell::kMaxPageCount),
        cells_(num_cells_, {PageCell::kMaxPageIndex, PageCell::State::kClear}),
        dirty_cell_indices_(num_cells_) {
    if (num_pages_ % PageCell::kMaxPageCount != 0) {
      cells_.back().last_page_index =
          (num_pages_ % PageCell::kMaxPageCount) - 1u;
    }
  }

  // Not copyable, not movable.
  TrackingRange(const TrackingRange&) = delete;
  TrackingRange(TrackingRange&&) = delete;
  TrackingRange& operator=(const TrackingRange&) = delete;
  TrackingRange& operator=(TrackingRange&&) = delete;

  // Traverse through all the memory address ranges marked as 'dirty' with the
  // given |clear_if| callback function and clear the state of the memory range
  // if the callback function returns true.
  void ForDirtyIntersects(
      uintptr_t addr, size_t size,
      std::function<bool(uintptr_t intersect_addr, size_t intersect_size)>
          clear_if) {
    uintptr_t end = addr + size;
    dirty_cell_indices_.ForEachMarked(
        [this, addr, end, &clear_if](const PageCellIndex& cid) -> bool {
          uintptr_t c_addr = GetCellAddr(cid);
          uintptr_t c_end = c_addr + GetCellSize(cid);
          uintptr_t i_addr = std::max(c_addr, addr);
          uintptr_t i_end = std::min(c_end, end);
          if (i_addr < i_end) {
            if (clear_if(c_addr, c_end - c_addr)) {
              cells_[cid].state = PageCell::State::kClear;
              return true;
            }
          }
          return false;
        });
  }

  // Set memory range 'dirty', the memory range to set is derived from the
  // given memory range specified with |addr| and |size|. The actual 'dirty'
  // range is guaranteed to fully cover the range of |addr| and |size| and is
  // page aligned. The given |on_set| callback will be called with the acutal
  // 'dirty' range. The |on_set| callback is enforce to be a function pointer
  // to avoid extra memory allocations (which is not safe in signal handler).
  bool SetDirty(uintptr_t addr, size_t size, OnSetDirty on_set) {
    PageCellIndex start_id = 0;
    PageCellIndex end_id = 0;
    if (!GetCellFor(addr, &start_id)) {
      return false;
    }
    if (!GetCellFor(addr + size - 1u, &end_id)) {
      return false;
    }
    bool result = true;
    for (PageCellIndex i = start_id; i <= end_id; i++) {
      if (cells_[i].state != PageCell::State::kDirty) {
        cells_[i].state = PageCell::State::kDirty;
        dirty_cell_indices_.Mark(i);
      }
      result &= on_set(GetCellAddr(i), GetCellSize(i));
    }
    return result;
  }

  // Returns true if the given memory address is in this tracking range and
  // is 'dirty'. Otherwise returns false.
  bool IsDirty(uintptr_t addr) const {
    PageCellIndex cid = 0;
    if (!GetCellFor(addr, &cid)) {
      return false;
    }
    return cells_[cid].state == PageCell::State::kDirty;
  }

  // Returns true if this tracking range overlaps with range [start, end).
  bool Overlaps(uintptr_t start, uintptr_t end) const {
    uintptr_t s = std::max(start_, start);
    uintptr_t e = std::min(start_ + size_, end);
    return s < e;
  }

  uintptr_t start() const { return start_; }
  uintptr_t end() const { return start_ + size_; }
  uintptr_t aligned_start() const { return aligned_start_; }
  uintptr_t aligned_size() const { return aligned_size_; }

 protected:
  // Get the beginning address represented by the cell at index |cid|
  uintptr_t GetCellAddr(PageCellIndex cid) const {
    return aligned_start_ + cid * PageCell::kMaxPageCount * GetPageSize();
  }
  // Get the memory space size represented by the cell at index |cid|
  uintptr_t GetCellSize(PageCellIndex cid) const {
    return (cells_[cid].last_page_index + size_t(1u)) * GetPageSize();
  }
  // Get the index of the cell which covers the memory address |addr|
  bool GetCellFor(uintptr_t addr, PageCellIndex* cid) const {
    if (!cid || addr < aligned_start_ ||
        addr >= aligned_start_ + aligned_size_) {
      return false;
    }
    *cid = (addr - aligned_start_) / GetPageSize() / PageCell::kMaxPageCount;
    return true;
  }

  const uintptr_t start_;
  const uintptr_t aligned_start_;
  const size_t size_;
  const size_t aligned_size_;
  const size_t num_pages_;
  const size_t num_cells_;

  // Contains the dirtiness state.
  std::vector<PageCell> cells_;
  // Contains the indices of dirty PageCells.
  MarkList<PageCellIndex> dirty_cell_indices_;
};

// MemoryTrackerImpl utilizes Segfault signal on Linux to track accesses to
// memories.
template <typename SpecificMemoryTracker>
class MemoryTrackerImpl : public SpecificMemoryTracker {
 public:
  using derived_tracker_type = SpecificMemoryTracker;
  // Assume mapped coherement memory range can not be larger than 4GB.
  using tracking_range_type = TrackingRange<uint32_t>;
  using tracking_range_list_allocator = core::StlCompatibleAllocator<
      std::pair<const uintptr_t, std::unique_ptr<tracking_range_type>>>;
  using tracking_range_list_type =
      std::map<uintptr_t, std::unique_ptr<tracking_range_type>,
               std::less<uintptr_t>, tracking_range_list_allocator>;

  // Creates a memory tracker to track memory write operations. If
  // |track_read| is set to true, also tracks the memory read operations.
  // By default |track_read| is set to false.
  MemoryTrackerImpl(bool track_read = false)
      : SpecificMemoryTracker([this](void* v) { return DoHandleSegfault(v); }),
        track_read_(track_read),
        l_(),
        tracking_ranges_(std::less<uintptr_t>(), &arena_),
#define CONSTRUCT_SIGNAL_SAFE(function) \
  function(this, &MemoryTrackerImpl::function##Impl, &l_, SIGSEGV)
        CONSTRUCT_SIGNAL_SAFE(TrackRange),
        CONSTRUCT_SIGNAL_SAFE(UntrackRange),
        CONSTRUCT_SIGNAL_SAFE(HandleAndClearDirtyIntersects),
        CONSTRUCT_SIGNAL_SAFE(EnableMemoryTracker),
        CONSTRUCT_SIGNAL_SAFE(DisableMemoryTracker),
#undef CONSTRUCT_SIGNAL_SAFE
#define CONSTRUCT_LOCKED(function) \
  function(this, &MemoryTrackerImpl::function##Impl, &l_)
        CONSTRUCT_LOCKED(HandleSegfault)
#undef CONSTRUCT_LOCKED
  {
  }
  // Not copyable, not movable.
  MemoryTrackerImpl(const MemoryTrackerImpl&) = delete;
  MemoryTrackerImpl(MemoryTrackerImpl&&) = delete;
  MemoryTrackerImpl& operator=(const MemoryTrackerImpl&) = delete;
  MemoryTrackerImpl& operator=(MemoryTrackerImpl&&) = delete;

  ~MemoryTrackerImpl() { DisableMemoryTracker(); }

 protected:
  // Adds an address range specified by |start| address and |size| for tracking
  // and set accessing permissions of the corresponding pages to track write
  // (and read, if track_read is true) operations. Returns true if the range
  // is added successfully, returns false when the operaion is not done
  // successfully, e.g. the range overlaps with exisiting ranges or the size is
  // 0. The ranges must not contain any pages that store the data of this
  // memory tracker object, otherwise the segfault signal will be handled by
  // the original handler and the application may crash. Safe memory ranges can
  // be obtained from calls to vkMapMemory() or posix_memalign()/memalign()
  // with the page size used as the alignment parameter.
  bool TrackRangeImpl(void* start, size_t size);

  // Removes an address range specified with |start| address and |size| from
  // tracking. Also recovers the write and read permission of the corresponding
  // memory pages. Returns true if such a tracking range does not exist or is
  // removed successfully, otherwise returns false. To have a successful
  // removal, the given range must match exactly with an existing tracking
  // range, i.e. both the starting address and the size must match.
  bool UntrackRangeImpl(void* start, size_t size);

  // HandleSegfaultImpl is the core of memory tracker function. It
  // checks wheter the fault address is within a tracking range. If so, it
  // records the page address and recovers the read/write permission of that
  // page and returns true. Otherwise, it return false.
  // Besides, recording of the dirty page address must not allocates new memory
  // space. If new space is required, fallback to the original segfault
  // handler.
  bool HandleSegfaultImpl(void* fault_addr);

  // Get all the intersects of 'dirty' memory ranges and the range specified
  // with |start| and |size|, traverse all such intersects with the given
  // |handle_dirty| callback function, then clear all of traversed 'dirty'
  // intersects.
  bool HandleAndClearDirtyIntersectsImpl(
      void* start, size_t size,
      std::function<void(void* dirty_addr, size_t dirty_size)> handle_dirty);

  bool DisableMemoryTrackerImpl() {
    // Loop over tracking_ranges_ but DO NOT use an iterator, as
    // UntrackRangeImpl() itself uses an iterator over tracking_ranges_ to
    // *delete* an element.

    struct range {
      uintptr_t start;
      uintptr_t end;
    };
    std::vector<struct range> ranges;
    ranges.reserve(tracking_ranges_.size());
    for (const auto& r : tracking_ranges_) {
      ranges.push_back({r.second->start(), r.second->end()});
    }

    bool result = true;
    for (auto r : ranges) {
      result &=
          UntrackRangeImpl(reinterpret_cast<void*>(r.start), r.end - r.start);
    }
    result &= derived_tracker_type::DisableMemoryTrackerImpl();
    return result;
  }

  // Placeholder function that we can pass down to the specific memory tracker.
  bool DoHandleSegfault(void* v) { return HandleSegfault(v); }

  // A helper function that returns the first tracking range that overlaps with
  // the given address specified by starting |addr| and |size|.
  tracking_range_list_type::iterator FirstOverlappedRange(uintptr_t addr,
                                                          size_t size);

  // A flag to indicate whether to track read operations on the tracking memory
  // ranges
  bool track_read_;

  // Spin lock to guard the accesses of shared data
  SpinLock l_;

  core::Arena arena_;

  // Ranges registered for tracking. Stored in ordered map according to their
  // end address.
  tracking_range_list_type tracking_ranges_;

 public:
// SignalSafe wrapped methods that access shared data and cannot be
// interrupted by SIGSEGV signal.
#define SIGNAL_SAFE(function)                                                 \
  SignalSafe<MemoryTrackerImpl, decltype(&MemoryTrackerImpl::function##Impl)> \
      function;
  SIGNAL_SAFE(TrackRange);
  SIGNAL_SAFE(UntrackRange);
  SIGNAL_SAFE(HandleAndClearDirtyIntersects);
  SIGNAL_SAFE(EnableMemoryTracker);
  SIGNAL_SAFE(DisableMemoryTracker);
#undef SIGNAL_SAFE

// SpinLockGuarded wrapped methods that access critical region.
#define LOCKED(function)                                        \
  SpinLockGuarded<MemoryTrackerImpl,                            \
                  decltype(&MemoryTrackerImpl::function##Impl)> \
      function;
  LOCKED(HandleSegfault);
#undef LOCKED
};
}  // namespace track_memory
}  // namespace gapii

#if IS_POSIX
#include "core/memory_tracker/cc/posix/memory_tracker.inc"
#else
#include "core/memory_tracker/cc/windows/memory_tracker.inc"
#endif

#endif  // COHERENT_TRACKING_ENABLED
#endif  // GAPII_MEMORY_TRACKER_H
