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

#ifndef CORE_ANDROID_TRACE_H
#define CORE_ANDROID_TRACE_H

#include <cstdint>

namespace core {

class TraceScope {
public:
    TraceScope(const char* name);

    ~TraceScope();
};

void TraceInt(const char* name, std::int32_t value);

}  // namespace core

#define GAPID_TRACE_CALL() core::TraceScope __gapidtrace(__FUNCTION__)

#define GAPID_TRACE_NAME(name) core::TraceScope __gapidtrace(name)

#define GAPID_TRACE_INT(name, value) core::TraceInt(name, value)

#endif  // CORE_ANDROID_TRACE_H
