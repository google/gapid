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

// TODO(qining): Add Windows support
#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>

#include <atomic>
#include <condition_variable>
#include <list>
#include <map>
#include <thread>

namespace gapii {
namespace track_memory {
namespace test {

using ::testing::Contains;

TEST(GetAlignedAddressTest, 4KAligned) {
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x0, 0x1000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x50, 0x1000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x100, 0x1000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x800, 0x1000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0xFFF, 0x1000));
  EXPECT_EQ((void*)0x1000, GetAlignedAddress((void*)0x1000, 0x1000));
  EXPECT_EQ((void*)0x1000, GetAlignedAddress((void*)0x1FFF, 0x1000));
  EXPECT_EQ((void*)0x2611000, GetAlignedAddress((void*)0x2611001, 0x1000));
  EXPECT_EQ((void*)0x2611000, GetAlignedAddress((void*)0x2611FFF, 0x1000));
  EXPECT_EQ((void*)0xFFFFF000, GetAlignedAddress((void*)0xFFFFFFFF, 0x1000));
}

TEST(GetAlignedAddressTest, 64KAligned) {
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x0, 0x10000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x50, 0x10000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x100, 0x10000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0x1800, 0x10000));
  EXPECT_EQ((void*)0x0, GetAlignedAddress((void*)0xFFFF, 0x10000));
  EXPECT_EQ((void*)0x10000, GetAlignedAddress((void*)0x10000, 0x10000));
  EXPECT_EQ((void*)0x10000, GetAlignedAddress((void*)0x1FFFF, 0x10000));
  EXPECT_EQ((void*)0x2610000, GetAlignedAddress((void*)0x2611001, 0x10000));
  EXPECT_EQ((void*)0xFFFF0000, GetAlignedAddress((void*)0xFFFFFFFF, 0x10000));
}

// If the alignment value is invalid, just make sure we don't crash, the
// results are undefined in such cases.
TEST(GetAlignedAddressTest, Invalid) {
  // Alignment is zero
  GetAlignedAddress((void*)0x12345678, 0x0);
  // Alignment is not a power of two
  GetAlignedAddress((void*)0x12345678, 0x7);
  GetAlignedAddress((void*)0x12345678, 0xFFFF);
}

TEST(GetAlignedSizeTest, 4KAligned) {
  EXPECT_EQ(0x0, GetAlignedSize((void*)0x0, 0x0, 0x1000));
  EXPECT_EQ(0x1000, GetAlignedSize((void*)0x0, 0x0FFF, 0x1000));
  EXPECT_EQ(0x1000, GetAlignedSize((void*)0x0, 0x1000, 0x1000));
  EXPECT_EQ(0x2000, GetAlignedSize((void*)0x0, 0x1FFF, 0x1000));
  EXPECT_EQ(0x2000, GetAlignedSize((void*)0x0, 0x2000, 0x1000));
  EXPECT_EQ(0x0, GetAlignedSize((void*)0x0FFF, 0x0, 0x1000));
  EXPECT_EQ(0x1000, GetAlignedSize((void*)0x0FFF, 0x1, 0x1000));
  EXPECT_EQ(0x2000, GetAlignedSize((void*)0x0FFF, 0x1000, 0x1000));
  EXPECT_EQ(0x1000, GetAlignedSize((void*)0xFFFFF000, 0x1, 0x1000));
  EXPECT_EQ(0x0, GetAlignedSize((void*)0xFFFFFFFF, 0x0, 0x1000));
}

