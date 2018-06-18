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

#ifndef CORE_STREAM_WRITER_H
#define CORE_STREAM_WRITER_H

#include <stdint.h>

namespace core {

// StreamWriter is a pure-virtual interface used to write data streams.
class StreamWriter {
 public:
  // write attempts to write size bytes from data to the stream, blocking
  // until all data is written. Returns the number of bytes successfully
  // written, which may be less than size if the stream was closed or there
  // was an error.
  virtual uint64_t write(const void* data, uint64_t size) = 0;

  // write attempts to write the bytes of s to the stream, returning true on
  // success or false if the write was partial or a complete failure.
  // Note: T must be a plain-old-data type.
  template <typename T>
  inline bool write(const T& s);

 protected:
  virtual ~StreamWriter() {}
};

template <typename T>
inline bool StreamWriter::write(const T& s) {
  return write(&s, sizeof(s)) == sizeof(s);
}

}  // namespace core

#endif  // CORE_STREAM_WRITER_H
