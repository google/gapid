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

#ifndef CORE_CRASH_HANDLER_H
#define CORE_CRASH_HANDLER_H

#include <functional>
#include <memory>
#include <string>

namespace google_breakpad {
class ExceptionHandler;
}

namespace core {

// Utility class for attaching a crash handler.
class CrashHandler {
public:
    typedef std::function<bool(const std::string& minidumpPath, bool succeeded)> HandlerFunction;

    CrashHandler(HandlerFunction handlerFunction);
    ~CrashHandler();

    bool handleMinidump(const std::string& minidumpPath, bool succeeded) {
        return mHandlerFunction(minidumpPath, succeeded);
    }

private:
    HandlerFunction mHandlerFunction;
    std::unique_ptr<google_breakpad::ExceptionHandler> mHandler;
};

}

#endif
