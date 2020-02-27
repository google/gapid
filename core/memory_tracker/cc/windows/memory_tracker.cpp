/*
 * Copyright (C) 2018 Google Inc.
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

#include "core/memory_tracker/cc/memory_tracker.h"

#if COHERENT_TRACKING_ENABLED
#include <Windows.h>
#include <atomic>
#include <functional>

namespace gapii {
namespace track_memory {

bool set_protection(void* p, size_t size, PageProtections prot) {
  DWORD oldProt;
  DWORD protections =
      (prot == PageProtections::kRead)
          ? PAGE_READONLY
          : (prot == PageProtections::kWrite)
                ? PAGE_READWRITE
                : (prot == PageProtections::kReadWrite) ? PAGE_READWRITE : 0;
  return VirtualProtect(p, size, protections, &oldProt);
}

// A static wrapper of HandleSegfault() as sigaction() asks for a static
// function.
long int __stdcall WindowsMemoryTracker::VectoredExceptionHandler(void* _info) {
  struct _EXCEPTION_POINTERS* info =
      reinterpret_cast<struct _EXCEPTION_POINTERS*>(_info);
  if (info->ExceptionRecord->ExceptionCode == EXCEPTION_ACCESS_VIOLATION &&
      unique_tracker) {
    if (unique_tracker->handle_segfault_(reinterpret_cast<void*>(
            info->ExceptionRecord->ExceptionInformation[1]))) {
      return EXCEPTION_CONTINUE_EXECUTION;
    }
  }
  return EXCEPTION_CONTINUE_SEARCH;
}

// EnableMemoryTrackerImpl calls sigaction() to register the new segfault
// handler to the thread (and affects all the child threads), stores the
// original segfault handler. This method sets the static pointer:
// unique_tracker to |this| pointer. The signal handler will not be set again
// if the signal handler has already been set by the same memory tracker
// instance before, all following calls to this function will just return
// true.
bool WindowsMemoryTracker::EnableMemoryTrackerImpl() {
  if (vectored_exception_handler_) {
    return true;
  }
  const uint32_t kCallFirst = 1;
  unique_tracker = this;
  PVECTORED_EXCEPTION_HANDLER handler =
      reinterpret_cast<PVECTORED_EXCEPTION_HANDLER>(&VectoredExceptionHandler);
  vectored_exception_handler_ =
      AddVectoredExceptionHandler(kCallFirst, handler);
  return vectored_exception_handler_ != nullptr;
}

// DisableMemoryTrackerImpl recovers the original segfault signal
// handler. Returns true if the handler is recovered successfully,
// othwerwise returns false.
bool WindowsMemoryTracker::DisableMemoryTrackerImpl() {
  if (vectored_exception_handler_) {
    ULONG ret = RemoveVectoredExceptionHandler(vectored_exception_handler_);
    vectored_exception_handler_ = nullptr;
    return ret != 0;
  }
  return true;
}

uint32_t GetPageSize() {
  static std::atomic<uint32_t> pageSize;
  int x = pageSize.load();
  if (x != 0) {
    return x;
  }
  SYSTEM_INFO si;
  GetSystemInfo(&si);
  pageSize.store(si.dwPageSize);
  return si.dwPageSize;
}

}  // namespace track_memory
}  // namespace gapii
#endif  // COHERENT_MEMORY_TRACKING_ENABLED
