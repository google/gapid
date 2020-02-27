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

#include "client/windows/handler/exception_handler.h"

#include "../log.h"

#include <windows.h>

#include <memory>

namespace {

static bool handleCrash(const wchar_t* minidumpDir, const wchar_t* minidumpId,
                        void* crashHandlerPtr, EXCEPTION_POINTERS* exinfo,
                        MDRawAssertionInfo* assertion, bool succeeded) {
  core::CrashHandler* crashHandler =
      reinterpret_cast<core::CrashHandler*>(crashHandlerPtr);
  // convert wchar_t to UTF-8
  size_t dirLen =
      WideCharToMultiByte(CP_UTF8, 0, minidumpDir, -1, NULL, 0, NULL, NULL);
  size_t IdLen =
      WideCharToMultiByte(CP_UTF8, 0, minidumpId, -1, NULL, 0, NULL, NULL);
  std::unique_ptr<char[]> dirBuffer(new char[dirLen]);
  std::unique_ptr<char[]> idBuffer(new char[IdLen]);
  WideCharToMultiByte(CP_UTF8, 0, minidumpDir, -1, dirBuffer.get(), dirLen,
                      NULL, NULL);
  WideCharToMultiByte(CP_UTF8, 0, minidumpId, -1, idBuffer.get(), IdLen, NULL,
                      NULL);
  std::string minidumpPath(dirBuffer.get());
  minidumpPath.append(idBuffer.get());
  minidumpPath.append(".dmp");
  return crashHandler->handleMinidump(minidumpPath, succeeded);
}

}  // namespace

namespace core {

CrashHandler::CrashHandler() : mNextHandlerID(0) {
  wchar_t tempPathBuf[MAX_PATH + 1];
  GetTempPathW(MAX_PATH + 1, tempPathBuf);
  std::wstring tempPath(tempPathBuf);
  mExceptionHandler = std::unique_ptr<google_breakpad::ExceptionHandler>(
      new google_breakpad::ExceptionHandler(
          tempPath, NULL, ::handleCrash, reinterpret_cast<void*>(this),
          google_breakpad::ExceptionHandler::HandlerType::HANDLER_ALL));

  registerHandler(defaultHandler);
}

CrashHandler::CrashHandler(const std::string& crashDir) : mNextHandlerID(0) {
  size_t length =
      MultiByteToWideChar(CP_UTF8, 0, crashDir.c_str(), -1, NULL, 0);
  std::unique_ptr<wchar_t[]> crashDirBuffer(new wchar_t[length]);
  MultiByteToWideChar(CP_UTF8, 0, crashDir.c_str(), -1, crashDirBuffer.get(),
                      length);
  std::wstring crashDirW(crashDirBuffer.get());

  std::unique_ptr<google_breakpad::ExceptionHandler>(
      new google_breakpad::ExceptionHandler(
          crashDirW, NULL, ::handleCrash, reinterpret_cast<void*>(this),
          google_breakpad::ExceptionHandler::HandlerType::HANDLER_ALL));

  registerHandler(defaultHandler);
}

// this prevents unique_ptr<CrashHandler> from causing an incomplete type error
// from inlining the destructor. The incomplete type is the previously forward
// declared google_breakpad::ExceptionHandler.
CrashHandler::~CrashHandler() = default;

}  // namespace core
