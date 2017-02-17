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

#ifndef LIBSPIRV_VAL_BASICBLOCK_H_
#define LIBSPIRV_VAL_BASICBLOCK_H_

#include "spirv/1.1/spirv.h"

#include <cstdint>

#include <bitset>
#include <functional>
#include <memory>
#include <vector>

namespace libspirv {

enum BlockType : uint32_t {
  kBlockTypeUndefined,
  kBlockTypeHeader,
  kBlockTypeLoop,
  kBlockTypeMerge,
  kBlockTypeBreak,
  kBlockTypeContinue,
  kBlockTypeReturn,
  kBlockTypeCOUNT  ///< Total number of block types. (must be the last element)
};

// This class represents a basic block in a SPIR-V module
class BasicBlock {
 public:
  /// Constructor for a BasicBlock
  ///
  /// @param[in] id The ID of the basic block
  explicit BasicBlock(uint32_t id);

  /// Returns the id of the BasicBlock
  uint32_t id() const { return id_; }

  /// Returns the predecessors of the BasicBlock
  const std::vector<BasicBlock*>* predecessors() const {
    return &predecessors_;
  }

  /// Returns the predecessors of the BasicBlock
  std::vector<BasicBlock*>* predecessors() { return &predecessors_; }

  /// Returns the successors of the BasicBlock
  const std::vector<BasicBlock*>* successors() const { return &successors_; }

  /// Returns the successors of the BasicBlock
  std::vector<BasicBlock*>* successors() { return &successors_; }

  /// Returns true if the block is reachable in the CFG
  bool reachable() const { return reachable_; }

  /// Returns true if BasicBlock is of the given type
  bool is_type(BlockType type) const {
    if (type == kBlockTypeUndefined) return type_.none();
    return type_.test(type);
  }

  /// Sets the reachability of the basic block in the CFG
  void set_reachable(bool reachability) { reachable_ = reachability; }

  /// Sets the type of the BasicBlock
  void set_type(BlockType type) {
    if (type == kBlockTypeUndefined)
      type_.reset();
    else
      type_.set(type);
  }

  /// Sets the immedate dominator of this basic block
  ///
  /// @param[in] dom_block The dominator block
  void SetImmediateDominator(BasicBlock* dom_block);

  /// Sets the immedate post dominator of this basic block
  ///
  /// @param[in] pdom_block The post dominator block
  void SetImmediatePostDominator(BasicBlock* pdom_block);

  /// Returns the immedate dominator of this basic block
  BasicBlock* immediate_dominator();

  /// Returns the immedate dominator of this basic block
  const BasicBlock* immediate_dominator() const;

  /// Returns the immedate post dominator of this basic block
  BasicBlock* immediate_post_dominator();

  /// Returns the immedate post dominator of this basic block
  const BasicBlock* immediate_post_dominator() const;

  /// Ends the block without a successor
  void RegisterBranchInstruction(SpvOp branch_instruction);

  /// Adds @p next BasicBlocks as successors of this BasicBlock
  void RegisterSuccessors(const std::vector<BasicBlock*>& next = {});

  /// Returns true if the id of the BasicBlock matches
  bool operator==(const BasicBlock& other) const { return other.id_ == id_; }

  /// Returns true if the id of the BasicBlock matches
  bool operator==(const uint32_t& other_id) const { return other_id == id_; }

  /// Returns true if this block dominates the other block.
  /// Assumes dominators have been computed.
  bool dominates(const BasicBlock& other) const;

  /// Returns true if this block postdominates the other block.
  /// Assumes dominators have been computed.
  bool postdominates(const BasicBlock& other) const;

  /// @brief A BasicBlock dominator iterator class
  ///
  /// This iterator will iterate over the (post)dominators of the block
  class DominatorIterator
      : public std::iterator<std::forward_iterator_tag, BasicBlock*> {
   public:
    /// @brief Constructs the end of dominator iterator
    ///
    /// This will create an iterator which will represent the element
    /// before the root node of the dominator tree
    DominatorIterator();

    /// @brief Constructs an iterator for the given block which points to
    ///        @p block
    ///
    /// @param block          The block which is referenced by the iterator
    /// @param dominator_func This function will be called to get the immediate
    ///                       (post)dominator of the current block
    DominatorIterator(
        const BasicBlock* block,
        std::function<const BasicBlock*(const BasicBlock*)> dominator_func);

    /// @brief Advances the iterator
    DominatorIterator& operator++();

    /// @brief Returns the current element
    const BasicBlock*& operator*();

    friend bool operator==(const DominatorIterator& lhs,
                           const DominatorIterator& rhs);

   private:
    const BasicBlock* current_;
    std::function<const BasicBlock*(const BasicBlock*)> dom_func_;
  };

  /// Returns a dominator iterator which points to the current block
  const DominatorIterator dom_begin() const;

  /// Returns a dominator iterator which points to the current block
  DominatorIterator dom_begin();

  /// Returns a dominator iterator which points to one element past the first
  /// block
  const DominatorIterator dom_end() const;

  /// Returns a dominator iterator which points to one element past the first
  /// block
  DominatorIterator dom_end();

  /// Returns a post dominator iterator which points to the current block
  const DominatorIterator pdom_begin() const;
  /// Returns a post dominator iterator which points to the current block
  DominatorIterator pdom_begin();

  /// Returns a post dominator iterator which points to one element past the
  /// last block
  const DominatorIterator pdom_end() const;

  /// Returns a post dominator iterator which points to one element past the
  /// last block
  DominatorIterator pdom_end();

 private:
  /// Id of the BasicBlock
  const uint32_t id_;

  /// Pointer to the immediate dominator of the BasicBlock
  BasicBlock* immediate_dominator_;

  /// Pointer to the immediate dominator of the BasicBlock
  BasicBlock* immediate_post_dominator_;

  /// The set of predecessors of the BasicBlock
  std::vector<BasicBlock*> predecessors_;

  /// The set of successors of the BasicBlock
  std::vector<BasicBlock*> successors_;

  /// The type of the block
  std::bitset<kBlockTypeCOUNT - 1> type_;

  /// True if the block is reachable in the CFG
  bool reachable_;
};

/// @brief Returns true if the iterators point to the same element or if both
///        iterators point to the @p dom_end block
bool operator==(const BasicBlock::DominatorIterator& lhs,
                const BasicBlock::DominatorIterator& rhs);

/// @brief Returns true if the iterators point to different elements and they
///        do not both point to the @p dom_end block
bool operator!=(const BasicBlock::DominatorIterator& lhs,
                const BasicBlock::DominatorIterator& rhs);

}  /// namespace libspirv

#endif  /// LIBSPIRV_VAL_BASICBLOCK_H_
