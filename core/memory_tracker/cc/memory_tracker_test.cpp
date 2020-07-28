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

#include <gmock/gmock.h>

#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>

#include <atomic>
#include <condition_variable>
#include <list>
#include <map>
#include <thread>
#include <unordered_map>

namespace gapii {
namespace track_memory {
namespace test {

using ::testing::Contains;

TEST(RoundUpAlignedAddress, 4KAligned) {
  EXPECT_EQ(0x0, RoundUpAlignedAddress(0x0, 0x1000));
  EXPECT_EQ(0x1000, RoundUpAlignedAddress(0x1, 0x1000));
  EXPECT_EQ(0x1000, RoundUpAlignedAddress(0x100, 0x1000));
  EXPECT_EQ(0x1000, RoundUpAlignedAddress(0x800, 0x1000));
  EXPECT_EQ(0x1000, RoundUpAlignedAddress(0xFFF, 0x1000));
  EXPECT_EQ(0x1000, RoundUpAlignedAddress(0x1000, 0x1000));
  EXPECT_EQ(0x2000, RoundUpAlignedAddress(0x1FFF, 0x1000));
  EXPECT_EQ(0x2612000, RoundUpAlignedAddress(0x2611001, 0x1000));
  EXPECT_EQ(0x2612000, RoundUpAlignedAddress(0x2611FFF, 0x1000));
  EXPECT_EQ(0xFFFFF000, RoundUpAlignedAddress(0xFFFFE001, 0x1000));
}

TEST(RoundUpAlignedAddress, 64KAligned) {
  EXPECT_EQ(0x0, RoundUpAlignedAddress(0x0, 0x10000));
  EXPECT_EQ(0x10000, RoundUpAlignedAddress(0x1, 0x10000));
  EXPECT_EQ(0x10000, RoundUpAlignedAddress(0x100, 0x10000));
  EXPECT_EQ(0x10000, RoundUpAlignedAddress(0x1800, 0x10000));
  EXPECT_EQ(0x10000, RoundUpAlignedAddress(0xFFFF, 0x10000));
  EXPECT_EQ(0x10000, RoundUpAlignedAddress(0x10000, 0x10000));
  EXPECT_EQ(0x20000, RoundUpAlignedAddress(0x1FFFF, 0x10000));
  EXPECT_EQ(0x2620000, RoundUpAlignedAddress(0x2610001, 0x10000));
  EXPECT_EQ(0xFFFF0000, RoundUpAlignedAddress(0xFFFE0001, 0x10000));
}

TEST(RoundDownAlignedAddress, 4KAligned) {
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x0, 0x1000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x50, 0x1000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x100, 0x1000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x800, 0x1000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0xFFF, 0x1000));
  EXPECT_EQ(0x1000, RoundDownAlignedAddress(0x1000, 0x1000));
  EXPECT_EQ(0x1000, RoundDownAlignedAddress(0x1FFF, 0x1000));
  EXPECT_EQ(0x2611000, RoundDownAlignedAddress(0x2611001, 0x1000));
  EXPECT_EQ(0x2611000, RoundDownAlignedAddress(0x2611FFF, 0x1000));
  EXPECT_EQ(0xFFFFF000, RoundDownAlignedAddress(0xFFFFFFFF, 0x1000));
}

TEST(RoundDownAlignedAddress, 64KAligned) {
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x0, 0x10000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x50, 0x10000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x100, 0x10000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0x1800, 0x10000));
  EXPECT_EQ(0x0, RoundDownAlignedAddress(0xFFFF, 0x10000));
  EXPECT_EQ(0x10000, RoundDownAlignedAddress(0x10000, 0x10000));
  EXPECT_EQ(0x10000, RoundDownAlignedAddress(0x1FFFF, 0x10000));
  EXPECT_EQ(0x2610000, RoundDownAlignedAddress(0x2611001, 0x10000));
  EXPECT_EQ(0xFFFF0000, RoundDownAlignedAddress(0xFFFFFFFF, 0x10000));
}

// If the alignment value is invalid, just make sure we don't crash, the
// results are undefined in such cases.
TEST(RoundUpAlignedAddress, Invalid) {
  // Alignment is zero
  RoundUpAlignedAddress(0x12345678, 0x0);
  // Alignment is not a power of two
  RoundUpAlignedAddress(0x12345678, 0x7);
  RoundUpAlignedAddress(0x12345678, 0xFFFF);
}

TEST(RoundDownAlignedAddress, Invalid) {
  // Alignment is zero
  RoundDownAlignedAddress(0x12345678, 0x0);
  // Alignment is not a power of two
  RoundDownAlignedAddress(0x12345678, 0x7);
  RoundDownAlignedAddress(0x12345678, 0xFFFF);
}

