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

#ifndef LIBSPIRV_OPT_TYPE_MANAGER_H_
#define LIBSPIRV_OPT_TYPE_MANAGER_H_

#include <memory>
#include <unordered_map>
#include <unordered_set>

#include "module.h"
#include "types.h"

namespace spvtools {
namespace opt {
namespace analysis {

// A class for managing the SPIR-V type hierarchy.
class TypeManager {
 public:
  using IdToTypeMap = std::unordered_map<uint32_t, std::unique_ptr<Type>>;
  using TypeToIdMap = std::unordered_map<const Type*, uint32_t>;
  using ForwardPointerVector = std::vector<std::unique_ptr<ForwardPointer>>;

  inline explicit TypeManager(const spvtools::ir::Module& module);
  TypeManager(const TypeManager&) = delete;
  TypeManager(TypeManager&&) = delete;
  TypeManager& operator=(const TypeManager&) = delete;
  TypeManager& operator=(TypeManager&&) = delete;

  // Returns the type for the given type |id|. Returns nullptr if the given |id|
  // does not define a type.
  Type* GetType(uint32_t id) const;
  // Returns the id for the given |type|. Returns 0 if can not find the given
  // |type|.
  uint32_t GetId(const Type* type) const;
  // Returns the number of types hold in this manager.
  size_t NumTypes() const { return id_to_type_.size(); }

  // Returns the forward pointer type at the given |index|.
  ForwardPointer* GetForwardPointer(uint32_t index) const;
  // Returns the number of forward pointer types hold in this manager.
  size_t NumForwardPointers() const { return forward_pointers_.size(); }

  Type* GetRecordIfTypeDefinition(const spvtools::ir::Instruction& inst) {
    return RecordIfTypeDefinition(inst);
  }

  // Returns the map from types to their ids.
  const TypeToIdMap& type_to_ids() const { return type_to_id_; }

 private:
  // Analyzes the types and decorations on types in the given |module|.
  void AnalyzeTypes(const spvtools::ir::Module& module);

  // Creates and returns a type from the given SPIR-V |inst|. Returns nullptr if
  // the given instruction is not for defining a type.
  Type* RecordIfTypeDefinition(const spvtools::ir::Instruction& inst);
  // Attaches the decoration encoded in |inst| to a type. Does nothing if the
  // given instruction is not a decoration instruction or not decorating a type.
  void AttachIfTypeDecoration(const spvtools::ir::Instruction& inst);

  IdToTypeMap id_to_type_;  // Mapping from ids to their type representations.
  TypeToIdMap type_to_id_;  // Mapping from types to their defining ids.
  ForwardPointerVector forward_pointers_;  // All forward pointer declarations.
  // All unresolved forward pointer declarations.
  // Refers the contents in the above vector.
  std::unordered_set<ForwardPointer*> unresolved_forward_pointers_;
};

inline TypeManager::TypeManager(const spvtools::ir::Module& module) {
  AnalyzeTypes(module);
}

}  // namespace analysis
}  // namespace opt
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_TYPE_MANAGER_H_
