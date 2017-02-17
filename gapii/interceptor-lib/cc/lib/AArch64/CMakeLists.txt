# Copyright (C) 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include_directories(${LLVM_MAIN_SRC_DIR}/lib/Target/AArch64)
include_directories(${LLVM_LIBRARY_DIR}/Target/AArch64)

add_llvm_library(InterceptorAArch64Target
  target_aarch64.cc)

add_dependencies(InterceptorAArch64Target
  LLVMAArch64AsmPrinter
  LLVMAArch64AsmParser
  LLVMAArch64Desc
  LLVMAArch64Disassembler
  LLVMAArch64Info)