namespace {
using SignalHandlerType = void (*)(int, siginfo_t*, void*);

// Registers a signal handler function for a specific signal and store the
// original signal action struct in |orig_action|. Returns true if the signal
// handler is installed successfully, otherwise returns false;
bool RegisterSignalHandler(int sig, SignalHandlerType h,
                           struct sigaction* orig_action) {
  struct sigaction sa;
  sa.sa_flags = SA_SIGINFO;
  sigemptyset(&sa.sa_mask);
  sa.sa_sigaction = h;
  return sigaction(sig, &sa, orig_action) != -1;
}

// A test fixture to ease two kinds of tests:
//   1) Two threads won't be accessing the same critical region
//   2) When a function is being executed in a normal thread, the signal
//   whose handler will call the same function should be blocked.
class TestFixture : public ::testing::Test {
 public:
  void Init() {
    mutex_thread_init_order_.lock();
    normal_thread_run_before_signal_handler_.lock();
    m_.lock();
    fake_lock_.store(false, std::memory_order_seq_cst);
    deadlocked_.store(false, std::memory_order_seq_cst);
    unique_test_ = this;
  }

  // The two given function will be executed in two threads, one is guaranteed
  // to be initiated before the other one.
  void RunInTwoThreads(std::function<void(void)> initiated_first,
                       std::function<void(void)> initiated_second) {
    std::thread thread_first([this, &initiated_first]() {
      mutex_thread_init_order_.unlock();
      initiated_first();
    });
    std::thread thread_second([this, &initiated_second]() {
      mutex_thread_init_order_.lock();
      initiated_second();
    });
    thread_first.join();
    thread_second.join();
  }

  // DoTask acquires the fake lock and runs the given function.
  void DoTask(std::function<void(void)> task) {
    FakeLock();
    normal_thread_run_before_signal_handler_
        .unlock();  // This is to make sure the signal will only be send after
                    // the normal threads
    task();
    FakeUnlock();
  }

  // RegisterHandlerAndTriggerSignal does the following things:
  //  1) Registers a signal handler to SIGUSR1 which acquires the fake lock.
  //  2) Creates a child thread which will acquire the fake lock.
  //  3) Waits for the child thread to run then sends a signal interrupt to the
  //  child thread.
  //  4) Waits until the child thread finishes and clean up.
  void RegisterHandlerAndTriggerSignal(void* (*child_thread_func)(void*)) {
    auto handler_func = [](int, siginfo_t*, void*) {
      // FakeLock is to simulate the case that the coherent memory tracker's
      // signal handler needs to access shared data.
      unique_test_->FakeLock();
      ;
      unique_test_->FakeUnlock();
    };
    struct sigaction orig_action;
    ASSERT_TRUE(RegisterSignalHandler(SIGUSR1, handler_func, &orig_action));
    pthread_t child_thread;
    EXPECT_EQ(
        0, pthread_create(&child_thread, nullptr, child_thread_func, nullptr));
    normal_thread_run_before_signal_handler_.lock();
    pthread_kill(child_thread, SIGUSR1);
    pthread_join(child_thread, nullptr);
    ASSERT_EQ(0, sigaction(SIGUSR1, &orig_action, nullptr));
  }

 protected:
  // Acquires the fake lock. If the fake lock has already been locked, i.e.
  // its value is true, sets the deadlocked_ flag.
  void FakeLock() {
    if (fake_lock_.load(std::memory_order_seq_cst)) {
      deadlocked_.exchange(true);
    }
    fake_lock_.exchange(true);
  }
  // Releases the fake lock by setting its value to false.
  void FakeUnlock() { fake_lock_.exchange(false); }

  std::mutex normal_thread_run_before_signal_handler_;  // A mutex to
                                                        // guarantee the normal
                                                        // thread is initiated
                                                        // before the signal
                                                        // handler is called.
  std::mutex mutex_thread_init_order_;  // A mutex to guarantee one thread is
                                        // initiated before the other one
  std::mutex m_;  // A mutex for tuning execution order in tests
  std::atomic<bool>
      fake_lock_;  // An atomic bool to simulate the lock/unlock state
  std::atomic<bool> deadlocked_;  // A flag to indicate deadlocked or not state

  static TestFixture* unique_test_;  // A static pointer of this test fixture,
                                     // required as signal handler must be a
                                     // static function, and we will need a
                                     // static pointer to access the member
                                     // functions.
};
TestFixture* TestFixture::unique_test_ = nullptr;
}  // namespace

// A helper function to ease void* pointer + offset calculation.
void* VoidPointerAdd(void* addr, ssize_t offset) {
  return reinterpret_cast<void*>(reinterpret_cast<uintptr_t>(addr) + offset);
}

using SpinLockTest = TestFixture;

