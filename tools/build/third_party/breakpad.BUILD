# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@//tools/build:rules.bzl", "mm_library", "cc_copts")

POSIX_SRCS = [
    "src/client/minidump_file_writer.cc",
    "src/common/convert_UTF.c",
    "src/common/convert_UTF.h",  # needs to be here, because of an unqualified import
    "src/common/md5.cc",
    "src/common/simple_string_dictionary.cc",
    "src/common/string_conversion.cc",
]

LINUX_SRCS = POSIX_SRCS + [
    "src/client/linux/crash_generation/crash_generation_client.cc",
    "src/client/linux/dump_writer_common/thread_info.cc",
    "src/client/linux/dump_writer_common/ucontext_reader.cc",
    "src/client/linux/handler/exception_handler.cc",
    "src/client/linux/handler/minidump_descriptor.cc",
    "src/client/linux/log/log.cc",
    "src/client/linux/microdump_writer/microdump_writer.cc",
    "src/client/linux/minidump_writer/linux_dumper.cc",
    "src/client/linux/minidump_writer/linux_ptrace_dumper.cc",
    "src/client/linux/minidump_writer/minidump_writer.cc",
    "src/common/linux/elfutils.cc",
    "src/common/linux/file_id.cc",
    "src/common/linux/guid_creator.cc",
    "src/common/linux/linux_libc_support.cc",
    "src/common/linux/memory_mapped_file.cc",
    "src/common/linux/safe_readlink.cc",
]

cc_library(
    name = "breakpad",
    srcs = select({
        "@//tools/build:linux": LINUX_SRCS,
        "@//tools/build:darwin": POSIX_SRCS + [
            "src/common/mac/arch_utilities.cc",
            "src/common/mac/bootstrap_compat.cc",
            "src/common/mac/file_id.cc",
            "src/common/mac/launch_reporter.cc",
            "src/common/mac/macho_id.cc",
            "src/common/mac/macho_utilities.cc",
            "src/common/mac/macho_walker.cc",
            "src/common/mac/string_utilities.cc",
            "src/client/mac/crash_generation/crash_generation_client.cc",
            "src/client/mac/crash_generation/crash_generation_server.cc",
            "src/client/mac/handler/breakpad_nlist_64.h",
            "src/client/mac/handler/breakpad_nlist_64.cc",
            "src/client/mac/handler/dynamic_images.cc",
            "src/client/mac/handler/exception_handler.cc",
            "src/client/mac/handler/minidump_generator.cc",
            "src/client/mac/handler/protected_memory_allocator.h",
            "src/client/mac/handler/protected_memory_allocator.cc",
        ],
        "@//tools/build:windows": [
            "src/client/windows/crash_generation/client_info.cc",
            "src/client/windows/crash_generation/crash_generation_client.cc",
            "src/client/windows/crash_generation/crash_generation_server.cc",
            "src/client/windows/crash_generation/minidump_generator.cc",
            "src/client/windows/handler/exception_handler.cc",
            "src/client/windows/sender/crash_report_sender.cc",
            "src/common/windows/guid_string.cc",
            "src/common/windows/http_upload.cc",
            "src/common/windows/string_utils.cc",
        ],
        # Android.
        "//conditions:default": LINUX_SRCS + [
            "src/common/android/breakpad_getcontext.S",
        ],
    }),
    hdrs = glob([
        "src/client/*.h",
        "src/common/*.h",
        "src/google_breakpad/**/*.h",
    ]) + select({
        "@//tools/build:linux": glob([
            "src/client/linux/**/*.h",
            "src/common/linux/**/*.h",
        ]),
        "@//tools/build:darwin": glob([
            "src/client/mac/**/*.h",
            "src/common/mac/**/*.h",
        ]) + ["src/common/linux/linux_libc_support.h"],  # no joke
        "@//tools/build:windows": glob([
            "src/client/windows/**/*.h",
            "src/common/windows/**/*.h",
        ]),
        # Android.
        "//conditions:default": glob([
            "src/common/android/*.h",
            "src/client/linux/**/*.h",
            "src/common/linux/**/*.h",
        ]),
    }),
    copts = cc_copts() + select({
        "@//tools/build:linux": [],
        "@//tools/build:darwin": [],
        "@//tools/build:windows": [
            "-D_UNICODE",
            "-DUNICODE",
        ],
        # Android.
        "//conditions:default": ["-D__STDC_FORMAT_MACROS"],
    }),
    linkopts = select({
        "@//tools/build:linux": ["-lpthread"],
        "@//tools/build:darwin": [],
        "@//tools/build:windows": ["-lwininet"],
        # Android.
        "//conditions:default": [],
    }),
    strip_include_prefix = "src",
    visibility = ["//visibility:public"],
    deps = select({
        "@//tools/build:linux": ["@lss"],
        "@//tools/build:darwin": [":breakpad_darwin"],
        "@//tools/build:windows": [],
        # Android.
        "//conditions:default": [
            ":breakpad_android_includes",
            "@lss",
        ],
    }),
)

mm_library(
    name = "breakpad_darwin",
    srcs = [
        "src/client/mac/Framework/Breakpad.h",
        "src/client/mac/Framework/Breakpad.mm",
        "src/client/mac/Framework/OnDemandServer.h",
        "src/client/mac/Framework/OnDemandServer.mm",
        "src/client/mac/crash_generation/ConfigFile.h",
        "src/client/mac/crash_generation/ConfigFile.mm",
        "src/common/mac/MachIPC.h",
        "src/common/mac/MachIPC.mm",
    ],
    hdrs = glob([
        "src/common/*.h",
        "src/common/mac/**/*.h",
        "src/client/*.h",
        "src/client/apple/**/*.h",
        "src/client/mac/**/*.h",
        "src/google_breakpad/**/*.h",
    ]),
    copts = cc_copts(),
    strip_include_prefix = "src",
    deps = [":breakpad_darwin_defines"],
)

# Needed because the BreakpadDefines is also included without a full path.
cc_library(
    name = "breakpad_darwin_defines",
    hdrs = ["src/client/apple/Framework/BreakpadDefines.h"],
    copts = cc_copts(),
    strip_include_prefix = "src/client/apple/Framework",
)

# Needed because breakpad "fakes" some system includes in src/common/android/include.
cc_library(
    name = "breakpad_android_includes",
    hdrs = glob(["src/common/android/include/**/*.h"]),
    copts = cc_copts(),
    strip_include_prefix = "src/common/android/include",
)
