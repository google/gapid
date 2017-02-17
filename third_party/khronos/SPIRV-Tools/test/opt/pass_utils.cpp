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

#include "pass_utils.h"

#include <algorithm>
#include <sstream>

namespace {

// Well, this is another place requiring the knowledge of the grammar and can be
// stale when SPIR-V is updated. It would be nice to automatically generate
// this, but the cost is just too high.

const char* kDebugOpcodes[] = {
    // clang-format off
    "OpSourceContinued", "OpSource", "OpSourceExtension",
    "OpName", "OpMemberName", "OpString",
    "OpLine", "OpNoLine", "OpModuleProcessed"
    // clang-format on
};

}  // anonymous namespace

namespace spvtools {

bool FindAndReplace(std::string* process_str, const std::string find_str,
                    const std::string replace_str) {
  if (process_str->empty() || find_str.empty()) {
    return false;
  }
  bool replaced = false;
  // Note this algorithm has quadratic time complexity. It is OK for test cases
  // with short strings, but might not fit in other contexts.
  for (size_t pos = process_str->find(find_str, 0); pos != std::string::npos;
       pos = process_str->find(find_str, pos)) {
    process_str->replace(pos, find_str.length(), replace_str);
    pos += replace_str.length();
    replaced = true;
  }
  return replaced;
}

bool ContainsDebugOpcode(const char* inst) {
  return std::any_of(std::begin(kDebugOpcodes), std::end(kDebugOpcodes),
                     [inst](const char* op) {
                       return std::string(inst).find(op) != std::string::npos;
                     });
}

std::string SelectiveJoin(const std::vector<const char*>& strings,
                          const std::function<bool(const char*)>& skip_dictator,
                          char delimiter) {
  std::ostringstream oss;
  for (const auto* str : strings) {
    if (!skip_dictator(str)) oss << str << delimiter;
  }
  return oss.str();
}

std::string JoinAllInsts(const std::vector<const char*>& insts) {
  return SelectiveJoin(insts, [](const char*) { return false; });
}

std::string JoinNonDebugInsts(const std::vector<const char*>& insts) {
  return SelectiveJoin(
      insts, [](const char* inst) { return ContainsDebugOpcode(inst); });
}

}  // namespace spvtools
