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

/*
 * Between Android K and L, some platform functions were changed from inline to
 * system library functions. We build against the Android 21 (L) NDK so we can
 * support 64-bit ARM devices. gtest makes calls to sigemptyset and getpagesize
 * which are part of these functions that were un-inlined. To keep the tests
 * working on older devices, stub these two functions here.
 */

#include <signal.h>

#include <cstring>

extern "C" {

/* TODO - delete this file?

int sigemptyset(sigset_t *set) {
    memset(set, 0, sizeof *set);
    return 0;
}

int getpagesize(void) {
  return 4096; // doesn't really matter - we're just satisfying the linker.
}
*/

}  // extern "C"