TEST(GetAlignedSizeTest, Invalid) {
  // Overflow
  EXPECT_EQ(0x0, GetAlignedSize((void*)(~(uintptr_t)0x0), 0x1, 0x1000));
  EXPECT_EQ(0x0, GetAlignedSize((void*)0x1, (~(size_t)0x0), 0x1000));
  // Alignment is zero
  EXPECT_EQ(0x0, GetAlignedSize((void*)0x12345678, 0x4321, 0x0));
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
    dummy_lock_.store(false, std::memory_order_seq_cst);
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

  // DoTask acquires the dummy lock and runs the given function.
  void DoTask(std::function<void(void)> task) {
    DummyLock();
    normal_thread_run_before_signal_handler_
        .unlock();  // This is to make sure the signal will only be send after
                    // the normal threads
    task();
    DummyUnlock();
  }

  // RegisterHandlerAndTriggerSignal does the following things:
  //  1) Registers a signal handler to SIGUSR1 which acquires the dummy lock.
  //  2) Creates a child thread which will acquire the dummy lock.
  //  3) Waits for the child thread to run then sends a signal interrupt to the
  //  child thread.
  //  4) Waits until the child thread finishes and clean up.
  void RegisterHandlerAndTriggerSignal(void* (*child_thread_func)(void*)) {
    auto handler_func = [](int, siginfo_t*, void*) {
      // DummyLock is to simulate the case that the coherent memory tracker's
      // signal handler needs to access shared data.
      unique_test_->DummyLock();
      ;
      unique_test_->DummyUnlock();
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
  // Acquires the dummy lock. If the dummy lock has already been locked, i.e.
  // its value is true, sets the deadlocked_ flag.
  void DummyLock() {
    if (dummy_lock_.load(std::memory_order_seq_cst)) {
      deadlocked_.exchange(true);
    }
    dummy_lock_.exchange(true);
  }
  // Releases the dummy lock by setting its value to false.
  void DummyUnlock() { dummy_lock_.exchange(false); }

  std::mutex normal_thread_run_before_signal_handler_;  // A mutex to
                                                        // guarantee the normal
                                                        // thread is initiated
                                                        // before the signal
                                                        // handler is called.
  std::mutex mutex_thread_init_order_;  // A mutex to guarantee one thread is
                                        // initiated before the other one
  std::mutex m_;  // A mutex for tuning execution order in tests
  std::atomic<bool>
      dummy_lock_;  // An atomic bool to simulate the lock/unlock state
  std::atomic<bool> deadlocked_;  // A flag to indicate deadlocked or not state

  static TestFixture* unique_test_;  // A static pointer of this test fixture,
                                     // required as signal handler must be a
                                     // static function, and we will need a
                                     // static pointer to access the member
                                     // functions.
};
TestFixture* TestFixture::unique_test_ = nullptr;
}

// A helper function to ease void* pointer + offset calculation.
void* VoidPointerAdd(void* addr, ssize_t offset) {
  return reinterpret_cast<void*>(reinterpret_cast<uintptr_t>(addr) + offset);
}

using SpinLockTest = TestFixture;

// Thread A sleeps first, then increases the counter, while Thread B will
// increase the counter before Thread A wakes up.
TEST_F(SpinLockTest, WithoutSpinLockGuard) {
  Init();
  uint32_t counter = 0u;
  RunInTwoThreads(
      // Thread A is initiated first and runs first.
      [this, &counter]() {
        m_.unlock();
        usleep(5000);
        counter++;
        EXPECT_EQ(2u, counter);
      },
      // Thread B is initiated secondly and runs after thread A runs.
      [this, &counter]() {
        m_.lock();
        counter++;
        EXPECT_EQ(1u, counter);
      });
  EXPECT_EQ(2u, counter);
}

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

namespace {
// A sub-class of DirtyPageTable that exposes the internal storage for test.
class DirtyPageTableForTest : public DirtyPageTable {
 public:
  const std::list<uintptr_t>& pages() const { return pages_; }
  const std::list<uintptr_t>::iterator& current() const { return current_; }
  size_t stored() const { return stored_; }
};
}

TEST(DirtyPageTableTest, Init) {
  DirtyPageTableForTest t;
  EXPECT_EQ(0u, t.stored());
  EXPECT_EQ(1u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());
}

TEST(DirtyPageTableTest, Reserve) {
  DirtyPageTableForTest t;
  t.Reserve(10u);
  EXPECT_EQ(11u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());

  t.Reserve(100u);
  EXPECT_EQ(111u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());

  t.Reserve(1000u);
  EXPECT_EQ(1111u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());
}

TEST(DirtyPageTableTest, Record) {
  DirtyPageTableForTest t;
  // Record should fail when there is no space available.
  EXPECT_FALSE(t.Record((void*)0x100));
  EXPECT_EQ(0u, t.stored());
  EXPECT_EQ(1u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());

  t.Reserve(1u);
  EXPECT_EQ(0u, t.stored());
  EXPECT_EQ(2u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());

  t.Reserve(3u);
  EXPECT_EQ(0u, t.stored());
  EXPECT_EQ(5u, t.pages().size());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_NE(t.pages().end(), t.current());

  EXPECT_TRUE(t.Record((void*)0x200));
  EXPECT_EQ(1u, t.stored());
  EXPECT_EQ(5u, t.pages().size());  // Record should not allocate new space.
  EXPECT_EQ(std::next(t.pages().begin(), 1), t.current());

  EXPECT_TRUE(t.Record((void*)0x300));
  EXPECT_TRUE(t.Record((void*)0x400));
  EXPECT_EQ(3u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 3), t.current());
  EXPECT_EQ(5u, t.pages().size());

  EXPECT_TRUE(t.Record((void*)0x500));
  EXPECT_EQ(4u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 4), t.current());
  EXPECT_EQ(5u, t.pages().size());

  EXPECT_FALSE(t.Record((void*)0x600));
  EXPECT_EQ(4u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 4), t.current());
  EXPECT_EQ(5u, t.pages().size());
}

