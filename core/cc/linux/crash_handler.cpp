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

#include "../crash_handler.h"

#include "client/linux/handler/exception_handler.h"

#include <stdlib.h>

namespace {

static bool handleCrash(const google_breakpad::MinidumpDescriptor& descriptor,
                        void* crashHandlerPtr, bool succeeded) {
  core::CrashHandler* crashHandler =
      reinterpret_cast<core::CrashHandler*>(crashHandlerPtr);
  std::string minidumpPath(descriptor.path());
  return crashHandler->handleMinidump(minidumpPath, succeeded);
}

}  // namespace

namespace core {

namespace {
const char* GetTempDir() {
  const char* tmpdir = getenv("TMPDIR");
  if (!tmpdir) {
    tmpdir = "/tmp";
  }
  return tmpdir;
}
}  // namespace

CrashHandler::CrashHandler()
    : mNextHandlerID(0),
      mExceptionHandler(new google_breakpad::ExceptionHandler(
          google_breakpad::MinidumpDescriptor(GetTempDir()), NULL,
          ::handleCrash, reinterpret_cast<void*>(this), true, -1)) {
  registerHandler(defaultHandler);
}

CrashHandler::CrashHandler(const std::string& crashDir)
    : mNextHandlerID(0),
      mExceptionHandler(new google_breakpad::ExceptionHandler(
          google_breakpad::MinidumpDescriptor(crashDir), NULL, ::handleCrash,
          reinterpret_cast<void*>(this), true, -1)) {
  registerHandler(defaultHandler);
}

// this prevents unique_ptr<CrashHandler> from causing an incomplete type error
// from inlining the destructor. The incomplete type is the previously forward
// declared google_breakpad::ExceptionHandler.
CrashHandler::~CrashHandler() = default;

}  // namespace core
