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

#ifndef CORE_ASSERT_H
#define CORE_ASSERT_H

#include "log.h"

#define GAPID_ASSERT(cond)                     \
  do {                                         \
    if (!(cond)) {                             \
      GAPID_FATAL("Assert: " GAPID_STR(cond)); \
    }                                          \
  } while (false)
#define GAPID_ASSERT_MSG(cond, msg, ...)                                 \
  do {                                                                   \
    if (!(cond)) {                                                       \
      GAPID_FATAL("Assert: <" GAPID_STR(cond) ">: " msg, ##__VA_ARGS__); \
    }                                                                    \
  } while (false)

#endif  // CORE_ASSERT_H