TEST(DirtyPageTableTest, Has) {
  DirtyPageTableForTest t;
  EXPECT_FALSE(t.Has((void*)0x100));
  EXPECT_FALSE(t.Has((void*)0x200));
  EXPECT_FALSE(t.Has((void*)0x300));

  t.Reserve(3u);
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_TRUE(t.Record((void*)0x200));
  EXPECT_TRUE(t.Has((void*)0x100));
  EXPECT_TRUE(t.Has((void*)0x200));
  EXPECT_FALSE(t.Has((void*)0x300));

  t.DumpAndClearAll();
  EXPECT_FALSE(t.Has((void*)0x100));
  EXPECT_FALSE(t.Has((void*)0x200));
  EXPECT_FALSE(t.Has((void*)0x300));
}

TEST(DirtyPageTableTest, DumpAndClearAll) {
  DirtyPageTableForTest t;
  auto empty_dump = t.DumpAndClearAll();
  EXPECT_EQ(std::vector<void*>(), empty_dump);

  std::vector<void*> addresses = {(void*)0x100, (void*)0x200, (void*)0x300,
                                  (void*)0x400, (void*)0x500};
  t.Reserve(5u);
  for (auto a : addresses) {
    EXPECT_TRUE(t.Record(a));
  }
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(5u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 5), t.current());

  auto dump = t.DumpAndClearAll();
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(0u, t.stored());
  EXPECT_EQ(t.pages().begin(), t.current());
  EXPECT_EQ(addresses, dump);
}

