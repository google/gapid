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

#include "gapir/cc/crash_uploader.h"

#include <stdio.h>

#include "core/cc/crash_handler.h"
#include "core/cc/file_reader.h"
#include "core/cc/log.h"
#include "gapir/cc/replay_service.h"

namespace gapir {

CrashUploader::CrashUploader(core::CrashHandler& crash_handler,
                             ReplayService* srv)
    : mSrv(srv) {
  mUnregister = crash_handler.registerHandler(
      [this](const std::string& minidumpPath, bool succeeded) {
        if (!succeeded) {
          GAPID_ERROR("Failed to write minidump out to %s",
                      minidumpPath.c_str());
        }

        core::FileReader minidumpFile(minidumpPath.c_str());
        if (const char* err = minidumpFile.error()) {
          GAPID_ERROR("Failed to open minidump file %s: %s",
                      minidumpPath.c_str(), err);
          return;
        }

        uint64_t minidumpSize = minidumpFile.size();
        if (minidumpSize == 0u) {
          GAPID_ERROR("Failed to get minidump file size %s",
                      minidumpPath.c_str());
          return;
        }
        std::unique_ptr<char[]> minidumpData =
            std::unique_ptr<char[]>(new char[minidumpSize]);
        uint64_t read = minidumpFile.read(minidumpData.get(), minidumpSize);
        if (read != minidumpSize) {
          GAPID_ERROR("Failed to read in the minidump file");
          return;
        }

        if (!mSrv->sendCrashDump(minidumpPath, minidumpData.get(),
                                 minidumpSize)) {
          GAPID_ERROR("Failed to send minidump to server");
          return;
        }
      });
}

CrashUploader::~CrashUploader() { mUnregister(); }

}  // namespace gapir
