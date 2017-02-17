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

#ifndef LIBSPIRV_ENUM_SET_H
#define LIBSPIRV_ENUM_SET_H

#include <cstdint>
#include <functional>
#include <memory>
#include <set>
#include <utility>

#include "spirv/1.1/spirv.h"

namespace libspirv {

// A set of values of a 32-bit enum type.
// It is fast and compact for the common case, where enum values
// are at most 63.  But it can represent enums with larger values,
// as may appear in extensions.
template <typename EnumType>
class EnumSet {
 private:
  // The ForEach method will call the functor on enum values in
  // enum value order (lowest to highest).  To make that easier, use
  // an ordered set for the overflow values.
  using OverflowSetType = std::set<uint32_t>;

 public:
  // Construct an empty set.
  EnumSet() = default;
  // Construct an set with just the given enum value.
  explicit EnumSet(EnumType c) { Add(c); }
  // Construct an set from an initializer list of enum values.
  EnumSet(std::initializer_list<EnumType> cs) {
    for (auto c : cs) Add(c);
  }
  // Copy constructor.
  EnumSet(const EnumSet& other) { *this = other; }
  // Move constructor.  The moved-from set is emptied.
  EnumSet(EnumSet&& other) {
    mask_ = other.mask_;
    overflow_ = std::move(other.overflow_);
    other.mask_ = 0;
    other.overflow_.reset(nullptr);
  }
  // Assignment operator.
  EnumSet& operator=(const EnumSet& other) {
    if (&other != this) {
      mask_ = other.mask_;
      overflow_.reset(other.overflow_ ? new OverflowSetType(*other.overflow_)
                                      : nullptr);
    }
    return *this;
  }

  // Adds the given enum value to the set.  This has no effect if the
  // enum value is already in the set.
  void Add(EnumType c) { Add(ToWord(c)); }
  // Adds the given enum value (as a 32-bit word) to the set.  This has no
  // effect if the enum value is already in the set.
  void Add(uint32_t word) {
    if (auto new_bits = AsMask(word)) {
      mask_ |= new_bits;
    } else {
      Overflow().insert(word);
    }
  }

  // Returns true if this enum value is in the set.
  bool Contains(EnumType c) const { return Contains(ToWord(c)); }
  // Returns true if the enum represented as a 32-bit word is in the set.
  bool Contains(uint32_t word) const {
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

  // Applies f to each enum in the set, in order from smallest enum
  // value to largest.
  void ForEach(std::function<void(EnumType)> f) const {
    for (uint32_t i = 0; i < 64; ++i) {
      if (mask_ & AsMask(i)) f(static_cast<EnumType>(i));
    }
    if (overflow_) {
      for (uint32_t c : *overflow_) f(static_cast<EnumType>(c));
    }
  }

 private:
  // Returns the enum value as a uint32_t.
  uint32_t ToWord(EnumType value) const {
    static_assert(sizeof(EnumType) <= sizeof(uint32_t),
                  "EnumType must statically castable to uint32_t");
    return static_cast<uint32_t>(value);
  }

  // Determines whether the given enum value can be represented
  // as a bit in a uint64_t mask. If so, then returns that mask bit.
  // Otherwise, returns 0.
  uint64_t AsMask(uint32_t word) const {
    if (word > 63) return 0;
    return uint64_t(1) << word;
  }

  // Ensures that overflow_set_ references a set.  A new empty set is
  // allocated if one doesn't exist yet.  Returns overflow_set_.
  OverflowSetType& Overflow() {
    if (overflow_.get() == nullptr) {
      overflow_.reset(new OverflowSetType);
    }
    return *overflow_;
  }

  // Enums with values up to 63 are stored as bits in this mask.
  uint64_t mask_ = 0;
  // Enums with values larger than 63 are stored in this set.
  // This set should normally be empty or very small.
  std::unique_ptr<OverflowSetType> overflow_ = {};
};

// A set of SpvCapability, optimized for small capability values.
using CapabilitySet = EnumSet<SpvCapability>;

}  // namespace libspirv

#endif  // LIBSPIRV_ENUM_SET_H
