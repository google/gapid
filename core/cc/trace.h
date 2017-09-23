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

#ifndef CORE_TRACE_H
#define CORE_TRACE_H

#ifdef GAPID_USE_TRACING
#if TARGET_OS == GAPID_OS_ANDROID
#define GAPID_TRACE_MACROS_DEFINED
#include "android/trace.h"
#endif  // TARGET_OS == GAPID_OS_ANDROID
#endif  // GAPID_USE_TRACING


#ifndef GAPID_TRACE_MACROS_DEFINED
#define GAPID_TRACE_CALL()
#define GAPID_TRACE_NAME(name)
#define GAPID_TRACE_INT(name)
#define GAPID_TRACE_ENABLED() false
#endif  // GAPID_TRACE_MACROS_DEFINED

#endif  // CORE_TRACE_H
