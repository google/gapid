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

#include "BasicBlock.h"

#include <algorithm>
#include <utility>
#include <vector>

using std::vector;

namespace libspirv {

BasicBlock::BasicBlock(uint32_t label_id)
    : id_(label_id),
      immediate_dominator_(nullptr),
      immediate_post_dominator_(nullptr),
      predecessors_(),
      successors_(),
      type_(0),
      reachable_(false) {}

void BasicBlock::SetImmediateDominator(BasicBlock* dom_block) {
  immediate_dominator_ = dom_block;
}

void BasicBlock::SetImmediatePostDominator(BasicBlock* pdom_block) {
  immediate_post_dominator_ = pdom_block;
}

const BasicBlock* BasicBlock::immediate_dominator() const {
  return immediate_dominator_;
}

const BasicBlock* BasicBlock::immediate_post_dominator() const {
  return immediate_post_dominator_;
}

BasicBlock* BasicBlock::immediate_dominator() { return immediate_dominator_; }
BasicBlock* BasicBlock::immediate_post_dominator() {
  return immediate_post_dominator_;
}

void BasicBlock::RegisterSuccessors(const vector<BasicBlock*>& next_blocks) {
  for (auto& block : next_blocks) {
    block->predecessors_.push_back(this);
    successors_.push_back(block);
    if (block->reachable_ == false) block->set_reachable(reachable_);
  }
}

void BasicBlock::RegisterBranchInstruction(SpvOp branch_instruction) {
  if (branch_instruction == SpvOpUnreachable) reachable_ = false;
  return;
}

bool BasicBlock::dominates(const BasicBlock& other) const {
  return (this == &other) ||
         !(other.dom_end() ==
           std::find(other.dom_begin(), other.dom_end(), this));
}

bool BasicBlock::postdominates(const BasicBlock& other) const {
  return (this == &other) ||
         !(other.pdom_end() ==
           std::find(other.pdom_begin(), other.pdom_end(), this));
}

BasicBlock::DominatorIterator::DominatorIterator() : current_(nullptr) {}

BasicBlock::DominatorIterator::DominatorIterator(
    const BasicBlock* block,
    std::function<const BasicBlock*(const BasicBlock*)> dominator_func)
    : current_(block), dom_func_(dominator_func) {}

BasicBlock::DominatorIterator& BasicBlock::DominatorIterator::operator++() {
  if (current_ == dom_func_(current_)) {
    current_ = nullptr;
  } else {
    current_ = dom_func_(current_);
  }
  return *this;
}

const BasicBlock::DominatorIterator BasicBlock::dom_begin() const {
  return DominatorIterator(
      this, [](const BasicBlock* b) { return b->immediate_dominator(); });
}

BasicBlock::DominatorIterator BasicBlock::dom_begin() {
  return DominatorIterator(
      this, [](const BasicBlock* b) { return b->immediate_dominator(); });
}

const BasicBlock::DominatorIterator BasicBlock::dom_end() const {
  return DominatorIterator();
}

BasicBlock::DominatorIterator BasicBlock::dom_end() {
  return DominatorIterator();
}

const BasicBlock::DominatorIterator BasicBlock::pdom_begin() const {
  return DominatorIterator(
      this, [](const BasicBlock* b) { return b->immediate_post_dominator(); });
}

BasicBlock::DominatorIterator BasicBlock::pdom_begin() {
  return DominatorIterator(
    this, [](const BasicBlock* b) { return b->immediate_post_dominator(); });
}

const BasicBlock::DominatorIterator BasicBlock::pdom_end() const {
  return DominatorIterator();
}

BasicBlock::DominatorIterator BasicBlock::pdom_end() {
  return DominatorIterator();
}

bool operator==(const BasicBlock::DominatorIterator& lhs,
                const BasicBlock::DominatorIterator& rhs) {
  return lhs.current_ == rhs.current_;
}

bool operator!=(const BasicBlock::DominatorIterator& lhs,
                const BasicBlock::DominatorIterator& rhs) {
  return !(lhs == rhs);
}

const BasicBlock*& BasicBlock::DominatorIterator::operator*() {
  return current_;
}
}  // namespace libspirv
