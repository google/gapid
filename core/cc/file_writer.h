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

#ifndef CORE_FILE_WRITER_H
#define CORE_FILE_WRITER_H

#include <stdio.h>

#include "stream_writer.h"

namespace core {

// FileWriter is an implementation of the StreamWriter interface that writes to
// a binary file.
class FileWriter : public StreamWriter {
 public:
  FileWriter(const char* path);
  ~FileWriter();

  // StreamWriter compliance
  virtual uint64_t write(const void* data, uint64_t size) override;

 private:
  FILE* mFile;
};

}  // namespace core

#endif  // CORE_FILE_WRITER_H
