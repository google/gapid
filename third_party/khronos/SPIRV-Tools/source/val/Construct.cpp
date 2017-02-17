// Copyright (c) 2015-2016 The Khronos Group Inc.
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

#include "val/Construct.h"

#include <cassert>
#include <cstddef>

namespace libspirv {

Construct::Construct(ConstructType construct_type,
                     BasicBlock* entry, BasicBlock* exit,
                     std::vector<Construct*> constructs)
    : type_(construct_type),
      corresponding_constructs_(constructs),
      entry_block_(entry),
      exit_block_(exit) {}

ConstructType Construct::type() const { return type_; }

const std::vector<Construct*>& Construct::corresponding_constructs() const {
  return corresponding_constructs_;
}
std::vector<Construct*>& Construct::corresponding_constructs() {
  return corresponding_constructs_;
}

bool ValidateConstructSize(ConstructType type, size_t size) {
  switch (type) {
    case ConstructType::kSelection: return size == 0;
    case ConstructType::kContinue:  return size == 1;
    case ConstructType::kLoop:      return size == 1;
    case ConstructType::kCase:      return size >= 1;
    default: assert(1 == 0 && "Type not defined");
  }
  return false;
}

void Construct::set_corresponding_constructs(
    std::vector<Construct*> constructs) {
  assert(ValidateConstructSize(type_, constructs.size()));
  corresponding_constructs_ = constructs;
}

const BasicBlock* Construct::entry_block() const { return entry_block_; }
BasicBlock* Construct::entry_block() { return entry_block_; }

const BasicBlock* Construct::exit_block() const { return exit_block_; }
BasicBlock* Construct::exit_block() { return exit_block_; }

void Construct::set_exit(BasicBlock* block) { exit_block_ = block; }
}  /// namespace libspirv
