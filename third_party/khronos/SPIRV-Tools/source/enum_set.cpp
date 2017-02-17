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

#include "enum_set.h"

#include "spirv/1.1/spirv.hpp"

namespace {

// Determines whether the given enum value can be represented
// as a bit in a uint64_t mask. If so, then returns that mask bit.
// Otherwise, returns 0.
uint64_t AsMask(uint32_t word) {
  if (word > 63) return 0;
  return uint64_t(1) << word;
}
}

namespace libspirv {

template<typename EnumType>
void EnumSet<EnumType>::Add(uint32_t word) {
  if (auto new_bits = AsMask(word)) {
    mask_ |= new_bits;
  } else {
    Overflow().insert(word);
  }
}

template<typename EnumType>
bool EnumSet<EnumType>::Contains(uint32_t word) const {
  // We shouldn't call Overflow() since this is a const method.
  if (auto bits = AsMask(word)) {
    return mask_ & bits;
  } else if (auto overflow = overflow_.get()) {
    return overflow->find(word) != overflow->end();
  }
  // The word is large, but the set doesn't have large members, so
  // it doesn't have an overflow set.
  return false;
}

// Applies f to each capability in the set, in order from smallest enum
// value to largest.
void CapabilitySet::ForEach(std::function<void(SpvCapability)> f) const {
  for (uint32_t i = 0; i < 64; ++i) {
    if (mask_ & AsMask(i)) f(static_cast<SpvCapability>(i));
  }
  if (overflow_) {
    for (uint32_t c : *overflow_) f(static_cast<SpvCapability>(c));
  }
}

}  // namespace libspirv