TEST(DirtyPageTableTest, DumpAndClearInRange) {
  DirtyPageTableForTest t;
  auto empty_dump = t.DumpAndClearInRange((void*)0x0, 0xFFFFFFFF);
  EXPECT_EQ(std::vector<void*>(), empty_dump);

  std::vector<void*> addresses = {(void*)0x100, (void*)0x200, (void*)0x300,
                                  (void*)0x400, (void*)0x500};
  t.Reserve(5u);
  for (auto a : addresses) {
    EXPECT_TRUE(t.Record(a));
  }
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(5u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 5), t.current());

  // Dump the dirty pages in range: [0x234, 0x447), expect 0x300 and 0x400 are
  // dumped and removed from the dirty page table.
  auto dump_start_0x234_size_0x213 = t.DumpAndClearInRange((void*)0x234, 0x213);
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(3u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 3), t.current());
  EXPECT_THAT(dump_start_0x234_size_0x213, Contains((void*)0x300));
  EXPECT_THAT(dump_start_0x234_size_0x213, Contains((void*)0x400));

  // Removed pages should not exist.
  empty_dump = t.DumpAndClearInRange((void*)0x234, 0x213);
  EXPECT_EQ(std::vector<void*>(), empty_dump);

  // Recording new pages should work
  EXPECT_TRUE(t.Record((void*)0x600));
  EXPECT_TRUE(t.Record((void*)0x700));
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(5u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 5), t.current());
  // Dump the dirty pages in range [0x100, 0x600), expect 0x100, 0x200, 0x500
  // are dumped and removed from the dirty page table.
  auto dump_start_0x100_size_0x500 = t.DumpAndClearInRange((void*)0x100, 0x500);
  EXPECT_EQ(6u, t.pages().size());
  EXPECT_EQ(2u, t.stored());
  EXPECT_EQ(std::next(t.pages().begin(), 2), t.current());
  EXPECT_THAT(dump_start_0x100_size_0x500, Contains((void*)0x100));
  EXPECT_THAT(dump_start_0x100_size_0x500, Contains((void*)0x200));
  EXPECT_THAT(dump_start_0x100_size_0x500, Contains((void*)0x500));
}

TEST(DirtyPageTableTest, RecollectIfPossible) {
  DirtyPageTableForTest t;
  // recollect all the spaces.
  t.Reserve(5u);
  EXPECT_EQ(6u, t.pages().size());
  t.RecollectIfPossible(5u);
  EXPECT_EQ(1u, t.pages().size());

  // cannot recollect any spaces.
  t.Reserve(5u);
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_EQ(std::next(t.pages().begin(), 5), t.current());
  EXPECT_EQ(5u, t.stored());
  EXPECT_EQ(6u, t.pages().size());
  t.RecollectIfPossible(5u);
  EXPECT_EQ(std::next(t.pages().begin(), 5), t.current());
  EXPECT_EQ(5u, t.stored());
  EXPECT_EQ(6u, t.pages().size());

  // recollect part of the spaces.
  t.Reserve(5u);
  EXPECT_TRUE(t.Record((void*)0x100));
  EXPECT_EQ(std::next(t.pages().begin(), 6), t.current());
  EXPECT_EQ(6u, t.stored());
  EXPECT_EQ(11u, t.pages().size());
  t.RecollectIfPossible(5u);
  EXPECT_EQ(std::next(t.pages().begin(), 6), t.current());
  EXPECT_EQ(6u, t.stored());
  EXPECT_EQ(7u, t.pages().size());
}

// Test helper function: MemoryTracker::IsInRange()
TEST(IsInRangesTest, IsInRangesNotPageAligned) {
  std::map<uintptr_t, size_t> test_ranges;
  // Empty ranges
  EXPECT_FALSE(IsInRanges(0x1000, test_ranges));

  // At starting address
  test_ranges[0x1000] = 0x100;
  EXPECT_TRUE(IsInRanges(0x1000, test_ranges));
  // Higher than staring address
  EXPECT_TRUE(IsInRanges(0x1001, test_ranges));
  EXPECT_FALSE(IsInRanges(0x1100, test_ranges));
  // Lower than staring address
  EXPECT_FALSE(IsInRanges(0x0FFF, test_ranges));

  // Multiple ranges
  test_ranges[0x208] = 0x10;
  EXPECT_FALSE(IsInRanges(0x207, test_ranges));
  EXPECT_TRUE(IsInRanges(0x20A, test_ranges));
  EXPECT_TRUE(IsInRanges(0x1000, test_ranges));
  EXPECT_TRUE(IsInRanges(0x1001, test_ranges));
  EXPECT_FALSE(IsInRanges(0x1100, test_ranges));
  EXPECT_FALSE(IsInRanges(0xFFF, test_ranges));
}

