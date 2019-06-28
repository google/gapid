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

#include "core/cc/crash_handler.h"

#include "core/cc/log.h"

namespace core {

CrashHandler::Handler CrashHandler::defaultHandler =
    [](const std::string& minidump_path, bool succeeded) {
      if (!succeeded) {
        GAPID_ERROR("Failed to write minidump out to %s",
                    minidump_path.c_str());
      }
    };

CrashHandler::Unregister CrashHandler::registerHandler(Handler handler) {
  auto id = mNextHandlerID++;
  mHandlers[id] = handler;
  return [this, id]() { mHandlers.erase(id); };
}

bool CrashHandler::handleMinidump(const std::string& minidumpPath,
                                  bool succeeded) {
  for (const auto& it : mHandlers) {
    it.second(minidumpPath, succeeded);
  }
  return succeeded;
}

}  // namespace core
