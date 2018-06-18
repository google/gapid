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

#ifndef CORE_STREAM_READER_H
#define CORE_STREAM_READER_H

#include <stdint.h>

namespace core {

// StreamReader is a pure-virtual interface used to read from data streams.
class StreamReader {
 public:
  // read attempts to read max_size bytes to the pointer data, blocking until
  // the data is available. Returns the number of bytes successfully read,
  // which may be less than size if the stream was closed or there was an
  // error.
  virtual uint64_t read(void* data, uint64_t max_size) = 0;

  // read attempts to read the s from the stream, returning true on success or
  // false if the read was partial or a complete failure.
  // Note: T must be a plain-old-data type.
  template <typename T>
  inline bool read(T& s);

 protected:
  virtual ~StreamReader() {}
};

template <typename T>
inline bool StreamReader::read(T& s) {
  return read(&s, sizeof(s)) == sizeof(s);
}

}  // namespace core

#endif  // CORE_STREAM_READER_H