TEST(IsInRangesTest, IsInRangesPageAligned) {
  std::map<uintptr_t, size_t> test_ranges;
  // Empty ranges
  EXPECT_FALSE(IsInRanges(0x1000, test_ranges, true));

  // At starting address
  test_ranges[0x1000] = 0x100;
  EXPECT_TRUE(IsInRanges(0x1000, test_ranges, true));
  // Higher than staring address
  EXPECT_TRUE(IsInRanges(0x1001, test_ranges, true));
  EXPECT_TRUE(IsInRanges(0x1100, test_ranges, true));
  EXPECT_FALSE(IsInRanges(0x2100, test_ranges, true));
  // Lower than staring address
  EXPECT_FALSE(IsInRanges(0x0FFF, test_ranges, true));

  // Multiple ranges
  test_ranges[0x208] = 0x10;
  EXPECT_TRUE(IsInRanges(0x207, test_ranges, true));
  EXPECT_TRUE(IsInRanges(0x20A, test_ranges, true));
  EXPECT_TRUE(IsInRanges(0x1000, test_ranges, true));
  EXPECT_TRUE(IsInRanges(0x1001, test_ranges, true));
  EXPECT_FALSE(IsInRanges(0x2100, test_ranges, true));
  EXPECT_TRUE(IsInRanges(0xFFF, test_ranges, true));
  EXPECT_FALSE(IsInRanges(0x3FFF, test_ranges, true));
}

namespace {
class AlignedMemory {
 public:
  AlignedMemory(size_t alignment, size_t size) : mem_(nullptr) {
    posix_memalign(&mem_, alignment, size);
  }
  ~AlignedMemory() { free(mem_); }
  void* mem() const { return mem_; }

 private:
  void* mem_;
};
}

