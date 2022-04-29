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
struct printer {
  virtual void begin_object(const char* name) = 0;
  virtual void end_object() = 0;
  virtual void begin_array(const char* name) = 0;
  virtual void end_array() = 0;
  virtual void handle_comma() = 0;
#define MAKE_PRINT(tp) virtual void print(const char* name, tp val) = 0;
  MAKE_PRINT(uint64_t)
  MAKE_PRINT(uint32_t)
  MAKE_PRINT(uint16_t)
  MAKE_PRINT(uint8_t)
  MAKE_PRINT(int64_t)
  MAKE_PRINT(int32_t)
  MAKE_PRINT(int16_t)
  MAKE_PRINT(int8_t)
  MAKE_PRINT(float)
  MAKE_PRINT(nullptr_t)
#undef MAKE_PRINT

  virtual void print_null(const char* name) = 0;
  virtual void print_char_array(const char* name, const char* val, size_t size) = 0;
  virtual void print_string(const char* name, const char* str) = 0;
};
}  // namespace gapid2