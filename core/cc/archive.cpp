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

#include "archive.h"

#include "assert.h"
#include "log.h"
#include "target.h"  // ftruncate

#ifdef _MSC_VER  // MSVC
#include <io.h>
#ifndef __GNUC__
// MSYS GCC allows fileno, MSVC itself does not.
#define fileno _fileno
#endif
#else  // not MSVC
#include <unistd.h>
#endif

#if GAPID_ARCHIVE_USE_MMAP

#include <fcntl.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/types.h>

#endif  //  GAPID_ARCHIVE_USE_MMAP

namespace {

void must_truncate(int fd, off_t length) {
  if (!ftruncate(fd, length)) {
    GAPID_ASSERT("Unable to truncate the achive file");
  }
}

}  // anonymous namespace

namespace core {

static const char* kIndexFileNameSuffix = ".index";
static const char* kDataFileNameSuffix = ".data";

// Use mmap-ed data file if it is available.
// fseek+fread might fail on some driver.
#if GAPID_ARCHIVE_USE_MMAP

Archive::RecordFile::RecordFile()
    : fd(-1), base(nullptr), end(0), capacity(0) {}

bool Archive::RecordFile::open(const std::string& filename) {
  fd = ::open(filename.c_str(), O_RDWR | O_CREAT, S_IRWXU);
  if (fd == -1) {
    return false;
  }
  struct stat st;
  ::fstat(fd, &st);
  end = st.st_size;
  capacity = end;
  return map();
}

void Archive::RecordFile::close() {
  if (fd == -1) return;
  if (!unmap()) {
    GAPID_FATAL("Unable to unmap archive file.");
  }
  // Set the proper file size when we close the file.
  if (::ftruncate(fd, end)) {
    GAPID_FATAL("Unable to truncate archive file.");
  }
  ::close(fd);
}

bool Archive::RecordFile::read(uint64_t offset, void* buf, size_t size) {
  if (offset + size > end) {
    return false;
  }
  memcpy(buf, at(offset), size);
  return true;
}

bool Archive::RecordFile::append(const void* buf, size_t size) {
  if (!reserve(end + size)) {
    return false;
  }
  memcpy(at(end), buf, size);
  end += size;
  return true;
}

uint64_t Archive::RecordFile::size() { return end; }

bool Archive::RecordFile::resize(uint64_t size) {
  if (size > capacity) {
    if (!reserve(size)) {
      return false;
    }
  }
  end = size;
  return true;
}

bool Archive::RecordFile::reserve(uint64_t requiredCapacity) {
  if (requiredCapacity <= capacity) {
    return true;
  }

  if (!unmap()) {
    GAPID_FATAL("Unable to unmap archive file.");
  }

  // Reserve at least 1.5 time the size.
  if (requiredCapacity < end * 3 / 2) {
    requiredCapacity = end * 3 / 2;
  }

  // Align to the next 4k boundary, not necessary but good to have.
  requiredCapacity = (requiredCapacity + 0xfff) & ~(uint64_t)0xfff;

  if (::ftruncate(fd, requiredCapacity)) {
    GAPID_FATAL("Unable to ftruncate(grow) archive file.");
  }

  capacity = requiredCapacity;

  if (!map()) {
    GAPID_FATAL("Unable to map archive file.");
  }

  return true;
}

bool Archive::RecordFile::unmap() {
  if (!base) return true;
  if (::munmap(base, capacity)) {
    return false;
  }
  base = nullptr;
  return true;
}

bool Archive::RecordFile::map() {
  if (base || capacity == 0) {
    return true;  // Already mapped or don't need to map.
  }
  base = ::mmap(nullptr, capacity, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
  return base != nullptr;
}

#else  // #if GAPID_ARCHIVE_USE_MMAP

Archive::RecordFile::RecordFile() : fp(nullptr) {}

bool Archive::RecordFile::open(const std::string& filename) {
  fp = fopen(filename.c_str(), "ab+");
  return fp;
}

void Archive::RecordFile::close() {
  if (fp) fclose(fp);
}

bool Archive::RecordFile::read(uint64_t offset, void* buf, size_t size) {
  fseek(fp, offset, SEEK_SET);
  return fread(buf, size, 1, fp) == 1;
}

bool Archive::RecordFile::append(const void* buf, size_t size) {
  fseek(fp, 0, SEEK_END);
  return fwrite(buf, size, 1, fp) == 1;
}

uint64_t Archive::RecordFile::size() {
  fseek(fp, 0, SEEK_END);
  return ftell(fp);
}

bool Archive::RecordFile::resize(uint64_t size) {
  must_truncate(fileno(fp), size);
  return true;
}

#endif  //  GAPID_ARCHIVE_USE_MMAP

Archive::Archive(const std::string& archiveName)
    : mDataFilePath(archiveName + kDataFileNameSuffix),
      mIndexFilePath(archiveName + kIndexFileNameSuffix) {
  // Open or create the archive data file in binary read/write mode.
  const std::string dataFilename(mDataFilePath);
  if (!mDataFile.open(dataFilename)) {
    GAPID_FATAL("Unable to open archive data file %s", dataFilename.c_str());
  }

  // Open or create the archive index file in binary read/write mode.
  const std::string indexFilename(mIndexFilePath);
  if (!(mIndexFile = fopen(indexFilename.c_str(), "ab+"))) {
    GAPID_FATAL("Unable to open archive index file %s", indexFilename.c_str());
  }

  // Linux fopen() with mode "a" leads to reads from the beginning of file, but
  // this is not true on Android, hence the explicit rewind() here
  rewind(mIndexFile);

  // Load the archive index in memory.
  for (;;) {
    uint32_t idSize;
    if (!fread(&idSize, sizeof(idSize), 1, mIndexFile)) break;

    std::string id(idSize, 0);
    uint64_t offset;
    uint32_t size;
    if (!fread(&id.front(), idSize, 1, mIndexFile) ||
        !fread(&offset, sizeof(offset), 1, mIndexFile) ||
        !fread(&size, sizeof(size), 1, mIndexFile)) {
      break;
    }

    mRecords.emplace(id, ArchiveRecord{offset, size});
  }

  // Make sure we're at the end of the index file, likely a no-op.
  fseek(mIndexFile, 0, SEEK_END);
}

Archive::~Archive() {
  mDataFile.close();
  if (mIndexFile) fclose(mIndexFile);
}

bool Archive::contains(const std::string& id) const {
  return mRecords.find(id) != mRecords.end();
}

bool Archive::read(const std::string& id, void* buffer, uint32_t size) {
  const auto r = mRecords.find(id);
  if (r == mRecords.end() || r->second.size != size) return false;

  return mDataFile.read(r->second.offset, buffer, size);
}

bool Archive::write(const std::string& id, const void* buffer, uint32_t size) {
  // Skip if we already have a record by this id.
  if (mRecords.find(id) != mRecords.end()) {
    return true;
  }

  // Update the archive data file.
  const uint64_t dataOffset = mDataFile.size();
  if (!mDataFile.append(buffer, size)) {
    GAPID_WARNING("Couldn't write '%s' to the archive data file, dropping it.",
                  id.c_str());
    return false;
  }

  // Update the archive index file.
  const uint32_t idSize = id.size();
  const uint64_t indexOffset = ftell(mIndexFile);
  if (!fwrite(&idSize, sizeof(idSize), 1, mIndexFile) ||
      !fwrite(&id.front(), id.size(), 1, mIndexFile) ||
      !fwrite(&dataOffset, sizeof(dataOffset), 1, mIndexFile) ||
      !fwrite(&size, sizeof(size), 1, mIndexFile)) {
    GAPID_WARNING("Couldn't write '%s' to the archive index file, dropping it.",
                  id.c_str());
    mDataFile.resize(dataOffset);
    must_truncate(fileno(mIndexFile), indexOffset);
    fseek(mIndexFile, 0, SEEK_END);
    return false;
  }

  // Update the memory index.
  mRecords.emplace(id, ArchiveRecord{dataOffset, size});
  return true;
}

}  // namespace core

extern "C" {

archive* archive_create(const char* archiveName) {
  return reinterpret_cast<archive*>(new core::Archive(archiveName));
}

void archive_destroy(archive* a) { delete reinterpret_cast<core::Archive*>(a); }

int archive_write(archive* a, const char* id, const void* buffer, size_t size) {
  return reinterpret_cast<core::Archive*>(a)->write(id, buffer, size) ? 1 : 0;
}

}  // extern "C"