// Allocates one page of memory, adds the memory range to the memory tracker,
// touches the allocated memory, the tracker should record the touched page.
TEST(MemoryTrackerTest, Basic) {
  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(m.mem(), t.page_size()));
  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  EXPECT_EQ(0xFF, *(uint8_t*)m.mem());
  // Clean up
  EXPECT_TRUE(t.ClearTrackingRanges());
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Test for GetDirtyPagesInRange and ResetPagesToTrack interfaces.
TEST(MemoryTrackerTest, GetDirtyPagesInRangeAndResetPagesToTrack) {
  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(m.mem(), t.page_size()));

  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> dirty_pages =
      t.GetDirtyPagesInRange(m.mem(), t.page_size());
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);

  // Touches the same page again, this should not be recorded.
  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> empty_dump =
      t.GetDirtyPagesInRange(m.mem(), t.page_size());
  EXPECT_EQ(0u, empty_dump.size());

  EXPECT_TRUE(t.ResetPagesToTrack(dirty_pages));
  memset(m.mem(), 0xFF, t.page_size());
  dirty_pages = t.GetDirtyPagesInRange(m.mem(), t.page_size());
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);

  EXPECT_TRUE(t.ClearTrackingRanges());
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Test for GetAndResetInRange interface.
TEST(MemoryTrackerTest, GetAndResetInRange) {
  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(m.mem(), t.page_size()));

  // Test getting dirty pages in a range
  // Exact same range as the tracking page:
  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> dirty_pages =
      t.GetAndResetDirtyPagesInRange(m.mem(), t.page_size());
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  dirty_pages = t.GetAndResetDirtyPagesInRange(m.mem(), t.page_size());
  EXPECT_EQ(0u, dirty_pages.size());
  // Starting at a lower address without no overlap:
  memset(m.mem(), 0xFF, t.page_size());
  dirty_pages = t.GetAndResetDirtyPagesInRange(
      VoidPointerAdd(m.mem(), (-1) * t.page_size()), t.page_size());
  EXPECT_EQ(0u, dirty_pages.size());
  // Starting at lower address with overlapping with the tracking page:
  memset(m.mem(), 0xFF, t.page_size());
  dirty_pages = t.GetAndResetDirtyPagesInRange(
      VoidPointerAdd(m.mem(), (-1) * t.page_size()), t.page_size() + 1u);
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  // Starting at a higher address with overlapping:
  memset(m.mem(), 0xFF, t.page_size());
  dirty_pages = t.GetAndResetDirtyPagesInRange(
      VoidPointerAdd(m.mem(), t.page_size() - 1u), t.page_size());
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  // Starting at a higher address without overlapping:
  memset(m.mem(), 0xFF, t.page_size());
  dirty_pages = t.GetAndResetDirtyPagesInRange(
      VoidPointerAdd(m.mem(), t.page_size()), t.page_size());
  EXPECT_EQ(0u, dirty_pages.size());

  EXPECT_TRUE(t.ClearTrackingRanges());
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, does not add the memory range to the memory
// tracker, touches the allocated memory, the tracker should not record
// anything.
TEST(MemoryTrackerTest, NoTrackingMemory) {
  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());
  // Still register the segfault handler, even though it should not be
  // triggered.
  ASSERT_TRUE(t.EnableMemoryTracker());

  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());

  EXPECT_EQ(0u, dirty_pages.size());
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, adds the same memory range to the memory
// tracker and touches the memory range to the tracker in two threads, the
// tracker should have just one record of the touched page.
TEST(MemoryTrackerTest, MultithreadSamePage) {
  MemoryTracker t;
  // allocates one pages.
  AlignedMemory m(t.page_size(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());

  std::thread t1([&t, &m]() {
    // Adding tracking range may return false
    t.AddTrackingRange(m.mem(), t.page_size());
    memset(m.mem(), 0xFF, t.page_size());
  });
  std::thread t2([&t, &m]() {
    // Adding tracking range may return false
    t.AddTrackingRange(m.mem(), t.page_size());
    memset(m.mem(), 0xFF, t.page_size());
  });
  t1.join();
  t2.join();
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());
  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates two pages of memory, adds the two pages and touches them in two
// threads respectively. The tracker should have the records of both touched
// pages.
TEST(MemoryTrackerTest, MultithreadDifferentPage) {
  MemoryTracker t;
  // allocates two pages.
  AlignedMemory m(t.page_size(), t.page_size() * 2);
  void* first_page = m.mem();
  void* second_page = VoidPointerAdd(m.mem(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());

  std::thread t1([&t, first_page]() {
    // touches the first page.
    EXPECT_TRUE(t.AddTrackingRange(first_page, t.page_size()));
    memset(first_page, 0xFF, t.page_size());
  });
  std::thread t2([&t, second_page]() {
    // touches the second page.
    EXPECT_TRUE(t.AddTrackingRange(second_page, t.page_size()));
    memset(second_page, 0xFF, t.page_size());
  });
  t1.join();
  t2.join();
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());
  EXPECT_EQ(2u, dirty_pages.size());
  EXPECT_THAT(dirty_pages, Contains(first_page));
  EXPECT_THAT(dirty_pages, Contains(second_page));
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates one page of memory, add part of the page as an un-aligned range to
// the tracker and touches the memory in that range. The tracker should have
// the record of that page.
TEST(MemoryTrackerTest, UnalignedRangeTrackingMemory) {
  const size_t start_offset = 128;
  const size_t range_size = 97;

  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());

  void* range_start = VoidPointerAdd(m.mem(), start_offset);

  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(range_start, range_size));
  memset(range_start, 0xFF, range_size);
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());

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
    succeeded_ =
        RegisterSignalHandler(SIG, IgnoreSignal, &old_sigaction_);
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
template<>
void SilentSignal<SIGSEGV>::IgnoreSignal(int, siginfo_t* info, void*) {
  void* page_start = GetAlignedAddress(info->si_addr, getpagesize());
  mprotect(page_start, getpagesize(), PROT_READ | PROT_WRITE);
}
}

