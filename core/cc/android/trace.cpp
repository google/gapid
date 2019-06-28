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

#include "trace.h"

#include <android/log.h>
#include <fcntl.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include <atomic>
#include <cstdio>
#include <mutex>

namespace core {
namespace {

constexpr size_t kBufferSize = 1024;
constexpr const char kTraceMarkerPath[] =
    "sys/kernel/debug/tracing/trace_marker";

int sTraceFD = -1;
volatile bool sInitialized = false;
std::once_flag sInitializeOnce;

void Initialize() {
  sTraceFD = open(kTraceMarkerPath, O_WRONLY);
  if (sTraceFD == -1) {
    __android_log_print(ANDROID_LOG_INFO, "TRACE",
                        "error opening trace file: %s (%d)", strerror(errno),
                        errno);
  }
  // Even if we've failed to open, we have opened the subsystem.
  sInitialized = true;
}

bool EnsureInitialized() {
  if (!sInitialized) {
    std::call_once(sInitializeOnce, Initialize);
  }

  return sTraceFD != -1;
}

}  // namespace

TraceScope::TraceScope(const char* name) {
  if (EnsureInitialized()) {
    char buffer[kBufferSize];
    size_t length = snprintf(buffer, kBufferSize, "B|%d|%s", getpid(), name);
    write(sTraceFD, buffer, length);
  }
}

TraceScope::~TraceScope() {
  if (EnsureInitialized()) {
    char value = 'E';
    write(sTraceFD, &value, sizeof(value));
  }
}

void TraceInt(const char* name, std::int32_t value) {
  if (EnsureInitialized()) {
    char buffer[kBufferSize];
    size_t length =
        snprintf(buffer, kBufferSize, "C|%d|%s|%d", getpid(), name, value);
    write(sTraceFD, buffer, length);
  }
}

}  // namespace core
