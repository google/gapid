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

#ifndef CORE_LOG_H
#define CORE_LOG_H

#include <vector>

// General logging functions. All logging should be done through these macros.
//
// The system supports the following log levels with the specified meanings:
// * LOG_LEVEL_FATAL:   Serious error. No recovery is possible and the system will die immediately.
// * LOG_LEVEL_ERROR:   Serious error. We can continue, but it should not happen during normal use.
// * LOG_LEVEL_WARNING: Possible issue. It is not technically an error, but it is suspicious.
// * LOG_LEVEL_INFO:    Normal behaviour. Small amount of messages that indicate program progress.
// * LOG_LEVEL_DEBUG    Used only for debugging. The amount of logging may slow down the program.
// * LOG_LEVEL_VERBOSE: Very verbose debug logging. It may log excessive amount of information.

#include "target.h"

// Levels of logging from least verbose to most verbose.
// These log levels correspond to Android log levels.
#define LOG_LEVEL_FATAL   0
#define LOG_LEVEL_ERROR   1
#define LOG_LEVEL_WARNING 2
#define LOG_LEVEL_INFO    3
#define LOG_LEVEL_DEBUG   4
#define LOG_LEVEL_VERBOSE 5

// If no log level specified then use the default one.
#ifndef LOG_LEVEL
#   define LOG_LEVEL LOG_LEVEL_INFO
#endif

#define GAPID_STR(S) GAPID_STR2(S)
#define GAPID_STR2(S) #S

#if TARGET_OS == GAPID_OS_WINDOWS
  #define PRIsize "Iu"
#else
  #define PRIsize "zu"
#endif

// Template meta-programming (tmp) used to strip absolute path from __FILE__.
// The main difficulty is that we need to make copy of the string since
// any live reference to __FILE__ will keep the whole path in the binary.
// We also want to avoid storing the string again for each log message.
namespace tmp {
  // Returns the filename and extension only without directories.
  constexpr const char* Basename(const char *str, unsigned i = 0) {
    return str[i] == '/' || str[i] == '\\' ? Basename(str + i + 1) :
           str[i] == '\0' ? str : Basename(str, i + 1);
  }
  // Returns the length of string literal excluding the zero-terminator.
  constexpr unsigned Strlen(const char *str) {
    return str[0] == '\0' ? 0 : 1 + Strlen(str + 1);
  }
  // Represents unique string literal. For example: String<'H','e','l','l','o'>
  template<char... C> struct String {
    static const char* value() { static const char str[] = {C...}; return str; }
  };
  // Represents consecutive index list. For example: Indices<0,1,2,3,4,5,6,7,8>
  template<int... I> struct Indices{};
  template<int N, int... I> struct MakeIndices : MakeIndices<N-1, N-1, I...>{};
  template<int... I> struct MakeIndices<0, I...> : Indices<I...>{};
  // Makes compile-time copy of a string (which is stored as boxed value in S).
  // Indices specify the characters to copy - that is, range 0 to strlen(S)+1
  template<typename S, int... I> constexpr const char* Strcpy(Indices<I...> i) {
    return String<S::value()[I]...>::value();
  }
}

#if TARGET_OS == GAPID_OS_ANDROID

#include <android/log.h>

