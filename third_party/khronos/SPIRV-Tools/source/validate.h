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

#ifndef LIBSPIRV_VALIDATE_H_
#define LIBSPIRV_VALIDATE_H_

#include <functional>
#include <utility>
#include <vector>

#include "instruction.h"
#include "spirv-tools/libspirv.h"
#include "table.h"

namespace libspirv {

class ValidationState_t;
class BasicBlock;

/// A function that returns a vector of BasicBlocks given a BasicBlock. Used to
/// get the successor and predecessor nodes of a CFG block
using get_blocks_func =
    std::function<const std::vector<BasicBlock*>*(const BasicBlock*)>;

/// @brief Depth first traversal starting from the \p entry BasicBlock
///
/// This function performs a depth first traversal from the \p entry
/// BasicBlock and calls the pre/postorder functions when it needs to process
/// the node in pre order, post order. It also calls the backedge function
/// when a back edge is encountered.
///
/// @param[in] entry      The root BasicBlock of a CFG
/// @param[in] successor_func  A function which will return a pointer to the
///                            successor nodes
/// @param[in] preorder   A function that will be called for every block in a
///                       CFG following preorder traversal semantics
/// @param[in] postorder  A function that will be called for every block in a
///                       CFG following postorder traversal semantics
/// @param[in] backedge   A function that will be called when a backedge is
///                       encountered during a traversal
/// NOTE: The @p successor_func and predecessor_func each return a pointer to a
/// collection such that iterators to that collection remain valid for the
/// lifetime of the algorithm.
void DepthFirstTraversal(
    const BasicBlock* entry, get_blocks_func successor_func,
    std::function<void(const BasicBlock*)> preorder,
    std::function<void(const BasicBlock*)> postorder,
    std::function<void(const BasicBlock*, const BasicBlock*)> backedge);

/// @brief Calculates dominator edges for a set of blocks
///
/// Computes dominators using the algorithm of Cooper, Harvey, and Kennedy
/// "A Simple, Fast Dominance Algorithm", 2001.
///
/// The algorithm assumes there is a unique root node (a node without
/// predecessors), and it is therefore at the end of the postorder vector.
///
/// This function calculates the dominator edges for a set of blocks in the CFG.
/// Uses the dominator algorithm by Cooper et al.
///
/// @param[in] postorder        A vector of blocks in post order traversal order
///                             in a CFG
/// @param[in] predecessor_func Function used to get the predecessor nodes of a
///                             block
///
/// @return the dominator tree of the graph, as a vector of pairs of nodes.
/// The first node in the pair is a node in the graph. The second node in the
/// pair is its immediate dominator in the sense of Cooper et.al., where a block
/// without predecessors (such as the root node) is its own immediate dominator.
std::vector<std::pair<BasicBlock*, BasicBlock*>> CalculateDominators(
    const std::vector<const BasicBlock*>& postorder,
    get_blocks_func predecessor_func);

/// @brief Performs the Control Flow Graph checks
///
/// @param[in] _ the validation state of the module
///
/// @return SPV_SUCCESS if no errors are found. SPV_ERROR_INVALID_CFG otherwise
spv_result_t PerformCfgChecks(ValidationState_t& _);

/// @brief Updates the use vectors of all instructions that can be referenced
///
/// This function will update the vector which define where an instruction was
/// referenced in the binary.
///
/// @param[in] _ the validation state of the module
///
/// @return SPV_SUCCESS if no errors are found.
spv_result_t UpdateIdUse(ValidationState_t& _);

/// @brief This function checks all ID definitions dominate their use in the
/// CFG.
///
/// This function will iterate over all ID definitions that are defined in the
/// functions of a module and make sure that the definitions appear in a
/// block that dominates their use.
///
/// @param[in] _ the validation state of the module
///
/// @return SPV_SUCCESS if no errors are found. SPV_ERROR_INVALID_ID otherwise
spv_result_t CheckIdDefinitionDominateUse(const ValidationState_t& _);

/// @brief Updates the immediate dominator for each of the block edges
///
/// Updates the immediate dominator of the blocks for each of the edges
/// provided by the @p dom_edges parameter
///
/// @param[in,out] dom_edges The edges of the dominator tree
/// @param[in] set_func This function will be called to updated the Immediate
///                     dominator
void UpdateImmediateDominators(
    const std::vector<std::pair<BasicBlock*, BasicBlock*>>& dom_edges,
    std::function<void(BasicBlock*, BasicBlock*)> set_func);

/// @brief Prints all of the dominators of a BasicBlock
///
/// @param[in] block The dominators of this block will be printed
void printDominatorList(BasicBlock& block);

/// Performs logical layout validation as described in section 2.4 of the SPIR-V
/// spec.
spv_result_t ModuleLayoutPass(ValidationState_t& _,
                              const spv_parsed_instruction_t* inst);

/// Performs Control Flow Graph validation of a module
spv_result_t CfgPass(ValidationState_t& _,
                     const spv_parsed_instruction_t* inst);

/// Performs Id and SSA validation of a module
spv_result_t IdPass(ValidationState_t& _, const spv_parsed_instruction_t* inst);

/// Performs instruction validation.
spv_result_t InstructionPass(ValidationState_t& _,
                             const spv_parsed_instruction_t* inst);

}  // namespace libspirv

/// @brief Validate the ID usage of the instruction stream
///
/// @param[in] pInsts stream of instructions
/// @param[in] instCount number of instructions
/// @param[in] opcodeTable table of specified Opcodes
/// @param[in] operandTable table of specified operands
/// @param[in] usedefs use-def info from module parsing
/// @param[in,out] position current position in the stream
/// @param[out] pDiag contains diagnostic on failure
///
/// @return result code
spv_result_t spvValidateInstructionIDs(const spv_instruction_t* pInsts,
                                       const uint64_t instCount,
                                       const spv_opcode_table opcodeTable,
                                       const spv_operand_table operandTable,
                                       const spv_ext_inst_table extInstTable,
                                       const libspirv::ValidationState_t& state,
                                       spv_position position,
                                       spv_diagnostic* pDiag);

/// @brief Validate the ID's within a SPIR-V binary
///
/// @param[in] pInstructions array of instructions
/// @param[in] count number of elements in instruction array
/// @param[in] bound the binary header
/// @param[in] opcodeTable table of specified Opcodes
/// @param[in] operandTable table of specified operands
/// @param[in,out] position current word in the binary
/// @param[out] pDiagnostic contains diagnostic on failure
///
/// @return result code
spv_result_t spvValidateIDs(const spv_instruction_t* pInstructions,
                            const uint64_t count, const uint32_t bound,
                            const spv_opcode_table opcodeTable,
                            const spv_operand_table operandTable,
                            const spv_ext_inst_table extInstTable,
                            spv_position position, spv_diagnostic* pDiagnostic);

#define spvCheckReturn(expression) \
  if (spv_result_t error = (expression)) return error;

#endif  // LIBSPIRV_VALIDATE_H_
