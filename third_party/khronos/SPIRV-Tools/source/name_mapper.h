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

#ifndef LIBSPIRV_NAME_MAPPER_H_
#define LIBSPIRV_NAME_MAPPER_H_

#include <functional>
#include <string>
#include <unordered_map>
#include <unordered_set>

#include "spirv-tools/libspirv.h"
#include "assembly_grammar.h"

namespace libspirv {

// A NameMapper maps SPIR-V Id values to names.  Each name is valid to use in
// SPIR-V assembly.  The mapping is one-to-one, i.e. no two Ids map to the same
// name.
using NameMapper = std::function<std::string(uint32_t)>;

// Returns a NameMapper which always maps an Id to its decimal representation.
NameMapper GetTrivialNameMapper();

// A FriendlyNameMapper parses a module upon construction.  If the parse is
// successful, then the NameForId method maps an Id to a friendly name
// while also satisfying the constraints on a NameMapper.
//
// The mapping is friendly in the following sense:
//  - If an Id has a debug name (via OpName), then that will be used when
//    possible.
//  - Well known scalar types map to friendly names.  For example,
//    OpTypeVoid should be %void.  Scalar types map to their names in OpenCL when
//    there is a correspondence, and otherwise as follows:
//    - unsigned integer type of n bits map to "u" followed by n
//    - signed integer type of n bits map to "i" followed by n
//    - floating point type of n bits map to "fp" followed by n
//  - Vector type names map to "v" followed by the number of components,
//    followed by the friendly name for the base type.
//  - Matrix type names map to "mat" followed by the number of columns,
//    followed by the friendly name for the base vector type.
//  - Pointer types map to "_ptr_", then the name of the storage class, then the
//    name for the pointee type.
//  - Exotic types like event, pipe, opaque, queue, reserve-id map to their own
//    human readable names.
//  - A struct type maps to "_struct_" followed by the raw Id number.  That's
//    pretty simplistic, but workable.
class FriendlyNameMapper {
 public:
  // Construct a friendly name mapper, and determine friendly names for each
  // defined Id in the specified module.  The module is specified by the code
  // wordCount, and should be parseable in the specified context.
  FriendlyNameMapper(const spv_const_context context, const uint32_t* code,
                     const size_t wordCount);

  // Returns a NameMapper which maps ids to the friendly names parsed from the
  // module provided to the constructor.
  NameMapper GetNameMapper() {
    return [this](uint32_t id) { return this->NameForId(id); };
  }

  // Returns the friendly name for the given id.  If the module parsed during
  // construction is valid, then the mapping satisfies the rules for a
  // NameMapper.
  std::string NameForId(uint32_t id);

 private:
  // Transforms the given string so that it is acceptable as an Id name in
  // assembly language.  Two distinct inputs can map to the same output.
  std::string Sanitize(const std::string& suggested_name);

  // Records a name for the given id.  Use the given suggested_name if it
  // hasn't already been taken, and otherwise generate a new (unused) name
  // based on the suggested name.
  void SaveName(uint32_t id, const std::string& suggested_name);

  // Collects information from the given parsed instruction to populate
  // name_for_id_.  Returns SPV_SUCCESS;
  spv_result_t ParseInstruction(const spv_parsed_instruction_t& inst);

  // Forwards a parsed-instruction callback from the binary parser into the
  // FriendlyNameMapper hidden inside the user_data parameter.
  static spv_result_t ParseInstructionForwarder(
      void* user_data, const spv_parsed_instruction_t* parsed_instruction) {
    return reinterpret_cast<FriendlyNameMapper*>(user_data)->ParseInstruction(
        *parsed_instruction);
  }

  // Returns the friendly name for an enumerant.
  std::string NameForEnumOperand(spv_operand_type_t type, uint32_t word);

  // Maps an id to its friendly name.  This will have an entry for each Id
  // defined in the module.
  std::unordered_map<uint32_t, std::string> name_for_id_;
  // The set of names that have a mapping in name_for_id_;
  std::unordered_set<std::string> used_names_;
  // The assembly grammar for the current context.
  const libspirv::AssemblyGrammar grammar_;
};

}  // namespace libspirv

#endif  // _LIBSPIRV_NAME_MAPPER_H_