// Thread A sleeps first, then increases the counter, but Thread B waits for
// the spin lock. So Thread A increases the counter before Thread B.
TEST_F(SpinLockTest, WithSpinLockGuard) {
  Init();
  uint32_t counter = 0u;
  SpinLock l;
  RunInTwoThreads(
      // Thread A is initiated first and runs first.
      [this, &counter, &l]() {
        SpinLockGuard g(&l);
        m_.unlock();
        usleep(5000);
        counter++;
        EXPECT_EQ(1u, counter);
      },
      // Thread B is initiated secondly and runs after thread A runs.
      [this, &counter, &l]() {
        m_.lock();  // make sure Thread B's SpinLockGuard is created after
                    // Thread A's
        SpinLockGuard g(&l);
        counter++;
        EXPECT_EQ(2u, counter);
      });
  EXPECT_EQ(2u, counter);
}

using SignalBlockerTest = TestFixture;

// A thread acquires the a lock first, then a signal interrupts the thread and
// acquires the lock again. This results into a deadlock.
TEST_F(SignalBlockerTest, WithoutBlocker) {
  Init();
  auto normal_thread_func = [](void*) -> void* {
    unique_test_->DoTask([]() { usleep(5000); });
    return nullptr;
  };

  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  // Expect dead locked
  EXPECT_TRUE(deadlocked_.load(std::memory_order_seq_cst));
}

// A thread acquires the a lock first, then a signal tries to interrupt the
// thread. But the signal is blocked, signal handler is called after the thread
// finishes its job. This avoids dead lock.
TEST_F(SignalBlockerTest, WithBlocker) {
  Init();
  auto normal_thread_func = [](void*) -> void* {
    SignalBlocker b(SIGUSR1);
    unique_test_->DoTask([]() { usleep(5000); });
    return nullptr;
  };

  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  // Expect dead locked
  EXPECT_FALSE(deadlocked_.load(std::memory_order_seq_cst));
}

// Signal is blocked before a thread is created. The child thread inherits the
// signal mask from the parent thread. Dead lock does not happen.
TEST_F(SignalBlockerTest, BlockerDoesAffectChildThread) {
  SignalBlocker b(SIGUSR1);
  Init();
  auto normal_thread_func = [](void*) -> void* {
    unique_test_->DoTask([]() { usleep(5000); });
    return nullptr;
  };

  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  // Expect dead locked
  EXPECT_FALSE(deadlocked_.load(std::memory_order_seq_cst));
}

// Signal is blocked in both the parent thread and the child thread.
TEST_F(SignalBlockerTest, BlockerRecursiveSafe) {
  SignalBlocker b(SIGUSR1);
  Init();
  auto normal_thread_func = [](void*) -> void* {
    SignalBlocker b(SIGUSR1);
    unique_test_->DoTask([]() { usleep(5000); });
    return nullptr;
  };

  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  // Expect dead locked
  EXPECT_FALSE(deadlocked_.load(std::memory_order_seq_cst));
}

using WrapperTest = TestFixture;

// Call the member function: DoTask() without spin lock guard.
TEST_F(WrapperTest, WithoutSpinLockGuardedWrapper) {
  Init();
  uint32_t counter = 0u;
  RunInTwoThreads(
      // Initiated first
      [this, &counter]() {
        DoTask([this, &counter]() {
          m_.unlock();
          usleep(5000);
          counter++;
          EXPECT_EQ(2u, counter);
        });
      },
      // Initiated secondly
      [this, &counter]() {
        DoTask([this, &counter]() {
          m_.lock();
          counter++;
          EXPECT_EQ(1u, counter);
        });
      });
  EXPECT_EQ(2u, counter);
}

// Call the member function: DoTask() with spin lock guarded.
TEST_F(WrapperTest, WithSpinLockGuardedWrapper) {
  Init();
  SpinLock l;
  int counter = 0;
  auto LockedDoTask =
      SpinLockGuarded<WrapperTest, decltype(&WrapperTest::DoTask)>(
          this, &WrapperTest::DoTask, &l);
  RunInTwoThreads(
      // Initiated first
      [this, &counter, &LockedDoTask]() {
        LockedDoTask([this, &counter]() {
          m_.unlock();  // This is to make sure the second initiated thread has
                        // its SpinLockGuard created after the first initiated
                        // thread
          usleep(5000);
          counter++;
          EXPECT_EQ(1u, counter);
        });
      },
      // Initiated secondly
      [this, &counter, &LockedDoTask]() {
        m_.lock();
        LockedDoTask([this, &counter]() {
          counter++;
          EXPECT_EQ(2u, counter);
        });
      });
  EXPECT_EQ(2u, counter);
}

// Without a signal safe wrapper, spin lock acquired in function may cause
// deadlock.
TEST_F(WrapperTest, WithoutSignalSafeWrapper) {
  Init();
  auto normal_thread_func = [](void*) -> void* {
    unique_test_->DoTask([]() { usleep(5000); });
    return nullptr;
  };
  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  EXPECT_TRUE(deadlocked_.load(std::memory_order_seq_cst));
}

