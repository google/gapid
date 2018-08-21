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

#ifndef CORE_TIMER_H
#define CORE_TIMER_H

#include <cstdint>

namespace core {

// Timer provides a timer that measures monotonic time between calls to Start()
// and Stop().
class Timer {
 public:
  // Begin the timer.
  void Start();

  // Stop the timer and report the time in nanoseconds since Start was called.
  uint64_t Stop();

 private:
  uint64_t mStartTime;  // Units dependent on platform.
};

uint64_t GetNanoseconds();

}  // namespace core

#endif  // CORE_TIMER_H
