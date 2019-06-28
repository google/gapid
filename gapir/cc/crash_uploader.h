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

#ifndef GAPIR_CRASH_REPORTER_H
#define GAPIR_CRASH_REPORTER_H

#include <memory>
#include <string>

#include "core/cc/crash_handler.h"
#include "gapir/cc/replay_service.h"

namespace gapir {

// CrashUploader uploads crash minidumps from a CrashHandler to GAPIS via a
// ServerConnection.
class CrashUploader {
 public:
  CrashUploader(core::CrashHandler& crash_handler, ReplayService* srv);
  ~CrashUploader();

 private:
  core::CrashHandler::Unregister mUnregister;
  ReplayService* mSrv;
};

}  // namespace gapir

#endif  // GAPIR_CRASH_REPORTER_H
