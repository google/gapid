/*
 * Copyright (C) 2022 Google Inc.
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

namespace gapid2 {
memory_tracker*& static_tracker() {
  static memory_tracker* _tracker;
  return _tracker;
}

LONG memory_tracker::handler(_EXCEPTION_POINTERS* ExceptionInfo) {
  if (ExceptionInfo->ExceptionRecord->ExceptionCode !=
      EXCEPTION_ACCESS_VIOLATION) {
    return EXCEPTION_CONTINUE_SEARCH;
  }
  if (ExceptionInfo->ExceptionRecord->NumberParameters < 2) {
    return EXCEPTION_CONTINUE_SEARCH;
  }
  bool read = ExceptionInfo->ExceptionRecord->ExceptionInformation[0] == 0;
  void* pptr = reinterpret_cast<void*>(
      ExceptionInfo->ExceptionRecord->ExceptionInformation[1]);
  if (!pptr) {
    return EXCEPTION_CONTINUE_SEARCH;
  }
  if (static_tracker()->handle_exception(reinterpret_cast<char*>(pptr), read)) {
    return EXCEPTION_CONTINUE_EXECUTION;
  }
  return EXCEPTION_CONTINUE_SEARCH;
}

}  // namespace gapid2