#define GAPID_LOGGER_INIT(...)
#define GAPID_LOG(LEVEL, ANDROID_LOG_LEVEL, FORMAT, ...)                                          \
  if (LOG_LEVEL >= LEVEL) {                                                                       \
    /* Literal string is very difficult pass to any function, but it works if we box it. */       \
    struct boxedStr { constexpr static const char* value() { return tmp::Basename(__FILE__); } }; \
    auto basename = tmp::Strcpy<boxedStr>(tmp::MakeIndices<tmp::Strlen(boxedStr{}.value())+1>{}); \
    if (LEVEL == LOG_LEVEL_FATAL) {                                                               \
      __android_log_assert(nullptr, "GAPID", "[%s:%u] " FORMAT,                                   \
                           basename, __LINE__, ##__VA_ARGS__);                                    \
    } else {                                                                                      \
      __android_log_print(ANDROID_LOG_LEVEL, "GAPID", "[%s:%u] " FORMAT,                          \
                          basename, __LINE__, ##__VA_ARGS__);                                     \
    }                                                                                             \
  }
#define GAPID_FATAL(FORMAT, ...) GAPID_LOG(LOG_LEVEL_FATAL, ANDROID_LOG_FATAL, FORMAT, ##__VA_ARGS__)
#define GAPID_ERROR(FORMAT, ...) GAPID_LOG(LOG_LEVEL_ERROR, ANDROID_LOG_ERROR, FORMAT, ##__VA_ARGS__)
#define GAPID_WARNING(FORMAT, ...) GAPID_LOG(LOG_LEVEL_WARNING, ANDROID_LOG_WARN, FORMAT, ##__VA_ARGS__)
#define GAPID_INFO(FORMAT, ...) GAPID_LOG(LOG_LEVEL_INFO, ANDROID_LOG_INFO, FORMAT, ##__VA_ARGS__)
#define GAPID_DEBUG(FORMAT, ...) GAPID_LOG(LOG_LEVEL_DEBUG, ANDROID_LOG_DEBUG, FORMAT, ##__VA_ARGS__)
#define GAPID_VERBOSE(FORMAT, ...) GAPID_LOG(LOG_LEVEL_VERBOSE, ANDROID_LOG_VERBOSE, FORMAT, ##__VA_ARGS__)

#else  // TARGET_OS == GAPID_OS_ANDROID

#include <stdio.h>

namespace core {

// Singleton logger implementation for PCs to write formatted log messages.
class Logger {
public:
    // Initializes the logger to write to the log file at path.
    static void init(unsigned level, const char* system, const char* path);

    // Write a log message to the log output with the specific log level. The location should
    // contain the place where the log is written from and the format is a standard C format string
    // If a message is logged with level LOG_LEVEL_FATAL, the program will terminate after the
    // message is printed.
    // Log messages take the form: <time> <level> <system> <file:line> : <message>
    void log(unsigned level, const char* file, unsigned line, const char* format, ...) const;

    static const Logger& instance() { return mInstance; }

    static unsigned level() { return mInstance.mLevel; }

private:
    // mInstance is the single logger instance.
    static Logger mInstance;

    Logger();
    ~Logger();

    unsigned mLevel;
    const char* mSystem;
    std::vector<FILE*> mFiles;
};

}  // namespace core

#define GAPID_LOGGER_INIT(...) ::core::Logger::init(__VA_ARGS__)
#define GAPID_LOG(LEVEL, FORMAT, ...)                                                             \
  if (::core::Logger::level() >= LEVEL) {                                                         \
    /* Literal string is very difficult pass to any function, but it works if we box it. */       \
    struct boxedStr { constexpr static const char* value() { return tmp::Basename(__FILE__); } }; \
    auto basename = tmp::Strcpy<boxedStr>(tmp::MakeIndices<tmp::Strlen(boxedStr{}.value())+1>{}); \
    ::core::Logger::instance().log(LEVEL, basename, __LINE__, FORMAT, ##__VA_ARGS__);             \
  }
#define GAPID_FATAL(FORMAT, ...) GAPID_LOG(LOG_LEVEL_FATAL, FORMAT, ##__VA_ARGS__)
#define GAPID_ERROR(FORMAT, ...) GAPID_LOG(LOG_LEVEL_ERROR, FORMAT, ##__VA_ARGS__)
#define GAPID_WARNING(FORMAT, ...) GAPID_LOG(LOG_LEVEL_WARNING, FORMAT, ##__VA_ARGS__)
#define GAPID_INFO(FORMAT, ...) GAPID_LOG(LOG_LEVEL_INFO, FORMAT, ##__VA_ARGS__)
#define GAPID_DEBUG(FORMAT, ...) GAPID_LOG(LOG_LEVEL_DEBUG, FORMAT, ##__VA_ARGS__)
#define GAPID_VERBOSE(FORMAT, ...) GAPID_LOG(LOG_LEVEL_VERBOSE, FORMAT, ##__VA_ARGS__)

#endif  // TARGET_OS == GAPID_OS_ANDROID

// Define the required log macros based on the specified log level

#endif  // CORE_LOG_H
