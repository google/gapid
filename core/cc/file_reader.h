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

#ifndef CORE_FILE_READER_H
#define CORE_FILE_READER_H

#include <stdint.h>
#include <stdio.h>

#include "stream_reader.h"

namespace core {

// FileReader is an implementation of the StreamReader interface that reads
// from binary files.
class FileReader : public StreamReader {
 public:
  FileReader(const char* path);
  ~FileReader();

  // error returns an error string if the reader has encountered an error.
  const char* error() const;

  // read attempts to read max_size bytes to the pointer data, blocking until
  // the data is available. Returns the number of bytes successfully read,
  // which may be less than size if the stream was closed or there was an
  // error.
  virtual uint64_t read(void* data, uint64_t max_size);

  // size returns the number of bytes in the underlying file.
  uint64_t size();

 private:
  FILE* mFile;
};

}  // namespace core

#endif  // CORE_STREAM_READER_H
