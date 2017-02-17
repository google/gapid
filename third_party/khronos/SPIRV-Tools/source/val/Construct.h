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

#ifndef LIBSPIRV_VAL_CONSTRUCT_H_
#define LIBSPIRV_VAL_CONSTRUCT_H_

#include <cstdint>
#include <vector>

namespace libspirv {

enum class ConstructType {
  kNone,
  /// The set of blocks dominated by a selection header, minus the set of blocks
  /// dominated by the header's merge block
  kSelection,
  /// The set of blocks dominated by an OpLoopMerge's Continue Target and post
  /// dominated by the corresponding back
  kContinue,
  ///  The set of blocks dominated by a loop header, minus the set of blocks
  ///  dominated by the loop's merge block, minus the loop's corresponding
  ///  continue construct
  kLoop,
  ///  The set of blocks dominated by an OpSwitch's Target or Default, minus the
  ///  set of blocks dominated by the OpSwitch's merge block (this construct is
  ///  only defined for those OpSwitch Target or Default that are not equal to
  ///  the OpSwitch's corresponding merge block)
  kCase
};

class BasicBlock;

/// @brief This class tracks the CFG constructs as defined in the SPIR-V spec
class Construct {
 public:
  Construct(ConstructType type, BasicBlock* dominator,
            BasicBlock* exit = nullptr,
            std::vector<Construct*> constructs = {});

  /// Returns the type of the construct
  ConstructType type() const;

  const std::vector<Construct*>& corresponding_constructs() const;
  std::vector<Construct*>& corresponding_constructs();
  void set_corresponding_constructs(std::vector<Construct*> constructs);

  /// Returns the dominator block of the construct.
  ///
  /// This is usually the header block or the first block of the construct.
  const BasicBlock* entry_block() const;

  /// Returns the dominator block of the construct.
  ///
  /// This is usually the header block or the first block of the construct.
  BasicBlock* entry_block();

  /// Returns the exit block of the construct.
  ///
  /// For a continue construct it is  the backedge block of the corresponding
  /// loop construct. For the case  construct it is the block that branches to
  /// the OpSwitch merge block or  other case blocks. Otherwise it is the merge
  /// block of the corresponding  header block
  const BasicBlock* exit_block() const;

  /// Returns the exit block of the construct.
  ///
  /// For a continue construct it is  the backedge block of the corresponding
  /// loop construct. For the case  construct it is the block that branches to
  /// the OpSwitch merge block or  other case blocks. Otherwise it is the merge
  /// block of the corresponding  header block
  BasicBlock* exit_block();

  /// Sets the exit block for this construct. This is useful for continue
  /// constructs which do not know the back-edge block during construction
  void set_exit(BasicBlock* exit_block);

 private:
  /// The type of the construct
  ConstructType type_;

  /// These are the constructs that are related to this construct. These
  /// constructs can be the continue construct, for the corresponding loop
  /// construct, the case construct that are part of the same OpSwitch
  /// instruction
  ///
  /// Here is a table that describes what constructs are included in
  /// @p corresponding_constructs_
  /// | this construct | corresponding construct          |
  /// |----------------|----------------------------------|
  /// | loop           | continue                         |
  /// | continue       | loop                             |
  /// | case           | other cases in the same OpSwitch |
  ///
  /// kContinue and kLoop constructs will always have corresponding
  /// constructs even if they are represented by the same block
  std::vector<Construct*> corresponding_constructs_;

  /// @brief Dominator block for the construct
  ///
  /// The dominator block for the construct. Depending on the construct this may
  /// be a selection header, a continue target of a loop, a loop header or a
  /// Target or Default block of a switch
  BasicBlock* entry_block_;

  /// @brief Exiting block for the construct
  ///
  /// The exit block for the construct. This can be a merge block for the loop
  /// and selection constructs, a back-edge block for a continue construct, or
  /// the branching block for the case construct
  BasicBlock* exit_block_;
};

}  /// namespace libspirv

#endif  /// LIBSPIRV_VAL_CONSTRUCT_H_
