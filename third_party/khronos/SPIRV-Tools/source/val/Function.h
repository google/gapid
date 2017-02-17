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

#ifndef LIBSPIRV_VAL_FUNCTION_H_
#define LIBSPIRV_VAL_FUNCTION_H_

#include <list>
#include <functional>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "spirv-tools/libspirv.h"
#include "spirv/1.1/spirv.h"
#include "val/BasicBlock.h"
#include "val/Construct.h"

namespace libspirv {

enum class FunctionDecl {
  kFunctionDeclUnknown,      /// < Unknown function declaration
  kFunctionDeclDeclaration,  /// < Function declaration
  kFunctionDeclDefinition    /// < Function definition
};

/// This class manages all function declaration and definitions in a module. It
/// handles the state and id information while parsing a function in the SPIR-V
/// binary.
class Function {
 public:
  Function(uint32_t id, uint32_t result_type_id,
           SpvFunctionControlMask function_control, uint32_t function_type_id);

  /// Registers a function parameter in the current function
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterFunctionParameter(uint32_t id, uint32_t type_id);

  /// Sets the declaration type of the current function
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterSetFunctionDeclType(FunctionDecl type);

  /// Registers a block in the current function. Subsequent block instructions
  /// will target this block
  /// @param id The ID of the label of the block
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterBlock(uint32_t id, bool is_definition = true);

  /// Registers a variable in the current block
  ///
  /// @param[in] type_id The type ID of the varaible
  /// @param[in] id      The ID of the varaible
  /// @param[in] storage The storage of the variable
  /// @param[in] init_id The initializer ID of the variable
  ///
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterBlockVariable(uint32_t type_id, uint32_t id,
                                     SpvStorageClass storage, uint32_t init_id);

  /// Registers a loop merge construct in the function
  ///
  /// @param[in] merge_id The merge block ID of the loop
  /// @param[in] continue_id The continue block ID of the loop
  ///
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterLoopMerge(uint32_t merge_id, uint32_t continue_id);

  /// Registers a selection merge construct in the function
  /// @return Returns SPV_SUCCESS if the call was successful
  spv_result_t RegisterSelectionMerge(uint32_t merge_id);

  /// Registers the end of the block
  ///
  /// @param[in] successors_list A list of ids to the block's successors
  /// @param[in] branch_instruction the branch instruction that ended the block
  void RegisterBlockEnd(std::vector<uint32_t> successors_list,
                        SpvOp branch_instruction);

  /// Registers the end of the function.  This is idempotent.
  void RegisterFunctionEnd();

  /// Returns true if the \p id block is the first block of this function
  bool IsFirstBlock(uint32_t id) const;

  /// Returns true if the \p merge_block_id is a BlockType of \p type
  bool IsBlockType(uint32_t merge_block_id, BlockType type) const;

  /// Returns a pair consisting of the BasicBlock with \p id and a bool
  /// which is true if the block has been defined, and false if it is
  /// declared but not defined. This function will return nullptr if the
  /// \p id was not declared and not defined at the current point in the binary
  std::pair<const BasicBlock*, bool> GetBlock(uint32_t id) const;
  std::pair<BasicBlock*, bool> GetBlock(uint32_t id);

  /// Returns the first block of the current function
  const BasicBlock* first_block() const;

  /// Returns the first block of the current function
  BasicBlock* first_block();

  /// Returns a vector of all the blocks in the function
  const std::vector<BasicBlock*>& ordered_blocks() const;

  /// Returns a vector of all the blocks in the function
  std::vector<BasicBlock*>& ordered_blocks();

  /// Returns a list of all the cfg constructs in the function
  const std::list<Construct>& constructs() const;

  /// Returns a list of all the cfg constructs in the function
  std::list<Construct>& constructs();

  /// Returns the number of blocks in the current function being parsed
  size_t block_count() const;

  /// Returns the id of the funciton
  uint32_t id() const { return id_; }

  /// Returns the number of blocks in the current function being parsed
  size_t undefined_block_count() const;
  const std::unordered_set<uint32_t>& undefined_blocks() const {
    return undefined_blocks_;
  }

  /// Returns the block that is currently being parsed in the binary
  BasicBlock* current_block();

  /// Returns the block that is currently being parsed in the binary
  const BasicBlock* current_block() const;

  // For dominance calculations, we want to analyze all the
  // blocks in the function, even in degenerate control flow cases
  // including unreachable blocks.  We therefore make an "augmented CFG"
  // which is the same as the ordinary CFG but adds:
  //  - A pseudo-entry node.
  //  - A pseudo-exit node.
  //  - A minimal set of edges so that a forward traversal from the
  //    pseudo-entry node will visit all nodes.
  //  - A minimal set of edges so that a backward traversal from the
  //    pseudo-exit node will visit all nodes.
  // In particular, the pseudo-entry node is the unique source of the
  // augmented CFG, and the psueo-exit node is the unique sink of the
  // augmented CFG.

  /// Returns the pseudo exit block
  BasicBlock* pseudo_entry_block() { return &pseudo_entry_block_; }

