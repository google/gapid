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


#ifndef GAPII_MEMORY_PROTECTIONS_H
#define GAPII_MEMORY_PROTECTIONS_H


namespace gapii {
namespace track_memory {

enum class PageProtections {
  kNone = 0x0,
  kRead = 0x1,
  kWrite = 0x2,
  kReadWrite = 0x1 | 0x2
};

} // namespace track_memory
} // namespace gapii

#endif // GAPII_MEMORY_PROTECTIONS_H