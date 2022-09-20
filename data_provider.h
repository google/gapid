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
class transform_data_provider {
 public:
  const uint64_t get_current_command_index() const {
    return m_current_command_index;
  }
  void set_current_command_index(uint64_t idx) {
    m_current_command_index = idx;
  }

 private:
  uint64_t m_current_command_index = 0;
};
}  // namespace gapid2