  /// Returns the pseudo exit block
  const BasicBlock* pseudo_entry_block() const { return &pseudo_entry_block_; }

  /// Returns the pseudo exit block
  BasicBlock* pseudo_exit_block() { return &pseudo_exit_block_; }

  /// Returns the pseudo exit block
  const BasicBlock* pseudo_exit_block() const { return &pseudo_exit_block_; }

  using GetBlocksFunction =
      std::function<const std::vector<BasicBlock*>*(const BasicBlock*)>;
  /// Returns the block successors function for the augmented CFG.
  GetBlocksFunction AugmentedCFGSuccessorsFunction() const;
  /// Like AugmentedCFGSuccessorsFunction, but also includes a forward edge from
  /// a loop header block to its continue target, if they are different blocks.
  GetBlocksFunction AugmentedCFGSuccessorsFunctionIncludingHeaderToContinueEdge() const;
  /// Returns the block predecessors function for the augmented CFG.
  GetBlocksFunction AugmentedCFGPredecessorsFunction() const;

  /// Prints a GraphViz digraph of the CFG of the current funciton
  void PrintDotGraph() const;

  /// Prints a directed graph of the CFG of the current funciton
  void PrintBlocks() const;

 private:
  // Computes the representation of the augmented CFG.
  // Populates augmented_successors_map_ and augmented_predecessors_map_.
  void ComputeAugmentedCFG();

  // Adds a copy of the given Construct, and tracks it by its entry block.
  // Returns a reference to the stored construct.
  Construct& AddConstruct(const Construct& new_construct);

  // Returns a reference to the construct corresponding to the given entry
  // block.
  Construct& FindConstructForEntryBlock(const BasicBlock* entry_block);

  /// The result id of the OpLabel that defined this block
  uint32_t id_;

  /// The type of the function
  uint32_t function_type_id_;

  /// The type of the return value
  uint32_t result_type_id_;

  /// The control fo the funciton
  SpvFunctionControlMask function_control_;

  /// The type of declaration of each function
  FunctionDecl declaration_type_;

  // Have we finished parsing this function?
  bool end_has_been_registered_;

  /// The blocks in the function mapped by block ID
  std::unordered_map<uint32_t, BasicBlock> blocks_;

  /// A list of blocks in the order they appeared in the binary
  std::vector<BasicBlock*> ordered_blocks_;

  /// Blocks which are forward referenced by blocks but not defined
  std::unordered_set<uint32_t> undefined_blocks_;

  /// The block that is currently being parsed
  BasicBlock* current_block_;

  /// A pseudo entry node used in dominance analysis.
  /// After the function end has been registered, the successor list of the
  /// pseudo entry node is the minimal set of nodes such that all nodes in the
  /// CFG can be reached by following successor lists.  That is, the successors
  /// will be:
  ///   - Any basic block without predecessors.  This includes the entry
  ///     block to the function.
  ///   - A single node from each otherwise unreachable cycle in the CFG, if
  ///     such cycles exist.
  /// The pseudo entry node does not appear in the predecessor or successor
  /// list of any ordinary block.
  /// It has no predecessors.
  /// It has Id 0.
  BasicBlock pseudo_entry_block_;

  /// A pseudo exit block used in dominance analysis.
  /// After the function end has been registered, the predecessor list of the
  /// pseudo exit node is the minimal set of nodes such that all nodes in the
  /// CFG can be reached by following predecessor lists.  That is, the
  /// predecessors will be:
  ///   - Any basic block without successors.  This includes any basic block
  ///     ending with an OpReturn, OpReturnValue or similar instructions.
  ///   - A single node from each otherwise unreachable cycle in the CFG, if
  ///     such cycles exist.
  /// The pseudo exit node does not appear in the predecessor or successor
  /// list of any ordinary block.
  /// It has no successors.
  BasicBlock pseudo_exit_block_;

  // Maps a block to its successors in the augmented CFG, if that set is
  // different from its successors in the ordinary CFG.
  std::unordered_map<const BasicBlock*, std::vector<BasicBlock*>>
      augmented_successors_map_;
  // Maps a block to its predecessors in the augmented CFG, if that set is
  // different from its predecessors in the ordinary CFG.
  std::unordered_map<const BasicBlock*, std::vector<BasicBlock*>>
      augmented_predecessors_map_;

  // Maps a structured loop header to its CFG successors and also its
  // continue target if that continue target is not the loop header
  // itself. This might have duplicates.
  std::unordered_map<const BasicBlock*, std::vector<BasicBlock*>>
      loop_header_successors_plus_continue_target_map_;

  /// The constructs that are available in this function
  std::list<Construct> cfg_constructs_;

  /// The variable IDs of the functions
  std::vector<uint32_t> variable_ids_;

  /// The function parameter ids of the functions
  std::vector<uint32_t> parameter_ids_;

  /// Maps a construct's entry block to the construct.
  std::unordered_map<const BasicBlock*, Construct*> entry_block_to_construct_;
};

}  /// namespace libspirv

#endif  /// LIBSPIRV_VAL_FUNCTION_H_
