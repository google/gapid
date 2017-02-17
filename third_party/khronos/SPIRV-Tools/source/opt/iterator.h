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

#ifndef LIBSPIRV_OPT_ITERATOR_H_
#define LIBSPIRV_OPT_ITERATOR_H_

#include <iterator>
#include <memory>
#include <type_traits>
#include <vector>

namespace spvtools {
namespace ir {

// An ad hoc iterator class for std::vector<std::unique_ptr<|ValueType|>>. The
// purpose of this iterator class is to provide transparent access to those
// std::unique_ptr managed elements in the vector, behaving like we are using
// std::vector<|ValueType|>.
template <typename ValueType, bool IsConst = false>
class UptrVectorIterator
    : public std::iterator<
          std::random_access_iterator_tag,
          typename std::conditional<IsConst, const ValueType, ValueType>::type,
          ptrdiff_t> {
 public:
  using super = std::iterator<
      std::random_access_iterator_tag,
      typename std::conditional<IsConst, const ValueType, ValueType>::type>;

  using pointer = typename super::pointer;
  using reference = typename super::reference;
  using difference_type = typename super::difference_type;

  // Type aliases. We need to apply constness properly if |IsConst| is true.
  using Uptr = std::unique_ptr<ValueType>;
  using UptrVector = typename std::conditional<IsConst, const std::vector<Uptr>,
                                               std::vector<Uptr>>::type;
  using UnderlyingIterator =
      typename std::conditional<IsConst, typename UptrVector::const_iterator,
                                typename UptrVector::iterator>::type;

  // Creates a new iterator from the given |container| and its raw iterator
  // |it|.
  UptrVectorIterator(UptrVector* container, const UnderlyingIterator& it)
      : container_(container), iterator_(it) {}
  UptrVectorIterator(const UptrVectorIterator&) = default;
  UptrVectorIterator& operator=(const UptrVectorIterator&) = default;

  inline UptrVectorIterator& operator++();
  inline UptrVectorIterator operator++(int);
  inline UptrVectorIterator& operator--();
  inline UptrVectorIterator operator--(int);

  reference operator*() const { return **iterator_; }
  pointer operator->() { return (*iterator_).get(); }
  reference operator[](ptrdiff_t index) { return **(iterator_ + index); }

  inline bool operator==(const UptrVectorIterator& that) const;
  inline bool operator!=(const UptrVectorIterator& that) const;

  inline ptrdiff_t operator-(const UptrVectorIterator& that) const;
  inline bool operator<(const UptrVectorIterator& that) const;

  // Inserts the given |value| to the position pointed to by this iterator
  // and returns an iterator to the newly iserted |value|.
  // If the underlying vector changes capacity, all previous iterators will be
  // invalidated. Otherwise, those previous iterators pointing to after the
  // insertion point will be invalidated.
  template <bool IsConstForMethod = IsConst>
  inline typename std::enable_if<!IsConstForMethod, UptrVectorIterator>::type
  InsertBefore(Uptr value);

 private:
  UptrVector* container_;        // The container we are manipulating.
  UnderlyingIterator iterator_;  // The raw iterator from the container.
};

// Handy class for a (begin, end) iterator pair.
template <typename IteratorType>
class IteratorRange {
 public:
  IteratorRange(IteratorType b, IteratorType e) : begin_(b), end_(e) {}

  IteratorType begin() const { return begin_; }
  IteratorType end() const { return end_; }

  bool empty() const { return begin_ == end_; }
  size_t size() const { return end_ - begin_; }

 private:
  IteratorType begin_;
  IteratorType end_;
};

// Returns a (begin, end) iterator pair for the given container.
template <typename ValueType,
          class IteratorType = UptrVectorIterator<ValueType>>
inline IteratorRange<IteratorType> make_range(
    std::vector<std::unique_ptr<ValueType>>& container) {
  return {IteratorType(&container, container.begin()),
          IteratorType(&container, container.end())};
}

// Returns a const (begin, end) iterator pair for the given container.
template <typename ValueType,
          class IteratorType = UptrVectorIterator<ValueType, true>>
inline IteratorRange<IteratorType> make_const_range(
    const std::vector<std::unique_ptr<ValueType>>& container) {
  return {IteratorType(&container, container.cbegin()),
          IteratorType(&container, container.cend())};
}

template <typename VT, bool IC>
inline UptrVectorIterator<VT, IC>& UptrVectorIterator<VT, IC>::operator++() {
  ++iterator_;
  return *this;
}

template <typename VT, bool IC>
inline UptrVectorIterator<VT, IC> UptrVectorIterator<VT, IC>::operator++(int) {
  auto it = *this;
  ++(*this);
  return it;
}

template <typename VT, bool IC>
inline UptrVectorIterator<VT, IC>& UptrVectorIterator<VT, IC>::operator--() {
  --iterator_;
  return *this;
}

template <typename VT, bool IC>
inline UptrVectorIterator<VT, IC> UptrVectorIterator<VT, IC>::operator--(int) {
  auto it = *this;
  --(*this);
  return it;
}

template <typename VT, bool IC>
inline bool UptrVectorIterator<VT, IC>::operator==(
    const UptrVectorIterator& that) const {
  return container_ == that.container_ && iterator_ == that.iterator_;
}

template <typename VT, bool IC>
inline bool UptrVectorIterator<VT, IC>::operator!=(
    const UptrVectorIterator& that) const {
  return !(*this == that);
}

template <typename VT, bool IC>
inline ptrdiff_t UptrVectorIterator<VT, IC>::operator-(
    const UptrVectorIterator& that) const {
  assert(container_ == that.container_);
  return iterator_ - that.iterator_;
}

template <typename VT, bool IC>
inline bool UptrVectorIterator<VT, IC>::operator<(
    const UptrVectorIterator& that) const {
  assert(container_ == that.container_);
  return iterator_ < that.iterator_;
}

template <typename VT, bool IC>
template <bool IsConstForMethod>
inline
    typename std::enable_if<!IsConstForMethod, UptrVectorIterator<VT, IC>>::type
    UptrVectorIterator<VT, IC>::InsertBefore(Uptr value) {
  auto index = iterator_ - container_->begin();
  container_->insert(iterator_, std::move(value));
  return UptrVectorIterator(container_, container_->begin() + index);
}

}  // namespace ir
}  // namespace spvtools

#endif  // LIBSPIRV_OPT_ITERATOR_H_
