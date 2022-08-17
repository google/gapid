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

load("@gapid//tools/build:rules.bzl", "cc_copts", "mm_library")

LIB_POSIX = [
    "src/client/minidump_file_writer.cc",
    "src/common/convert_UTF.c",
    "src/common/convert_UTF.h",  # needs to be here, because of an unqualified import
    "src/common/md5.cc",
    "src/common/simple_string_dictionary.cc",
    "src/common/string_conversion.cc",
]

LIB_LINUX = LIB_POSIX + [
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

LIB_MACOS = LIB_POSIX + [
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
    "src/client/mac/handler/breakpad_nlist_64.h",  # unqualified import
    "src/client/mac/handler/breakpad_nlist_64.cc",
    "src/client/mac/handler/dynamic_images.cc",
    "src/client/mac/handler/exception_handler.cc",
    "src/client/mac/handler/minidump_generator.cc",
    "src/client/mac/handler/protected_memory_allocator.h",  # unqualified import
    "src/client/mac/handler/protected_memory_allocator.cc",
]

LIB_WINDOWS = [
    "src/client/windows/crash_generation/client_info.cc",
    "src/client/windows/crash_generation/crash_generation_client.cc",
    "src/client/windows/crash_generation/crash_generation_server.cc",
    "src/client/windows/crash_generation/minidump_generator.cc",
    "src/client/windows/handler/exception_handler.cc",
    "src/client/windows/sender/crash_report_sender.cc",
    "src/common/windows/guid_string.cc",
    "src/common/windows/http_upload.cc",
    "src/common/windows/string_utils.cc",
]

cc_library(
    name = "breakpad",
    srcs = select({
        "@gapid//tools/build:linux": LIB_LINUX,
        "@gapid//tools/build:darwin": LIB_MACOS,
        "@gapid//tools/build:windows": LIB_WINDOWS,
        # Android.
        "//conditions:default": LIB_LINUX,
    }),
    hdrs = glob(["src/**/*.h"]),
    copts = cc_copts() + select({
        "@gapid//tools/build:linux": [
            "-Wno-maybe-uninitialized",
            "-Wno-deprecated",
            "-Wno-array-bounds",
        ],
        "@gapid//tools/build:darwin": [],
        "@gapid//tools/build:windows": [
            "-D_UNICODE",
            "-DUNICODE",
            "-Wno-conversion-null",
        ],
        # Android.
        "//conditions:default": ["-D__STDC_FORMAT_MACROS"],
    }),
    linkopts = select({
        "@gapid//tools/build:linux": ["-lpthread"],
        "@gapid//tools/build:darwin": [],
        "@gapid//tools/build:windows": ["-lwininet"],
        # Android.
        "//conditions:default": [],
    }),
    strip_include_prefix = "src",
    visibility = ["//visibility:public"],
    deps = select({
        "@gapid//tools/build:linux": ["@lss"],
        "@gapid//tools/build:darwin": [":breakpad_darwin"],
        "@gapid//tools/build:windows": [],
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
    copts = cc_copts() + ["-Wno-deprecated-declarations"],
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

DUMP_SYMS_POSIX = [
    "src/common/dwarf/bytereader.cc",
    "src/common/dwarf/dwarf2diehandler.cc",
    "src/common/dwarf/dwarf2reader.cc",
    "src/common/dwarf/elf_reader.cc",
    "src/common/dwarf/elf_reader.h",  # unqualified import
    "src/common/dwarf_cfi_to_module.cc",
    "src/common/dwarf_cu_to_module.cc",
    "src/common/dwarf_line_to_module.cc",
    "src/common/language.cc",
    "src/common/module.cc",
    "src/common/path_helper.cc",
    "src/common/stabs_reader.cc",
    "src/common/stabs_to_module.cc",
]

DUMP_SYMS_LINUX = DUMP_SYMS_POSIX + [
    "src/common/linux/crc32.cc",
    "src/common/linux/dump_symbols.cc",
    "src/common/linux/elf_symbols_to_module.cc",
    "src/common/linux/elfutils.cc",
    "src/common/linux/file_id.cc",
    "src/common/linux/linux_libc_support.cc",
    "src/common/linux/memory_mapped_file.cc",
    "src/common/linux/safe_readlink.cc",
    "src/tools/linux/dump_syms/dump_syms.cc",
]

DUMP_SYMS_MACOS = DUMP_SYMS_POSIX + [
    "src/common/mac/arch_utilities.cc",
    "src/common/mac/dump_syms.cc",
    "src/common/mac/file_id.cc",
    "src/common/mac/macho_id.cc",
    "src/common/mac/macho_reader.cc",
    "src/common/mac/macho_utilities.cc",
    "src/common/mac/macho_walker.cc",
    "src/common/md5.cc",
    "src/tools/mac/dump_syms/dump_syms_tool.cc",
]

DUMP_SYMS_WINDOWS = [
    "src/tools/windows/dump_syms/dump_syms_pe.cc",
    "src/common/module.cc",
    "src/common/path_helper.cc",
    "src/common/dwarf/bytereader.cc",
    "src/common/dwarf/dwarf2diehandler.cc",
    "src/common/dwarf/dwarf2reader.cc",
    "src/common/dwarf_cfi_to_module.cc",
    "src/common/dwarf_cu_to_module.cc",
    "src/common/dwarf_line_to_module.cc",
    "src/common/language.cc",
]

cc_library(
    name = "dump_syms-lib",
    srcs = select({
        "@gapid//tools/build:linux": DUMP_SYMS_LINUX,
        "@gapid//tools/build:darwin": DUMP_SYMS_MACOS,
        "@gapid//tools/build:windows": DUMP_SYMS_WINDOWS,
    }),
    hdrs = glob(["src/**/*.h"]),
    copts = cc_copts() + [
        "-DN_UNDF=0x0",
    ] + select({
        "@gapid//tools/build:windows": ["-DNO_STABS_SUPPORT"],
        "@gapid//tools/build:linux": ["-Wno-maybe-uninitialized"],
        "//conditions:default": [],
    }),
    linkopts = select({
        "@gapid//tools/build:windows": ["-limagehlp"],
        "//conditions:default": [],
    }),
    strip_include_prefix = "src",
    deps = select({
        "@gapid//tools/build:linux": ["@lss"],
        "//conditions:default": [],
    }),
)

cc_binary(
    name = "dump_syms",
    visibility = ["//visibility:public"],
    deps = [":dump_syms-lib"],
)
