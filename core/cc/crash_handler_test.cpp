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

#include "crash_handler.h"

#include <gtest/gtest.h>
#include <stdlib.h>

#include <iostream>

#include "core/cc/target.h"

namespace core {
namespace test {

#if TARGET_OS != GAPID_OS_OSX  // Work around for
                               // https://github.com/google/gapid/issues/1788
TEST(CrashHandlerTest, HandleCrash) {
  CrashHandler crashHandler;
  crashHandler.registerHandler(
      [](const std::string& minidumpPath, bool succeeded) {
        if (succeeded) {
          std::cerr << "crash handled.";
        } else {
          std::cerr << "crash not handled.";
        }
      });

  EXPECT_DEATH({ (void)*((volatile int*)(0)); }, "crash handled.");
}
#endif  // TARGET_OS != GAPID_OS_OSX

}  // namespace test
}  // namespace core
