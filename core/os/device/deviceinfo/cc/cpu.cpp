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

#include "query.h"

#include "core/cc/target.h"

#if (defined(__x86_64) || defined(__i386)) && (TARGET_OS != GAPID_OS_ANDROID)
#if !defined(_MSC_VER) || defined(__GNUC__)
#include <cpuid.h>

namespace query {

const char* cpuName() {
    static union {
        uint32_t reg[12];
        char str[49];
    };
    if (__get_cpuid(0x80000002, &reg[0], &reg[1], &reg[2],  &reg[3]) &&
        __get_cpuid(0x80000003, &reg[4], &reg[5], &reg[6],  &reg[7]) &&
        __get_cpuid(0x80000004, &reg[8], &reg[9], &reg[10], &reg[11])) {

        return str;
    }
    return "";
}

const char* cpuVendor() {
    static union {
        uint32_t reg[3];
        char str[13];
    };
    uint32_t eax = 0;
    if (__get_cpuid(0, &eax, &reg[0], &reg[2], &reg[1])) {
        return str;
    }
    return "";
}

device::Architecture cpuArchitecture() {
    return device::X86_64;
}

}  // namespace query
#else // !defined(_MSC_VER) || defined(__GNUC__)
// If we are using MSVC (rather than MSYS) we cannot use __get_cpuid
namespace query {

const char* cpuName() {
    return "";
}

const char* cpuVendor() {
    return "";
}

device::Architecture cpuArchitecture() {
#ifdef _WIN64
    return device::X86_64;
#elif defined _WIN32
    return device::X86;
#endif
}
}
#endif
#endif