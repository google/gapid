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

#include "unify_const_pass.h"

#include <unordered_map>
#include <utility>

#include "def_use_manager.h"
#include "make_unique.h"

namespace spvtools {
namespace opt {

namespace {

// The trie that stores a bunch of result ids and, for a given instruction,
// searches the result id that has been defined with the same opcode, type and
// operands.
class ResultIdTrie {
 public:
  ResultIdTrie() : root_(new Node) {}

  // For a given instruction, extracts its opcode, type id and operand words
  // as an array of keys, looks up the trie to find a result id which is stored
  // with the same opcode, type id and operand words. If none of such result id
  // is found, creates a trie node with those keys, stores the instruction's
  // result id and returns that result id. If an existing result id is found,
  // returns the existing result id.
  uint32_t LookupEquivalentResultFor(const ir::Instruction& inst) {
    auto keys = GetLookUpKeys(inst);
    auto* node = root_.get();
    for (uint32_t key : keys) {
      node = node->GetOrCreateTrieNodeFor(key);
    }
    if (node->result_id() == 0) {
      node->SetResultId(inst.result_id());
    }
    return node->result_id();
  }

 private:
  // The trie node to store result ids.
  class Node {
   public:
    using TrieNodeMap = std::unordered_map<uint32_t, std::unique_ptr<Node>>;

    Node() : result_id_(0), next_() {}
    uint32_t result_id() const { return result_id_; }

    // Sets the result id stored in this node.
    void SetResultId(uint32_t id) { result_id_ = id; }

    // Searches for the child trie node with the given key. If the node is
    // found, returns that node. Otherwise creates an empty child node with
    // that key and returns that newly created node.
    Node* GetOrCreateTrieNodeFor(uint32_t key) {
      auto iter = next_.find(key);
      if (iter == next_.end()) {
        // insert a new node and return the node.
        return next_.insert(std::make_pair(key, MakeUnique<Node>()))
            .first->second.get();
      }
      return iter->second.get();
    }

   private:
    // The result id stored in this node. 0 means this node is empty.
    uint32_t result_id_;
    // The mapping from the keys to the child nodes of this node.
    TrieNodeMap next_;
  };

  // Returns a vector of the opcode followed by the words in the raw SPIR-V
  // instruction encoding but without the result id.
  std::vector<uint32_t> GetLookUpKeys(const ir::Instruction& inst) {
    std::vector<uint32_t> keys;
    // Need to use the opcode, otherwise there might be a conflict with the
    // following case when <op>'s binary value equals xx's id:
    //  OpSpecConstantOp tt <op> yy zz
    //  OpSpecConstantComposite tt xx yy zz;
    keys.push_back(static_cast<uint32_t>(inst.opcode()));
    for (const auto& operand : inst) {
      if (operand.type == SPV_OPERAND_TYPE_RESULT_ID) continue;
      keys.insert(keys.end(), operand.words.cbegin(), operand.words.cend());
    }
    return keys;
  }

  std::unique_ptr<Node> root_;  // The root node of the trie.
};
}  // anonymous namespace

bool UnifyConstantPass::Process(ir::Module* module) {
  bool modified = false;
  ResultIdTrie defined_constants;
  analysis::DefUseManager def_use_mgr(module);

  for (ir::Instruction& inst : module->types_values()) {
    // Do not handle the instruction when there are decorations upon the result
    // id.
    if (def_use_mgr.GetAnnotations(inst.result_id()).size() != 0) {
      continue;
    }

    // The overall algorithm is to store the result ids of all the eligible
    // constants encountered so far in a trie. For a constant defining
    // instruction under consideration, use its opcode, result type id and
    // words in operands as an array of keys to lookup the trie. If a result id
    // can be found for that array of keys, a constant with exactly the same
    // value must has been defined before, the constant under processing
    // should be replaced by the constant previously defined. If no such result
    // id can be found for that array of keys, this must be the first time a
    // constant with its value be defined, we then create a new trie node to
    // store the result id with the keys. When replacing a duplicated constant
    // with a previously defined constant, all the uses of the duplicated
    // constant, which must be placed after the duplicated constant defining
    // instruction, will be updated. This way, the descendants of the
    // previously defined constant and the duplicated constant will both refer
    // to the previously defined constant. So that the operand ids which are
    // used in key arrays will be the ids of the unified constants, when
    // processing is up to a descendant. This makes comparing the key array
    // always valid for judging duplication.
    switch (inst.opcode()) {
      case SpvOp::SpvOpConstantTrue:
      case SpvOp::SpvOpConstantFalse:
      case SpvOp::SpvOpConstant:
      case SpvOp::SpvOpConstantNull:
      case SpvOp::SpvOpConstantSampler:
      case SpvOp::SpvOpConstantComposite:
      // Only spec constants defined with OpSpecConstantOp and
      // OpSpecConstantComposite should be processed in this pass. Spec
      // constants defined with OpSpecConstant{|True|False} are decorated with
      // 'SpecId' decoration and all of them should be treated as unique.
      // 'SpecId' is not applicable to SpecConstants defined with
      // OpSpecConstant{Op|Composite}, their values are not necessary to be
      // unique. When all the operands/compoents are the same between two
      // OpSpecConstant{Op|Composite} results, their result values must be the
      // same so are unifiable.
      case SpvOp::SpvOpSpecConstantOp:
      case SpvOp::SpvOpSpecConstantComposite: {
        uint32_t id = defined_constants.LookupEquivalentResultFor(inst);
        if (id != inst.result_id()) {
          // The constant is a duplicated one, use the cached constant to
          // replace the uses of this duplicated one, then turn it to nop.
          def_use_mgr.ReplaceAllUsesWith(inst.result_id(), id);
          def_use_mgr.KillInst(&inst);
          modified = true;
        }
        break;
      }
      default:
        break;
    }
  }
  return modified;
}

}  // opt
}  // namespace spvtools
