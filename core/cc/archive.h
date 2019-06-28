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

#ifdef __cplusplus

#include <stdint.h>
#include <stdio.h>

#include <string>
#include <unordered_map>

#include "id.h"
#include "target.h"

#if TARGET_OS == GAPID_OS_LINUX
#define GAPID_ARCHIVE_USE_MMAP 1
#else
#define GAPID_ARCHIVE_USE_MMAP 0
#endif

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

  // Returns the path of the index file.
  std::string indexFilePath() const { return mIndexFilePath; }

  // Returns the path of the data file.
  std::string dataFilePath() const { return mDataFilePath; }

 protected:
  struct ArchiveRecord {
    uint64_t offset;
    uint32_t size;
  };

  struct RecordFile {
    RecordFile();
    bool open(const std::string& filename);
    void close();
    bool read(uint64_t offset, void* buf, size_t size);
    bool append(const void* buf, size_t size);
    uint64_t size();
    bool resize(uint64_t size);

   private:
#if GAPID_ARCHIVE_USE_MMAP
    bool reserve(uint64_t requiredCapacity);
    bool unmap();
    bool map();
    void* at(uint64_t offset) { return static_cast<char*>(base) + offset; }

    int fd;
    void* base;
    uint64_t end;
    uint64_t capacity;
#else
    FILE* fp;
#endif
  };

  RecordFile mDataFile;
  FILE* mIndexFile;
  std::unordered_map<std::string, ArchiveRecord> mRecords;
  const std::string mDataFilePath;
  const std::string mIndexFilePath;
};

}  // namespace core

extern "C" {
#endif
typedef struct archive_t archive;
archive* archive_create(const char* archiveName);
void archive_destroy(archive* a);
int archive_write(archive* a, const char* id, const void* buffer, size_t size);
#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // CORE_ARCHIVE_H
