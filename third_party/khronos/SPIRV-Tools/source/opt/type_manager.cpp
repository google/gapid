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

#include <algorithm>
#include <cassert>

#include "reflect.h"
#include "type_manager.h"

namespace spvtools {
namespace opt {
namespace analysis {

Type* TypeManager::GetType(uint32_t id) const {
  auto iter = id_to_type_.find(id);
  if (iter != id_to_type_.end()) return (*iter).second.get();
  return nullptr;
}

uint32_t TypeManager::GetId(const Type* type) const {
  auto iter = type_to_id_.find(type);
  if (iter != type_to_id_.end()) return (*iter).second;
  return 0;
}

ForwardPointer* TypeManager::GetForwardPointer(uint32_t index) const {
  if (index >= forward_pointers_.size()) return nullptr;
  return forward_pointers_.at(index).get();
}

void TypeManager::AnalyzeTypes(const spvtools::ir::Module& module) {
  for (const auto* inst : module.GetTypes()) RecordIfTypeDefinition(*inst);
  for (const auto& inst : module.annotations()) AttachIfTypeDecoration(inst);
}

Type* TypeManager::RecordIfTypeDefinition(
    const spvtools::ir::Instruction& inst) {
  if (!spvtools::ir::IsTypeInst(inst.opcode())) return nullptr;

  Type* type = nullptr;
  switch (inst.opcode()) {
    case SpvOpTypeVoid:
      type = new Void();
      break;
    case SpvOpTypeBool:
      type = new Bool();
      break;
    case SpvOpTypeInt:
      type = new Integer(inst.GetSingleWordInOperand(0),
                         inst.GetSingleWordInOperand(1));
      break;
    case SpvOpTypeFloat:
      type = new Float(inst.GetSingleWordInOperand(0));
      break;
    case SpvOpTypeVector:
      type = new Vector(GetType(inst.GetSingleWordInOperand(0)),
                        inst.GetSingleWordInOperand(1));
      break;
    case SpvOpTypeMatrix:
      type = new Matrix(GetType(inst.GetSingleWordInOperand(0)),
                        inst.GetSingleWordInOperand(1));
      break;
    case SpvOpTypeImage: {
      const SpvAccessQualifier access =
          inst.NumInOperands() < 8
              ? SpvAccessQualifierReadOnly
              : static_cast<SpvAccessQualifier>(inst.GetSingleWordInOperand(7));
      type = new Image(
          GetType(inst.GetSingleWordInOperand(0)),
          static_cast<SpvDim>(inst.GetSingleWordInOperand(1)),
          inst.GetSingleWordInOperand(2), inst.GetSingleWordInOperand(3),
          inst.GetSingleWordInOperand(4), inst.GetSingleWordInOperand(5),
          static_cast<SpvImageFormat>(inst.GetSingleWordInOperand(6)), access);
    } break;
    case SpvOpTypeSampler:
      type = new Sampler();
      break;
    case SpvOpTypeSampledImage:
      type = new SampledImage(GetType(inst.GetSingleWordInOperand(0)));
      break;
    case SpvOpTypeArray:
      type = new Array(GetType(inst.GetSingleWordInOperand(0)),
                       inst.GetSingleWordInOperand(1));
      break;
    case SpvOpTypeRuntimeArray:
      type = new RuntimeArray(GetType(inst.GetSingleWordInOperand(0)));
      break;
    case SpvOpTypeStruct: {
      std::vector<Type*> element_types;
      for (uint32_t i = 0; i < inst.NumInOperands(); ++i) {
        element_types.push_back(GetType(inst.GetSingleWordInOperand(i)));
      }
      type = new Struct(element_types);
    } break;
    case SpvOpTypeOpaque: {
      const uint32_t* data = inst.GetInOperand(0).words.data();
      type = new Opaque(reinterpret_cast<const char*>(data));
    } break;
    case SpvOpTypePointer: {
      auto* ptr = new Pointer(
          GetType(inst.GetSingleWordInOperand(1)),
          static_cast<SpvStorageClass>(inst.GetSingleWordInOperand(0)));
      // Let's see if somebody forward references this pointer.
      for (auto* fp : unresolved_forward_pointers_) {
        if (fp->target_id() == inst.result_id()) {
          fp->SetTargetPointer(ptr);
          unresolved_forward_pointers_.erase(fp);
          break;
        }
      }
      type = ptr;
    } break;
    case SpvOpTypeFunction: {
      Type* return_type = GetType(inst.GetSingleWordInOperand(0));
      std::vector<Type*> param_types;
      for (uint32_t i = 1; i < inst.NumInOperands(); ++i) {
        param_types.push_back(GetType(inst.GetSingleWordInOperand(i)));
      }
      type = new Function(return_type, param_types);
    } break;
    case SpvOpTypeEvent:
      type = new Event();
      break;
    case SpvOpTypeDeviceEvent:
      type = new DeviceEvent();
      break;
    case SpvOpTypeReserveId:
      type = new ReserveId();
      break;
    case SpvOpTypeQueue:
      type = new Queue();
      break;
    case SpvOpTypePipe:
      type = new Pipe(
          static_cast<SpvAccessQualifier>(inst.GetSingleWordInOperand(0)));
      break;
    case SpvOpTypeForwardPointer: {
      // Handling of forward pointers is different from the other types.
      auto* fp = new ForwardPointer(
          inst.GetSingleWordInOperand(0),
          static_cast<SpvStorageClass>(inst.GetSingleWordInOperand(1)));
      forward_pointers_.emplace_back(fp);
      unresolved_forward_pointers_.insert(fp);
      return fp;
    }
    case SpvOpTypePipeStorage:
      type = new PipeStorage();
      break;
    case SpvOpTypeNamedBarrier:
      type = new NamedBarrier();
      break;
    default:
      assert(0 && "unhandled type found");
      break;
  }

  uint32_t id = inst.result_id();
  if (id == 0) {
    assert(inst.opcode() == SpvOpTypeForwardPointer &&
           "instruction without result id found");
  } else {
    assert(type != nullptr && "type should not be nullptr at this point");
    id_to_type_[id].reset(type);
    type_to_id_[type] = id;
  }
  return type;
}

void TypeManager::AttachIfTypeDecoration(const ir::Instruction& inst) {
  const SpvOp opcode = inst.opcode();
  if (!ir::IsAnnotationInst(opcode)) return;
  const uint32_t id = inst.GetSingleWordOperand(0);
  // Do nothing if the id to be decorated is not for a known type.
  if (!id_to_type_.count(id)) return;

  Type* target_type = id_to_type_[id].get();
  switch (opcode) {
    case SpvOpDecorate: {
      const auto count = inst.NumOperands();
      std::vector<uint32_t> data;
      for (uint32_t i = 1; i < count; ++i) {
        data.push_back(inst.GetSingleWordOperand(i));
      }
      target_type->AddDecoration(std::move(data));
    } break;
    case SpvOpMemberDecorate: {
      const auto count = inst.NumOperands();
      const uint32_t index = inst.GetSingleWordOperand(1);
      std::vector<uint32_t> data;
      for (uint32_t i = 2; i < count; ++i) {
        data.push_back(inst.GetSingleWordOperand(i));
      }
      if (Struct* st = target_type->AsStruct()) {
        st->AddMemeberDecoration(index, std::move(data));
      } else {
        assert(0 && "OpMemberDecorate on non-struct type");
      }
    } break;
    case SpvOpDecorationGroup:
    case SpvOpGroupDecorate:
    case SpvOpGroupMemberDecorate:
      assert(0 && "unhandled decoration");
      break;
    default:
      assert(0 && "unreachable");
      break;
  }
}

}  // namespace analysis
}  // namespace opt
}  // namespace spvtools