// With a signal safe wrapper, signal interrupt will be blocked, so signal
// handler will not cause a deadlock.
TEST_F(WrapperTest, WithSignalSafeWrapper) {
  Init();
  SpinLock l;
  auto wrapped_do_task =
      SignalSafe<WrapperTest, decltype(&WrapperTest::DoTask)>(
          this, &WrapperTest::DoTask, &l, SIGUSR1);
  static auto* wrapped_func_ptr = &wrapped_do_task;
  auto normal_thread_func = [](void*) -> void* {
    (*wrapped_func_ptr)([]() { usleep(5000); });
    return nullptr;
  };
  std::thread start_normal_thread_and_send_signal(
      [this, &normal_thread_func]() {
        RegisterHandlerAndTriggerSignal(normal_thread_func);
      });
  start_normal_thread_and_send_signal.join();
  EXPECT_FALSE(deadlocked_.load(std::memory_order_seq_cst));
}

template <typename T>
class MarkListTest : public ::testing::Test {
 protected:
  MarkList<T>* CreateMarkList(size_t size) {
    m_.reset(new MarkList<T>(size));
    return m_.get();
  }
  std::unique_ptr<MarkList<T>> m_;
};

using MarkListTestTypes = ::testing::Types<uint32_t, uint64_t>;

TYPED_TEST_CASE(MarkListTest, MarkListTestTypes);

TYPED_TEST(MarkListTest, NoSpace) {
  auto m = this->CreateMarkList(0);
  EXPECT_FALSE(m->Mark(TypeParam()));
  size_t marked = 0;
  m->ForEachMarked([&marked](const TypeParam&) -> bool {
    marked += 1;
    return false;
  });
  EXPECT_EQ(0, marked);
}

TEST(MarkListTestUint32, Mark) {
  MarkList<uint32_t> m(4);
  EXPECT_TRUE(m.Mark(1));
  EXPECT_TRUE(m.Mark(2));
  EXPECT_TRUE(m.Mark(3));
  // Mark 3 twice
  EXPECT_TRUE(m.Mark(3));
  // 5 should fail.
  EXPECT_FALSE(m.Mark(5));

  std::unordered_map<uint32_t, size_t> expected;
  expected[1] = 1;
  expected[2] = 1;
  expected[3] = 2;

  std::unordered_map<uint32_t, size_t> actual;
  m.ForEachMarked([&actual](const uint32_t& t) -> bool {
    if (actual.find(t) == actual.end()) {
      actual[t] = 0;
    }
    actual[t] += 1;
    return false;
  });
  EXPECT_THAT(expected, ::testing::ContainerEq(actual));
}

TYPED_TEST(MarkListTest, DoNotClear) {
  auto m = this->CreateMarkList(4);
  TypeParam item = TypeParam();
  EXPECT_TRUE(m->Mark(item));
  EXPECT_TRUE(m->Mark(item));
  EXPECT_TRUE(m->Mark(item));

  size_t marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return false;
  });
  EXPECT_EQ(3u, marked);

  EXPECT_TRUE(m->Mark(item));
  marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return false;
  });
  EXPECT_EQ(4u, marked);

  EXPECT_FALSE(m->Mark(item));
  marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return false;
  });
  EXPECT_EQ(4u, marked);
}

TYPED_TEST(MarkListTest, Clear) {
  auto m = this->CreateMarkList(4);
  TypeParam item = TypeParam();
  EXPECT_TRUE(m->Mark(item));
  EXPECT_TRUE(m->Mark(item));
  EXPECT_TRUE(m->Mark(item));
  EXPECT_TRUE(m->Mark(item));

  size_t marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return true;
  });
  EXPECT_EQ(4u, marked);

  marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return true;
  });
  EXPECT_EQ(0u, marked);

  EXPECT_TRUE(m->Mark(item));
  marked = 0;
  m->ForEachMarked([&marked](const TypeParam& t) -> bool {
    marked += 1;
    return true;
  });
  EXPECT_EQ(1u, marked);
}

namespace {
class AlignedMemory {
 public:
  AlignedMemory(size_t alignment, size_t size) : mem_(nullptr) {
    if (auto err = posix_memalign(&mem_, alignment, size) != 0) {
      std::cerr << "posix_memalign failed with " << err;
    }
  }
  ~AlignedMemory() { free(mem_); }
  void* mem() const { return mem_; }

 private:
  void* mem_;
};

// Expose the protected methods for testing helper functions
class MemoryTrackerForHelperTest : public MemoryTracker {
 public:
  tracking_range_list_type::iterator FirstOverlappedRange(uintptr_t addr,
                                                          size_t size) {
    return MemoryTracker::FirstOverlappedRange(addr, size);
  }

