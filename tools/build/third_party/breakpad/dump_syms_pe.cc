/*
 * Copyright (C) 2018 Google Inc.
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

// Windows utility to dump the line number data from a .exe built
// with debug info with MinGW's gcc to the minidump text-based
// format for breakpad.

// Much of this is inspired from Breakpad's
//    src/common/linux/dump_symbols.cc,
//    src/common/windows/pdb_source_line_writer.cc
// which have the following license:
//
// Copyright (c) 2006, Google Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

#include <cstdint>
#include <cstring>
#include <iomanip>
#include <iostream>
#include <fstream>
#include <sstream>
#include <string>
#include <windows.h>
#include <DbgHelp.h>
#include <Imagehlp.h>

#include "common/dwarf/bytereader-inl.h"
#include "common/dwarf/dwarf2diehandler.h"
#include "common/dwarf_cfi_to_module.h"
#include "common/dwarf_cu_to_module.h"
#include "common/dwarf_line_to_module.h"
#include "common/linux/dump_symbols.h"
#include "common/module.h"
#include "common/path_helper.h"

#define SYMBOL_SIZE 18

namespace {

// See http://www.debuginfo.com/articles/debuginfomatch.html#pdbfiles
typedef struct DEBUG_DATA {
  DWORD signature;
  struct {
    DWORD a;
    WORD b;
    WORD c;
    BYTE d[8];
  } uuid;
  DWORD age;
} DEBUG_DATA;

// Prints the given messsage, followed by the error message of GetLastError().
void PrintError(const std::string& msg, ...) {
  va_list args;
  va_start(args, msg);
  vfprintf(stderr, msg.c_str(), args);
  va_end(args);

  DWORD error = GetLastError();
  VOID* msgBuf;
  FormatMessage(FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
      nullptr, error, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), (LPTSTR)&msgBuf, 0, nullptr);
  fprintf(stderr, ": (error code %x) %s", error, msgBuf);
  LocalFree(msgBuf);
}

// This is a straight copy of the samenamed class in the Linux symbol dumper.
class DumperLineToModule: public google_breakpad::DwarfCUToModule::LineToModuleHandler {
 public:
  explicit DumperLineToModule(dwarf2reader::ByteReader *byte_reader)
      : byte_reader_(byte_reader) { }

  void StartCompilationUnit(const string& compilation_dir) {
    compilation_dir_ = compilation_dir;
  }

  void ReadProgram(const uint8_t *program, uint64 length,
                   google_breakpad::Module* module, std::vector<google_breakpad::Module::Line>* lines) {
    google_breakpad::DwarfLineToModule handler(module, compilation_dir_, lines);
    dwarf2reader::LineInfo parser(program, length, byte_reader_, &handler);
    parser.Start();
  }

 private:
  string compilation_dir_;
  dwarf2reader::ByteReader *byte_reader_;
};

class PEFile {
 public:
  explicit PEFile() : loaded_(false) {
  }

  ~PEFile() {
    if (loaded_) {
      UnMapAndLoad(&image_);
    }
  }

  // Loads the given executable file and initializes this PEFile.
  bool Load(const std::string& in) {
    if (!MapAndLoad(in.c_str(), nullptr /* search path */, &image_, false /* .exe */, true /* read only */)) {
      PrintError("Failed to load %s", in.c_str());
      return false;
    }
    loaded_ = true;

    IMAGE_FILE_HEADER* hdr = &image_.FileHeader->FileHeader;
    IMAGE_OPTIONAL_HEADER* opt = &image_.FileHeader->OptionalHeader;

    if (hdr->Machine != IMAGE_FILE_MACHINE_AMD64) {
      std::cerr << "Unsupported machine type: %x" << std::hex << hdr->Machine << std::endl;
      return false;
    }

    // Init the string table fields.
    if (hdr->NumberOfSymbols && hdr->PointerToSymbolTable) {
      DWORD stringTableStart = hdr->PointerToSymbolTable + hdr->NumberOfSymbols * SYMBOL_SIZE;
      void* stringTable = reinterpret_cast<void*>(image_.MappedAddress + stringTableStart);
      DWORD stringTableSize = *reinterpret_cast<DWORD*>(stringTable);
      if (stringTableSize > 4) {
        strings_.size = stringTableSize;
        strings_.data = reinterpret_cast<const char*>(stringTable);
      } else {
        strings_.size = 0;
      }
    } else {
      strings_.size = 0;
    }

    // Look for the debug directory.
    if (opt->NumberOfRvaAndSizes <= IMAGE_DIRECTORY_ENTRY_DEBUG) {
      std::cerr << "No debug directory: not enough directory entries" << std::endl;
      return false;
    }
    if (!opt->DataDirectory[IMAGE_DIRECTORY_ENTRY_DEBUG].VirtualAddress) {
      std::cerr << "No debug directory: address 0" << std::endl;
      return false;
    }

    ULONG debugSize;
    VOID* debugDirPtr = ImageDirectoryEntryToDataEx(image_.MappedAddress, false, IMAGE_DIRECTORY_ENTRY_DEBUG, &debugSize, nullptr);
    if (!debugDirPtr) {
      PrintError("Failed to load debug directory entry");
      return false;
    }
    if (debugSize < sizeof(IMAGE_DEBUG_DIRECTORY)) {
      std::cerr << "Debug directory too small: " << debugSize << " < " << sizeof(IMAGE_DEBUG_DIRECTORY) << std::endl;;
      return false;
    }
    IMAGE_DEBUG_DIRECTORY* debugDir = reinterpret_cast<IMAGE_DEBUG_DIRECTORY*>(debugDirPtr);

    // Look for the codeview debut information.
    if (debugDir->Type != IMAGE_DEBUG_TYPE_CODEVIEW) {
      std::cerr << "Unsupported debug data type: " << debugDir->Type << " != " << IMAGE_DEBUG_TYPE_CODEVIEW << std::endl;
      return false;
    }
    if (!debugDir->PointerToRawData) {
      std::cerr << "No debug data: address 0" << std::endl;
      return false;
    }
    if (debugDir->SizeOfData < sizeof(DEBUG_DATA)) {
      std::cerr << "Debug data too small: " << debugDir->SizeOfData << " < " << sizeof(DEBUG_DATA) << std::endl;
      return false;
    }
    DEBUG_DATA* debugData = reinterpret_cast<DEBUG_DATA*>(image_.MappedAddress + debugDir->PointerToRawData);

    // Build the debug identifier from the debug UUID and age.
    std::stringstream debugIdentifier;
    debugIdentifier << std::setfill('0') << std::hex << std::uppercase
        << std::setw(8) << debugData->uuid.a
        << std::setw(4) << debugData->uuid.b
        << std::setw(4) << debugData->uuid.c
        << std::setw(2) << int(debugData->uuid.d[0])
        << std::setw(2) << int(debugData->uuid.d[1])
        << std::setw(2) << int(debugData->uuid.d[2])
        << std::setw(2) << int(debugData->uuid.d[3])
        << std::setw(2) << int(debugData->uuid.d[4])
        << std::setw(2) << int(debugData->uuid.d[5])
        << std::setw(2) << int(debugData->uuid.d[6])
        << std::setw(2) << int(debugData->uuid.d[7])
        << std::setw(0) << debugData->age;
    debugId = debugIdentifier.str();

    // Build the code identifier from the timestamp and size.
    std::stringstream codeIdentifier;
    codeIdentifier << std::setfill('0') << std::hex << std::uppercase
        << std::setw(8) << hdr->TimeDateStamp
        << std::setw(0) << opt->SizeOfImage;
    codeId = codeIdentifier.str();

    return true;
  }

  // Loads the DWARF .debug_info section and adds the symbol/line number data to the given module.
  // This is similiar to the samenamed method in the Linux symbol dumper.
  void LoadDwarf(const std::string& name, google_breakpad::Module* module) const {
    dwarf2reader::ByteReader byte_reader(dwarf2reader::ENDIANNESS_LITTLE);
    google_breakpad::DwarfCUToModule::FileContext file_context(name, module, true);
    for (ULONG i = 0; i < image_.NumberOfSections; i++) {
      IMAGE_SECTION_HEADER* section = &image_.Sections[i];
      void* addr = ImageRvaToVa(image_.FileHeader, image_.MappedAddress, section->VirtualAddress, nullptr);
      if (addr) {
        file_context.AddSectionToSectionMap(GetSectionName(image_.Sections[i]),
            reinterpret_cast<const uint8_t*>(addr), section->Misc.VirtualSize);
      }
    }

    DumperLineToModule line_to_module(&byte_reader);
    dwarf2reader::SectionMap::const_iterator debug_info_entry = file_context.section_map().find(".debug_info");
    assert(debug_info_entry != file_context.section_map().end());
    const std::pair<const uint8_t *, uint64>& debug_info_section = debug_info_entry->second;
    assert(debug_info_section.first);
    const uint64 debug_info_length = debug_info_section.second;
    for (uint64 offset = 0; offset < debug_info_length;) {
      google_breakpad::DwarfCUToModule::WarningReporter reporter(name, offset);
      google_breakpad::DwarfCUToModule root_handler(&file_context, &line_to_module, &reporter);
      dwarf2reader::DIEDispatcher die_dispatcher(&root_handler);
      dwarf2reader::CompilationUnit reader(name, file_context.section_map(), offset, &byte_reader, &die_dispatcher);
      offset += reader.Start();
    }
  }

  // Loads the stack unwinding information from .debug_frame and adds the data to the given module.
  // This is similiar to the samenamed method in the Linux symbol dumper.
  void LoadDwarfCFI(const std::string& name, google_breakpad::Module* module) const {
    IMAGE_SECTION_HEADER* debugFrame = FindSectionByName(".debug_frame");
    if (debugFrame != nullptr) {
      std::vector<std::string> register_names = google_breakpad::DwarfCFIToModule::RegisterNames::X86_64();
      dwarf2reader::ByteReader byte_reader(dwarf2reader::ENDIANNESS_LITTLE);
      uint8_t* cfi = reinterpret_cast<uint8_t*>(
        ImageRvaToVa(image_.FileHeader, image_.MappedAddress, debugFrame->VirtualAddress, nullptr));
      google_breakpad::DwarfCFIToModule::Reporter module_reporter(name, ".debug_frame");
      google_breakpad::DwarfCFIToModule handler(module, register_names, &module_reporter);
      byte_reader.SetAddressSize(8);
      byte_reader.SetCFIDataBase(debugFrame->VirtualAddress, cfi);
      dwarf2reader::CallFrameInfo::Reporter dwarf_reporter(name, ".debug_frame");
      dwarf2reader::CallFrameInfo parser(cfi, debugFrame->Misc.VirtualSize, &byte_reader, &handler, &dwarf_reporter, false);
      parser.Start();
    }
  }

 private:
  // Returns the name of the given section, possibly looking it up from the string table,
  // if the name is longer than 8 characters.
  std::string GetSectionName(const IMAGE_SECTION_HEADER& section) const {
    std::string name = std::string(reinterpret_cast<const char*>(section.Name), 0, 8);
    if (name[0] != '/') {
      return name;
    }

    int off = std::atoi(&name[1]);
    if (off > strings_.size) {
      return "";
    }
    return reinterpret_cast<const char*>(strings_.data + off);
  }

  // Returns the header of the section with the given name, or nullptr, if no such section
  // exists in the file.
  IMAGE_SECTION_HEADER* FindSectionByName(const std::string& name) const {
    for (ULONG i = 0; i < image_.NumberOfSections; i++) {
      if (GetSectionName(image_.Sections[i]) == name) {
        return &image_.Sections[i];
      }
    }
    return nullptr;
  }

  LOADED_IMAGE image_;
  bool loaded_;
  struct {
    const char* data;
    DWORD size;
  } strings_;

 public:
  std::string debugId;
  std::string codeId;
};

} // namespace

int main(int argc, char **argv) {
  if (argc != 2) {
    std::cerr << "Usage: " << argv[0] << " <file.exe>" << std::endl << std::endl
      << "Dumps crashpad symbol information from the given PE executable to stdout." << std::endl;
    return 1;
  }

  PEFile file;
  if (!file.Load(argv[1])) {
    return 1;
  }

  google_breakpad::Module module(google_breakpad::BaseName(argv[1]), "windows", "x86_64", file.debugId, file.codeId);
  file.LoadDwarf(argv[1], &module);
  file.LoadDwarfCFI(argv[1], &module);

  google_breakpad::DumpOptions options(ALL_SYMBOL_DATA, true);
  bool result = module.Write(std::cout, options.symbol_data);
  return result ? 0 : 1;
}
