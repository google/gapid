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

CrashHandler::HandlerFunction CrashHandler::defaultHandlerFunction =
        [] (const std::string& minidump_path, bool succeeded) {
            if (!succeeded) {
                GAPID_ERROR("Failed to write minidump out to %s", minidump_path.c_str());
            }
            return succeeded;
        };

void CrashHandler::setHandlerFunction(HandlerFunction newHandlerFunction) {
    mHandlerFunction = newHandlerFunction;
}

void CrashHandler::unsetHandlerFunction() {
    mHandlerFunction = defaultHandlerFunction;
}

bool CrashHandler::handleMinidump(const std::string& minidumpPath, bool succeeded) {
    return mHandlerFunction(minidumpPath, succeeded);
}


} // namespace core