  tracking_range_list_type& ranges() { return tracking_ranges_; }
  void insert(uintptr_t addr, size_t size) {
    tracking_ranges_[addr + size] =
        std::unique_ptr<MemoryTracker::tracking_range_type>(
            new MemoryTracker::tracking_range_type(addr, size));
  }
};
}  // namespace

TEST(MemoryTrackerTest, FirstOverlappedRange) {
  MemoryTrackerForHelperTest t;
  auto it = t.FirstOverlappedRange(0x0, 0x0);
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x100, 0x10);
  EXPECT_EQ(it, t.ranges().end());

  t.insert(0x123, 0x456);                   // [0x123 - 0x579)
  it = t.FirstOverlappedRange(0x0, 0x100);  // [0x0, 0x100)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x100, 0x100);  // [0x100, 0x200)
  EXPECT_EQ(it, t.ranges().find(0x579));
  it = t.FirstOverlappedRange(0x500, 0x100);  // [0x500, 0x600)
  EXPECT_EQ(it, t.ranges().find(0x579));
  it = t.FirstOverlappedRange(0x200, 0x100);  // [0x200, 0x300)
  EXPECT_EQ(it, t.ranges().find(0x579));
  it = t.FirstOverlappedRange(0x100, 0x500);  // [0x100, 0x600)
  EXPECT_EQ(it, t.ranges().find(0x579));
  it = t.FirstOverlappedRange(0x600, 0x100);  // [0x600, 0x700)
  EXPECT_EQ(it, t.ranges().end());

  t.ranges().erase(t.ranges().find(0x579));
  it = t.FirstOverlappedRange(0x0, 0x100);  // [0x0, 0x100)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x100, 0x100);  // [0x100, 0x200)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x500, 0x100);  // [0x500, 0x600)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x200, 0x100);  // [0x200, 0x300)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x100, 0x500);  // [0x100, 0x600)
  EXPECT_EQ(it, t.ranges().end());
  it = t.FirstOverlappedRange(0x600, 0x100);  // [0x600, 0x700)
  EXPECT_EQ(it, t.ranges().end());

  t.insert(0x100, 0x100);                   // [0x100, 0x200)
  t.insert(0x200, 0x200);                   // [0x200, 0x400)
  it = t.FirstOverlappedRange(0x0, 0x200);  // [0x0, 0x200)
  EXPECT_EQ(it, t.ranges().find(0x200));
  it = t.FirstOverlappedRange(0x1FF, 0x200);  // [0x1FF, 0x3FF)
  EXPECT_EQ(it, t.ranges().find(0x200));
  it = t.FirstOverlappedRange(0x1FF, 0x300);  // [0x1FF, 0x4FF)
  EXPECT_EQ(it, t.ranges().find(0x200));
  it = t.FirstOverlappedRange(0x200, 0x200);  // [0x200, 0x400)
  EXPECT_EQ(it, t.ranges().find(0x400));
  it = t.FirstOverlappedRange(0x0, 0x400);  // [0x0, 0x400)
  EXPECT_EQ(it, t.ranges().find(0x200));
}

