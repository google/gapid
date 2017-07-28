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

#include "log.h"

#include "core/cc/target.h"

#if TARGET_OS != GAPID_OS_ANDROID

#include <errno.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <chrono>
#include <ctime> // Required for MSVC.

namespace core {

Logger Logger::mInstance = Logger();

void Logger::init(unsigned level, const char* system, const char* path) {
    mInstance.mLevel = level;
    mInstance.mSystem = system;
    if (path != nullptr) {
        if (FILE* f = fopen(path, "w")) {
            GAPID_INFO("Logging to %s", path);
            mInstance.mFiles.push_back(f);
        } else {
            GAPID_WARNING("Can't open file for logging (%s): %s", path, strerror(errno));
        }
    }
}

Logger::Logger() : mLevel(LOG_LEVEL_INFO), mSystem("") {
    mFiles.push_back(stdout);
}

Logger::~Logger() {
    for (FILE* file : mFiles) {
        fclose(file);
    }
}

void Logger::log(unsigned level, const char* src_file, unsigned src_line, const char* format, ...) const {
    // Get the current time with milliseconds precision
    auto t = std::chrono::system_clock::now();
    std::time_t now = std::chrono::system_clock::to_time_t(t);
    std::tm* loc = std::localtime(&now);
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(t.time_since_epoch());

    for (FILE* file : mFiles) {
        // Print out the common part of the log messages
        fprintf(file, "%02d:%02d:%02d.%03d %c %s: [%s:%u] ", loc->tm_hour, loc->tm_min, loc->tm_sec,
                static_cast<int>(ms.count() % 1000), "FEWIDV"[level], mSystem, src_file, src_line);

        // Print out the actual log message
        va_list args;
        va_start(args, format);
        vfprintf(file, format, args);
        va_end(args);

        // Always finish with a newline
        fprintf(file, "\n");

        // Flush the log to ensure that every message is written out even if the application crashes
        fflush(file);
    }

    if (level == LOG_LEVEL_FATAL) {
        exit(EXIT_FAILURE);
    }
}

}  // namespace core

#endif  // TARGET_OS != GAPID_OS_ANDROID
