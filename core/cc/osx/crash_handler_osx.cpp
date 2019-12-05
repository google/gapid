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

#include "../debugger.h"

#include "client/mac/handler/exception_handler.h"

#include <stdlib.h>

#include <execinfo.h>
#include <stdio.h>

namespace {

static bool handleCrash(const char* minidumpDir, const char* minidumpId,
                        void* crashHandlerPtr, bool succeeded) {
  core::CrashHandler* crashHandler =
      reinterpret_cast<core::CrashHandler*>(crashHandlerPtr);
  std::string minidumpPath(minidumpDir);
  minidumpPath.append(minidumpId);
  minidumpPath.append(".dmp");
  return crashHandler->handleMinidump(minidumpPath, succeeded);
}

}  // namespace

namespace core {

namespace {
const char* GetTempDir() {
  const char* tmpdir = getenv("TMPDIR");
  if (!tmpdir) {
    tmpdir = "/tmp/";
  }
  return tmpdir;
}
}  // namespace

CrashHandler::CrashHandler() : mNextHandlerID(0), mExceptionHandler(nullptr) {
  if (!Debugger::isAttached()) {
    mExceptionHandler = std::unique_ptr<google_breakpad::ExceptionHandler>(
        new google_breakpad::ExceptionHandler(
            GetTempDir(), nullptr, ::handleCrash, reinterpret_cast<void*>(this),
            true, nullptr));
  }
  registerHandler(defaultHandler);
}

CrashHandler::CrashHandler(const std::string& crashDir)
    : mNextHandlerID(0), mExceptionHandler(nullptr) {
  if (!Debugger::isAttached()) {
    mExceptionHandler = std::unique_ptr<google_breakpad::ExceptionHandler>(
        new google_breakpad::ExceptionHandler(crashDir, nullptr, ::handleCrash,
                                              reinterpret_cast<void*>(this),
                                              true, nullptr));
  }
  registerHandler(defaultHandler);
}

// this prevents unique_ptr<CrashHandler> from causing an incomplete type error
// from inlining the destructor. The incomplete type is the previously forward
// declared google_breakpad::ExceptionHandler.
CrashHandler::~CrashHandler() = default;

}  // namespace core
