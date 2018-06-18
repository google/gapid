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

#include "file_reader.h"

#include "log.h"

namespace core {

FileReader::FileReader(const char* path) { mFile = fopen(path, "rb"); }
FileReader::~FileReader() { fclose(mFile); }

const char* FileReader::error() const {
  if (mFile == 0) {
    return "File did not open";
  }
  return 0;
}

uint64_t FileReader::read(void* data, uint64_t max_size) {
  return fread(data, 1, max_size, mFile);
}

uint64_t FileReader::size() {
  // TODO: this is not portable, need to move to platform file.
  if (fseek(mFile, 0, SEEK_END) != 0) {
    GAPID_ERROR("Failed to seek to the end of file");
    return 0u;
  }

  long int sizeTemp = ftell(mFile);
  if (sizeTemp == -1L) {
    GAPID_ERROR("Failed to get size of file")
    rewind(mFile);
    return 0u;
  }
  uint64_t size = static_cast<uint64_t>(sizeTemp);
  rewind(mFile);
  return size;
}

}  // namespace core
