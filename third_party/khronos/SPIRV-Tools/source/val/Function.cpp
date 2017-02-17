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

#include "val/Function.h"

#include <cassert>

#include <algorithm>
#include <unordered_set>
#include <unordered_map>
#include <utility>

#include "val/BasicBlock.h"
#include "val/Construct.h"
#include "validate.h"

using std::ignore;
using std::list;
using std::make_pair;
using std::pair;
using std::tie;
using std::vector;

namespace {

using libspirv::BasicBlock;

// Computes a minimal set of root nodes required to traverse, in the forward
// direction, the CFG represented by the given vector of blocks, and successor
// and predecessor functions.  When considering adding two nodes, each having
// predecessors, favour using the one that appears earlier on the input blocks
// list.
std::vector<BasicBlock*> TraversalRoots(const std::vector<BasicBlock*>& blocks,
                                        libspirv::get_blocks_func succ_func,
                                        libspirv::get_blocks_func pred_func) {
  // The set of nodes which have been visited from any of the roots so far.
  std::unordered_set<const BasicBlock*> visited;

  auto mark_visited = [&visited](const BasicBlock* b) { visited.insert(b); };
  auto ignore_block = [](const BasicBlock*) {};
  auto ignore_blocks = [](const BasicBlock*, const BasicBlock*) {};

  auto traverse_from_root = [&mark_visited, &succ_func, &ignore_block,
                             &ignore_blocks](const BasicBlock* entry) {
    DepthFirstTraversal(entry, succ_func, mark_visited, ignore_block,
                        ignore_blocks);
  };

  std::vector<BasicBlock*> result;

  // First collect nodes without predecessors.
  for (auto block : blocks) {
    if (pred_func(block)->empty()) {
      assert(visited.count(block) == 0 && "Malformed graph!");
      result.push_back(block);
      traverse_from_root(block);
    }
  }

  // Now collect other stranded nodes.  These must be in unreachable cycles.
  for (auto block : blocks) {
    if (visited.count(block) == 0) {
      result.push_back(block);
      traverse_from_root(block);
    }
  }

  return result;
}

} // anonymous namespace

