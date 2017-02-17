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

include_directories(${LLVM_MAIN_SRC_DIR}/lib/Target/X86)
include_directories(${LLVM_LIBRARY_DIR}/Target/X86)

add_llvm_library(InterceptorX86Target
  target_x86.cc)

add_dependencies(InterceptorX86Target
  LLVMX86AsmPrinter
  LLVMX86AsmParser
  LLVMX86Desc
  LLVMX86Disassembler
  LLVMX86Info)
