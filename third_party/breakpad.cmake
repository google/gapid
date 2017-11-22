# Copyright (C) 2017 Google Inc.
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

set(breakpad_dir "third_party/breakpad/src")

if(APPLE)
    set(breakpad_srcs
        "client/minidump_file_writer.cc"
        "common/convert_UTF.c"
        "common/mac/arch_utilities.cc"
        "common/mac/bootstrap_compat.cc"
        "common/mac/file_id.cc"
        "common/mac/launch_reporter.cc"
        "common/mac/macho_id.cc"
        "common/mac/macho_utilities.cc"
        "common/mac/macho_walker.cc"
        "common/mac/MachIPC.mm"
        "common/mac/string_utilities.cc"
        "common/md5.cc"
        "common/simple_string_dictionary.cc"
        "common/string_conversion.cc"
        "client/mac/crash_generation/ConfigFile.mm"
        "client/mac/crash_generation/crash_generation_client.cc"
        "client/mac/crash_generation/crash_generation_server.cc"
        "client/mac/handler/breakpad_nlist_64.cc"
        "client/mac/handler/dynamic_images.cc"
        "client/mac/handler/exception_handler.cc"
        "client/mac/handler/minidump_generator.cc"
        "client/mac/handler/protected_memory_allocator.cc"
        "client/mac/Framework/Breakpad.mm"
        "client/mac/Framework/OnDemandServer.mm"
    )
elseif(WIN32)
    set(breakpad_srcs
        "client/windows/crash_generation/client_info.cc"
        "client/windows/crash_generation/crash_generation_client.cc"
        "client/windows/crash_generation/crash_generation_server.cc"
        "client/windows/crash_generation/minidump_generator.cc"
        "client/windows/handler/exception_handler.cc"
        "client/windows/sender/crash_report_sender.cc"
        "common/windows/guid_string.cc"
        "common/windows/http_upload.cc"
        "common/windows/string_utils.cc"
    )
else()
    set(breakpad_srcs
        "client/linux/crash_generation/crash_generation_client.cc"
        "client/linux/dump_writer_common/thread_info.cc"
        "client/linux/dump_writer_common/ucontext_reader.cc"
        "client/linux/handler/exception_handler.cc"
        "client/linux/handler/minidump_descriptor.cc"
        "client/linux/log/log.cc"
        "client/linux/microdump_writer/microdump_writer.cc"
        "client/linux/minidump_writer/linux_dumper.cc"
        "client/linux/minidump_writer/linux_ptrace_dumper.cc"
        "client/linux/minidump_writer/minidump_writer.cc"
        "client/minidump_file_writer.cc"
        "common/convert_UTF.c"
        "common/linux/elfutils.cc"
        "common/linux/file_id.cc"
        "common/linux/guid_creator.cc"
        "common/linux/linux_libc_support.cc"
        "common/linux/memory_mapped_file.cc"
        "common/linux/safe_readlink.cc"
        "common/md5.cc"
        "common/string_conversion.cc"
    )
endif()

abs_list(breakpad_srcs ${breakpad_dir})

if(MSVC_GAPIR AND WIN32)
    add_cmake_step("gapir-msvc" breakpad DEPENDS ${breakpad_srcs})
endif()

if(NOT DISABLED_CXX)
    add_library(breakpad STATIC ${breakpad_srcs})
    target_include_directories(breakpad PUBLIC ${breakpad_dir})

    if(ANDROID)
        set_property(SOURCE "${breakpad_dir}/common/android/breakpad_getcontext.S" PROPERTY LANGUAGE C)
        target_sources(breakpad PRIVATE "${breakpad_dir}/common/android/breakpad_getcontext.S")
        target_include_directories(breakpad SYSTEM BEFORE PRIVATE "${breakpad_dir}/common/android/include")
        target_compile_definitions(breakpad PUBLIC "__STDC_FORMAT_MACROS")
    elseif(APPLE)
        find_library(CARBON_LIBRARY Carbon)
        find_library(FOUNDATION_LIBRARY Foundation)
        find_library(COCOA_LIBRARY Cocoa)

        set(dump_syms_srcs
            "common/dwarf/dwarf2diehandler.cc"
            "common/dwarf/elf_reader.cc"
            "common/dwarf/dwarf2reader.cc"
            "common/dwarf/bytereader.cc"
            "common/dwarf_cfi_to_module.cc"
            "common/dwarf_cu_to_module.cc"
            "common/dwarf_line_to_module.cc"
            "common/language.cc"
            "common/mac/arch_utilities.cc"
            "common/mac/dump_syms.cc"
            "common/mac/file_id.cc"
            "common/mac/macho_id.cc"
            "common/mac/macho_reader.cc"
            "common/mac/macho_utilities.cc"
            "common/mac/macho_walker.cc"
            "common/md5.cc"
            "common/module.cc"
            "common/path_helper.cc"
            "common/stabs_reader.cc"
            "common/stabs_to_module.cc"
            "tools/mac/dump_syms/dump_syms_tool.cc"
        )
        abs_list(dump_syms_srcs ${breakpad_dir})
        add_executable(dump_syms ${dump_syms_srcs})
        target_include_directories(dump_syms PUBLIC ${breakpad_dir})
        target_compile_definitions(dump_syms PRIVATE "N_UNDF=0x0")
        target_compile_options(dump_syms PRIVATE "-Wno-deprecated")
        target_link_libraries(dump_syms ${FOUNDATION_LIBRARY})

        target_include_directories(breakpad PUBLIC "${breakpad_dir}/client/apple/Framework")
        target_compile_options(breakpad PRIVATE "-Wno-deprecated")
        target_link_libraries(breakpad  ${CARBON_LIBRARY} ${FOUNDATION_LIBRARY} ${COCOA_LIBRARY})

    elseif(WIN32)
        target_compile_definitions(breakpad PRIVATE "_UNICODE" "UNICODE")
        if(NOT MSVC)
            target_compile_options(breakpad PRIVATE "-Wno-conversion-null")
        endif()
    endif()
endif()
