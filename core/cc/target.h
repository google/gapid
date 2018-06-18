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

#define GAPID_OS_LINUX 1
#define GAPID_OS_OSX 2
#define GAPID_OS_WINDOWS 3
#define GAPID_OS_ANDROID 4

#define LINUX_ONLY(x)
#define OSX_ONLY(x)
#define WINDOWS_ONLY(x)
#define ANDROID_ONLY(x)

#if defined(TARGET_OS_LINUX)
#define TARGET_OS GAPID_OS_LINUX
#define STDCALL
#define EXPORT __attribute__((visibility("default")))
#define PATH_DELIMITER '/'
#define PATH_DELIMITER_STR "/"
#undef LINUX_ONLY
#define LINUX_ONLY(x) x
#endif

#if defined(TARGET_OS_OSX)
#define TARGET_OS GAPID_OS_OSX
#define STDCALL
#define EXPORT __attribute__((visibility("default")))
#define PATH_DELIMITER '/'
#define PATH_DELIMITER_STR "/"
#undef OSX_ONLY
#define OSX_ONLY(x) x
#include <stdint.h>
using size_val = uint64_t;
#else  // defined(TARGET_OS_OSX)
#include <stddef.h>
using size_val = size_t;
#endif

#if defined(TARGET_OS_ANDROID)
#define TARGET_OS GAPID_OS_ANDROID
#define STDCALL
#define EXPORT __attribute__((visibility("default")))
#define PATH_DELIMITER '/'
#define PATH_DELIMITER_STR "/"
#undef ANDROID_ONLY
#define ANDROID_ONLY(x) x
#endif

#if defined(TARGET_OS_WINDOWS)
#define TARGET_OS GAPID_OS_WINDOWS
#define STDCALL __stdcall
#define EXPORT __declspec(dllexport)
#define PATH_DELIMITER '\\'
#define PATH_DELIMITER_STR "\\"
#undef WINDOWS_ONLY
#define WINDOWS_ONLY(x) x
#endif

#ifndef TARGET_OS
#error "OS not defined correctly."
#error \
    "Exactly one of the following macro have to be defined:" \
           "TARGET_OS_LINUX, TARGET_OS_OSX, TARGET_OS_WINDOWS, TARGET_OS_ANDROID"
#endif

#ifdef _MSC_VER  // MSVC
#define ftruncate _chsize
#define alignof __alignof
#define _ALLOW_KEYWORD_MACROS 1
#define LIKELY(expr) expr
#define UNLIKELY(expr) expr
#if !defined(__GNUC__)
// MSVC itself does not have ssize_t, although
// msys mingw does.
typedef long long ssize_t;
#endif  //! defined(__GNUC__)
#else   // _MSC_VER
#define LIKELY(expr) __builtin_expect(expr, true)
#define UNLIKELY(expr) __builtin_expect(expr, false)
#endif  // _MSC_VER

#endif  // CORE_TARGET_H
