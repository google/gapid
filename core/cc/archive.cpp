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
#include "log.h"
#include "target.h" // ftruncate

#ifdef _MSC_VER // MSVC
#   include <io.h>
#ifndef __GNUC__
// MSYS GCC allows fileno, MSVC itself does not.
#define fileno _fileno
#endif
#else // not MSVC
#   include <unistd.h>
#endif

namespace core {

Archive::Archive(const std::string& archiveName) {
    // Open or create the archive data file in binary read/write mode.
    const std::string dataFilename(archiveName + ".data");
    if (!(mDataFile = fopen(dataFilename.c_str(), "ab+"))) {
        GAPID_FATAL("Unable to open archive data file %s", dataFilename.c_str());
    }

    // Open or create the archive index file in binary read/write mode.
    const std::string indexFilename(archiveName + ".index");
    if (!(mIndexFile = fopen(indexFilename.c_str(), "ab+"))) {
        GAPID_FATAL("Unable to open archive index file %s", indexFilename.c_str());
    }

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

        mRecords.emplace(id, ArchiveRecord{ offset, size });
    }

    // Make sure we're at the end of the index file, likely a no-op.
    fseek(mIndexFile, 0, SEEK_END);
}

Archive::~Archive() {
    if (mDataFile) fclose(mDataFile);
    if (mIndexFile) fclose(mIndexFile);
}

bool Archive::contains(const std::string& id) const {
    return mRecords.find(id) != mRecords.end();
}

bool Archive::read(const std::string& id, void* buffer, uint32_t size) {
    const auto r = mRecords.find(id);
    if (r == mRecords.end() || r->second.size != size) return false;

    fseek(mDataFile, r->second.offset, SEEK_SET);
    return fread(buffer, size, 1, mDataFile) == 1;
}

bool Archive::write(const std::string& id, const void* buffer, uint32_t size) {
    // Skip if we already have a record by this id.
    if (mRecords.find(id) != mRecords.end()) {
        return true;
    }

    // Update the archive data file.
    fseek(mDataFile, 0, SEEK_END);
    const uint64_t dataOffset = ftell(mDataFile);
    if (!fwrite(buffer, size, 1, mDataFile)) {
        GAPID_WARNING("Couldn't write '%s' to the archive data file, dropping it.", id.c_str());
        ftruncate(fileno(mDataFile), dataOffset);
        return false;
    }

    // Update the archive index file.
    const uint32_t idSize = id.size();
    const uint64_t indexOffset = ftell(mIndexFile);
    if (!fwrite(&idSize, sizeof(idSize), 1, mIndexFile) ||
        !fwrite(&id.front(), id.size(), 1, mIndexFile) ||
        !fwrite(&dataOffset, sizeof(dataOffset), 1, mIndexFile) ||
        !fwrite(&size, sizeof(size), 1, mIndexFile)) {
        GAPID_WARNING("Couldn't write '%s' to the archive index file, dropping it.", id.c_str());
        ftruncate(fileno(mDataFile), dataOffset);
        ftruncate(fileno(mIndexFile), indexOffset);
        fseek(mIndexFile, 0, SEEK_END);
        return false;
    }

    // Update the memory index.
    mRecords.emplace(id, ArchiveRecord{ dataOffset, size });
    return true;
}

}  // namespace core