namespace libspirv {

// Universal Limit of ResultID + 1
static const uint32_t kInvalidId = 0x400000;

Function::Function(uint32_t function_id, uint32_t result_type_id,
                   SpvFunctionControlMask function_control,
                   uint32_t function_type_id)
    : id_(function_id),
      function_type_id_(function_type_id),
      result_type_id_(result_type_id),
      function_control_(function_control),
      declaration_type_(FunctionDecl::kFunctionDeclUnknown),
      end_has_been_registered_(false),
      blocks_(),
      current_block_(nullptr),
      pseudo_entry_block_(0),
      pseudo_exit_block_(kInvalidId),
      cfg_constructs_(),
      variable_ids_(),
      parameter_ids_() {}

bool Function::IsFirstBlock(uint32_t block_id) const {
  return !ordered_blocks_.empty() && *first_block() == block_id;
}

spv_result_t Function::RegisterFunctionParameter(uint32_t parameter_id,
                                                 uint32_t type_id) {
  assert(current_block_ == nullptr &&
         "RegisterFunctionParameter can only be called when parsing the binary "
         "ouside of a block");
  // TODO(umar): Validate function parameter type order and count
  // TODO(umar): Use these variables to validate parameter type
  (void)parameter_id;
  (void)type_id;
  return SPV_SUCCESS;
}

spv_result_t Function::RegisterLoopMerge(uint32_t merge_id,
                                         uint32_t continue_id) {
  RegisterBlock(merge_id, false);
  RegisterBlock(continue_id, false);
  BasicBlock& merge_block = blocks_.at(merge_id);
  BasicBlock& continue_target_block = blocks_.at(continue_id);
  assert(current_block_ &&
         "RegisterLoopMerge must be called when called within a block");

  current_block_->set_type(kBlockTypeLoop);
  merge_block.set_type(kBlockTypeMerge);
  continue_target_block.set_type(kBlockTypeContinue);
  Construct& loop_construct =
      AddConstruct({ConstructType::kLoop, current_block_, &merge_block});
  Construct& continue_construct =
      AddConstruct({ConstructType::kContinue, &continue_target_block});
  continue_construct.set_corresponding_constructs({&loop_construct});
  loop_construct.set_corresponding_constructs({&continue_construct});

  return SPV_SUCCESS;
}

spv_result_t Function::RegisterSelectionMerge(uint32_t merge_id) {
  RegisterBlock(merge_id, false);
  BasicBlock& merge_block = blocks_.at(merge_id);
  current_block_->set_type(kBlockTypeHeader);
  merge_block.set_type(kBlockTypeMerge);

  AddConstruct({ConstructType::kSelection, current_block(), &merge_block});

  return SPV_SUCCESS;
}

spv_result_t Function::RegisterSetFunctionDeclType(FunctionDecl type) {
  assert(declaration_type_ == FunctionDecl::kFunctionDeclUnknown);
  declaration_type_ = type;
  return SPV_SUCCESS;
}

spv_result_t Function::RegisterBlock(uint32_t block_id, bool is_definition) {
  assert(
      declaration_type_ == FunctionDecl::kFunctionDeclDefinition &&
      "RegisterBlocks can only be called after declaration_type_ is defined");

  std::unordered_map<uint32_t, BasicBlock>::iterator inserted_block;
  bool success = false;
  tie(inserted_block, success) =
      blocks_.insert({block_id, BasicBlock(block_id)});
  if (is_definition) {  // new block definition
    assert(current_block_ == nullptr &&
           "Register Block can only be called when parsing a binary outside of "
           "a BasicBlock");

    undefined_blocks_.erase(block_id);
    current_block_ = &inserted_block->second;
    ordered_blocks_.push_back(current_block_);
    if (IsFirstBlock(block_id)) current_block_->set_reachable(true);
  } else if (success) {  // Block doesn't exsist but this is not a definition
    undefined_blocks_.insert(block_id);
  }

  return SPV_SUCCESS;
}

void Function::RegisterBlockEnd(vector<uint32_t> next_list,
                                SpvOp branch_instruction) {
  assert(
      current_block_ &&
      "RegisterBlockEnd can only be called when parsing a binary in a block");
  vector<BasicBlock*> next_blocks;
  next_blocks.reserve(next_list.size());

  std::unordered_map<uint32_t, BasicBlock>::iterator inserted_block;
  bool success;
  for (uint32_t successor_id : next_list) {
    tie(inserted_block, success) =
        blocks_.insert({successor_id, BasicBlock(successor_id)});
    if (success) {
      undefined_blocks_.insert(successor_id);
    }
    next_blocks.push_back(&inserted_block->second);
  }

  if (current_block_->is_type(kBlockTypeLoop)) {
    // For each loop header, record the set of its successors, and include
    // its continue target if the continue target is not the loop header
    // itself.
    std::vector<BasicBlock*>& next_blocks_plus_continue_target =
        loop_header_successors_plus_continue_target_map_[current_block_];
    next_blocks_plus_continue_target = next_blocks;
    auto continue_target = FindConstructForEntryBlock(current_block_)
                               .corresponding_constructs()
                               .back()
                               ->entry_block();
    if (continue_target != current_block_) {
      next_blocks_plus_continue_target.push_back(continue_target);
    }
  }

  current_block_->RegisterBranchInstruction(branch_instruction);
  current_block_->RegisterSuccessors(next_blocks);
  current_block_ = nullptr;
  return;
}

void Function::RegisterFunctionEnd() {
  if (!end_has_been_registered_) {
    end_has_been_registered_ = true;

    ComputeAugmentedCFG();
  }
}

size_t Function::block_count() const { return blocks_.size(); }

size_t Function::undefined_block_count() const {
  return undefined_blocks_.size();
}

const vector<BasicBlock*>& Function::ordered_blocks() const {
  return ordered_blocks_;
}
vector<BasicBlock*>& Function::ordered_blocks() { return ordered_blocks_; }

const BasicBlock* Function::current_block() const { return current_block_; }
BasicBlock* Function::current_block() { return current_block_; }

const list<Construct>& Function::constructs() const { return cfg_constructs_; }
list<Construct>& Function::constructs() { return cfg_constructs_; }

const BasicBlock* Function::first_block() const {
  if (ordered_blocks_.empty()) return nullptr;
  return ordered_blocks_[0];
}
BasicBlock* Function::first_block() {
  if (ordered_blocks_.empty()) return nullptr;
  return ordered_blocks_[0];
}

bool Function::IsBlockType(uint32_t merge_block_id, BlockType type) const {
  bool ret = false;
  const BasicBlock* block;
  tie(block, ignore) = GetBlock(merge_block_id);
  if (block) {
    ret = block->is_type(type);
  }
  return ret;
}

pair<const BasicBlock*, bool> Function::GetBlock(uint32_t block_id) const {
  const auto b = blocks_.find(block_id);
  if (b != end(blocks_)) {
    const BasicBlock* block = &(b->second);
    bool defined =
        undefined_blocks_.find(block->id()) == end(undefined_blocks_);
    return make_pair(block, defined);
  } else {
    return make_pair(nullptr, false);
  }
}

pair<BasicBlock*, bool> Function::GetBlock(uint32_t block_id) {
  const BasicBlock* out;
  bool defined;
  tie(out, defined) = const_cast<const Function*>(this)->GetBlock(block_id);
  return make_pair(const_cast<BasicBlock*>(out), defined);
}

Function::GetBlocksFunction Function::AugmentedCFGSuccessorsFunction() const {
  return [this](const BasicBlock* block) {
    auto where = augmented_successors_map_.find(block);
    return where == augmented_successors_map_.end() ? block->successors()
                                                    : &(*where).second;
  };
}

Function::GetBlocksFunction
Function::AugmentedCFGSuccessorsFunctionIncludingHeaderToContinueEdge() const {
  return [this](const BasicBlock* block) {
    auto where = loop_header_successors_plus_continue_target_map_.find(block);
    return where == loop_header_successors_plus_continue_target_map_.end()
               ? AugmentedCFGSuccessorsFunction()(block)
               : &(*where).second;
  };
}

Function::GetBlocksFunction Function::AugmentedCFGPredecessorsFunction() const {
  return [this](const BasicBlock* block) {
    auto where = augmented_predecessors_map_.find(block);
    return where == augmented_predecessors_map_.end() ? block->predecessors()
                                                      : &(*where).second;
  };
}

void Function::ComputeAugmentedCFG() {
  // Compute the successors of the pseudo-entry block, and
  // the predecessors of the pseudo exit block.
  auto succ_func = [](const BasicBlock* b) { return b->successors(); };
  auto pred_func = [](const BasicBlock* b) { return b->predecessors(); };
  auto sources = TraversalRoots(ordered_blocks_, succ_func, pred_func);

  // For the predecessor traversals, reverse the order of blocks.  This
  // will affect the post-dominance calculation as follows:
  //  - Suppose you have blocks A and B, with A appearing before B in
  //    the list of blocks.
  //  - Also, A branches only to B, and B branches only to A.
  //  - We want to compute A as dominating B, and B as post-dominating B.
  // By using reversed blocks for predecessor traversal roots discovery,
  // we'll add an edge from B to the pseudo-exit node, rather than from A.
  // All this is needed to correctly process the dominance/post-dominance
  // constraint when A is a loop header that points to itself as its
  // own continue target, and B is the latch block for the loop.
  std::vector<BasicBlock*> reversed_blocks(ordered_blocks_.rbegin(),
                                           ordered_blocks_.rend());
  auto sinks = TraversalRoots(reversed_blocks, pred_func, succ_func);

  // Wire up the pseudo entry block.
  augmented_successors_map_[&pseudo_entry_block_] = sources;
  for (auto block : sources) {
    auto& augmented_preds = augmented_predecessors_map_[block];
    const auto& preds = *block->predecessors();
    augmented_preds.reserve(1 + preds.size());
    augmented_preds.push_back(&pseudo_entry_block_);
    augmented_preds.insert(augmented_preds.end(), preds.begin(), preds.end());
  }

  // Wire up the pseudo exit block.
  augmented_predecessors_map_[&pseudo_exit_block_] = sinks;
  for (auto block : sinks) {
    auto& augmented_succ = augmented_successors_map_[block];
    const auto& succ = *block->successors();
    augmented_succ.reserve(1 + succ.size());
    augmented_succ.push_back(&pseudo_exit_block_);
    augmented_succ.insert(augmented_succ.end(), succ.begin(), succ.end());
  }
};

Construct& Function::AddConstruct(const Construct& new_construct) {
  cfg_constructs_.push_back(new_construct);
  auto& result = cfg_constructs_.back();
  entry_block_to_construct_[new_construct.entry_block()] = &result;
  return result;
}

Construct& Function::FindConstructForEntryBlock(const BasicBlock* entry_block) {
  auto where = entry_block_to_construct_.find(entry_block);
  assert(where != entry_block_to_construct_.end());
  auto construct_ptr = (*where).second;
  assert(construct_ptr);
  return *construct_ptr;
}

}  /// namespace libspirv
