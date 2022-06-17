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

#pragma once

namespace gapid2 {

enum class flags : uint64_t {
  NONE = 0,
  MID_EXECUTION = 1 << 0
};

inline flags operator|(const flags& a, const flags& b) {
  return static_cast<flags>(static_cast<uint64_t>(a) |
                            static_cast<uint64_t>(b));
}

inline flags operator&(const flags& a, const flags& b) {
  return static_cast<flags>(static_cast<uint64_t>(a) |
                            static_cast<uint64_t>(b));
}

inline flags operator^(const flags& a, const flags& b) {
  return static_cast<flags>(static_cast<uint64_t>(a) |
                            static_cast<uint64_t>(b));
}

}  // namespace gapid2