TEST(MemoryTrackerTest, RegisterAndUnregister) {
  SilentSignal<SIGSEGV> ss;
  SilentSignal<SIGTRAP> st;
  ASSERT_TRUE(st.succeeded());
  ASSERT_TRUE(ss.succeeded());

  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());
  ASSERT_TRUE(t.EnableMemoryTracker());
  // A second call to register segfault handler should return true.
  EXPECT_TRUE(t.EnableMemoryTracker());
  // Register segfault handler with the same tracker in another thread should
  // also return true.
  std::thread another_thread(
      [&t]() { EXPECT_TRUE(t.EnableMemoryTracker()); });
  another_thread.join();

  EXPECT_TRUE(t.AddTrackingRange(m.mem(), t.page_size()));
  EXPECT_TRUE(t.DisableMemoryTracker());
  // Although multiple calls to register segfault handler are made, the handler
  // should only be set once, and by one call to unregister segfault handler,
  // the disposition of segfault signal should be recovered to the previous
  // handler.
  memset(m.mem(), 0xFF, t.page_size());
  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_EQ(0u, dirty_pages.size());
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
  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());

  void* second_range_start = VoidPointerAdd(m.mem(), second_range_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(m.mem(), range_size));
  EXPECT_FALSE(t.AddTrackingRange(second_range_start, range_size));

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

  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());

  void* range_start = VoidPointerAdd(m.mem(), start_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(range_start, range_size));

  void* touch_start = VoidPointerAdd(range_start, range_size);
  const size_t touch_size = range_size;
  memset(touch_start, 0xFF, touch_size);

  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());

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

  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());

  void* range_start = VoidPointerAdd(m.mem(), start_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(range_start, range_size));

  void* touch_start = reinterpret_cast<void*>(
      reinterpret_cast<uintptr_t>(range_start) - range_size);
  const size_t touch_size = range_size;
  memset(touch_start, 0xFF, touch_size);

  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());

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

  const size_t second_offset = 1024;
  const size_t second_size = 97;

  MemoryTracker t;
  AlignedMemory m(t.page_size(), t.page_size());

  void* first_start = VoidPointerAdd(m.mem(), first_offset);
  void* second_start = VoidPointerAdd(m.mem(), second_offset);
  ASSERT_TRUE(t.EnableMemoryTracker());
  EXPECT_TRUE(t.AddTrackingRange(first_start, first_size));
  EXPECT_TRUE(t.AddTrackingRange(second_start, second_size));

  EXPECT_TRUE(t.RemoveTrackingRange(first_start, first_size));

  memset(second_start, 0xFF, second_size);

  std::vector<void*> dirty_pages = t.GetAndResetAllDirtyPages();
  EXPECT_TRUE(t.ClearTrackingRanges());

  EXPECT_EQ(1u, dirty_pages.size());
  EXPECT_EQ(m.mem(), dirty_pages[0]);
  ASSERT_TRUE(t.DisableMemoryTracker());
}

// Allocates a lot of pages and tracking/touches them in multiple threads.
TEST(MemoryTrackerTest, ManyPagesMultithread) {
  const size_t num_threads = 128;
  const size_t num_pages_per_thread = 4;
  const size_t num_pages = num_pages_per_thread * num_threads;
  const size_t page_size = getpagesize();

  MemoryTracker t;
  ASSERT_TRUE(t.EnableMemoryTracker());
  AlignedMemory m(page_size, num_pages * page_size);
  void* mem_start_addr = m.mem();

  std::vector<std::thread> threads;
  threads.reserve(num_threads);
  // Every thread is responsible for 4 continuous pages.
  for (uint32_t ti = 0; ti < num_threads; ti++) {
    threads.emplace_back(std::thread([mem_start_addr, num_pages_per_thread, ti,
                                      page_size, &t]() {
      size_t thread_range_size = num_pages_per_thread * page_size;
      void* thread_range_start =
          VoidPointerAdd(mem_start_addr, ti * thread_range_size);
      EXPECT_TRUE(t.AddTrackingRange(thread_range_start, thread_range_size));
      memset(thread_range_start, 0xFF, thread_range_size);
    }));
  }
  std::for_each(threads.begin(), threads.end(),
                [](std::thread& t) { t.join(); });

  // All the pages should have been recorded.
  auto dirty_pages = t.GetAndResetAllDirtyPages();
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
