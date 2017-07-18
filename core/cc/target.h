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

#ifndef CORE_TARGET_H
#define CORE_TARGET_H

#define GAPID_OS_LINUX   1
#define GAPID_OS_OSX     2
#define GAPID_OS_WINDOWS 3
#define GAPID_OS_ANDROID 4

#if defined(TARGET_OS_LINUX)
#   define TARGET_OS GAPID_OS_LINUX
#   define STDCALL
#   define EXPORT __attribute__ ((visibility ("default")))
#   define PATH_DELIMITER '/'
#   define PATH_DELIMITER_STR "/"
#endif

#if defined(TARGET_OS_OSX)
#   define TARGET_OS GAPID_OS_OSX
#   define STDCALL
#   define EXPORT __attribute__ ((visibility ("default")))
#   define PATH_DELIMITER '/'
#   define PATH_DELIMITER_STR "/"
#   include <stdint.h>
    using size_val = uint64_t;
#else  // defined(TARGET_OS_OSX)
#   include <stddef.h>
    using size_val = size_t;
#endif

#if defined(TARGET_OS_ANDROID)
#   define TARGET_OS GAPID_OS_ANDROID
#   define STDCALL
#   define EXPORT __attribute__ ((visibility ("default")))
#   define PATH_DELIMITER '/'
#   define PATH_DELIMITER_STR "/"
#endif

#if defined(TARGET_OS_WINDOWS)
#   define TARGET_OS GAPID_OS_WINDOWS
#   define STDCALL __stdcall
#   define EXPORT __declspec(dllexport)
#   define PATH_DELIMITER '\\'
#   define PATH_DELIMITER_STR "\\"
#endif

#ifndef TARGET_OS
#   error "OS not defined correctly."
#   error "Exactly one of the following macro have to be defined:" \
           "TARGET_OS_LINUX, TARGET_OS_OSX, TARGET_OS_WINDOWS, TARGET_OS_ANDROID"
#endif

#ifdef _MSC_VER // MSVC
#   define ftruncate _chsize
#ifdef __GNUC__
// MSYS needs this, MSVC will complain
// if we #define snprintf
#   define snprintf _snprintf
#endif
#   define alignof __alignof
#   define _ALLOW_KEYWORD_MACROS 1
#   define LIKELY(expr) expr
#   define UNLIKELY(expr) expr
#else
#   define LIKELY(expr) __builtin_expect(expr, true)
#   define UNLIKELY(expr) __builtin_expect(expr, false)
#endif // _MSC_VER

#endif  // CORE_TARGET_H