// Allocates one page of memory, adds the memory range to the memory tracker,
// touches the allocated memory, the tracker should record the touched page.
TEST(MemoryTrackerTest, BasicUse) {
  MemoryTracker t;
  const size_t page_size = GetPageSize();
  AlignedMemory m(page_size, page_size);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(m.mem(), page_size));

  memset(m.mem(), 0xFF, page_size);
  t.HandleAndClearDirtyIntersects(nullptr, std::numeric_limits<size_t>::max(),
                                  [&m, &page_size](void* addr, size_t size) {
                                    EXPECT_EQ(m.mem(), addr);
                                    EXPECT_EQ(0xFF, *(uint8_t*)m.mem());
                                    EXPECT_EQ(page_size, size);
                                  });

  // Clean up
  EXPECT_TRUE(t.UntrackRange(m.mem(), page_size));
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Test for HandleAndClearDirtyIntersects.
TEST(MemoryTrackerTest, HandleAndClear) {
  MemoryTracker t;
  const size_t page_size = GetPageSize();
  AlignedMemory m(page_size, page_size);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(m.mem(), page_size));

  memset(m.mem(), 0xAB, page_size);
  // Touches the same page again with updated value.
  memset(m.mem(), 0xCD, page_size);
  size_t num_called = 0;
  EXPECT_TRUE(t.HandleAndClearDirtyIntersects(
      m.mem(), page_size,
      [&m, page_size, &num_called](void* addr, size_t size) {
        EXPECT_EQ(m.mem(), addr);
        EXPECT_EQ(0xCD, *(uint8_t*)m.mem());
        EXPECT_EQ(page_size, size);
        num_called++;
      }));
  // Only one intersect should be found.
  EXPECT_EQ(1u, num_called);

  // Starting at a lower address without no overlap:
  memset(m.mem(), 0x12, page_size);
  t.HandleAndClearDirtyIntersects(
      VoidPointerAdd(m.mem(), (-1) * page_size), page_size,
      [&num_called](void*, size_t) { num_called++; });
  EXPECT_EQ(1u, num_called);

  // Starting at lower address with overlapping with the tracking page:
  memset(m.mem(), 0x34, page_size);
  t.HandleAndClearDirtyIntersects(VoidPointerAdd(m.mem(), (-1) * page_size),
                                  page_size + 1u,
                                  [&num_called, &m](void* addr, size_t) {
                                    EXPECT_EQ(m.mem(), addr);
                                    EXPECT_EQ(0x34, *(uint8_t*)m.mem());
                                    num_called++;
                                  });
  EXPECT_EQ(2u, num_called);

  // Starting at a higher address with overlapping:
  memset(m.mem(), 0x56, page_size);
  t.HandleAndClearDirtyIntersects(VoidPointerAdd(m.mem(), page_size - 1u),
                                  page_size,
                                  [&num_called, &m](void* addr, size_t) {
                                    EXPECT_EQ(m.mem(), addr);
                                    EXPECT_EQ(0x56, *(uint8_t*)m.mem());
                                    num_called++;
                                  });
  EXPECT_EQ(3u, num_called);

  // Starting at a higher address without overlapping:
  memset(m.mem(), 0x78, page_size);
  t.HandleAndClearDirtyIntersects(
      VoidPointerAdd(m.mem(), page_size), page_size,
      [&num_called](void*, size_t) { num_called++; });

  EXPECT_EQ(3u, num_called);

  EXPECT_TRUE(t.UntrackRange(m.mem(), page_size));
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, does not add the memory range to the memory
// tracker, touches the allocated memory, the tracker should not record
// anything.
TEST(MemoryTrackerTest, NoTrackingMemory) {
  MemoryTracker t;
  const size_t page_size = GetPageSize();
  AlignedMemory m(page_size, page_size);
  // Still register the segfault handler, even though it should not be
  // triggered.
  ASSERT_TRUE(t.EnableMemoryTracker());
  memset(m.mem(), 0xFF, page_size);

  size_t num_called = 0;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size, [&num_called](void*, size_t) { num_called++; });
  EXPECT_EQ(0u, num_called);

  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, adds the same memory range to the memory
// tracker and touches the memory range to the tracker in two threads, the
// tracker should have just one record of the touched page.
TEST(MemoryTrackerTest, MultithreadSamePage) {
  MemoryTracker t;
  const size_t page_size = GetPageSize();
  // allocates one pages.
  AlignedMemory m(page_size, page_size);
  ASSERT_TRUE(t.EnableMemoryTracker());

  std::thread t1([&t, &m, page_size]() {
    // Adding tracking range may return false
    t.TrackRange(m.mem(), page_size);
    memset(m.mem(), 0xFF, page_size);
  });
  std::thread t2([&t, &m, page_size]() {
    // Adding tracking range may return false
    t.TrackRange(m.mem(), page_size);
    memset(m.mem(), 0xFF, page_size);
  });
  t1.join();
  t2.join();
  size_t num_called = 0;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size,
      [&num_called, &m, page_size](void* addr, size_t size) {
        EXPECT_EQ(m.mem(), addr);
        EXPECT_EQ(page_size, size);
        num_called++;
      });
  EXPECT_EQ(1u, num_called);

  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates two pages of memory, adds the two pages and touches them in two
// threads respectively. The tracker should have the records of both touched
// pages.
TEST(MemoryTrackerTest, MultithreadDifferentPage) {
  MemoryTracker t;
  const size_t page_size = GetPageSize();
  // allocates two pages.
  AlignedMemory m(page_size, page_size * 2);
  void* first_page = m.mem();
  void* second_page = VoidPointerAdd(m.mem(), page_size);
  ASSERT_TRUE(t.EnableMemoryTracker());

  std::thread t1([&t, first_page, page_size]() {
    // touches the first page.
    EXPECT_TRUE(t.TrackRange(first_page, page_size));
    memset(first_page, 0x12, page_size);
  });
  std::thread t2([&t, second_page, page_size]() {
    // touches the second page.
    EXPECT_TRUE(t.TrackRange(second_page, page_size));
    memset(second_page, 0x34, page_size);
  });
  t1.join();
  t2.join();
  uintptr_t dirty_start = std::numeric_limits<uintptr_t>::max();
  uintptr_t dirty_end = 0;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size * 2u,
      [&dirty_start, &dirty_end](void* addr, size_t size) {
        uintptr_t casted_addr = reinterpret_cast<uintptr_t>(addr);
        if (casted_addr < dirty_start) {
          dirty_start = casted_addr;
        }
        if (casted_addr + size > dirty_end) {
          dirty_end = casted_addr + size;
        }
      });

  EXPECT_EQ(m.mem(), (void*)dirty_start);
  EXPECT_EQ(2u * page_size, dirty_end - dirty_start);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, add part of the page as an un-aligned range to
