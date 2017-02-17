// Copyright (c) 2016 Google Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

#include "name_mapper.h"

#include <algorithm>
#include <iterator>
#include <sstream>
#include <string>
#include <unordered_map>
#include <unordered_set>

#include "spirv-tools/libspirv.h"
#include "spirv/1.1/spirv.h"

namespace {

// Converts a uint32_t to its string decimal representation.
std::string to_string(uint32_t id) {
  // Use stringstream, since some versions of Android compilers lack
  // std::to_string.
  std::stringstream os;
  os << id;
  return os.str();
}

}  // anonymous namespace

namespace libspirv {

NameMapper GetTrivialNameMapper() { return to_string; }

FriendlyNameMapper::FriendlyNameMapper(const spv_const_context context,
                                       const uint32_t* code,
                                       const size_t wordCount)
    : grammar_(libspirv::AssemblyGrammar(context)) {
  spv_diagnostic diag = nullptr;
  // We don't care if the parse fails.
  spvBinaryParse(context, this, code, wordCount, nullptr,
                 ParseInstructionForwarder, &diag);
  spvDiagnosticDestroy(diag);
}

std::string FriendlyNameMapper::NameForId(uint32_t id) {
  auto iter = name_for_id_.find(id);
  if (iter == name_for_id_.end()) {
    // It must have been an invalid module, so just return a trivial mapping.
    // We don't care about uniqueness.
    return to_string(id);
  } else {
    return iter->second;
  }
}

std::string FriendlyNameMapper::Sanitize(const std::string& suggested_name) {
  if (suggested_name.empty()) return "_";
  // Otherwise, replace invalid characters by '_'.
  std::string result;
  std::string valid =
      "abcdefghijklmnopqrstuvwxyz"
      "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
      "_0123456789";
  std::transform(suggested_name.begin(), suggested_name.end(),
                 std::back_inserter(result), [&valid](const char c) {
                   return (std::string::npos == valid.find(c)) ? '_' : c;
                 });
  return result;
}

void FriendlyNameMapper::SaveName(uint32_t id,
                                  const std::string& suggested_name) {
  if (name_for_id_.find(id) != name_for_id_.end()) return;

  const std::string sanitized_suggested_name = Sanitize(suggested_name);
  std::string name = sanitized_suggested_name;
  auto inserted = used_names_.insert(name);
  if (!inserted.second) {
    const std::string base_name = sanitized_suggested_name + "_";
    for (uint32_t index = 0; !inserted.second; ++index) {
      name = base_name + to_string(index);
      inserted = used_names_.insert(name);
    }
  }
  name_for_id_[id] = name;
}

spv_result_t FriendlyNameMapper::ParseInstruction(
    const spv_parsed_instruction_t& inst) {
  const auto result_id = inst.result_id;
  switch (inst.opcode) {
    case SpvOpName:
      SaveName(inst.words[1], reinterpret_cast<const char*>(inst.words + 2));
      break;
    case SpvOpTypeVoid:
      SaveName(result_id, "void");
      break;
    case SpvOpTypeBool:
      SaveName(result_id, "bool");
      break;
    case SpvOpTypeInt: {
      std::string signedness;
      std::string root;
      const auto bit_width = inst.words[2];
      switch (bit_width) {
        case 8:
          root = "char";
          break;
        case 16:
          root = "short";
          break;
        case 32:
          root = "int";
          break;
        case 64:
          root = "long";
          break;
        default:
          root = to_string(bit_width);
          signedness = "i";
          break;
      }
      if (0 == inst.words[3]) signedness = "u";
      SaveName(result_id, signedness + root);
    } break;
    case SpvOpTypeFloat: {
      const auto bit_width = inst.words[2];
      switch (bit_width) {
        case 16:
          SaveName(result_id, "half");
          break;
        case 32:
          SaveName(result_id, "float");
          break;
        case 64:
          SaveName(result_id, "double");
          break;
        default:
          SaveName(result_id, std::string("fp") + to_string(bit_width));
          break;
      }
    } break;
    case SpvOpTypeVector:
      SaveName(result_id, std::string("v") + to_string(inst.words[3]) +
                              NameForId(inst.words[2]));
      break;
    case SpvOpTypeMatrix:
      SaveName(result_id, std::string("mat") + to_string(inst.words[3]) +
                              NameForId(inst.words[2]));
      break;
    case SpvOpTypeArray:
      SaveName(result_id, std::string("_arr_") + NameForId(inst.words[2]) +
                              "_" + NameForId(inst.words[3]));
      break;
    case SpvOpTypeRuntimeArray:
      SaveName(result_id,
               std::string("_runtimearr_") + NameForId(inst.words[2]));
      break;
    case SpvOpTypePointer:
      SaveName(result_id, std::string("_ptr_") +
                              NameForEnumOperand(SPV_OPERAND_TYPE_STORAGE_CLASS,
                                                 inst.words[2]) +
                              "_" + NameForId(inst.words[3]));
      break;
    case SpvOpTypePipe:
      SaveName(result_id,
               std::string("Pipe") +
                   NameForEnumOperand(SPV_OPERAND_TYPE_ACCESS_QUALIFIER,
                                      inst.words[2]));
      break;
    case SpvOpTypeEvent:
      SaveName(result_id, "Event");
      break;
    case SpvOpTypeDeviceEvent:
      SaveName(result_id, "DeviceEvent");
      break;
    case SpvOpTypeReserveId:
      SaveName(result_id, "ReserveId");
      break;
    case SpvOpTypeQueue:
      SaveName(result_id, "Queue");
      break;
    case SpvOpTypeOpaque:
      SaveName(result_id,
               std::string("Opaque_") +
                   Sanitize(reinterpret_cast<const char*>(inst.words + 2)));
      break;
    case SpvOpTypePipeStorage:
      SaveName(result_id, "PipeStorage");
      break;
    case SpvOpTypeNamedBarrier:
      SaveName(result_id, "NamedBarrier");
      break;
    case SpvOpTypeStruct:
      // Structs are mapped rather simplisitically. Just indicate that they
      // are a struct and then give the raw Id number.
      SaveName(result_id, std::string("_struct_") + to_string(result_id));
      break;
    default:
      // If this instruction otherwise defines an Id, then save a mapping for
      // it.  This is needed to ensure uniqueness in there is an OpName with
      // string something like "1" that might collide with this result_id.
      // We should only do this if a name hasn't already been registered by some
      // previous forward reference.
      if (result_id && name_for_id_.find(result_id) == name_for_id_.end())
        SaveName(result_id, to_string(result_id));
      break;
  }
  return SPV_SUCCESS;
}

std::string FriendlyNameMapper::NameForEnumOperand(spv_operand_type_t type,
                                                   uint32_t word) {
  spv_operand_desc desc = nullptr;
  if (SPV_SUCCESS == grammar_.lookupOperand(type, word, &desc)) {
    return desc->name;
  } else {
    // Invalid input.  Just give something sane.
    return std::string("StorageClass") + to_string(word);
  }
}

}  // namespace libspirv
