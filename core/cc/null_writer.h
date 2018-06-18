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

#ifndef CORE_NULL_WRITER_H
#define CORE_NULL_WRITER_H

#include "core/cc/stream_writer.h"

namespace core {

// NullWriter implements StreamWriter, but outputs nothing.
class NullWriter : public core::StreamWriter {
 public:
  uint64_t write(const void* data, uint64_t size) { return size; }
  ~NullWriter() {}
};

}  // namespace core

#endif  // CORE_NULL_WRITER_H
