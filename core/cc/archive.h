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

#ifndef CORE_ARCHIVE_H
#define CORE_ARCHIVE_H

#include "id.h"

#include <string>
#include <unordered_map>

#include <stdint.h>
#include <stdio.h>

namespace core {

class Archive {
 public:
  // Opens or creates an archive at the specified location archiveName (full
  // path).
  Archive(const std::string& archiveName);
  ~Archive();

  // Checks if the archive contains a record for the given id.
  bool contains(const std::string& id) const;

  // Reads the resource keyed by id into buffer if it exists and if its size
  // matches.
  bool read(const std::string& id, void* buffer, uint32_t size);

  // Write a resource of size size keyed by id from buffer into the archive.
  bool write(const std::string& id, const void* buffer, uint32_t size);

 protected:
  struct ArchiveRecord {
    uint64_t offset;
    uint32_t size;
  };

  FILE* mDataFile;
  FILE* mIndexFile;
  std::unordered_map<std::string, ArchiveRecord> mRecords;
};

}  // namespace core

#endif  // CORE_ARCHIVE_H