// the tracker and touches the memory in that range. The tracker should have
// the record of that page.
TEST(MemoryTrackerTest, UnalignedRangeTrackingMemory) {
  const size_t start_offset = 128;
  const size_t range_size = 97;
  const size_t page_size = GetPageSize();

  MemoryTracker t;
  AlignedMemory m(page_size, page_size);

  void* range_start = VoidPointerAdd(m.mem(), start_offset);

  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(range_start, range_size));
  memset(range_start, 0xFF, range_size);
  std::vector<void*> dirty_pages;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size, [&dirty_pages, page_size](void* addr, size_t size) {
        dirty_pages.push_back(addr);
        EXPECT_EQ(page_size, size);
      });
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

namespace {
// A helper class that, when being constructed, sets the signal handler of
// signal |SIG| to just silently ignore the signal. When being destoryed,
// recovers the original segfault handler.
template <int SIG>
class SilentSignal {
 public:
  SilentSignal() : succeeded_(false), old_sigaction_{0} {
    succeeded_ = RegisterSignalHandler(SIG, IgnoreSignal, &old_sigaction_);
  }
  ~SilentSignal() {
    if (succeeded_) {
      sigaction(SIG, &old_sigaction_, nullptr);
    }
  }
  bool succeeded() const { return succeeded_; }

 private:
  static void IgnoreSignal(int, siginfo_t* info, void*) {}
  bool succeeded_;
  struct sigaction old_sigaction_;
};

// To silently ignore a SIGSEGV signal, the permission of the fault page
// will be set to read-write before return.
template <>
void SilentSignal<SIGSEGV>::IgnoreSignal(int, siginfo_t* info, void*) {
  void* page_start = (void*)RoundDownAlignedAddress(
      reinterpret_cast<uintptr_t>(info->si_addr), GetPageSize());
  mprotect(page_start, GetPageSize(), PROT_READ | PROT_WRITE);
}
}  // namespace

TEST(MemoryTrackerTest, RegisterAndUnregister) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;
  ASSERT_TRUE(st.succeeded());
  ASSERT_TRUE(ss.succeeded());
  const size_t page_size = GetPageSize();

  MemoryTracker t;
  AlignedMemory m(page_size, page_size);
  ASSERT_TRUE(t.EnableMemoryTracker());
  // A second call to register segfault handler should return true.
  EXPECT_TRUE(t.EnableMemoryTracker());
  // Register segfault handler with the same tracker in another thread should
  // also return true.
  std::thread another_thread([&t]() { EXPECT_TRUE(t.EnableMemoryTracker()); });
  another_thread.join();

  EXPECT_TRUE(t.TrackRange(m.mem(), page_size));
  EXPECT_TRUE(t.DisableMemoryTracker());
  // Although multiple calls to register segfault handler are made, the handler
  // should only be set once, and by one call to unregister segfault handler,
  // the disposition of segfault signal should be recovered to the previous
  // handler.
  memset(m.mem(), 0xFF, page_size);
  size_t num_called = 0;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size, [&num_called](void*, size_t) { num_called++; });
  EXPECT_EQ(0u, num_called);
}

// Allocates one page of memory, add a memory range to the tracker, add another
// memory range which has overlapping region with the first one. The second
// range should not be accepted by the tracker.
TEST(MemoryTrackerTest, OverlappedTrackingRange) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;
  ASSERT_TRUE(st.succeeded());
  ASSERT_TRUE(ss.succeeded());

  const size_t range_size = 2048;
  const size_t second_range_offset = 1024;
  const size_t page_size = GetPageSize();
  MemoryTracker t;
  AlignedMemory m(page_size, page_size);

  void* second_range_start = VoidPointerAdd(m.mem(), second_range_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(m.mem(), range_size));
  EXPECT_FALSE(t.TrackRange(second_range_start, range_size));

  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates a page of memory, add a not-aligned range of space to the tracker,
// touches an address higher than the tracking range, the tracker should record
// the touched page.
TEST(MemoryTrackerTest, UnalignedRangeTrackingHigherAddress) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;
  ASSERT_TRUE(st.succeeded());
  ASSERT_TRUE(ss.succeeded());

  const size_t start_offset = 128;
  const size_t range_size = 97;
  const size_t page_size = GetPageSize();

  MemoryTracker t;
  AlignedMemory m(page_size, page_size);

  void* range_start = VoidPointerAdd(m.mem(), start_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(range_start, range_size));

  void* touch_start = VoidPointerAdd(range_start, range_size);
  const size_t touch_size = range_size;
  memset(touch_start, 0xFF, touch_size);

  std::vector<void*> dirty_pages;
  t.HandleAndClearDirtyIntersects(
      range_start, range_size,
      [&dirty_pages, page_size](void* addr, size_t size) {
        dirty_pages.push_back(addr);
        EXPECT_THAT(page_size, size);
      });

  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates a page of memory, add a not-aligned range of space to the tracker,
// touches an address lower than the tracking range, the tracker should record
// the touched page.
TEST(MemoryTrackerTest, UnalignedRangeNotTrackingLowerAddress) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;

  const size_t start_offset = 128;
  const size_t range_size = 97;
  const size_t page_size = GetPageSize();

  MemoryTracker t;
  AlignedMemory m(page_size, page_size);

  void* range_start = VoidPointerAdd(m.mem(), start_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(range_start, range_size));

  void* touch_start = reinterpret_cast<void*>(
      reinterpret_cast<uintptr_t>(range_start) - range_size);
  const size_t touch_size = range_size;
  memset(touch_start, 0xFF, touch_size);

  std::vector<void*> dirty_pages;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size, [&dirty_pages, page_size](void* addr, size_t size) {
        dirty_pages.push_back(addr);
        EXPECT_THAT(page_size, size);
      });

  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Two not overlapping ranges in the same page. Removing just one of them
// should not affect tracking of the other one.
TEST(MemoryTrackerTest, RemoveOneRangeShouldNotAffectOthersInSamePage) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;

  const size_t first_offset = 128;
  const size_t first_size = 97;
  const size_t page_size = GetPageSize();

  const size_t second_offset = 1024;
  const size_t second_size = 97;

  MemoryTracker t;
  AlignedMemory m(page_size, page_size);

  void* first_start = VoidPointerAdd(m.mem(), first_offset);
  void* second_start = VoidPointerAdd(m.mem(), second_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.TrackRange(first_start, first_size));
  EXPECT_TRUE(t.TrackRange(second_start, second_size));

  EXPECT_TRUE(t.UntrackRange(first_start, first_size));

  memset(second_start, 0xFF, second_size);

  std::vector<void*> dirty_pages;
  t.HandleAndClearDirtyIntersects(
      m.mem(), page_size, [&dirty_pages, page_size](void* addr, size_t size) {
        dirty_pages.push_back(addr);
        EXPECT_THAT(page_size, size);
      });

  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates a lot of pages and tracking/touches them in multiple threads.
TEST(MemoryTrackerTest, ManyPagesMultithread) {
  const size_t num_threads = 128;
  const size_t num_pages_per_thread = 16;
  const size_t num_pages = num_pages_per_thread * num_threads;
  const size_t page_size = GetPageSize();

  MemoryTracker t;
  ASSERT_TRUE(t.EnableMemoryTracker());
  AlignedMemory m(page_size, num_pages * page_size);
  void* mem_start_addr = m.mem();

  std::vector<std::thread> threads;
  threads.reserve(num_threads);
  // Every thread is responsible for 4 continuous pages.
  for (uint32_t ti = 0; ti < num_threads; ti++) {
    threads.emplace_back(std::thread(
        [mem_start_addr, num_pages_per_thread, ti, page_size, &t]() {
          size_t thread_range_size = num_pages_per_thread * page_size;
          void* thread_range_start =
              VoidPointerAdd(mem_start_addr, ti * thread_range_size);
          EXPECT_TRUE(t.TrackRange(thread_range_start, thread_range_size));
          memset(thread_range_start, 0xFF, thread_range_size);
        }));
  }
  std::for_each(threads.begin(), threads.end(),
                [](std::thread& t) { t.join(); });

  // All the pages should have been recorded.
  std::vector<void*> dirty_pages;
  t.HandleAndClearDirtyIntersects(
      m.mem(), num_pages * page_size,
      [&dirty_pages, page_size](void* addr, size_t size) {
        uintptr_t casted_addr = reinterpret_cast<uintptr_t>(addr);
        for (size_t i = 0; i < size / GetPageSize(); i++) {
          dirty_pages.push_back(
              reinterpret_cast<void*>(casted_addr + i * GetPageSize()));
        }
        EXPECT_EQ(0u, size % GetPageSize());
      });
  EXPECT_EQ(num_pages, dirty_pages.size());
  for (uint32_t i = 0; i < num_pages; i++) {
    void* page = VoidPointerAdd(mem_start_addr, i * page_size);
    EXPECT_THAT(dirty_pages, Contains(page));
  }

  ASSERT_TRUE(t.DisableMemoryTracker());
}
}  // namespace test
}  // namespace track_memory
}  // namespace gapii
#endif  // COHERENT_TRACKING_ENABLED
